package assets

import "github.com/HahyeonJeon/gobble"

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
		mod := AddModule(p, row.Sample)
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

func copyProofFile(parent Parent, name string, in gobble.Handle, out gobble.PathSpec) *gobble.Task {
	return AddTask(parent, gobble.TaskSpec{
		Name:    name,
		Command: []string{"cp", mustCommandPath(in.Spec()), mustCommandPath(out)},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: out}},
	})
}
