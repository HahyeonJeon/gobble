package engine

import (
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/modulecommand"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

// ReviewStep hashes the complete internal task, including fields omitted from
// public Plan JSON. Recognition verifies exact canonical argv, never a label.
// Unknown task fields automatically affect fingerprints and added-step checks.
func ReviewStep(t TaskPlan) pipelinereview.Step {
	out := pipelinereview.Step{ID: t.ID, Fingerprint: pipelinereview.Digest(t), Residual: pipelinereview.Digest(t), Settings: []pipelinereview.Setting{}}
	if o, ok := reviewedTrim(t); ok {
		out.Recipe = "trim-galore-v1"
		value := func(n int) *int {
			if n == 0 {
				return nil
			}
			return &n
		}
		out.Settings = []pipelinereview.Setting{{Key: "quality", Label: "Quality threshold", Unit: "Phred", Value: value(o.Quality)}, {Key: "length", Label: "Minimum length", Unit: "bp", Value: value(o.Length)}}
		o.Quality, o.Length = 0, 0
		t.Command = modulecommand.Trim(o)
		out.Residual = pipelinereview.Digest(t)
	} else if reviewedFastQC(t) {
		out.Recipe = "fastqc-v1"
	}
	return out
}

func reviewedTrim(t TaskPlan) (modulecommand.TrimOptions, bool) {
	o := modulecommand.TrimOptions{}
	if t.Script != "" || t.Image != modulecommand.TrimImage || t.Resources.CPU != 4 || t.Resources.Memory != "2g" || len(t.Inputs) != 1 || len(t.Outputs) != 2 || t.Name != "trim_galore" || !simpleReviewIO(t.Inputs[0]) || !simpleReviewIO(t.Outputs[0]) || !simpleReviewIO(t.Outputs[1]) {
		return o, false
	}
	if t.Inputs[0].Name != "read1" || t.Outputs[0].Name != "trimmed_read1" || t.Outputs[1].Name != "report1" {
		return o, false
	}
	const suffix = "_trimmed.fq.gz"
	output := path.Base(t.Outputs[0].Path)
	if !strings.HasSuffix(output, suffix) {
		return o, false
	}
	o = modulecommand.TrimOptions{Read1: t.Inputs[0].Path, OutDir: path.Dir(t.Outputs[0].Path), Prefix: strings.TrimSuffix(output, suffix), Cores: 1}
	// Parse only the two supported options in their canonical position. Compare
	// the entire regenerated argv below; any extra argument or wrapper fails.
	if len(t.Command) < 9 {
		return o, false
	}
	args := t.Command[8 : len(t.Command)-1]
	for len(args) > 0 {
		if len(args) < 2 {
			return o, false
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			return o, false
		}
		switch args[0] {
		case "--quality":
			if o.Quality != 0 {
				return o, false
			}
			o.Quality = n
		case "--length":
			if o.Length != 0 {
				return o, false
			}
			o.Length = n
		default:
			return o, false
		}
		args = args[2:]
	}
	return o, reflect.DeepEqual(t.Command, modulecommand.Trim(o))
}

func simpleReviewIO(b IO) bool {
	if (b.Kind != "" && b.Kind != "file") || b.Path == "" || (b.Source != "" && b.Source != b.Path) || b.Spec.BadLit {
		return false
	}
	// Reject unrepresented IO behavior, including future nonzero fields.
	expected := IO{Name: b.Name, Kind: b.Kind, Path: b.Path, Source: b.Source, Spec: b.Spec}
	if !reflect.DeepEqual(b, expected) {
		return false
	}
	resolved, err := b.Spec.Render()
	return err == nil && resolved == b.Path
}

func reviewedFastQC(t TaskPlan) bool {
	if t.Name != "fastqc" || t.Image != modulecommand.FastQCImage || t.Resources.CPU != 2 || t.Resources.Memory != "1g" || len(t.Inputs) != 1 || len(t.Outputs) != 2 {
		return false
	}
	for _, b := range append(append([]IO{}, t.Inputs...), t.Outputs...) {
		if !simpleReviewIO(b) {
			return false
		}
	}
	read := t.Inputs[0]
	html, zip := t.Outputs[0], t.Outputs[1]
	dir := path.Dir(html.Path)
	stem := modulecommand.FastQCStem(read.Path)
	if read.Name != "reads" || html.Name != "html" || zip.Name != "zip" || html.Path != path.Join(dir, stem+".html") || zip.Path != path.Join(dir, stem+".zip") {
		return false
	}
	if t.Backend != "" && t.Backend != "local" {
		return false
	}
	expected := TaskPlan{ID: t.ID, Name: "fastqc", Module: t.Module, Display: t.Display, Command: modulecommand.FastQC(read.Path, dir, 2), Image: modulecommand.FastQCImage, Backend: t.Backend, Resources: ResourcePlan{2, "1g"}, Inputs: t.Inputs, Outputs: t.Outputs}
	return reflect.DeepEqual(t, expected)
}
