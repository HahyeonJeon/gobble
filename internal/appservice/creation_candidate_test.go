package appservice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

func creationFixture(t *testing.T, run func(context.Context, []string) ([]byte, error)) (*Service, Project, creationScope) {
	t.Helper()
	s, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if s.ctx.Err() == nil {
			if err := s.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "reads.fastq.gz"), []byte("metadata only"))
	p := testProject(t, s, root)
	file := findFile(t, s, p, "reads.fastq.gz")
	d, err := s.store.createDraft(p.ProjectID, createDraftInput{"req_draft", "Read quality"})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_input", 1, "Read quality", file, "single-end"})
	if err != nil {
		t.Fatal(err)
	}
	scaffold, err := pipelinereview.ExportCreationScaffold("module github.com/HahyeonJeon/gobble\n\ngo 1.26\n", "sum\n")
	if err != nil {
		t.Fatal(err)
	}
	s.runtime.command = inspectionCommand(func(ctx context.Context, args []string) ([]byte, error) {
		if args[len(args)-1] == "creation-scaffold" {
			return json.Marshal(scaffold)
		}
		return run(ctx, args)
	})
	b := RuntimeBinding{"unix:///fixture/docker.sock", "test-daemon", "sha256:" + strings.Repeat("a", 64), "linux/amd64", "/gobble/project", "/gobble/project/work"}
	if _, err = s.bindCreationRuntime(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	scope, err := s.creationAuthoring(p.ProjectID, d.DraftID, selected.Generation)
	if err != nil {
		t.Fatal(err)
	}
	return s, p, scope
}
func mockCreationCheck() []byte {
	hash := "sha256:" + strings.Repeat("b", 64)
	steps := []pipelinereview.Step{{ID: "trim", Fingerprint: hash, Residual: hash, Recipe: "trim-galore-v1", Settings: []pipelinereview.Setting{}}, {ID: "qc", Fingerprint: hash, Residual: hash, Recipe: "fastqc-v1", Settings: []pipelinereview.Setting{}}}
	edges := []pipelinereview.Edge{{FromPort: "reads", ToTask: "trim", ToPort: "read1", Wait: []string{pipelinereview.CreationReadPath}}, {FromTask: "trim", FromPort: "trimmed_read1", ToTask: "qc", ToPort: "reads", Wait: []string{"work/trim-galore/sample_trimmed.fq.gz"}}}
	def := pipelinereview.Definition{SchemaVersion: 1, Context: hash, Steps: steps, Edges: edges}
	flow, _ := json.Marshal(map[string]any{"schemaVersion": 2, "name": "fixture", "inputs": []any{map[string]string{"name": "reads", "path": pipelinereview.CreationReadPath}}, "steps": []any{map[string]string{"id": "trim"}, map[string]string{"id": "qc"}}, "connections": edges})
	v := creationCheck{1, pipelinereview.CreationScope, pipelinereview.CreationReadPath, checkedReview{1, flow, def}, []string{}}
	raw, _ := json.Marshal(v)
	return raw
}
func awaitCreation(t *testing.T, s *Service, scope creationScope, id string) creationCandidate {
	t.Helper()
	deadline := time.Now().Add(inspectionTimeout + 10*time.Second)
	for time.Now().Before(deadline) {
		v, err := s.creationCandidate(scope.Draft.ProjectID, scope.Draft.DraftID, id)
		if err != nil {
			t.Fatal(err)
		}
		if v.State != "checking" {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("creation check did not finish")
	return creationCandidate{}
}
func submitCreationFixture(t *testing.T, s *Service, scope creationScope, id string) creationCandidateInput {
	t.Helper()
	input := creationCandidateInput{id, scope.Draft.Generation, scope.ScopeID, "Trim and inspect read quality", scope.Files}
	if _, err := s.submitCreation(scope.Draft.ProjectID, scope.Draft.DraftID, input); err != nil {
		t.Fatal(err)
	}
	return input
}
func TestCreationCandidateRetainsScopeRetryAndHistory(t *testing.T) {
	s, p, scope := creationFixture(t, func(_ context.Context, args []string) ([]byte, error) {
		joined := strings.Join(args, " ")
		for _, flag := range []string{"--network=none", "--read-only", "--pull=never", "creation-review", "./pipeline"} {
			if !strings.Contains(joined, flag) {
				t.Errorf("missing isolation argument %s", flag)
			}
		}
		if strings.Contains(joined, "docker.sock") || strings.Contains(joined, ".fastq.gz,") {
			t.Error("research data/socket mounted")
		}
		return mockCreationCheck(), nil
	})
	input := submitCreationFixture(t, s, scope, "req_candidate")
	v := awaitCreation(t, s, scope, input.RequestID)
	if v.State != "ready" || v.Artifact == nil || v.Artifact.Input != *scope.Draft.Input {
		t.Fatalf("%+v", v)
	}
	retry, err := s.submitCreation(p.ProjectID, scope.Draft.DraftID, input)
	if err != nil || retry.Artifact.ArtifactID != v.Artifact.ArtifactID {
		t.Fatalf("retry %+v %v", retry, err)
	}
	input.Summary = "different"
	_, err = s.submitCreation(p.ProjectID, scope.Draft.DraftID, input)
	wantCode(t, err, "request_conflict")
	other := testProject(t, s, t.TempDir())
	_, err = s.creationCandidate(other.ProjectID, scope.Draft.DraftID, input.RequestID)
	wantCode(t, err, "not_found")
	if len(s.store.data.Pipelines) != 0 || len(s.store.data.Runs) != 0 {
		t.Fatal("check adopted or ran")
	}
	if _, err = s.store.discardDraft(p.ProjectID, scope.Draft.DraftID, discardDraftInput{"req_discard", scope.Draft.Generation}); err != nil {
		t.Fatal(err)
	}
	history, err := s.creationCandidate(p.ProjectID, scope.Draft.DraftID, input.RequestID)
	listed, listErr := s.listCreationCandidates(p.ProjectID, scope.Draft.DraftID)
	if listErr != nil || len(listed) != 1 {
		t.Fatal("candidate history", listErr)
	}
	if err != nil || history.Artifact.ArtifactID != v.Artifact.ArtifactID {
		t.Fatal("historical artifact lost", err)
	}
	// A fresh service restores the same profile-owned runtime and immutable artifact.
	profile := filepath.Dir(s.store.dir)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	history, err = reopened.creationCandidate(p.ProjectID, scope.Draft.DraftID, input.RequestID)
	if err != nil || history.Artifact.ArtifactID != v.Artifact.ArtifactID {
		t.Fatal("restart changed artifact", err)
	}
}
func TestCreationCandidateGenerationDataAndCancellationFences(t *testing.T) {
	for _, action := range []string{"update", "discard", "data", "cancel", "runtime", "source"} {
		t.Run(action, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			s, p, scope := creationFixture(t, func(ctx context.Context, _ []string) ([]byte, error) {
				close(started)
				select {
				case <-release:
					return mockCreationCheck(), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			})
			submitCreationFixture(t, s, scope, "req_candidate")
			<-started
			var err error
			switch action {
			case "update":
				_, err = s.store.updateDraft(p.ProjectID, scope.Draft.DraftID, updateDraftInput{"req_update", scope.Draft.Generation, "changed brief", scope.Draft.Input.ResourceID, "single-end"})
			case "discard":
				_, err = s.store.discardDraft(p.ProjectID, scope.Draft.DraftID, discardDraftInput{"req_discard", scope.Draft.Generation})
			case "data":
				project, _ := s.store.project(p.ProjectID)
				err = os.WriteFile(filepath.Join(project.Root, "reads.fastq.gz"), []byte("changed metadata and size"), 0600)
			case "cancel":
				_, err = s.cancelCreation(p.ProjectID, scope.Draft.DraftID, "req_candidate")
			case "runtime":
				s.mu.Lock()
				err = os.Remove(filepath.Join(s.store.dir, "creation-runtime.json"))
				s.mu.Unlock()
			case "source":
				err = os.WriteFile(filepath.Join(s.creationDirectory(scope.Draft.DraftID, "req_candidate"), "project", "extra.go"), []byte("package extra"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			close(release)
			v := awaitCreation(t, s, scope, "req_candidate")
			if v.State == "ready" || v.Artifact != nil {
				t.Fatalf("late result published for %s: %+v", action, v)
			}
		})
	}
}
func TestCreationSourceRefusesSetupAndExtraFiles(t *testing.T) {
	s, _, scope := creationFixture(t, func(context.Context, []string) ([]byte, error) { return mockCreationCheck(), nil })
	runtime, err := s.readCreationRuntime()
	if err != nil {
		t.Fatal(err)
	}
	for _, files := range [][]proposalFile{{{"../escape.go", "x"}}, {{"go.mod", "module other"}}, {{"creation-input.json", "{}"}}, {{pipelinereview.CreationSourcePath, "x"}, {"extra.go", "x"}}, {{pipelinereview.CreationSourcePath, "\x00"}}} {
		_, err := retainCreation(scope, runtime, files, t.TempDir())
		wantCode(t, err, "forbidden")
	}
	dir := t.TempDir()
	m, err := retainCreation(scope, runtime, scope.Files, dir)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "project")
	if err = verifyCreationSource(source, m); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(source, "pipeline", "pipe.go")
	if err = os.Chmod(target, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(target, []byte("altered"), 0600); err != nil {
		t.Fatal(err)
	}
	if verifyCreationSource(source, m) == nil {
		t.Fatal("altered seal accepted")
	}
}
func TestCreationRuntimeRejectsUnavailableOrChangedScaffold(t *testing.T) {
	s, _, _ := creationFixture(t, func(context.Context, []string) ([]byte, error) { return mockCreationCheck(), nil })
	runtime, err := s.readCreationRuntime()
	if err != nil {
		t.Fatal(err)
	}
	runtime.Scaffold.Files[0].Path = "../go.mod"
	if validateCreationScaffold(runtime.Scaffold) == nil {
		t.Fatal("escaping scaffold accepted")
	}
	runtime.Scaffold.Files[0].Path = "go.mod"
	s.runtime.command = func(context.Context, []string, []string) ([]byte, error) {
		return nil, problem("runtime_unavailable", "offline")
	}
	_, err = s.bindCreationRuntime(context.Background(), runtime.Binding)
	wantCode(t, err, "runtime_unavailable")
	retained, err := s.readCreationRuntime()
	if err != nil || retained.ID != runtime.ID {
		t.Fatal("failed rebind lost previous selection")
	}
}

func TestLiveCreationCheck(t *testing.T) {
	image := os.Getenv("GOBBLE_CREATION_IMAGE")
	if image == "" {
		t.Skip("set GOBBLE_CREATION_IMAGE to a pinned local creation runtime")
	}
	s, p, file, d := draftFixture(t)
	d, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_input", 1, "Trim and inspect", file, "single-end"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	endpoint, err := s.runtime.command(ctx, []string{"context", "inspect", "--format", "{{.Endpoints.docker.Host}}"}, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	daemon, err := s.runtime.commandAt(ctx, strings.TrimSpace(string(endpoint)), "info", "--format", "{{.ID}}")
	if err != nil {
		t.Fatal(err)
	}
	binding := RuntimeBinding{strings.TrimSpace(string(endpoint)), strings.TrimSpace(string(daemon)), image, "linux/amd64", "/gobble/project", "/gobble/project/work"}
	if _, err = s.bindCreationRuntime(ctx, binding); err != nil {
		t.Fatal(err)
	}
	scope, err := s.creationAuthoring(p.ProjectID, d.DraftID, d.Generation)
	if err != nil {
		t.Fatal(err)
	}
	scope.Files[0].Content = strings.Replace(scope.Files[0].Content, "Quality: 0, Length: 0", "Quality: 25, Length: 40", 1)
	submitCreationFixture(t, s, scope, "req_live_good")
	good := awaitCreation(t, s, scope, "req_live_good")
	if good.State != "ready" || good.Artifact == nil || len(good.Artifact.Check.Gaps) != 0 {
		t.Fatalf("real complete check: %+v", good)
	}
	// The original file deliberately is not valid FASTQ. Successful metadata-only
	// checking therefore makes no claim about data validity or a completed Run.
	bad := scope
	bad.Files = append([]proposalFile(nil), scope.Files...)
	bad.Files[0].Content = strings.Replace(bad.Files[0].Content, "Quality: 25", "Options: modules.Options{ExtraArgs: []string{\"--quiet\"}}, Quality: 25", 1)
	submitCreationFixture(t, s, bad, "req_live_bad")
	rejected := awaitCreation(t, s, bad, "req_live_bad")
	if rejected.State != "unsupported" || rejected.Artifact == nil || len(rejected.Artifact.Check.Gaps) == 0 {
		t.Fatalf("unrepresented command accepted: %+v", rejected)
	}
	// Rebinding a new generation preserves the input's declared uncompressed format.
	project, err := s.store.project(p.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(project.Root, "plain.fastq"), []byte("metadata only, uncompressed"))
	plainFile := findFile(t, s, p, "plain.fastq")
	plainDraft, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_plain_input", d.Generation, "Inspect plain reads", plainFile, "single-end"})
	if err != nil {
		t.Fatal(err)
	}
	plainScope, err := s.creationAuthoring(p.ProjectID, d.DraftID, plainDraft.Generation)
	if err != nil {
		t.Fatal(err)
	}
	submitCreationFixture(t, s, plainScope, "req_live_plain")
	plain := awaitCreation(t, s, plainScope, "req_live_plain")
	if plain.State != "ready" || plain.Artifact == nil || plain.Artifact.Check.InputPath != "inputs/reads.fastq" {
		t.Fatalf("plain input binding: %+v", plain)
	}
	t.Logf("runtime=%s artifact=%s steps=%d gaps=%v", image, good.Artifact.ArtifactID, len(good.Artifact.Check.Review.Definition.Steps), rejected.Artifact.Check.Gaps)
}

func TestCreationCheckRejectsMismatchedEvidence(t *testing.T) {
	for _, mutation := range []func(*creationCheck){
		func(v *creationCheck) { v.Review.Flow = nil },
		func(v *creationCheck) { v.Review.Definition.Steps[0].ID = "different" },
		func(v *creationCheck) { v.Review.Definition.Edges = v.Review.Definition.Edges[:1] },
		func(v *creationCheck) { v.Gaps = nil },
		func(v *creationCheck) { v.Scope = "future" },
	} {
		var v creationCheck
		if err := decodeCreationJSON(mockCreationCheck(), &v); err != nil {
			t.Fatal(err)
		}
		if err := validateCreationCheck(v); err != nil {
			t.Fatal(err)
		}
		mutation(&v)
		if validateCreationCheck(v) == nil {
			t.Fatal("mismatched evidence accepted")
		}
	}
}
