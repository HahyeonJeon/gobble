// Package featurecountsmergematrices owns one Python featureCounts-matrix merge command.
package featurecountsmergematrices

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Python image selected by nf-core/atacseq 2.1.2 local utilities.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.8.3@sha256:4965e8f9078ba50c7148d49dcbc41c1827f21cb74329013deeca366204f0e317"

const mergeScript = `
import sys
output, *paths = sys.argv[1:]
metadata = None
count_headers = []
count_columns = []
for path in paths:
    with open(path) as handle:
        rows = [line.rstrip("\n").split("\t") for line in handle if line.strip() and not line.startswith("#")]
    if not rows or len(rows[0]) < 7:
        raise SystemExit("invalid featureCounts matrix: " + path)
    header, body = rows[0], rows[1:]
    current_metadata = [row[:6] for row in body]
    if metadata is None:
        metadata = current_metadata
    elif current_metadata != metadata:
        raise SystemExit("featureCounts matrices have different consensus membership")
    count_headers.extend(header[6:])
    count_columns.append([row[6:] for row in body])
with open(output, "w") as out:
    out.write("\t".join(["Geneid", "Chr", "Start", "End", "Strand", "Length"] + count_headers) + "\n")
    for index, row in enumerate(metadata):
        counts = []
        for matrix in count_columns:
            counts.extend(matrix[index])
        out.write("\t".join(row + counts) + "\n")
`

// Options controls one matrix merge.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the final mixed-mode-safe featureCounts matrix.
type Ports struct{ Counts gobble.Handle }

// Add records one strict merge over one or more same-consensus matrices.
func Add(parent modules.Parent, matrices []gobble.Handle, options Options) (Ports, error) {
	const unit = "featurecounts_merge_matrices"
	if len(matrices) == 0 || len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "matrix membership must not be empty and ExtraArgs are unsupported")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/featurecounts")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "consensus"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".featureCounts.txt"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "merged featureCounts path is invalid")
	}
	command := []string{"python3", "-c", mergeScript, outputPath}
	inputs := make([]gobble.Bind, len(matrices))
	for i, matrix := range matrices {
		path, pathErr := modules.HandlePath(unit, matrix)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, path)
		inputs[i] = gobble.Bind{Name: "matrix_" + strconv.Itoa(i), From: matrix}
	}
	base := options.Options
	base.ExtraArgs = nil
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "counts", Spec: output}}})
	return Ports{Counts: task.Out("counts")}, nil
}
