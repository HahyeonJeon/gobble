package assets

import (
	"github.com/HahyeonJeon/gobble"
)

// bwaImage is copied from tests/wgs-e2e/wgs_e2e_thin_test.go.
// Do not import that package.
const bwaImage = "quay.io/biocontainers/bwa:0.7.18--h577a1d6_2"

const bwaIndexTaskName = "bwa_index"

var bwaIndexMemberNames = []string{"amb", "ann", "bwt", "pac", "sa"}

// BWAIndexOptions are typed bwa index settings. ExtraArgs are argv
// tokens appended after named flags and before the FASTA path.
//
// bwa index has no thread flag. Resources.CPU is not copied into Command.
type BWAIndexOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
}

// BWAIndexPorts are the declared bwa index Group output.
type BWAIndexPorts struct {
	Index gobble.Handle
}

// AddBWAIndex records one bwa index task on parent. Index siblings are
// a Group. The parent folder is PathSpec.Dir, not a directory port.
// The shared builder does not call AddInput.
func AddBWAIndex(parent Parent, fasta gobble.Handle, opts BWAIndexOptions) BWAIndexPorts {
	return addBWAIndex(parent, fasta, opts)
}

// BWAIndexPipeline returns a standalone bwa index pipeline. It AddInputs
// fasta, then calls the same builder as AddBWAIndex.
func BWAIndexPipeline(fasta gobble.PathSpec, opts BWAIndexOptions) *gobble.Pipeline {
	return Standalone("bwa-index", []Input{{Name: "fasta", Spec: fasta}}, func(parent Parent, hs []gobble.Handle) {
		addBWAIndex(parent, hs[0], opts)
	})
}

func addBWAIndex(parent Parent, fasta gobble.Handle, opts BWAIndexOptions) BWAIndexPorts {
	cmd := []string{"bwa", "index"}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)
	if path, err := CommandPath(fasta.Spec()); err == nil {
		cmd = append(cmd, path)
	}

	task := AddTask(parent, gobble.TaskSpec{
		Name:      bwaIndexTaskName,
		Command:   cmd,
		Image:     bwaImage,
		Inputs:    []gobble.Bind{{Name: "fasta", From: fasta}},
		Outputs:   []gobble.Bind{{Name: "index", Group: bwaIndexGroup(fasta.Spec())}},
		Resources: opts.Resources,
	})
	return BWAIndexPorts{Index: task.Out("index")}
}

func bwaIndexGroup(fasta gobble.PathSpec) gobble.Group {
	g := make(gobble.Group, 0, len(bwaIndexMemberNames))
	for _, name := range bwaIndexMemberNames {
		g = append(g, gobble.Member{Name: name, Spec: fasta.Append("." + name)})
	}
	return g
}

func bwaIndexGroupFrom() gobble.Group {
	g := make(gobble.Group, 0, len(bwaIndexMemberNames))
	for _, name := range bwaIndexMemberNames {
		g = append(g, gobble.Member{Name: name})
	}
	return g
}
