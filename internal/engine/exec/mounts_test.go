package exec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestControllerMountMapping(t *testing.T) {
	project := t.TempDir()
	attempt := filepath.Join(project, "runs", "한글 space", "attempt")
	if err := os.MkdirAll(attempt, 0o755); err != nil {
		t.Fatal(err)
	}
	mounts := []controllerMount{{Type: "bind", Destination: project, Source: "/run/desktop/mnt/host/c/My data", RW: true}}
	got, err := mapControllerPath(attempt, mounts)
	want := "/run/desktop/mnt/host/c/My data/runs/한글 space/attempt"
	if err != nil || got != want {
		t.Fatalf("map = %q %v; want %q", got, err, want)
	}
	// An overlaid read-only mount must override a writable parent, not inherit
	// its permissions or point a task at a different underlying directory.
	mounts = append(mounts, controllerMount{Type: "bind", Destination: filepath.Dir(attempt), Source: "/read-only", RW: false})
	if _, err := mapControllerPath(attempt, mounts); err == nil {
		t.Fatal("read-only child accepted")
	}
	if _, err := mapControllerPath(t.TempDir(), mounts); err == nil {
		t.Fatal("unmounted path accepted")
	}
	link := filepath.Join(project, "escape")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	if _, err := mapControllerPath(link, mounts[:1]); err == nil {
		t.Fatal("escaping symlink accepted")
	}
}
