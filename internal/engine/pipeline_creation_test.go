package engine

import (
	"testing"

	"github.com/HahyeonJeon/gobble/internal/modulecommand"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

func creationDocument() Document {
	io := func(name, p string) IO { return IO{Name: name, Path: p, Spec: Path{Literal: true, Opaque: p}} }
	read := pipelinereview.CreationReadPath
	trimmed := "work/trim-galore/sample_trimmed.fq.gz"
	tr := TaskPlan{ID: "trim", Name: "trim_galore", Command: modulecommand.Trim(modulecommand.TrimOptions{Read1: read, OutDir: "work/trim-galore", Prefix: "sample", Cores: 1, Quality: 25, Length: 40}), Image: modulecommand.TrimImage, Resources: ResourcePlan{4, "2g"}, Inputs: []IO{io("read1", read)}, Outputs: []IO{io("trimmed_read1", trimmed), io("report1", "work/trim-galore/reads.fastq.gz_trimming_report.txt")}}
	qc := TaskPlan{ID: "qc", Name: "fastqc", Command: modulecommand.FastQC(trimmed, "work/fastqc", 2), Image: modulecommand.FastQCImage, Resources: ResourcePlan{2, "1g"}, Inputs: []IO{io("reads", trimmed)}, Outputs: []IO{io("html", "work/fastqc/sample_trimmed_fastqc.html"), io("zip", "work/fastqc/sample_trimmed_fastqc.zip")}}
	return Document{Name: "read_quality", Tasks: []TaskPlan{tr, qc}, Edges: []Edge{{FromPort: "reads", ToTask: "trim", ToPort: "read1", Wait: []string{read}}, {FromTask: "trim", FromPort: "trimmed_read1", ToTask: "qc", ToPort: "reads", Wait: []string{trimmed}}}}
}
func TestCreationRejectsCompleteTaskAndTopologyMutations(t *testing.T) {
	if gaps := QualifyCreation(creationDocument(), pipelinereview.CreationReadPath); len(gaps) != 0 {
		t.Fatal(gaps)
	}
	mutations := map[string]func(*Document){
		"hidden step":    func(d *Document) { d.Tasks = append(d.Tasks, TaskPlan{ID: "hidden"}) },
		"env":            func(d *Document) { d.Tasks[0].Env = map[string]string{"MODE": "other"} },
		"env digest":     func(d *Document) { d.Tasks[0].EnvDigest = "opaque" },
		"script":         func(d *Document) { d.Tasks[0].Script = "wrapper" },
		"extra command":  func(d *Document) { d.Tasks[0].Command = append(d.Tasks[0].Command, "--quiet") },
		"resource":       func(d *Document) { d.Tasks[0].Resources.CPU = 8 },
		"image":          func(d *Document) { d.Tasks[1].Image = "other" },
		"control":        func(d *Document) { d.Tasks[0].When = "maybe" },
		"params":         func(d *Document) { d.Tasks[0].Params = []ParamPlan{{Name: "extra", Value: "1"}} },
		"executable":     func(d *Document) { d.Tasks[0].ExecutablePath = "hidden" },
		"replace":        func(d *Document) { d.Tasks[0].Replace = true },
		"scatter":        func(d *Document) { d.Tasks[0].ScatterMemberSpecs = []Path{{Opaque: "hidden"}} },
		"input source":   func(d *Document) { d.Tasks[0].Inputs[0].Source = "hidden.fastq" },
		"report":         func(d *Document) { d.Tasks[0].Outputs[1].Path = "other.txt" },
		"connection":     func(d *Document) { d.Edges[1].FromPort = "report1" },
		"wait":           func(d *Document) { d.Edges[1].Wait = append(d.Edges[1].Wait, "hidden") },
		"duplicate edge": func(d *Document) { d.Edges[1] = d.Edges[0] },
		"module":         func(d *Document) { d.Tasks[1].Module = "hidden" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			d := creationDocument()
			mutate(&d)
			if len(QualifyCreation(d, pipelinereview.CreationReadPath)) == 0 {
				t.Fatal("unsupported behavior qualified")
			}
		})
	}
}
