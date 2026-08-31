// Package igvsession owns one Python IGV-session generation command.
package igvsession

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Python image selected by nf-core/atacseq 2.1.2 IGV generation.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.8.3@sha256:4965e8f9078ba50c7148d49dcbc41c1827f21cb74329013deeca366204f0e317"

const sessionScript = `
import sys, xml.etree.ElementTree as ET
output, fasta, *resources = sys.argv[1:]
session = ET.Element("Session", genome=fasta, version="8")
container = ET.SubElement(session, "Resources")
for path in resources:
    ET.SubElement(container, "Resource", path=path)
ET.ElementTree(session).write(output, encoding="utf-8", xml_declaration=True)
`

// Options controls one IGV session.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports contains the IGV session XML.
type Ports struct{ XML gobble.Handle }

// Add records one strict session fan-in over every final track, peak, and consensus file.
func Add(parent modules.Parent, fasta, fai gobble.Handle, resources []gobble.Handle, options Options) (Ports, error) {
	const unit = "igv_session"
	if len(resources) == 0 || len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "IGV resources must not be empty and ExtraArgs are unsupported")
	}
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	if _, err = modules.HandlePath(unit, fai); err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/igv")
	}
	output := gobble.PathSpec{Dir: outDir, Base: "igv_session", Ext: ".xml"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "IGV output path is invalid")
	}
	command := []string{"python3", "-c", sessionScript, outputPath, fastaPath}
	inputs := []gobble.Bind{{Name: "fasta", From: fasta}, {Name: "fai", From: fai}}
	for i, resource := range resources {
		path, pathErr := modules.HandlePath(unit, resource)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, path)
		inputs = append(inputs, gobble.Bind{Name: "resource_" + strconv.Itoa(i), From: resource})
	}
	base := options.Options
	base.ExtraArgs = nil
	command, image, resourcesPolicy, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resourcesPolicy, Inputs: inputs, Outputs: []gobble.Bind{{Name: "xml", Spec: output}}})
	return Ports{XML: task.Out("xml")}, nil
}
