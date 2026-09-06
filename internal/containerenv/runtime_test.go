package containerenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentPinAndMismatchPreservesIdentity(t *testing.T) {
	root := t.TempDir()
	want := Lock{Format: 1, Image: "sha256:first", Daemon: "engine-one", Reference: "registry/runtime@sha256:manifest"}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() { if err := Pin(root, want); err != nil { t.Error(err) } })
	}
	wg.Wait()
	before, err := os.ReadFile(filepath.Join(root, LockFile))
	if err != nil { t.Fatal(err) }
	for _, changed := range []Lock{
		{Format: 1, Image: "sha256:other", Daemon: want.Daemon},
		{Format: 1, Image: want.Image, Daemon: "other-engine"},
	} {
		if err := Pin(root, changed); err == nil { t.Fatal("identity change accepted") }
	}
	after, _ := os.ReadFile(filepath.Join(root, LockFile))
	var got Lock
	if string(before) != string(after) || json.Unmarshal(after, &got) != nil || got != want { t.Fatalf("lock changed: %s", after) }
}

func TestNestedAndOverlaidMounts(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "project", "runs")
	if err := os.MkdirAll(nested, 0o755); err != nil { t.Fatal(err) }
	mounts := []Mount{{Type: "bind", Source: "/run/desktop/mnt/host/c/My data", Destination: root, RW: true}}
	got, err := MapPath(nested, mounts)
	if err != nil || got != "/run/desktop/mnt/host/c/My data/project/runs" { t.Fatalf("%q %v", got, err) }
	mounts = append(mounts, Mount{Type: "bind", Source: "/private", Destination: filepath.Dir(nested), RW: false})
	if _, err := MapPath(nested, mounts); err == nil { t.Fatal("read-only overlay accepted") }
}
