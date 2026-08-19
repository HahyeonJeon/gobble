package assets

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestAddTaskParents(t *testing.T) {
	p := gobble.NewPipeline("helpers")
	in := p.AddInput("in", gobble.PathSpec{Name: "in", Ext: ".txt"})
	spec := func(name, out string) gobble.TaskSpec {
		return gobble.TaskSpec{
			Name:    name,
			Command: []string{"echo"},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Name: out, Ext: ".txt"}}},
		}
	}

	if got := AddTask(p, spec("root", "root")); got == nil {
		t.Fatalf("AddTask(pipeline) = nil, want task")
	}
	mod := AddModule(p, "mod")
	if mod == nil {
		t.Fatalf("AddModule(pipeline) = nil, want module")
	}
	if got := AddTask(mod, spec("modtask", "mod")); got == nil {
		t.Fatalf("AddTask(module) = nil, want task")
	}
	br := p.Branch("br")
	if got := AddTask(br, spec("brtask", "br")); got == nil {
		t.Fatalf("AddTask(branch) = nil, want task")
	}
	mg := p.Merge("mg", br)
	if got := AddTask(mg, spec("mgtask", "mg")); got == nil {
		t.Fatalf("AddTask(merge) = nil, want task")
	}

	if _, err := gobble.Compose(p); err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
}

func TestCommandPath(t *testing.T) {
	spec := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "sample", Ext: ".fastq.gz"}
	got, err := CommandPath(spec)
	if err != nil {
		t.Fatalf("CommandPath() error = %v, want nil", err)
	}
	want, err := spec.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got != want {
		t.Fatalf("CommandPath() = %q, want %q", got, want)
	}

	if _, err := CommandPath(gobble.PathSpec{Name: "."}); err == nil {
		t.Fatalf("CommandPath(invalid) error = nil, want error")
	}
}

func TestStandalone(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "sample", Ext: ".txt"}
	out := gobble.PathSpec{Dir: gobble.Dir("out"), Name: "sample", Ext: ".txt"}
	token, err := CommandPath(reads)
	if err != nil {
		t.Fatalf("CommandPath() error = %v", err)
	}

	var seen gobble.Handle
	p := Standalone("wrap", []Input{{Name: "reads", Spec: reads}}, func(parent Parent, hs []gobble.Handle) {
		if len(hs) != 1 {
			t.Fatalf("build handles got %d, want 1", len(hs))
		}
		seen = hs[0]
		cmd := AppendExtraArgs([]string{"cat", token}, []string{"--flag"})
		AddTask(parent, gobble.TaskSpec{
			Name:    "cat",
			Command: cmd,
			Inputs:  []gobble.Bind{{Name: "reads", From: hs[0]}},
			Outputs: []gobble.Bind{{Name: "out", Spec: out}},
		})
	})
	if p.Name() != "wrap" {
		t.Fatalf("Standalone name = %q, want %q", p.Name(), "wrap")
	}
	if seen.IsZero() {
		t.Fatalf("build handle IsZero() = true, want false")
	}
	if seen.Name() != "reads" {
		t.Fatalf("build handle Name() = %q, want %q", seen.Name(), "reads")
	}
	if !seen.Spec().Equal(reads) {
		t.Fatalf("build handle Spec() = %+v, want %+v", seen.Spec(), reads)
	}

	if _, err := gobble.Compose(p); err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
}

func TestStandaloneGroup(t *testing.T) {
	idx := gobble.Group{
		{Name: "amb", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Name: "ref", Ext: ".amb"}},
		{Name: "ann", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Name: "ref", Ext: ".ann"}},
	}
	p := Standalone("wrap-group", []Input{{Name: "idx", Group: idx}}, func(parent Parent, hs []gobble.Handle) {
		if len(hs) != 1 || hs[0].IsZero() || hs[0].Name() != "idx" {
			t.Fatalf("build handle = %+v, want idx", hs)
		}
		AddTask(parent, gobble.TaskSpec{
			Name:    "use",
			Command: []string{"true"},
			Inputs: []gobble.Bind{{
				Name:  "idx",
				From:  hs[0],
				Group: gobble.Group{{Name: "amb"}, {Name: "ann"}},
			}},
			Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Name: "ok", Ext: ".txt"}}},
		})
	})
	if _, err := gobble.Compose(p); err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
}
