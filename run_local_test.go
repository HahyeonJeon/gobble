package gobble_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

const (
	runLocalGolden   = "testdata/run-local/plan.json"
	runLocalInput    = "testdata/run-local/in/sample.txt"
	runLocalImage    = "alpine:3.21"
	workflowCaseFile = "testdata/workflow-case/plan.json"
)

func TestRunLocalPlanGolden(t *testing.T) {
	g := mustCompose(runLocalFixturePipeline)(t)
	var buf bytes.Buffer
	if _, err := gobble.BuildPlan(g, gobble.WriteTo(&buf)); err != nil {
		t.Fatalf("BuildPlan() error = %v, want nil", err)
	}
	want, err := os.ReadFile(runLocalGolden)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", runLocalGolden, err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("BuildPlan JSON != %s\ngot:\n%s\nwant:\n%s", runLocalGolden, buf.Bytes(), want)
	}
}

func runLocalFixturePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("run-local")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "image",
		Image:   runLocalImage,
		Command: []string{"sh", "-c", "pwd > out/docker/pwd.txt && cp in/sample.txt out/docker/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{
			{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "sample", Ext: ".txt"}},
			{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "pwd", Ext: ".txt"}},
		},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Resources: gobble.Resources{CPU: 1, Memory: "256m"},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "host",
		Command: []string{"cp", "in/sample.txt", "out/process/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out/process"), Base: "sample", Ext: ".txt"},
		}},
	})
	return p
}

func runLocalBadImagePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("run-local-bad")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "image",
		Image:   "gobble-missing-image:not-a-tag",
		Command: []string{"cp", "in/sample.txt", "out/docker/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "sample", Ext: ".txt"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "host",
		Command: []string{"cp", "in/sample.txt", "out/process/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out/process"), Base: "sample", Ext: ".txt"},
		}},
	})
	return p
}

func copyRunLocalInput(t *testing.T, workspace string) {
	t.Helper()
	data, err := os.ReadFile(runLocalInput)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", runLocalInput, err)
	}
	dst := filepath.Join(workspace, "in", "sample.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", dst, err)
	}
}
