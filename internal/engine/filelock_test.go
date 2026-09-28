package engine

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Run with TMPDIR on a Docker Desktop bind mount to exercise VirtioFS too.
func TestFreshOccupancyLockIsObservable(t *testing.T) {
	workspace := t.TempDir()
	first, defects := claimOccupy(filepath.Join(workspace, ControlDir))
	if len(defects) != 0 {
		t.Fatal(defects)
	}
	t.Cleanup(func() { first.Close() })
	if !flockHeld(workspace) {
		t.Error("a different descriptor must observe the freshly created occupancy lock")
	}
	second, defects := claimOccupy(filepath.Join(workspace, ControlDir))
	if second != nil {
		second.Close()
		t.Error("a second claimant acquired the occupied workspace")
	}
	if len(defects) != 1 || defects[0].Code != DefectOccupiedWorkspace {
		t.Errorf("second claim defects = %v", defects)
	}
	first.Close()
	if flockHeld(workspace) {
		t.Error("closed occupancy must no longer appear live")
	}
}

func TestFreshCheckpointLockExcludesReaders(t *testing.T) {
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, ControlDir), 0o755); err != nil {
		t.Fatal(err)
	}
	writer, err := checkpointLock(workspace, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { writer.Close() })
	reader, err := os.Open(filepath.Join(workspace, ControlDir, checkpointLockFile))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	err = syscall.Flock(int(reader.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
		t.Fatalf("reader must conflict with fresh writer; got %v", err)
	}
	writer.Close()
	if err := syscall.Flock(int(reader.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		t.Fatalf("reader after writer closes: %v", err)
	}
}
