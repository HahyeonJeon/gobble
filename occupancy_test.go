package gobble_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	t.Run("occupying-process Release", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if err := gobble.Release(dir); err != nil {
			t.Fatalf("occupying-process Release() error = %v", err)
		}
		if occupancySnapshot(t, dir) != "closed" {
			t.Fatalf("occupancy got %s, want closed", occupancySnapshot(t, dir))
		}
	})
	t.Run("live-occupancy", func(t *testing.T) {
		dir := readyRunWorkspace(t)
		if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		gobble.ForgetHeldLease(dir)
		err := gobble.Release(dir)
		requireReleaseError(t, "later process", err, gobble.DefectLiveOccupancy, "")
		if occupancySnapshot(t, dir) != "active" {
			t.Fatalf("live Release closed occupancy")
		}
		gobble.DropHeldLease(dir)
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

func TestConcurrentSetSampleSheetThenComposeRun(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			dir := readyRunWorkspace(t)
			sheet := filepath.Join(dir, fmt.Sprintf("sheet-%d.csv", i))
			body := fmt.Sprintf("sample,read1,read2\ns%d,reads/r1.fq,reads/r2.fq\n", i)
			if err := os.WriteFile(sheet, []byte(body), 0o644); err != nil {
				errs <- err
				return
			}
			gobble.SetSampleSheetPath(sheet)
			loaded, err := gobble.LoadSampleSheet()
			if err != nil {
				errs <- err
				return
			}
			if loaded.Path != sheet || loaded.Rows[0].Sample != fmt.Sprintf("s%d", i) {
				errs <- fmt.Errorf("sheet got path %q sample %q", loaded.Path, loaded.Rows[0].Sample)
				return
			}
			g := mustCompose(processCopyPipeline)(t)
			errs <- gobble.Run(t.Context(), g, dir, 0)
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent SetSampleSheetPath Compose/Run: %v", err)
		}
	}
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
