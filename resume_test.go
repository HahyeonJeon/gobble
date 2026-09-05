package gobble_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/internal/testutil"
)

func TestResumeMissingRunDoesNotOccupy(t *testing.T) {
	dir := readyRunWorkspace(t)
	before := snapshotWorkspace(t, dir)
	err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t))
	requireResumeError(t, "missing run", err, gobble.DefectNothingToResume, "")
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("missing run mutated workspace")
	}
	if _, statErr := os.Stat(filepath.Join(dir, engine.ControlDir)); !os.IsNotExist(statErr) {
		t.Fatalf("missing run occupied %s", engine.ControlDir)
	}
}

func TestResumeRequiresWithIdentity(t *testing.T) {
	err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), t.TempDir(), 0)
	requireResumeError(t, "missing resume identity", err, gobble.DefectInvalidRequest, "identity")
}

func TestResumeAfterSchedulerExit(t *testing.T) {
	dir := readyRunWorkspace(t)
	g := mustCompose(processCopyPipeline)(t)
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	err := gobble.Resume(t.Context(), g, dir, 0, testOccupyOption(t))
	if err != nil {
		t.Fatalf("automatic reconciliation: %v", err)
	}
}

func TestResumeUnsupportedSchemaNoOccupy(t *testing.T) {
	dir := readyRunWorkspace(t)
	writeRunFile(t, testutil.ControlPath(t, filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile)), `{
  "schema_version": 1,
  "id": "run-1",
  "status": "failed",
  "started": "2026-01-01T00:00:00Z",
  "occupancy": {"active": false, "closed": "2026-01-01T00:00:01Z"}
}
`)
	before := snapshotWorkspace(t, dir)
	err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t))
	requireResumeError(t, "unsupported schema", err, gobble.DefectUnsupportedSchema, "")
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("unsupported schema occupied workspace")
	}
}

func TestResumeNilGraphAndCap(t *testing.T) {
	dir := readyRunWorkspace(t)
	err := gobble.Resume(t.Context(), nil, dir, 0, testOccupyOption(t))
	requireResumeError(t, "nil graph", err, gobble.DefectInvalidRequest, "")
	err = gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, -1, testOccupyOption(t))
	requireResumeError(t, "cap below 1", err, gobble.DefectInvalidValue, "")
	err = gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 65, testOccupyOption(t))
	requireResumeError(t, "cap above 64", err, gobble.DefectInvalidValue, "")
}

func TestResumeOp(t *testing.T) {
	err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), t.TempDir(), 0, testOccupyOption(t))
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if ge.Op != "resume" {
		t.Fatalf("Error.Op got %q, want resume", ge.Op)
	}
}

func TestResumeGraphDiffChangeClasses(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T) string
		resume    func() *gobble.Pipeline
		ident     string
		want      string
		wantDec   string
		wantExist string
	}{
		{
			name:    "Added",
			setup:   func(t *testing.T) string { return readyReleasedRun(t, processCopyPipeline) },
			resume:  processCopyPlusPipeline,
			ident:   "extra",
			want:    "Added",
			wantDec: "rerun",
		},
		{
			name:      "Removed",
			setup:     func(t *testing.T) string { return readyReleasedRun(t, processCopyPlusPipeline) },
			resume:    processCopyPipeline,
			ident:     "extra",
			want:      "Removed",
			wantExist: "out/extra.txt",
		},
		{
			name: "Rewired",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processCopyChainPipeline("cp in/sample.txt out/a.txt", "cp out/a.txt out/b.txt"))
			},
			resume:  processCopyChainRewiredPipeline,
			ident:   "b",
			want:    "Rewired",
			wantDec: "rerun",
		},
		{
			name:    "Repathed dest",
			setup:   func(t *testing.T) string { return readyReleasedRun(t, processCopyDestPipeline("sample")) },
			resume:  processCopyDestPipeline("renamed"),
			ident:   "copy",
			want:    "Repathed",
			wantDec: "rerun",
		},
		{
			name: "Repathed wait-only",
			setup: func(t *testing.T) string {
				dir := readyReleasedRun(t, processCopyPipeline)
				writeRunFile(t, filepath.Join(dir, "in", "other.txt"), "reads")
				return dir
			},
			resume:  processCopyOtherInputPipeline,
			ident:   "copy",
			want:    "Repathed",
			wantDec: "rerun",
		},
		{
			name: "IdentityChanged",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processScriptCopyPipeline("cp in/sample.txt out/sample.txt"))
			},
			resume:  processScriptCopyPipeline("cp in/sample.txt out/sample.txt\n# v2"),
			ident:   "copy",
			want:    "IdentityChanged",
			wantDec: "rerun",
		},
		{
			name:    "Unchanged",
			setup:   func(t *testing.T) string { return readyReleasedRun(t, processCopyPipeline) },
			resume:  processCopyPipeline,
			ident:   "copy",
			want:    "Unchanged",
			wantDec: "reused",
		},
		{
			name:    "resources-only Unchanged",
			setup:   func(t *testing.T) string { return readyReleasedRun(t, processCopyPipeline) },
			resume:  processCopyResourcePipelineHeavy,
			ident:   "copy",
			want:    "Unchanged",
			wantDec: "reused",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.setup(t)
			if err := gobble.Resume(t.Context(), mustCompose(tc.resume)(t), dir, 2, testOccupyOption(t)); err != nil {
				t.Fatalf("Resume() error = %v", err)
			}
			got := latestTaskChange(t, dir, tc.ident)
			if got != tc.want {
				t.Fatalf("change got %q, want %q", got, tc.want)
			}
			if tc.wantDec != "" {
				reuse := instanceByID(mustInspectJSONL(t, dir, "reuse", ""))
				if reuse[tc.ident]["change"] != tc.want || reuse[tc.ident]["decision"] != tc.wantDec {
					t.Fatalf("reuse got %#v, want change %s decision %s", reuse[tc.ident], tc.want, tc.wantDec)
				}
			}
			if tc.wantExist != "" {
				if _, statErr := os.Stat(filepath.Join(dir, tc.wantExist)); statErr != nil {
					t.Fatalf("removed dest missing: %v", statErr)
				}
			}
		})
	}
}

func TestResumeFirstRunOmitsChange(t *testing.T) {
	dir := readyRunWorkspace(t)
	if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := latestTaskChange(t, dir, "copy"); got != "" {
		t.Fatalf("first Run change got %q, want omitted", got)
	}
	if remaining := mustInspectJSONL(t, dir, "remaining", ""); len(remaining) != 0 {
		t.Fatalf("remaining got %#v, want empty", remaining)
	}
	if reuse := mustInspectJSONL(t, dir, "reuse", ""); len(reuse) != 0 {
		t.Fatalf("reuse got %#v, want empty", reuse)
	}

	failDir := readyRunWorkspace(t)
	err := gobble.Run(t.Context(), mustCompose(processContainPipeline)(t), failDir, 2, testOccupyOption(t))
	requireRunError(t, "contained failure", err, gobble.DefectFailed, "fail")
	for _, rec := range mustInspectJSONL(t, failDir, "remaining", "") {
		if _, ok := rec["change"]; ok {
			t.Fatalf("first-Run remaining emitted change: %#v", rec)
		}
	}
	if got := latestTaskChange(t, failDir, "fail"); got != "" {
		t.Fatalf("first-Run failed change got %q, want omitted", got)
	}
}

func TestResumeAddedDestCollisionOutputExists(t *testing.T) {
	dir := readyReleasedRun(t, processCopyPlusPipeline)
	before := occupancySnapshot(t, dir)
	err := gobble.Resume(t.Context(), mustCompose(processCopyOtherWriterPipeline)(t), dir, 0, testOccupyOption(t))
	requireResumeError(t, "added dest collision", err, gobble.DefectOutputExists, "other.out")
	if occupancySnapshot(t, dir) != before {
		t.Fatalf("output-exists occupied workspace")
	}
	got, readErr := os.ReadFile(filepath.Join(dir, "out", "extra.txt"))
	if readErr != nil || string(got) == "" {
		t.Fatalf("removed dest mutated: %s err=%v", got, readErr)
	}
}

func TestResumeAllSuccessReuses(t *testing.T) {
	dir := readyReleasedRun(t, processCopyPipeline)
	g := mustCompose(processCopyPipeline)(t)
	if err := gobble.Resume(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	run := mustInspectObject(t, dir, "run", "")
	if run["status"] != engine.StatusSucceeded {
		t.Fatalf("run.status got %#v, want succeeded", run["status"])
	}
	if run["schema_version"] != float64(engine.SchemaVersion) {
		t.Fatalf("schema_version got %#v, want %d", run["schema_version"], engine.SchemaVersion)
	}
	occ, _ := run["occupancy"].(map[string]any)
	if occ["active"] != true {
		t.Fatalf("occupancy got %#v, want active", occ)
	}
	reuse := mustInspectJSONL(t, dir, "reuse", "")
	if len(reuse) != 1 {
		t.Fatalf("reuse records got %d, want 1", len(reuse))
	}
	if reuse[0]["decision"] != "reused" || reuse[0]["reason"] != "reused-identity-matched" {
		t.Fatalf("reuse record got %#v", reuse[0])
	}
	instances := mustInspectJSONL(t, dir, "instances", "")
	if instances[0]["attempt"] != float64(1) {
		t.Fatalf("reused attempt got %#v, want 1", instances[0]["attempt"])
	}
	if _, err := os.Stat(filepath.Join(dir, engine.ControlDir, "tasks", "copy", "_", "0", "2", "work")); !os.IsNotExist(err) {
		t.Fatalf("reuse executed a new attempt")
	}
	err := gobble.Resume(t.Context(), g, dir, 0, testOccupyOption(t))
	if err != nil {
		t.Fatalf("second Resume: %v", err)
	}
	err = gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
	requireRunError(t, "run during resume occupy", err, gobble.DefectOccupiedWorkspace, "")
}

func TestResumeFailedRerunAndBlockedUpstream(t *testing.T) {
	dir := readyReleasedRun(t, processContainPipeline)
	err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2, testOccupyOption(t))
	requireResumeError(t, "contained resume", err, gobble.DefectFailed, "fail")
	instances := mustInspectJSONL(t, dir, "instances", "")
	byID := instanceByID(instances)
	if byID["fail"]["attempt"] != float64(2) || byID["fail"]["decision"] != "rerun" {
		t.Fatalf("fail instance got %#v", byID["fail"])
	}
	if byID["fail"]["status"] != engine.StatusFailed {
		t.Fatalf("fail status got %#v", byID["fail"]["status"])
	}
	if byID["dep"]["attempt"] != float64(1) || byID["dep"]["decision"] != "blocked-upstream" {
		t.Fatalf("dep instance got %#v, want blocked-upstream attempt 1", byID["dep"])
	}
	if byID["dep"]["status"] != engine.StatusBlocked {
		t.Fatalf("dep status got %#v, want blocked", byID["dep"]["status"])
	}
	if _, err := os.Stat(filepath.Join(dir, engine.ControlDir, "tasks", "dep", "_", "0", "2", "work")); !os.IsNotExist(err) {
		t.Fatalf("blocked-upstream created a new attempt")
	}
	if byID["ok"]["decision"] != "reused" || byID["ok"]["attempt"] != float64(1) {
		t.Fatalf("ok instance got %#v", byID["ok"])
	}
	if latestTaskChange(t, dir, "fail") != "IdentityChanged" || latestTaskChange(t, dir, "dep") != "IdentityChanged" {
		t.Fatalf("contained resume change fail=%q dep=%q, want IdentityChanged", latestTaskChange(t, dir, "fail"), latestTaskChange(t, dir, "dep"))
	}
	if latestTaskChange(t, dir, "ok") != "Unchanged" {
		t.Fatalf("ok change got %q, want Unchanged", latestTaskChange(t, dir, "ok"))
	}
	remaining := instanceByID(mustInspectJSONL(t, dir, "remaining", ""))
	if remaining["fail"]["change"] != "IdentityChanged" || remaining["dep"]["change"] != "IdentityChanged" {
		t.Fatalf("remaining change fail=%#v dep=%#v", remaining["fail"], remaining["dep"])
	}
	if mustInspectObject(t, dir, "run", "")["status"] != engine.StatusFailed {
		t.Fatalf("run status got %#v, want failed", mustInspectObject(t, dir, "run", "")["status"])
	}
}

func TestResumeIncompleteNewAttempt(t *testing.T) {
	dir := readyRunWorkspace(t)
	g := mustCompose(processCopyPipeline)(t)
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	forcePublicDeadOwner(t, dir)
	markCopyRunning(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	instances := mustInspectJSONL(t, dir, "instances", "")
	if instances[0]["attempt"] != float64(2) || instances[0]["status"] != engine.StatusSucceeded {
		t.Fatalf("incomplete resume instance got %#v", instances[0])
	}
	if instances[0]["decision"] != "rerun" {
		t.Fatalf("incomplete resume decision got %#v", instances[0]["decision"])
	}
}

func TestResumeReusedDestsNotOutputExists(t *testing.T) {
	dir := readyReleasedRun(t, processCopyPipeline)
	if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); err != nil {
		t.Fatalf("expected reused dest: %v", err)
	}
	if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
}

func TestResumeUnattributedDestOutputExists(t *testing.T) {
	dir := readyReleasedRun(t, processContainPipeline)
	writeRunFile(t, filepath.Join(dir, "out", "dep.txt"), "stray")
	beforeOcc := occupancySnapshot(t, dir)
	err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2, testOccupyOption(t))
	requireResumeError(t, "unattributed dest", err, gobble.DefectOutputExists, "dep.out")
	if occupancySnapshot(t, dir) != beforeOcc {
		t.Fatalf("unattributed dest occupied workspace")
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "dep.txt"))
	if err != nil || string(got) != "stray" {
		t.Fatalf("unattributed dest mutated: %s err=%v", got, err)
	}
}

func TestResumeFailedIdentityForeignDestOutputExists(t *testing.T) {
	dir := readyReleasedRun(t, processContainPipeline)
	writeRunFile(t, filepath.Join(dir, "out", "fail.txt"), "foreign")
	beforeOcc := occupancySnapshot(t, dir)
	err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2, testOccupyOption(t))
	requireResumeError(t, "foreign dest", err, gobble.DefectOutputExists, "fail.out")
	if occupancySnapshot(t, dir) != beforeOcc {
		t.Fatalf("foreign dest occupied workspace")
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "fail.txt"))
	if err != nil || string(got) != "foreign" {
		t.Fatalf("foreign dest mutated: %s err=%v", got, err)
	}
}

func TestResumeDestReplaceTable(t *testing.T) {
	t.Run("failed attempt keeps prior dest", func(t *testing.T) {
		dir := readyReleasedRun(t, processCopyPipeline)
		prior, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
		if err != nil {
			t.Fatal(err)
		}
		err = gobble.Resume(t.Context(), mustCompose(processCopyCmdPipeline("exit 1"))(t), dir, 0, testOccupyOption(t))
		requireResumeError(t, "failed rerun", err, gobble.DefectFailed, "copy")
		got, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
		if err != nil || string(got) != string(prior) {
			t.Fatalf("failed attempt dest got %q, want %q", got, prior)
		}
	})
	t.Run("authorized replace publishes new dest", func(t *testing.T) {
		dir := readyReleasedRun(t, processCopyPipeline)
		err := gobble.Resume(t.Context(), mustCompose(processCopyCmdPipeline("exit 1"))(t), dir, 0, testOccupyOption(t))
		requireResumeError(t, "failed rerun", err, gobble.DefectFailed, "copy")
		forcePublicDeadOwner(t, dir)
		if err := gobble.Release(dir); err != nil {
			t.Fatalf("Release() error = %v", err)
		}
		if err := gobble.Resume(t.Context(), mustCompose(processCopyCmdPipeline("pwd > out/pwd.txt && echo new > out/sample.txt"))(t), dir, 0, testOccupyOption(t)); err != nil {
			t.Fatalf("successful replace Resume() error = %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
		if err != nil || string(got) != "new\n" {
			t.Fatalf("replaced dest got %q, want new", got)
		}
	})
	t.Run("reused dests not output-exists", func(t *testing.T) {
		dir := readyReleasedRun(t, processCopyPipeline)
		if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t)); err != nil {
			t.Fatalf("Resume() error = %v", err)
		}
	})
}

func TestResumeFailedAttemptKeepsPriorDestThenReplace(t *testing.T) {
	dir := readyReleasedRun(t, processCopyPipeline)
	prior, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatal(err)
	}
	err = gobble.Resume(t.Context(), mustCompose(processCopyCmdPipeline("exit 1"))(t), dir, 0, testOccupyOption(t))
	requireResumeError(t, "failed rerun", err, gobble.DefectFailed, "copy")
	got, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil || string(got) != string(prior) {
		t.Fatalf("failed attempt dest got %q, want %q", got, prior)
	}
	instances := mustInspectJSONL(t, dir, "instances", "")
	if instances[0]["attempt"] != float64(2) || instances[0]["status"] != engine.StatusFailed {
		t.Fatalf("failed new attempt got %#v", instances[0])
	}
	forcePublicDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), mustCompose(processCopyCmdPipeline("pwd > out/pwd.txt && echo new > out/sample.txt"))(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("successful replace Resume() error = %v", err)
	}
	got, err = os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil || string(got) != "new\n" {
		t.Fatalf("replaced dest got %q, want new", got)
	}
	instances = mustInspectJSONL(t, dir, "instances", "")
	if instances[0]["attempt"] != float64(3) || instances[0]["status"] != engine.StatusSucceeded {
		t.Fatalf("replaced attempt got %#v", instances[0])
	}
}

func TestResumeResourceOnlyDoesNotRerun(t *testing.T) {
	dir := readyReleasedRun(t, processCopyPipeline)
	if err := gobble.Resume(t.Context(), mustCompose(processCopyResourcePipelineHeavy)(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	reuse := mustInspectJSONL(t, dir, "reuse", "")
	if len(reuse) != 1 || reuse[0]["decision"] != "reused" {
		t.Fatalf("resource-only reuse got %#v", reuse)
	}
	if mustInspectJSONL(t, dir, "instances", "")[0]["attempt"] != float64(1) {
		t.Fatalf("resource-only executed a new attempt")
	}
}

func TestResumeAfterReleaseRunStillOutputExists(t *testing.T) {
	dir := readyReleasedRun(t, processCopyPipeline)
	if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	forcePublicDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t))
	requireRunError(t, "Run after released Resume", err, gobble.DefectOutputExists, "copy.out")
}

func TestResumeScriptChangeReruns(t *testing.T) {
	dir := readyReleasedRun(t, processScriptCopyPipeline("cp in/sample.txt out/sample.txt"))
	if err := gobble.Resume(t.Context(), mustCompose(processScriptCopyPipeline("cp in/sample.txt out/sample.txt\n# v2"))(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	instances := mustInspectJSONL(t, dir, "instances", "")
	if len(instances) != 1 || instances[0]["attempt"] != float64(2) || instances[0]["decision"] != "rerun" {
		t.Fatalf("script resume instance got %#v", instances)
	}
	if instances[0]["reuse_reason"] != "command-or-script-changed" {
		t.Fatalf("script resume reason got %#v", instances[0])
	}
}

func TestResumeEnvChangeReruns(t *testing.T) {
	dir := readyReleasedRun(t, processEnvCopyPipeline)
	if err := gobble.Resume(t.Context(), mustCompose(processEnvCopyHomePipeline("/tmp/gobble-home-2"))(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	instances := mustInspectJSONL(t, dir, "instances", "")
	if len(instances) != 1 || instances[0]["attempt"] != float64(2) || instances[0]["decision"] != "rerun" {
		t.Fatalf("env resume instance got %#v", instances)
	}
	if instances[0]["reuse_reason"] != "env-changed" {
		t.Fatalf("env resume reason got %#v", instances[0])
	}
}

func TestResumeSequentialRerunUsesNewUpstreamDest(t *testing.T) {
	dir := readyReleasedRun(t, processCopyChainPipeline("cp in/sample.txt out/a.txt", "cp out/a.txt out/b.txt"))
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "new")
	if err := gobble.Resume(t.Context(), mustCompose(processCopyChainPipeline("sh -c 'cp in/sample.txt out/a.txt'", "sh -c 'cp out/a.txt out/b.txt'"))(t), dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "b.txt"))
	if err != nil || string(got) != "new" {
		t.Fatalf("b dest got %q err=%v, want new", got, err)
	}
	byID := instanceByID(mustInspectJSONL(t, dir, "instances", ""))
	if byID["a"]["attempt"] != float64(2) || byID["b"]["attempt"] != float64(2) {
		t.Fatalf("chain attempts got a=%#v b=%#v", byID["a"], byID["b"])
	}
}

func TestResumeDestRenameDoesNotReuse(t *testing.T) {
	dir := readyReleasedRun(t, processCopyDestPipeline("sample"))
	err := gobble.Resume(t.Context(), mustCompose(processCopyDestPipeline("renamed"))(t), dir, 0, testOccupyOption(t))
	reuse := mustInspectJSONL(t, dir, "reuse", "")
	if len(reuse) != 1 || reuse[0]["decision"] == "reused" {
		t.Fatalf("dest rename reuse got %#v err=%v", reuse, err)
	}
	instances := mustInspectJSONL(t, dir, "instances", "")
	if len(instances) != 1 || instances[0]["decision"] == "reused" {
		t.Fatalf("dest rename instance got %#v", instances)
	}
}

func TestResumeFanoutAllSuccessReuses(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "a.txt"), "a")
	writeRunFile(t, filepath.Join(dir, "in", "b.txt"), "b")
	if err := gobble.Run(t.Context(), mustCompose(processFanoutPipeline)(t), dir, 2, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	forcePublicDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), mustCompose(processFanoutPipeline)(t), dir, 2, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	byID := instanceByID(mustInspectJSONL(t, dir, "reuse", ""))
	if len(byID) != 2 {
		t.Fatalf("fanout reuse records got %#v", byID)
	}
	for _, id := range []string{"left", "right"} {
		if byID[id]["decision"] != "reused" || byID[id]["reason"] != "reused-identity-matched" {
			t.Fatalf("%s reuse got %#v", id, byID[id])
		}
	}
	instances := instanceByID(mustInspectJSONL(t, dir, "instances", ""))
	if instances["left"]["attempt"] != float64(1) || instances["right"]["attempt"] != float64(1) {
		t.Fatalf("fanout attempts got left=%#v right=%#v", instances["left"], instances["right"])
	}
}

func processCopyPlusPipeline() *gobble.Pipeline {
	p := processCopyPipeline()
	p.AddTask(gobble.TaskSpec{
		Name:    "extra",
		Command: []string{"sh", "-c", "echo extra > out/extra.txt"},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "extra", Ext: ".txt"},
		}},
	})
	return p
}

func processCopyOtherWriterPipeline() *gobble.Pipeline {
	p := processCopyPipeline()
	p.AddTask(gobble.TaskSpec{
		Name:    "other",
		Command: []string{"sh", "-c", "echo other > out/extra.txt"},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "extra", Ext: ".txt"},
		}},
	})
	return p
}

func processCopyChainRewiredPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("chain")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "a",
		Command: []string{"sh", "-c", "cp in/sample.txt out/a.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "a", Ext: ".txt"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "b",
		Command: []string{"sh", "-c", "cp in/sample.txt out/b.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "b", Ext: ".txt"},
		}},
	})
	return p
}

func latestTaskChange(t *testing.T, dir, ident string) string {
	t.Helper()
	data, err := os.ReadFile(testutil.ControlPath(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile)))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	var best map[string]any
	bestAttempt := -1
	for _, st := range file.Tasks {
		id, _ := st["id"].(string)
		if id != ident {
			continue
		}
		attempt := 0
		if v, ok := st["attempt"].(float64); ok {
			attempt = int(v)
		}
		if attempt >= bestAttempt {
			bestAttempt = attempt
			best = st
		}
	}
	if best == nil {
		t.Fatalf("identity %s missing from tasks.json", ident)
	}
	ch, _ := best["change"].(string)
	return ch
}

func processCopyCmdPipeline(cmd string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		p := gobble.NewPipeline("copy")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		p.AddTask(gobble.TaskSpec{
			Name:    "copy",
			Command: []string{"sh", "-c", cmd},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{
				{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
				{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "pwd", Ext: ".txt"}},
			},
			Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
			Resources: gobble.Resources{CPU: 1, Memory: "512m"},
		})
		return p
	}
}

func processScriptCopyPipeline(script string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		p := gobble.NewPipeline("copy")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		p.AddTask(gobble.TaskSpec{
			Name:   "copy",
			Script: script,
			Inputs: []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{
				{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
			},
		})
		return p
	}
}

func processEnvCopyHomePipeline(home string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		return processEnvCopyWithHome(home)
	}
}

func processEnvCopyWithHome(home string) *gobble.Pipeline {
	p := gobble.NewPipeline("copy")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"sh", "-c", "pwd > out/pwd.txt && cp in/sample.txt out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{
			{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
			{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "pwd", Ext: ".txt"}},
		},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Env:       map[string]string{"HOME": home},
		Resources: gobble.Resources{CPU: 1, Memory: "512m"},
	})
	return p
}

func processCopyChainPipeline(aCmd, bCmd string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		p := gobble.NewPipeline("chain")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		a := p.AddTask(gobble.TaskSpec{
			Name:    "a",
			Command: []string{"sh", "-c", aCmd},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{{
				Name: "out",
				Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "a", Ext: ".txt"},
			}},
		})
		p.AddTask(gobble.TaskSpec{
			Name:    "b",
			Command: []string{"sh", "-c", bCmd},
			Inputs:  []gobble.Bind{{Name: "in", From: a.Out("out")}},
			Outputs: []gobble.Bind{{
				Name: "out",
				Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "b", Ext: ".txt"},
			}},
		})
		return p
	}
}

func processCopyDestPipeline(name string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		p := gobble.NewPipeline("copy")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		p.AddTask(gobble.TaskSpec{
			Name:    "copy",
			Command: []string{"cp", "in/sample.txt", "out/" + name + ".txt"},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{{
				Name: "out",
				Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: name, Ext: ".txt"},
			}},
		})
		return p
	}
}

func processCopyOtherInputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("copy")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "other", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"sh", "-c", "pwd > out/pwd.txt && cp in/other.txt out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{
			{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
			{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "pwd", Ext: ".txt"}},
		},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Resources: gobble.Resources{CPU: 1, Memory: "512m"},
	})
	return p
}

func processCopyResourcePipelineHeavy() *gobble.Pipeline {
	p := gobble.NewPipeline("copy")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"sh", "-c", "pwd > out/pwd.txt && cp in/sample.txt out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{
			{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
			{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "pwd", Ext: ".txt"}},
		},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Resources: gobble.Resources{CPU: 2, Memory: "1g"},
	})
	return p
}

func readyReleasedRun(t *testing.T, pipe func() *gobble.Pipeline) string {
	t.Helper()
	dir := readyRunWorkspace(t)
	err := gobble.Run(t.Context(), mustCompose(pipe)(t), dir, 2, testOccupyOption(t))
	if err != nil {
		var ge *gobble.Error
		if !errors.As(err, &ge) {
			t.Fatalf("Run() error = %v", err)
		}
	}
	forcePublicDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	return dir
}

func occupancySnapshot(t *testing.T, dir string) string {
	t.Helper()
	path := testutil.ControlPath(t, filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "ABSENT"
		}
		t.Fatal(err)
	}
	return occupancySnapshotFrom(string(data))
}

func occupancySnapshotFrom(raw string) string {
	var run map[string]any
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		return raw
	}
	occ, _ := run["occupancy"].(map[string]any)
	if occ == nil {
		return "none"
	}
	if occ["active"] == true {
		return "active"
	}
	return "closed"
}

func markCopyRunning(t *testing.T, dir string) {
	t.Helper()
	path := testutil.ControlPath(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	tasks, _ := doc["tasks"].([]any)
	for _, raw := range tasks {
		st, _ := raw.(map[string]any)
		if st["id"] == "copy" {
			st["status"] = engine.StatusIncomplete
			st["ended"] = "2026-01-01T00:00:01Z"
			st["runtime_id"] = ""
			st["reason"] = "released"
		}
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func instanceByID(recs []map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(recs))
	for _, rec := range recs {
		id, _ := rec["identity"].(string)
		out[id] = rec
	}
	return out
}

func requireResumeError(t *testing.T, name string, err error, code gobble.DefectCode, unit string) *gobble.Error {
	t.Helper()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("case %s: error = %v, want *Error", name, err)
	}
	if ge.Op != "resume" {
		t.Fatalf("case %s: Error.Op got %q, want resume", name, ge.Op)
	}
	found := false
	for _, d := range ge.Defects {
		if d.Code == code && (unit == "" || d.Unit == unit) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("case %s: defects %#v, want code %s unit %q", name, ge.Defects, code, unit)
	}
	return ge
}
