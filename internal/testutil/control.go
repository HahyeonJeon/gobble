// Package testutil contains shared workspace fixtures for Gobble's tests.
package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ControlPath resolves a logical control filename to the current generation.
// Tests that intentionally corrupt a committed record use this helper; normal
// application callers should use Inspect instead of depending on disk layout.
func ControlPath(t testing.TB, path string) string {
	t.Helper()
	if filepath.Base(filepath.Dir(path)) != ".gobble" {
		return path
	}
	switch filepath.Base(path) {
	case "run.json", "plan.json", "tasks.json":
	default:
		return path
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), "current.json"))
	if os.IsNotExist(err) {
		return path
	}
	if err != nil {
		t.Fatal(err)
	}
	var ptr struct {
		Current string `json:"current"`
	}
	if err := json.Unmarshal(raw, &ptr); err != nil {
		t.Fatal(err)
	}
	if ptr.Current == "" || filepath.Base(ptr.Current) != ptr.Current {
		t.Fatal("invalid test checkpoint pointer")
	}
	return filepath.Join(filepath.Dir(path), "checkpoints", ptr.Current, filepath.Base(path))
}
