package gobble

import "testing"

func TestRestageSpecClearsLiteralOpacity(t *testing.T) {
	from := Literal("sample.html").WithDir(Dir("work"))
	spec := PathSpec{Name: "report", Ext: ".txt"}
	got := restageSpec(spec, from)
	rendered, err := got.Render()
	if err != nil {
		t.Fatalf("case restage-literal: restageSpec().Render() error = %v, want nil", err)
	}
	if rendered != "work/report.txt" {
		t.Fatalf("case restage-literal: restageSpec().Render() got %q, want %q", rendered, "work/report.txt")
	}
	classified := classifySpec(spec, from, DeriveAppend)
	classRendered, err := classified.Render()
	if err != nil {
		t.Fatalf("case restage-literal: classifySpec().Render() error = %v, want nil", err)
	}
	if classRendered != "work/report.txt" {
		t.Fatalf("case restage-literal: classifySpec().Render() got %q, want %q", classRendered, "work/report.txt")
	}
}

func TestComposeRestageFromLiteralRendersNewFields(t *testing.T) {
	p := NewPipeline("restage")
	src := p.AddTask(TaskSpec{
		Name:    "src",
		Command: []string{"echo"},
		Outputs: []Bind{{
			Name: "html",
			Spec: Literal("sample.html").WithDir(Dir("work")),
		}},
	})
	p.AddTask(TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []Bind{{Name: "in", From: src.Out("html")}},
		Outputs: []Bind{{
			Name: "out",
			From: src.Out("html"),
			Spec: PathSpec{Name: "report", Ext: ".txt"},
		}},
	})
	g, err := Compose(p)
	if err != nil {
		t.Fatalf("case compose-restage: Compose() error = %v, want nil", err)
	}
	var spec PathSpec
	found := false
	for _, task := range g.tasks {
		if task.id != "copy" {
			continue
		}
		for _, b := range task.outputs {
			if b.name == "out" {
				spec = b.spec
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("case compose-restage: Graph missing copy.out")
	}
	got, err := spec.Render()
	if err != nil {
		t.Fatalf("case compose-restage: Graph copy.out.Render() error = %v, want nil", err)
	}
	if got != "work/report.txt" {
		t.Fatalf("case compose-restage: Graph copy.out.Render() got %q, want %q (opaque parent would be work/sample.html)", got, "work/report.txt")
	}
	if err := Validate(g); err != nil {
		t.Fatalf("case compose-restage: Validate() error = %v, want nil", err)
	}
}
