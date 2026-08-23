package assets

import (
	"fmt"

	"github.com/HahyeonJeon/gobble"
)

const (
	syntheticCohortSize = 24
	scatterMemberScript = `f=$(find . -type f ! -name '*.out' ! -name '.gobble-tree.json' | head -1); mkdir -p "$(dirname "$f")"; cp "$f" "$f.out"`
	gatherJoinScript    = "mkdir -p out; find . -name '*.out' -exec cat {} + > out/all.txt"
)

// ScatterGather returns a static Group scatter-and-gather proof.
func ScatterGather() *gobble.Pipeline {
	p := gobble.NewPipeline("scatter-gather")
	samples := p.AddInputGroup("samples", proofSampleGroup())
	item := AddTask(p.Scatter("each").From(samples), gobble.TaskSpec{
		Name:    "copy",
		Script:  scatterMemberScript,
		Inputs:  []gobble.Bind{{Name: "in", From: samples}},
		Outputs: []gobble.Bind{{Name: "out", From: samples, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	AddTask(p.Gather("all"), gobble.TaskSpec{
		Name:    "join",
		Script:  gatherJoinScript,
		Inputs:  []gobble.Bind{{Name: "parts", From: item.Out("out")}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "all", Ext: ".txt"}}},
	})
	return p
}

// ConditionalSkip returns a When proof. keep must be the declared
// boolean param "true" or "false".
func ConditionalSkip(keep string) *gobble.Pipeline {
	p := gobble.NewPipeline("conditional-skip")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	AddTask(p.When("opt").SkipIfFalse("keep"), gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
		Params:  []gobble.Param{{Name: "keep", Value: keep}},
	})
	return p
}

// DynamicExpansion returns a data-dependent Tree scatter proof.
// Membership is recorded after the producer succeeds.
func DynamicExpansion() *gobble.Pipeline {
	p := gobble.NewPipeline("dynamic-expansion")
	split := AddTask(p, gobble.TaskSpec{
		Name:    "split",
		Command: []string{"sh", "-c", "mkdir -p work/tree && echo one > work/tree/s1.txt && echo two > work/tree/s2.txt"},
		Outputs: []gobble.Bind{{Name: "tree", Tree: gobble.DeclareTree(gobble.Dir("work/tree"))}},
	})
	tree := split.Out("tree")
	item := AddTask(p.Scatter("each").From(tree), gobble.TaskSpec{
		Name:   "copy",
		Script: scatterMemberScript,
		Inputs: []gobble.Bind{{
			Name: "in",
			From: tree,
			Tree: gobble.DeclareTree(gobble.Dir("work/tree")),
		}},
		Outputs: []gobble.Bind{{Name: "out", From: tree, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	AddTask(p.Gather("all"), gobble.TaskSpec{
		Name:    "join",
		Script:  gatherJoinScript,
		Inputs:  []gobble.Bind{{Name: "parts", From: item.Out("out")}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "all", Ext: ".txt"}}},
	})
	return p
}

// TreeGroup returns a Group and Tree publication proof.
func TreeGroup() *gobble.Pipeline {
	p := gobble.NewPipeline("tree-group")
	idx := p.AddInputGroup("idx", gobble.Group{
		{Name: "amb", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "ref", Ext: ".amb"}},
		{Name: "ann", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "ref", Ext: ".ann"}},
	})
	AddTask(p, gobble.TaskSpec{
		Name:    "copy_group",
		Command: []string{"sh", "-c", "cp in/ref.amb out/out.amb && cp in/ref.ann out/out.ann"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  idx,
			Group: gobble.Group{{Name: "amb"}, {Name: "ann"}},
		}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Group: gobble.Group{
				{Name: "amb", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "out", Ext: ".amb"}},
				{Name: "ann", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "out", Ext: ".ann"}},
			},
		}},
	})
	tree := AddTask(p, gobble.TaskSpec{
		Name:    "make_tree",
		Command: []string{"sh", "-c", "mkdir -p work/idx && echo x > work/idx/SA && echo y > work/idx/ann"},
		Outputs: []gobble.Bind{{Name: "idx", Tree: gobble.DeclareTree(gobble.Dir("work/idx"))}},
	})
	AddTask(p, gobble.TaskSpec{
		Name:    "use_tree",
		Command: []string{"sh", "-c", "test -d work/idx && test -f work/idx/SA && echo ok > out/ok.txt"},
		Inputs: []gobble.Bind{{
			Name: "idx",
			From: tree.Out("idx"),
			Tree: gobble.DeclareTree(gobble.Dir("work/idx")),
		}},
		Outputs: []gobble.Bind{{
			Name: "ok",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "ok", Ext: ".txt"},
		}},
	})
	return p
}

// ModuleFanout returns two sibling modules as a module fan-out proof.
func ModuleFanout() *gobble.Pipeline {
	p := gobble.NewPipeline("module-fanout")
	a := p.AddInput("a", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "a", Ext: ".txt"})
	b := p.AddInput("b", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "b", Ext: ".txt"})
	copyProofFile(AddModule(p, "left"), "copy", a, gobble.PathSpec{Dir: gobble.Dir("out"), Base: "a", Ext: ".txt"})
	copyProofFile(AddModule(p, "right"), "copy", b, gobble.PathSpec{Dir: gobble.Dir("out"), Base: "b", Ext: ".txt"})
	return p
}

// SyntheticCohort returns one larger local-scale proof. It is not a
// latency SLA. Size is syntheticCohortSize independent modules.
func SyntheticCohort() *gobble.Pipeline {
	p := gobble.NewPipeline("synthetic-cohort")
	for i := 1; i <= syntheticCohortSize; i++ {
		name := fmt.Sprintf("s%02d", i)
		in := p.AddInput(name, gobble.PathSpec{Dir: gobble.Dir("in"), Base: name, Ext: ".txt"})
		out := gobble.PathSpec{Dir: gobble.Dir("work/" + name), Base: "out", Ext: ".txt"}
		copyProofFile(AddModule(p, name), "copy", in, out)
	}
	return p
}

func proofSampleGroup() gobble.Group {
	return gobble.Group{
		{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}},
		{Name: "s2", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s2", Ext: ".txt"}},
	}
}
