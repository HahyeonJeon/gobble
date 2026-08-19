package assets_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
)

// Copied dual-entry helper shape. This file must not import package assets.

type dummyParent interface {
	AddTask(spec gobble.TaskSpec) *gobble.Task
}

func dummyAddTask(parent dummyParent, spec gobble.TaskSpec) *gobble.Task {
	return parent.AddTask(spec)
}

type dummyInput struct {
	Name string
	Spec gobble.PathSpec
}

func dummyStandalone(name string, inputs []dummyInput, build func(dummyParent, []gobble.Handle)) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	handles := make([]gobble.Handle, len(inputs))
	for i, in := range inputs {
		handles[i] = p.AddInput(in.Name, in.Spec)
	}
	build(p, handles)
	return p
}

func dummyAppendExtraArgs(command, extra []string) []string {
	out := make([]string, 0, len(command)+len(extra))
	out = append(out, command...)
	out = append(out, extra...)
	return out
}

func TestDummyAssetCompose(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"}
	out := gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}
	token, err := reads.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	p := dummyStandalone("dummy", []dummyInput{{Name: "reads", Spec: reads}}, func(parent dummyParent, hs []gobble.Handle) {
		cmd := dummyAppendExtraArgs([]string{"copy", token}, []string{"--quiet"})
		dummyAddTask(parent, gobble.TaskSpec{
			Name:    "copy",
			Command: cmd,
			Inputs:  []gobble.Bind{{Name: "reads", From: hs[0]}},
			Outputs: []gobble.Bind{{Name: "out", Spec: out}},
		})
	})

	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("Compose() graph = nil, want non-nil")
	}
}
