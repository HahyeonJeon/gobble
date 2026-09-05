package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTranslatePathsPreservesArguments(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "runs", "한글, space")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	args, mounts, err := translateArgs(root, []string{"run", ".", "--workspace", workspace, "--cap=2"})
	want := []string{"run", ".", "--workspace", "/gobble/project/runs/한글, space", "--cap=2"}
	if err != nil || len(mounts) != 0 || !reflect.DeepEqual(args, want) {
		t.Fatalf("args=%q mounts=%q err=%v", args, mounts, err)
	}
	external := t.TempDir()
	args, mounts, err = translateArgs(root, []string{"stop", "--workspace=" + external})
	if err != nil || args[1] != "--workspace=/gobble/workspace" || len(mounts) != 2 {
		t.Fatalf("external: %q %q %v", args, mounts, err)
	}
	if _, _, err := translateArgs(root, []string{"run", ".", "--sample", filepath.Join(external, "sheet.csv")}); err == nil {
		t.Fatal("unmounted sample accepted")
	}
}

func TestMountCSVWithWindowsDriveAndComma(t *testing.T) {
	source := `C:\Users\Researcher\한글, data`
	fields, err := csv.NewReader(strings.NewReader(bindMount(source, "/gobble/project"))).Read()
	want := []string{"type=bind", "src=" + source, "dst=/gobble/project"}
	if err != nil || !reflect.DeepEqual(fields, want) {
		t.Fatalf("mount=%q %v", fields, err)
	}
}
