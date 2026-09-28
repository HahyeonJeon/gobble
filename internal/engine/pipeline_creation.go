package engine

import (
	"path"
	"reflect"

	"github.com/HahyeonJeon/gobble/internal/modulecommand"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

// QualifyCreation checks every TaskPlan field, including fields absent from flow
// JSON. It is intentionally narrower than refinement's unchanged-residual rule.
func QualifyCreation(doc Document, inputPath string) []string {
	reject := func(message string) []string { return []string{message} }
	if !pipelinereview.ValidCreationPath(inputPath) || len(doc.Tasks) != 2 || len(doc.Edges) != 2 {
		return reject("Creation supports exactly Trim Galore followed by FastQC with one read input.")
	}
	trim, qc := doc.Tasks[0], doc.Tasks[1]
	o, ok := reviewedTrim(trim)
	if !ok || o.Read1 != inputPath || o.OutDir != "work/trim-galore" || o.Prefix != "sample" || trim.Outputs[1].Path != "work/trim-galore/"+path.Base(inputPath)+"_trimming_report.txt" {
		return reject("The trimming step differs from the supported single-end design.")
	}
	expected := TaskPlan{ID: trim.ID, Name: "trim_galore", Display: trim.Display, Command: modulecommand.Trim(o), Image: modulecommand.TrimImage, Backend: trim.Backend, Resources: ResourcePlan{4, "2g"}, Inputs: trim.Inputs, Outputs: trim.Outputs}
	if (trim.Backend != "" && trim.Backend != "local") || !reflect.DeepEqual(trim, expected) {
		return reject("The trimming step contains unsupported execution or control behavior.")
	}
	if !reviewedFastQC(qc) || qc.Module != "" || qc.Inputs[0].Path != trim.Outputs[0].Path || qc.Outputs[0].Path != "work/fastqc/sample_trimmed_fastqc.html" || qc.Outputs[1].Path != "work/fastqc/sample_trimmed_fastqc.zip" {
		return reject("The quality check differs from the supported trimmed-read design.")
	}
	expectedEdges := []Edge{{FromPort: "reads", ToTask: trim.ID, ToPort: "read1", Wait: []string{inputPath}}, {FromTask: trim.ID, FromPort: "trimmed_read1", ToTask: qc.ID, ToPort: "reads", Wait: []string{trim.Outputs[0].Path}}}
	// Edge ordering has no execution meaning; multiplicity and every field do.
	remaining := map[string]int{}
	for _, e := range expectedEdges {
		remaining[pipelinereview.Digest(e)]++
	}
	for _, e := range doc.Edges {
		remaining[pipelinereview.Digest(e)]--
	}
	for _, n := range remaining {
		if n != 0 {
			return reject("The complete input or step connections differ from the supported design.")
		}
	}
	return []string{}
}
