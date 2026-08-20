package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestInspectViewsMatchLibrary(t *testing.T) {
	dir := occupiedWorkspace(t)
	views := []gobble.View{
		gobble.ViewRun,
		gobble.ViewInstances,
		gobble.ViewErrors,
		gobble.ViewLogs,
		gobble.ViewTiming,
		gobble.ViewDAG,
		gobble.ViewLineage,
		gobble.ViewRemaining,
		gobble.ViewReuse,
	}
	for _, view := range views {
		t.Run(string(view), func(t *testing.T) {
			want, err := gobble.Inspect(dir, view, "")
			if err != nil {
				t.Fatalf("Inspect() error = %v", err)
			}
			res := runCLI("inspect", string(view), "--workspace", dir)
			if res.code != 0 {
				t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
			}
			if len(res.stderr) != 0 {
				t.Fatalf("stderr = %q, want empty", res.stderr)
			}
			if !bytes.Equal(res.stdout, want) {
				t.Fatalf("stdout %q != Inspect %q", res.stdout, want)
			}
		})
	}
}

func TestInspectEmptyJSONL(t *testing.T) {
	dir := occupiedWorkspace(t)
	for _, view := range []string{"remaining", "reuse"} {
		t.Run(view, func(t *testing.T) {
			want, err := gobble.Inspect(dir, gobble.View(view), "")
			if err != nil {
				t.Fatalf("Inspect() error = %v", err)
			}
			if len(want) != 0 {
				t.Fatalf("Inspect(%s) = %q, want empty JSONL", view, want)
			}
			res := runCLI("inspect", view, "--workspace="+dir)
			if res.code != 0 {
				t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
			}
			if len(res.stdout) != 0 {
				t.Fatalf("stdout = %q, want empty", res.stdout)
			}
		})
	}
}

func TestInspectObjectViewHasNoAddedNewline(t *testing.T) {
	dir := occupiedWorkspace(t)
	want, err := gobble.Inspect(dir, gobble.ViewRun, "")
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if len(want) == 0 || want[len(want)-1] == '\n' {
		t.Fatalf("library object view already ends in newline: %q", want)
	}
	res := runCLI("inspect", "run", "--workspace", dir)
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	if !bytes.Equal(res.stdout, want) {
		t.Fatalf("stdout %q != Inspect %q", res.stdout, want)
	}
	if res.stdout[len(res.stdout)-1] == '\n' {
		t.Fatalf("CLI added newline to object view")
	}
}

func TestInspectMissingWorkspace(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	_, libErr := gobble.Inspect(missing, gobble.ViewRun, "")
	res := runCLI("inspect", "run", "--workspace", missing)
	requireDomainError(t, res, libErr)
	if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
		t.Fatalf("inspect created missing workspace")
	}
}

func TestInspectUnknownView(t *testing.T) {
	dir := occupiedWorkspace(t)
	_, libErr := gobble.Inspect(dir, gobble.View("events"), "")
	res := runCLI("inspect", "events", "--workspace", dir)
	requireDomainError(t, res, libErr)
}

func TestInspectInstance(t *testing.T) {
	dir := occupiedWorkspace(t)
	want, err := gobble.Inspect(dir, gobble.ViewInstances, "copy")
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	res := runCLI("inspect", "instances", "--workspace", dir, "--instance", "copy")
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	if !bytes.Equal(res.stdout, want) {
		t.Fatalf("stdout %q != Inspect %q", res.stdout, want)
	}
}

func occupiedWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	in := filepath.Join(dir, "in", "sample.txt")
	if err := os.MkdirAll(filepath.Dir(in), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in, []byte("reads"), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := gobble.Compose(processEnvCopyPipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return dir
}

func processEnvCopyPipeline() *gobble.Pipeline {
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
		Env:       map[string]string{"HOME": "/tmp/gobble-home"},
		Resources: gobble.Resources{CPU: 1, Memory: "512m"},
	})
	return p
}

func requireDomainError(t *testing.T, res cliResult, libErr error) {
	t.Helper()
	if len(res.stdout) != 0 {
		t.Fatalf("stdout = %q, want empty", res.stdout)
	}
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1\nstderr: %s", res.code, res.stderr)
	}
	var ge *gobble.Error
	if !errors.As(libErr, &ge) {
		t.Fatalf("library error = %v, want *Error", libErr)
	}
	want, err := json.Marshal(ge)
	if err != nil {
		t.Fatal(err)
	}
	got := bytes.TrimSpace(res.stderr)
	if !bytes.Equal(got, want) {
		t.Fatalf("stderr = %s, want %s", got, want)
	}
}
