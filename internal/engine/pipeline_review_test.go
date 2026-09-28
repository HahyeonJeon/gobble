package engine

import (
	"github.com/HahyeonJeon/gobble/internal/modulecommand"
	"testing"
)

func TestReviewFingerprintCoversHiddenExecutionFields(t *testing.T) {
	base := TaskPlan{ID: "task", Name: "custom", Command: []string{"tool"}, Inputs: []IO{}, Outputs: []IO{}}
	mutations := map[string]func(*TaskPlan){
		"script":             func(v *TaskPlan) { v.Script = "other" },
		"env":                func(v *TaskPlan) { v.Env = map[string]string{"SECRET": "changed"} },
		"environment digest": func(v *TaskPlan) { v.EnvDigest = "changed" },
		"params":             func(v *TaskPlan) { v.Params = []ParamPlan{{Name: "option", Value: "changed"}} },
		"executable":         func(v *TaskPlan) { v.ExecutableSHA256 = "changed" },
		"replace":            func(v *TaskPlan) { v.Replace = true },
		"condition":          func(v *TaskPlan) { v.When = "changed" },
		"input source":       func(v *TaskPlan) { v.Inputs = []IO{{Source: "other"}} },
		"scatter spec":       func(v *TaskPlan) { v.ScatterMemberSpecs = []Path{{Opaque: "other", Literal: true}} },
	}
	for name, change := range mutations {
		t.Run(name, func(t *testing.T) {
			next := base
			change(&next)
			if ReviewStep(base).Fingerprint == ReviewStep(next).Fingerprint {
				t.Fatal("hidden field omitted")
			}
		})
	}
}

func TestFastQCReviewRejectsUnrepresentedFields(t *testing.T) {
	read := IO{Name: "reads", Path: "sample.fastq", Spec: Path{Base: "sample", Ext: ".fastq"}}
	html := IO{Name: "html", Path: "work/fastqc/sample_fastqc.html", Spec: Path{Dir: "work/fastqc", Base: "sample_fastqc", Ext: ".html"}}
	zip := IO{Name: "zip", Path: "work/fastqc/sample_fastqc.zip", Spec: Path{Dir: "work/fastqc", Base: "sample_fastqc", Ext: ".zip"}}
	base := TaskPlan{ID: "fastqc", Name: "fastqc", Command: modulecommand.FastQC(read.Path, "work/fastqc", 2), Image: modulecommand.FastQCImage, Resources: ResourcePlan{2, "1g"}, Inputs: []IO{read}, Outputs: []IO{html, zip}}
	if ReviewStep(base).Recipe != "fastqc-v1" {
		t.Fatal("canonical FastQC rejected")
	}
	for _, mutate := range []func(*TaskPlan){func(v *TaskPlan) { v.Env = map[string]string{"MODE": "other"} }, func(v *TaskPlan) { v.Replace = true }, func(v *TaskPlan) { v.Script = "wrapper" }, func(v *TaskPlan) { v.ExecutablePath = "other" }, func(v *TaskPlan) { v.When = "conditional" }} {
		next := base
		mutate(&next)
		if ReviewStep(next).Recipe != "" {
			t.Fatal("unrepresented behavior qualified")
		}
	}
}
