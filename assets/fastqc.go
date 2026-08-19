package assets

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
)

// fastqcImage is the nf-core biocontainers pin from modules/nf-core/fastqc.
const fastqcImage = "quay.io/biocontainers/fastqc:0.12.1--hdfd78af_0"

const fastqcTaskName = "fastqc"

// FastQCOptions are typed FastQC settings. ExtraArgs are argv tokens
// appended after named flags.
//
// --threads copies Resources.CPU when CPU is at least 1, as an integer.
// Resources.Memory is not copied into FastQC --memory.
type FastQCOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// FastQCPorts are the declared FastQC regular-file outputs.
type FastQCPorts struct {
	HTML gobble.Handle
	Zip  gobble.Handle
}

// AddFastQC records one FastQC task on parent. The local task name is
// fastqc. Instance identity is the parent module name. The shared
// builder does not call AddInput.
func AddFastQC(parent Parent, reads gobble.Handle, opts FastQCOptions) FastQCPorts {
	return addFastQC(parent, reads, opts)
}

// FastQCPipeline returns a standalone FastQC pipeline. It AddInputs
// reads, then calls the same builder as AddFastQC.
func FastQCPipeline(reads gobble.PathSpec, opts FastQCOptions) *gobble.Pipeline {
	return Standalone("fastqc", []Input{{Name: "reads", Spec: reads}}, func(parent Parent, hs []gobble.Handle) {
		addFastQC(parent, hs[0], opts)
	})
}

func addFastQC(parent Parent, reads gobble.Handle, opts FastQCOptions) FastQCPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/fastqc")
	}
	stem := fastqcStem(reads.Spec())
	htmlSpec := gobble.PathSpec{Dir: outDir, Base: stem, Ext: ".html"}
	zipSpec := gobble.PathSpec{Dir: outDir, Base: stem, Ext: ".zip"}

	cmd := []string{"fastqc", "--outdir", outDir.String(), "--noextract"}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "--threads", strconv.Itoa(n))
	}
	cmd = append(cmd, mustCommandPath(reads.Spec()))
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)

	task := AddTask(parent, gobble.TaskSpec{
		Name:    fastqcTaskName,
		Command: cmd,
		Image:   fastqcImage,
		Inputs:  []gobble.Bind{{Name: "reads", From: reads}},
		Outputs: []gobble.Bind{
			{Name: "html", Spec: htmlSpec},
			{Name: "zip", Spec: zipSpec},
		},
		Resources: opts.Resources,
	})
	return FastQCPorts{HTML: task.Out("html"), Zip: task.Out("zip")}
}

// fastqcStem is the FastQC output prefix, including the _fastqc suffix.
// FastQC strips one compression suffix and one sequence suffix from the
// input basename.
func fastqcStem(spec gobble.PathSpec) string {
	path := mustCommandPath(spec)
	base := spec.Base
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	} else if path != "" {
		base = path
	}
	if base == "" {
		base = "reads"
	}
	lower := strings.ToLower(base)
	for _, suf := range []string{".gz", ".bz2", ".xz"} {
		if strings.HasSuffix(lower, suf) {
			base = base[:len(base)-len(suf)]
			lower = strings.ToLower(base)
			break
		}
	}
	for _, suf := range []string{".fastq", ".fq", ".sam", ".bam", ".txt"} {
		if strings.HasSuffix(lower, suf) {
			base = base[:len(base)-len(suf)]
			break
		}
	}
	return base + "_fastqc"
}

func threadCount(cpu float64) int {
	if cpu < 1 {
		return 0
	}
	return int(cpu)
}
