package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func inspectionFixture(t *testing.T) (*Service, projectRecord, PipelineDefinition) {
	t.Helper()
	s, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	root := t.TempDir()
	for name, value := range map[string]string{
		"pipeline.go": entrySource, "go.mod": "module example.com/analysis\ngo 1.26\n",
		".gobble-runtime.json":    `{"format":1,"image":"sha256:` + strings.Repeat("a", 64) + `","daemon":"test-daemon"}`,
		".gobble-inspection.json": `{"schemaVersion":1,"files":["pipeline.go"],"sample":""}`,
		"private.txt":             "not declared",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.store.registerProject("req_project", root, "Analysis")
	if err != nil {
		t.Fatal(err)
	}
	pipeline, err := s.store.registerPipeline(p.ProjectID, p.RootResourceID, "req_register", "Analysis")
	if err != nil {
		t.Fatal(err)
	}
	project, err := s.store.project(p.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	return s, project, pipeline
}

func inspectionCommand(run func(context.Context, []string) ([]byte, error)) commandRunner {
	return func(ctx context.Context, args, env []string) ([]byte, error) {
		switch args[0] {
		case "context":
			return []byte("unix:///fixture/docker.sock"), nil
		case "info":
			return []byte("test-daemon linux"), nil
		case "image":
			return []byte("sha256:" + strings.Repeat("a", 64) + " linux/amd64"), nil
		case "rm":
			return nil, nil
		default:
			return run(ctx, args)
		}
	}
}

var emptyFlow = []byte(`{"schemaVersion":1,"name":"analysis","inputs":[],"steps":[],"connections":[]}`)

func awaitInspection(t *testing.T, s *Service, p PipelineDefinition) pipelineInspection {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		value, err := s.pipelineInspection(p.ProjectID, p.PipelineID)
		if err != nil {
			t.Fatal(err)
		}
		if value.State != "checking" {
			return value
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("inspection did not finish")
	return pipelineInspection{}
}

func TestInspectionRetentionAndRuntimeBoundary(t *testing.T) {
	s, project, pipeline := inspectionFixture(t)
	var calls atomic.Int32
	s.runtime.command = inspectionCommand(func(ctx context.Context, args []string) ([]byte, error) {
		calls.Add(1)
		for _, required := range []string{"--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=128", "--memory=768m", "--cpus=1", "--pull=never", "10001:10001", "/usr/bin/timeout", "110s", "--signal=KILL", "GOWORK=off", "GOPROXY=off", "/tmp:rw,exec,nosuid,nodev,size=512m,mode=1777"} {
			found := false
			for _, arg := range args {
				if arg == required {
					found = true
				}
			}
			if !found {
				t.Errorf("missing evaluator restriction %s", required)
			}
		}
		for _, arg := range args {
			if strings.Contains(arg, project.Root) {
				t.Error("mutable research Project mounted")
			}
		}
		return emptyFlow, nil
	})
	first, err := s.checkPipeline(project.ProjectID, pipeline.PipelineID, "req_check")
	if err != nil || first.State != "checking" {
		t.Fatal(first, err)
	}
	value := awaitInspection(t, s, pipeline)
	if value.State != "ready" || value.Artifact == nil {
		t.Fatal(value)
	}
	if _, err = s.checkPipeline(project.ProjectID, pipeline.PipelineID, "req_check"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("idempotent check evaluated twice")
	}
	dir := filepath.Join(s.inspectionDirectory(pipeline.PipelineID), "candidates", "req_check")
	if _, err := os.Stat(filepath.Join(dir, "project/private.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("undeclared file retained")
	}
	retained, _ := os.ReadFile(filepath.Join(dir, "project/pipeline.go"))
	if string(retained) != entrySource {
		t.Fatal("source bytes changed")
	}
	if len(s.store.data.Runs) != 0 {
		t.Fatal("inspection created a Run")
	}
	old := value.Artifact.ArtifactID
	os.WriteFile(filepath.Join(project.Root, "pipeline.go"), []byte(entrySource+"\n// new version\n"), 0o600)
	if _, err = s.checkPipeline(project.ProjectID, pipeline.PipelineID, "req_second"); err != nil {
		t.Fatal(err)
	}
	value = awaitInspection(t, s, pipeline)
	if value.State != "ready" || value.Artifact.ArtifactID == old {
		t.Fatal("changed source reused artifact identity")
	}
	retainedAfter, err := os.ReadFile(filepath.Join(dir, "project/pipeline.go"))
	if err != nil || string(retainedAfter) != entrySource {
		t.Fatal("historical source changed")
	}
}

func TestInspectionFailureCancellationAndStaleResults(t *testing.T) {
	s, project, pipeline := inspectionFixture(t)
	s.runtime.command = inspectionCommand(func(context.Context, []string) ([]byte, error) { return emptyFlow, nil })
	_, _ = s.checkPipeline(project.ProjectID, pipeline.PipelineID, "req_good")
	good := awaitInspection(t, s, pipeline)
	if good.Artifact == nil {
		t.Fatal(good)
	}
	for _, mode := range []string{"failure", "changed", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			s.runtime.command = inspectionCommand(func(ctx context.Context, args []string) ([]byte, error) {
				close(entered)
				select {
				case <-release:
				case <-ctx.Done():
				}
				if mode == "failure" {
					return nil, errors.New("invalid pipeline")
				}
				return emptyFlow, nil
			})
			job := "req_" + mode
			if _, err := s.checkPipeline(project.ProjectID, pipeline.PipelineID, job); err != nil {
				t.Fatal(err)
			}
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("worker did not enter")
			}
			if mode == "changed" {
				os.WriteFile(filepath.Join(project.Root, "pipeline.go"), []byte(entrySource+"\n// changed while checking"), 0o600)
			}
			if mode == "cancel" {
				value, err := s.cancelPipeline(project.ProjectID, pipeline.PipelineID, job)
				if err != nil || value.State != "cancelled" {
					t.Fatal(value, err)
				}
			}
			close(release)
			s.workers.Wait()
			value := awaitInspection(t, s, pipeline)
			if value.State == "ready" || value.Artifact == nil || value.Artifact.ArtifactID != good.Artifact.ArtifactID {
				t.Fatal("late/invalid candidate replaced current flow", value)
			}
		})
	}
	_, err := s.cancelPipeline(project.ProjectID, pipeline.PipelineID, "req_foreign")
	wantCode(t, err, "stale_revision")
	_, err = s.pipelineInspection("prj_foreign", pipeline.PipelineID)
	wantCode(t, err, "not_found")
}

func TestInspectionSetupCannotEscapeProject(t *testing.T) {
	s, project, pipeline := inspectionFixture(t)
	s.runtime.command = inspectionCommand(func(context.Context, []string) ([]byte, error) {
		t.Error("invalid setup executed")
		return emptyFlow, nil
	})
	for i, files := range [][]string{{"../outside"}, {"missing"}, {"pipeline.go", "../private"}} {
		raw, _ := json.Marshal(inspectionSetup{1, files, ""})
		os.WriteFile(filepath.Join(project.Root, ".gobble-inspection.json"), raw, 0o600)
		_, err := s.checkPipeline(project.ProjectID, pipeline.PipelineID, []string{"req_a", "req_b", "req_c"}[i])
		if err != nil {
			t.Fatal(err)
		}
		if value := awaitInspection(t, s, pipeline); value.State != "failed" || value.Artifact != nil {
			t.Fatal(value)
		}
	}
}

func TestPipelineImportResolvesAliasesAndRejectsOutsideFolders(t *testing.T) {
	s, project, pipeline := inspectionFixture(t)
	alias := filepath.Join(t.TempDir(), "analysis")
	if err := os.Symlink(project.Root, alias); err != nil {
		t.Fatal(err)
	}
	got, err := s.store.importPipeline(project.ProjectID, "req_alias", alias)
	if err != nil || got.PipelineID != pipeline.PipelineID {
		t.Fatal(got, err)
	}
	_, err = s.store.importPipeline(project.ProjectID, "req_outside", t.TempDir())
	wantCode(t, err, "invalid_request")
}

// Opt-in native-service + pinned-Linux integration. The caller supplies only an
// owned qualification Project, never a real research directory.
func TestLivePipelineInspection(t *testing.T) {
	root := os.Getenv("GOBBLE_FLOW_QUALIFICATION_PROJECT")
	if root == "" {
		t.Skip("owned pinned-runtime fixture not supplied")
	}
	s, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p, err := s.store.registerProject("req_project", root, "Flow review")
	if err != nil {
		t.Fatal(err)
	}
	pipeline, err := s.store.importPipeline(p.ProjectID, "req_import", filepath.Join(root, "rnaseq"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.checkPipeline(p.ProjectID, pipeline.PipelineID, "req_live"); err != nil {
		t.Fatal(err)
	}
	s.workers.Wait()
	value, err := s.pipelineInspection(p.ProjectID, pipeline.PipelineID)
	if err != nil || value.State != "ready" {
		t.Fatal(value, err)
	}
	var flow struct {
		Steps       []any `json:"steps"`
		Connections []any `json:"connections"`
	}
	json.Unmarshal(value.Artifact.Flow, &flow)
	if len(flow.Steps) != 5 || len(flow.Connections) != 7 || len(s.store.data.Runs) != 0 {
		t.Fatal("unexpected real flow or Run side effect", len(flow.Steps), len(flow.Connections))
	}
	if output := os.Getenv("GOBBLE_FLOW_QUALIFICATION_OUTPUT"); output != "" {
		if err := os.WriteFile(output, value.Artifact.Flow, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("Real Gobble flow: %d steps, %d exact connections, no Run; artifact %s", len(flow.Steps), len(flow.Connections), value.Artifact.ArtifactID)
}

func TestLivePipelineModuleSettings(t *testing.T) {
	root := os.Getenv("GOBBLE_FLOW_QUALIFICATION_PROJECT")
	if root == "" {
		t.Skip("owned pinned-runtime fixture not supplied")
	}
	s, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p, err := s.store.registerProject("req_project", root, "Module settings review")
	if err != nil {
		t.Fatal(err)
	}
	pipeline, err := s.store.importPipeline(p.ProjectID, "req_import", filepath.Join(root, "trim-review"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.checkPipeline(p.ProjectID, pipeline.PipelineID, "req_live"); err != nil {
		t.Fatal(err)
	}
	s.workers.Wait()
	value, err := s.pipelineInspection(p.ProjectID, pipeline.PipelineID)
	if err != nil || value.State != "ready" {
		t.Fatal(value, err)
	}
	var flow struct {
		SchemaVersion int
		Steps         []struct {
			Settings []struct {
				Key   string
				Value *int
			}
		}
	}
	if err = json.Unmarshal(value.Artifact.Flow, &flow); err != nil {
		t.Fatal(err)
	}
	if flow.SchemaVersion != 2 || len(flow.Steps) != 2 || len(flow.Steps[0].Settings) != 2 || len(s.store.data.Runs) != 0 {
		t.Fatal("module metadata missing or execution side effect")
	}
	settings := flow.Steps[0].Settings
	if settings[0].Key != "quality" || settings[0].Value == nil || *settings[0].Value != 25 || settings[1].Key != "length" || settings[1].Value == nil || *settings[1].Value != 40 {
		t.Fatal("typed option values lost")
	}
	if output := os.Getenv("GOBBLE_PIPELINE_SETTINGS_OUTPUT"); output != "" {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(output, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("Real Trim Galore metadata: quality 25 Phred, minimum length 40 bp; no Run.")
}
