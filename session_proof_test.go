package gobble_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestSessionProofCrashedOccupyReleaseResume(t *testing.T) {
	dir := readyRunWorkspace(t)
	g := mustCompose(processCopyPipeline)(t)
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	forcePublicDeadOwner(t, dir)
	markCopyRunning(t, dir)

	if occupancySnapshot(t, dir) != "active" {
		t.Fatalf("crashed occupy occupancy got %s, want active", occupancySnapshot(t, dir))
	}

	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	run := mustInspectObject(t, dir, "run", "")
	occ, _ := run["occupancy"].(map[string]any)
	if occ["active"] != false || occ["closed"] == "" {
		t.Fatalf("released occupancy got %#v", occ)
	}
	instances := mustInspectJSONL(t, dir, "instances", "")
	if len(instances) != 1 || instances[0]["status"] != engine.StatusIncomplete {
		t.Fatalf("released instance got %#v, want incomplete", instances)
	}
	remaining := mustInspectJSONL(t, dir, "remaining", "")
	if len(remaining) != 1 || remaining[0]["identity"] != "copy" || remaining[0]["remaining"] != true {
		t.Fatalf("remaining after crash release got %#v", remaining)
	}

	if err := gobble.Resume(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() after Release error = %v", err)
	}
	run = mustInspectObject(t, dir, "run", "")
	occ, _ = run["occupancy"].(map[string]any)
	if occ["active"] != true {
		t.Fatalf("resumed occupancy got %#v, want active", occ)
	}
	instances = mustInspectJSONL(t, dir, "instances", "")
	if len(instances) != 1 {
		t.Fatalf("resumed instances got %#v", instances)
	}
	if instances[0]["status"] != engine.StatusSucceeded || instances[0]["attempt"] != float64(2) {
		t.Fatalf("resumed remaining instance got %#v", instances[0])
	}
	if instances[0]["decision"] != "rerun" {
		t.Fatalf("resumed remaining decision got %#v, want rerun", instances[0])
	}
}

func TestSessionProofReuseDecisionsVisible(t *testing.T) {
	t.Run("reused-identity-matched", func(t *testing.T) {
		dir := readyReleasedRun(t, processCopyPipeline)
		if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0, testOccupyOption(t)); err != nil {
			t.Fatalf("Resume() error = %v", err)
		}
		requireReuseRecords(t, dir, map[string]decisionWant{
			"copy": {decision: "reused", reason: "reused-identity-matched"},
		})
		if remaining := mustInspectJSONL(t, dir, "remaining", ""); len(remaining) != 0 {
			t.Fatalf("remaining got %#v, want empty", remaining)
		}
	})

	t.Run("rerun-incomplete", func(t *testing.T) {
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
		requireReuseRecords(t, dir, map[string]decisionWant{
			"copy": {decision: "rerun", reason: "previous-incomplete"},
		})
	})

	t.Run("rerun-unsuccessful-and-blocked-upstream", func(t *testing.T) {
		dir := readyReleasedRun(t, processContainPipeline)
		err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2, testOccupyOption(t))
		requireResumeError(t, "contained resume", err, gobble.DefectFailed, "fail")
		requireReuseRecords(t, dir, map[string]decisionWant{
			"fail": {decision: "rerun", reason: "previous-unsuccessful"},
			"dep":  {decision: "blocked-upstream", reason: "downstream-of-rerun"},
			"ok":   {decision: "reused", reason: "reused-identity-matched"},
		})
		remaining := instanceByID(mustInspectJSONL(t, dir, "remaining", ""))
		if remaining["fail"]["remaining"] != true || remaining["dep"]["remaining"] != true {
			t.Fatalf("remaining got %#v", remaining)
		}
		if remaining["fail"]["reason"] == "" || remaining["dep"]["reason"] == "" {
			t.Fatalf("remaining missing reason: %#v", remaining)
		}
		if remaining["ok"] != nil {
			t.Fatalf("ok listed as remaining: %#v", remaining["ok"])
		}
	})

	t.Run("unattributed-dest-output-exists", func(t *testing.T) {
		dir := readyReleasedRun(t, processContainPipeline)
		writeRunFile(t, filepath.Join(dir, "out", "dep.txt"), "stray")
		beforeOcc := occupancySnapshot(t, dir)
		beforeReuse := mustInspectJSONL(t, dir, "reuse", "")
		err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2, testOccupyOption(t))
		requireResumeError(t, "unattributed dest", err, gobble.DefectOutputExists, "dep.out")
		if occupancySnapshot(t, dir) != beforeOcc {
			t.Fatalf("output-exists occupied workspace")
		}
		got, err := os.ReadFile(filepath.Join(dir, "out", "dep.txt"))
		if err != nil || string(got) != "stray" {
			t.Fatalf("unattributed dest mutated: %s err=%v", got, err)
		}
		afterReuse := mustInspectJSONL(t, dir, "reuse", "")
		if len(afterReuse) != len(beforeReuse) {
			t.Fatalf("output-exists wrote reuse records: before %#v after %#v", beforeReuse, afterReuse)
		}
	})
}

type decisionWant struct {
	decision string
	reason   string
}

func requireReuseRecords(t *testing.T, dir string, want map[string]decisionWant) {
	t.Helper()
	reuse := instanceByID(mustInspectJSONL(t, dir, "reuse", ""))
	if len(reuse) != len(want) {
		t.Fatalf("reuse records got %d %#v, want %d", len(reuse), reuse, len(want))
	}
	instances := instanceByID(mustInspectJSONL(t, dir, "instances", ""))
	for ident, exp := range want {
		rec := reuse[ident]
		if rec == nil {
			t.Fatalf("reuse missing %s: %#v", ident, reuse)
		}
		if rec["decision"] != exp.decision || rec["reason"] != exp.reason {
			t.Fatalf("%s reuse got %#v, want decision %s reason %s", ident, rec, exp.decision, exp.reason)
		}
		inst := instances[ident]
		if inst == nil || inst["decision"] != exp.decision || inst["reuse_reason"] != exp.reason {
			t.Fatalf("%s instance got %#v, want decision %s reason %s", ident, inst, exp.decision, exp.reason)
		}
	}
}
