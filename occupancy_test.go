package gobble_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestOccupancyTable(t *testing.T) {
	t.Run("occupy after Run", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if occupancySnapshot(t, dir) != "active" {
			t.Fatalf("occupancy got %s, want active", occupancySnapshot(t, dir))
		}
		run := mustInspectObject(t, dir, "run", "")
		occ, _ := run["occupancy"].(map[string]any)
		if occ["active"] != true || occ["live"] != true {
			t.Fatalf("run occupancy got %#v", occ)
		}
	})
	t.Run("Inspect allowed while occupied", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		before := snapshotWorkspace(t, dir)
		if _, err := gobble.Inspect(dir, gobble.ViewRun, ""); err != nil {
			t.Fatalf("Inspect while occupied error = %v", err)
		}
		after := snapshotWorkspace(t, dir)
		if before != after {
			t.Fatalf("Inspect mutated occupied workspace")
		}
	})
	t.Run("second Run occupied-workspace", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		g := mustCompose(processCopyPipeline)(t)
		if err := gobble.Run(t.Context(), g, dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		err := gobble.Run(t.Context(), g, dir, 0)
		requireRunError(t, "second Run", err, gobble.DefectOccupiedWorkspace, "")
	})
	t.Run("Resume occupied-workspace", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		g := mustCompose(processCopyPipeline)(t)
		if err := gobble.Run(t.Context(), g, dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		err := gobble.Resume(t.Context(), g, dir, 0)
		requireResumeError(t, "resume occupied", err, gobble.DefectOccupiedWorkspace, "")
	})
}

func TestReleaseTable(t *testing.T) {
	t.Run("not-found", func(t *testing.T) {
		err := gobble.Release(t.TempDir())
		requireReleaseError(t, "missing run", err, gobble.DefectNotFound, "")
	})
	t.Run("live-occupancy", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		err := gobble.Release(dir)
		requireReleaseError(t, "live owner", err, gobble.DefectLiveOccupancy, "")
		if occupancySnapshot(t, dir) != "active" {
			t.Fatalf("live Release closed occupancy")
		}
	})
	t.Run("dead owner then already-released", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		forcePublicDeadOwner(t, dir)
		if err := gobble.Release(dir); err != nil {
			t.Fatalf("Release() error = %v", err)
		}
		if occupancySnapshot(t, dir) != "closed" {
			t.Fatalf("occupancy got %s, want closed", occupancySnapshot(t, dir))
		}
		err := gobble.Release(dir)
		requireReleaseError(t, "already released", err, gobble.DefectAlreadyReleased, "")
	})
	t.Run("foreign-host", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		forcePublicDeadOwner(t, dir)
		patchOccupancyHost(t, dir, "other-host")
		err := gobble.Release(dir)
		requireReleaseError(t, "foreign host", err, gobble.DefectForeignHost, "")
		if occupancySnapshot(t, dir) != "active" {
			t.Fatalf("foreign-host Release closed occupancy")
		}
	})
}

func patchOccupancyHost(t *testing.T, dir, host string) {
	t.Helper()
	path := filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var run map[string]any
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatal(err)
	}
	occ, _ := run["occupancy"].(map[string]any)
	if occ == nil {
		t.Fatal("occupancy missing")
	}
	occ["host"] = host
	out, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
