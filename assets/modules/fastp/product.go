package fastp

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 FastP image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/fastp:1.1.0--08aa7c5662a30d57@sha256:6babb3aed64bb6e594df3b499dbc8696888fc1e06df4934b9505e801819475ec"

// Options controls one lifted paired-end FastP command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains prepared reads and FastP reports.
type Ports struct {
	Read1 gobble.Handle
	Read2 gobble.Handle
	JSON  gobble.Handle
	HTML  gobble.Handle
}

// Add records one validated paired-end FastP command.
func Add(parent modules.Parent, read1, read2 gobble.Handle, options Options) (Ports, error) {
	const unit = "fastp"
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	read2Path, err := modules.HandlePath(unit, read2)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/fastp")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "reads"
	}
	prepared1 := gobble.PathSpec{Dir: outDir, Base: prefix + "_R1", Ext: ".fastp.fastq.gz"}
	prepared2 := gobble.PathSpec{Dir: outDir, Base: prefix + "_R2", Ext: ".fastp.fastq.gz"}
	json := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".fastp.json"}
	html := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".fastp.html"}
	paths := make([]string, 4)
	for i, spec := range []gobble.PathSpec{prepared1, prepared2, json, html} {
		paths[i], err = spec.Render()
		if err != nil {
			return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "FastP output path is invalid")
		}
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "4g"}
	}
	command := []string{"fastp", "--in1", read1Path, "--in2", read2Path, "--out1", paths[0], "--out2", paths[1], "--json", paths[2], "--html", paths[3], "--detect_adapter_for_pe"}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "--thread", strconv.Itoa(n))
	}
	protected := []string{"--in1", "--in2", "--out1", "--out2", "--json", "--html", "--detect_adapter_for_pe", "--thread", "--interleaved_in", "--stdout"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "read1", From: read1}, {Name: "read2", From: read2}},
		Outputs: []gobble.Bind{{Name: "prepared_read1", Spec: prepared1}, {Name: "prepared_read2", Spec: prepared2}, {Name: "json", Spec: json}, {Name: "html", Spec: html}},
	})
	return Ports{Read1: task.Out("prepared_read1"), Read2: task.Out("prepared_read2"), JSON: task.Out("json"), HTML: task.Out("html")}, nil
}

// ProductPipeline returns a standalone validated lifted FastP module.
func ProductPipeline(read1, read2 gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("fastp", []modules.Input{{Name: "read1", Spec: read1}, {Name: "read2", Spec: read2}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
