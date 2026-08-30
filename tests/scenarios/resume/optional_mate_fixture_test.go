package resume

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// OptionalMate returns a samplesheet-driven optional-mate proof.
// Every row copies read1. A non-empty read2 also copies the mate.
// Empty read2 is single-end. Tasks are local processes.
func OptionalMate() *gobble.Pipeline {
	p := gobble.NewPipeline("optional-mate")
	sheet, err := gobble.LoadSampleSheet()
	if err != nil {
		p.RecordComposeError(err)
		return p
	}
	for _, row := range sheet.Rows {
		mod := p.AddModule(row.Sample)
		work := gobble.Dir("work/" + row.Sample)
		r1 := p.AddInput(row.Sample+"_r1", sheetFileSpec(row.Read1))
		copyProofFile(mod, "copy_r1", r1, gobble.PathSpec{Dir: work, Base: "r1", Ext: ".txt"})
		if row.Read2 == "" {
			continue
		}
		r2 := p.AddInput(row.Sample+"_r2", sheetFileSpec(row.Read2))
		copyProofFile(mod, "copy_r2", r2, gobble.PathSpec{Dir: work, Base: "r2", Ext: ".txt"})
	}
	return p
}

func copyProofFile(parent modules.Parent, name string, in gobble.Handle, out gobble.PathSpec) *gobble.Task {
	return parent.AddTask(gobble.TaskSpec{
		Name:    name,
		Command: []string{"cp", modules.MustCommandPath(in.Spec()), modules.MustCommandPath(out)},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: out}},
	})
}

func sheetFileSpec(path string) gobble.PathSpec {
	dir, file := "", path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		dir, file = path[:i], path[i+1:]
	}
	base, ext := file, ""
	lower := strings.ToLower(file)
	for _, candidate := range []string{".fastq.gz", ".fq.gz"} {
		if strings.HasSuffix(lower, candidate) {
			base = file[:len(file)-len(candidate)]
			ext = file[len(base):]
			return proofFileSpec(dir, base, ext)
		}
	}
	if i := strings.LastIndex(file, "."); i > 0 {
		base, ext = file[:i], file[i:]
	}
	return proofFileSpec(dir, base, ext)
}

func proofFileSpec(dir, base, ext string) gobble.PathSpec {
	spec := gobble.PathSpec{Base: base, Ext: ext}
	if dir != "" {
		spec.Dir = gobble.Dir(dir)
	}
	return spec
}
