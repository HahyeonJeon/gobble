// Package sampleretentiontrimmed owns one trimmed-read retention gate.
package sampleretentiontrimmed

import (
	"fmt"
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 coreutils/gzip image resolved for
// linux/amd64. The pinned image also contains mawk for the gate expression.
const DefaultImage modules.Image = "community.wave.seqera.io/library/coreutils_grep_gzip_lbzip2_pruned:838ba80435a629f8@sha256:63c2c6b22e83b2f656e88fbb1553e595da4e9e58794e3bfcb98b20b3837f328a"

// Options controls one trimmed-read retention command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the accepted fragment count prerequisite.
type Ports struct{ Accepted gobble.Handle }

// Add records one gate that accepts a FASTQ when its first mate contains at
// least minimum complete records. One first-mate record represents one
// single-end read or one paired-end fragment.
func Add(parent modules.Parent, read1 gobble.Handle, minimum int64, options Options) (Ports, error) {
	const unit = "sample_retention_trimmed"
	if minimum < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "minimum trimmed reads must be non-negative")
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "the trimmed-read retention gate does not accept ExtraArgs")
	}
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/sample-retention-trimmed")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "trimmed_reads"
	}
	accepted := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".accepted.txt"}
	acceptedPath, err := accepted.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "trimmed-read retention output path is invalid")
	}
	script := fmt.Sprintf(`count=$(gzip -cd %s | awk 'END {if (NR %% 4 != 0) exit 2; print NR / 4}')
awk -v count="$count" -v minimum=%s 'BEGIN {exit !(count >= minimum)}'
printf '%%s\n' "$count" > %s`, modules.ShellQuote(read1Path), strconv.FormatInt(minimum, 10), modules.ShellQuote(acceptedPath))
	base := options.Options
	base.ExtraArgs = nil
	_, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, []string{"sh", "-c"}, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "read1", From: read1}},
		Outputs: []gobble.Bind{{Name: "accepted", Spec: accepted}},
	})
	return Ports{Accepted: task.Out("accepted")}, nil
}

// Pipeline returns a standalone validated trimmed-read retention module.
func Pipeline(read1 gobble.PathSpec, minimum int64, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("sample-retention-trimmed", []modules.Input{{Name: "read1", Spec: read1}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], minimum, options)
		return err
	})
}
