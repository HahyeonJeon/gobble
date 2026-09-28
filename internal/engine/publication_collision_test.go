package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func TestExclusivePublicationRollbackPreservesConcurrentWriter(t *testing.T) {
	ws := t.TempDir()
	isolate := filepath.Join(ws, "isolate")
	task := TaskPlan{Outputs: []IO{{Name: "first", Path: "out/first"}, {Name: "second", Path: "out/second"}}}
	for _, out := range task.Outputs {
		writeCheckFile(t, filepath.Join(isolate, out.Path), "task result")
	}
	first, second := filepath.Join(ws, "out/first"), filepath.Join(ws, "out/second")
	err := publishAllUsing(ws, isolate, task, func(tmp, dst string) error {
		if dst == second {
			writeCheckFile(t, dst, "concurrent result")
		}
		return exec.InstallExclusive(tmp, dst)
	})
	if err == nil {
		t.Fatal("collision succeeded")
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("own partial publication remained", err)
	}
	content, err := os.ReadFile(second)
	if err != nil || string(content) != "concurrent result" {
		t.Fatal("rollback removed another writer's result", err)
	}
	files, err := filepath.Glob(filepath.Join(ws, "out/.gobble-pub-*"))
	if err != nil || len(files) != 0 {
		t.Fatal("temporary publication files leaked", files, err)
	}
}
