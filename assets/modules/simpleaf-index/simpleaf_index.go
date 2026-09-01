// Package simpleafindex owns one Simpleaf index command.
package simpleafindex

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/scrnaseq 4.2.0 SIMPLEAF_INDEX image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/simpleaf:0.19.5--ha6fb395_0@sha256:3e2971957942246f54d8fe55b43a6dfae242a641114805f20aca147a687d73f9"

var protectedExtraArgs = []string{
	"--threads", "-t",
	"--ref-seq", "--refseq",
	"--fasta", "-f", "--gtf", "-g", "--gff3-format", "--rlen", "-r",
	"--dedup", "--spliced", "--unspliced", "--feature-csv", "--probe-csv",
	"--decoy-paths", "--work-dir",
	"--output", "-o",
	"--no-piscem", "--use-piscem", "--sparse", "-p", "--use-selective-alignment",
}

// Options controls one transcript-reference Simpleaf index.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports contains the complete produced Simpleaf index directory Tree.
type Ports struct{ Index gobble.Handle }

// ProtectedExtraArg returns the first Simpleaf index option that competes with
// the module-owned inputs, output, resources, mapper route, or filesystem state.
// It follows the pinned Simpleaf 0.19.5 IndexOpts aliases.
func ProtectedExtraArg(extraArgs []string) string {
	return modules.MatchProtectedExtraArg(extraArgs, protectedExtraArgs)
}

// Add records Simpleaf path setup and one index subcommand. The path setup is
// required by the benchmark image and does not select another graph stage.
func Add(parent modules.Parent, transcriptFASTA gobble.Handle, options Options) (Ports, error) {
	const unit = "simpleaf_index"
	transcriptPath, err := modules.HandlePath(unit, transcriptFASTA)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/scrnaseq/reference/simpleaf_index")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "8g"}
	}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protectedExtraArgs); err != nil {
		return Ports{}, err
	}
	command := []string{"simpleaf", "index", "--threads", strconv.Itoa(modules.ThreadCount(resources.CPU)), "--ref-seq", transcriptPath, "--output", outDir.String()}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protectedExtraArgs)
	if err != nil {
		return Ports{}, err
	}
	script := "'simpleaf' 'set-paths'\n" + modules.ShellCommand(command)
	indexDir := outDir.Join("index")
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources,
		Env:     map[string]string{"ALEVIN_FRY_HOME": "."},
		Inputs:  []gobble.Bind{{Name: "transcript_fasta", From: transcriptFASTA}},
		Outputs: []gobble.Bind{{Name: "index", Tree: gobble.DeclareTree(indexDir)}},
	})
	return Ports{Index: task.Out("index")}, nil
}

// Pipeline returns a standalone validated Simpleaf index module.
func Pipeline(transcriptFASTA gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("simpleaf-index", []modules.Input{{Name: "transcript_fasta", Spec: transcriptFASTA}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
