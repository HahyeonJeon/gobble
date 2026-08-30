// Package sampleretentionmapped owns one mapped-read retention gate.
package sampleretentionmapped

import (
	"fmt"
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 coreutils image resolved for
// linux/amd64. The pinned image contains mawk for parsing STAR's final log.
const DefaultImage modules.Image = "community.wave.seqera.io/library/coreutils_grep_gzip_lbzip2_pruned:838ba80435a629f8@sha256:63c2c6b22e83b2f656e88fbb1553e595da4e9e58794e3bfcb98b20b3837f328a"

// Options controls one mapped-read retention command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the accepted mapping percentage prerequisite.
type Ports struct{ Accepted gobble.Handle }

// Add records one gate over STAR's uniquely mapped read percentage.
func Add(parent modules.Parent, starLog, prerequisite gobble.Handle, minimum float64, options Options) (Ports, error) {
	const unit = "sample_retention_mapped"
	if minimum < 0 || minimum > 100 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "minimum mapped percent must be between 0 and 100")
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "the mapped-read retention gate does not accept ExtraArgs")
	}
	logPath, err := modules.HandlePath(unit, starLog)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/sample-retention-mapped")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "mapped_reads"
	}
	accepted := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".accepted.txt"}
	acceptedPath, err := accepted.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "mapped-read retention output path is invalid")
	}
	script := fmt.Sprintf(`mapped=$(awk -F'|' '/Uniquely mapped reads %%/ {v=$2; gsub(/[ %%]/, "", v); print v; exit}' %s)
test -n "$mapped"
awk -v mapped="$mapped" -v minimum=%s 'BEGIN {exit !(mapped >= minimum)}'
printf '%%s\n' "$mapped" > %s`, modules.ShellQuote(logPath), strconv.FormatFloat(minimum, 'g', -1, 64), modules.ShellQuote(acceptedPath))
	base := options.Options
	base.ExtraArgs = nil
	_, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, []string{"sh", "-c"}, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	inputs := []gobble.Bind{{Name: "star_log", From: starLog}}
	if !prerequisite.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "prerequisite", From: prerequisite})
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources,
		Inputs: inputs, Outputs: []gobble.Bind{{Name: "accepted", Spec: accepted}},
	})
	return Ports{Accepted: task.Out("accepted")}, nil
}

// Pipeline returns a standalone validated mapped-read retention module.
func Pipeline(starLog gobble.PathSpec, minimum float64, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("sample-retention-mapped", []modules.Input{{Name: "star_log", Spec: starLog}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], gobble.Handle{}, minimum, options)
		return err
	})
}
