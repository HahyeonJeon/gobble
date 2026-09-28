package pipelinereview

import (
	"fmt"
	"strings"
)

const CreationScope = "single-end-trim-fastqc-v1"
const CreationReadPath = "inputs/reads.fastq.gz"
const CreationInputFile = "creation-input.json"

type CreationInput struct {
	SchemaVersion int    `json:"schemaVersion"`
	Path          string `json:"path"`
	ReadLayout    string `json:"readLayout"`
}

// CreationPath preserves declared compression/extension without reading data.
func CreationPath(file string) (string, error) {
	for _, suffix := range []string{".fastq.gz", ".fq.gz", ".fastq", ".fq"} {
		if strings.HasSuffix(strings.ToLower(file), suffix) {
			return "inputs/reads" + suffix, nil
		}
	}
	return "", fmt.Errorf("unsupported read descriptor")
}
func ValidCreationPath(path string) bool {
	canonical, err := CreationPath(path)
	return err == nil && canonical == path
}

const CreationSourcePath = "pipeline/pipe.go"

type ScaffoldFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
type CreationScaffold struct {
	SchemaVersion int            `json:"schemaVersion"`
	Scope         string         `json:"scope"`
	Package       string         `json:"package"`
	InputPath     string         `json:"inputPath"`
	Editable      []string       `json:"editable"`
	Files         []ScaffoldFile `json:"files"`
	Guidance      string         `json:"guidance"`
}

// ExportCreationScaffold copies dependency setup from the qualified installation,
// never from an Agent or a research Project. It performs no I/O or initialization.
func ExportCreationScaffold(mod, sum string) (CreationScaffold, error) {
	const module = "module github.com/HahyeonJeon/gobble\n"
	if !strings.HasPrefix(mod, module) || len(mod) > 64<<10 || len(sum) > 256<<10 || strings.Contains(mod, "replace ") {
		return CreationScaffold{}, fmt.Errorf("creation scaffold needs the canonical installed module setup")
	}
	mod = "module gobble.local/creation\n" + strings.TrimPrefix(mod, module) + "\nrequire github.com/HahyeonJeon/gobble v0.0.0\nreplace github.com/HahyeonJeon/gobble => /opt/gobble\n"
	return CreationScaffold{1, CreationScope, "pipeline", CreationReadPath, []string{CreationSourcePath}, []ScaffoldFile{{"go.mod", mod}, {"go.sum", sum}, {CreationSourcePath, creationSource}, {CreationInputFile, `{"schemaVersion":1,"path":"inputs/reads.fastq.gz","readLayout":"single-end"}`}}, "Edit only pipeline/pipe.go. Preserve the single reads input returned by gobble.ReadCreationInput() and the Trim Galore then FastQC topology. Supported edits: Quality and Length nonnegative integer options and descriptive stage labels. Zero means an unknown tool default. Keep all other options, outputs and setup unchanged. The input is a logical descriptor bound by the service to selected Project data; its bytes are unavailable during this check. This check never runs analysis or adopts a Pipeline."}, nil
}

const creationSource = `package pipeline

import (
 "github.com/HahyeonJeon/gobble"
 "github.com/HahyeonJeon/gobble/assets/modules"
 "github.com/HahyeonJeon/gobble/assets/modules/fastqc"
 trim "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
)

func Pipeline() *gobble.Pipeline {
 p := gobble.NewPipeline("read_quality")
 inputPath, err := gobble.ReadCreationInput()
 if err != nil { p.RecordComposeError(err); return p }
 reads := p.AddInput("reads", gobble.Literal(inputPath))
 trimmed, err := trim.Add(modules.WithDisplay(p, gobble.TaskDisplay{Stage: "Trim adapters and low-quality bases"}), reads, gobble.Handle{}, trim.Options{Quality: 0, Length: 0})
 if err != nil { p.RecordComposeError(err); return p }
 _, err = fastqc.Add(modules.WithDisplay(p, gobble.TaskDisplay{Stage: "Inspect trimmed read quality"}), trimmed.Read1, fastqc.Options{})
 if err != nil { p.RecordComposeError(err); return p }
 return p
}
`
