package workflowcase

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("workflow-case")
	readsR1 := p.AddInput("reads_r1", gobble.PathSpec{
		Dir:    gobble.Dir("in"),
		Prefix: "sample_S1_L001_R1_",
		Base:   "001",
		Ext:    ".fastq.gz",
	})
	readsR2 := p.AddInput("reads_r2", gobble.PathSpec{
		Dir:    gobble.Dir("in"),
		Prefix: "sample_S1_L001_R2_",
		Base:   "001",
		Ext:    ".fastq.gz",
	})

	prep := p.AddModule("prep")
	fastp := addFastp(prep, readsR1, readsR2)

	call := p.AddModule("call")
	align := call.Branch("align")
	qc := call.Branch("qc")
	join := call.Merge("join")

	bam := gobble.PathSpec{
		Dir:      gobble.Dir("work/align"),
		Base:     "sample",
		Suffixes: []string{"sorted"},
		Ext:      ".bam",
	}
	bwa := align.AddTask(gobble.TaskSpec{
		Name:    "bwa",
		Command: []string{"bwa"},
		Image:   "example/bwa:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: fastp.Out("clean_r1")},
			{Name: "r2", From: fastp.Out("clean_r2")},
		},
		Outputs: []gobble.Bind{
			{Name: "bam", Spec: bam},
			{Name: "bai", Spec: bam.AppendExt(".bai")},
		},
	})

	fastqc := qc.AddTask(gobble.TaskSpec{
		Name:    "fastqc",
		Command: []string{"fastqc"},
		Image:   "example/fastqc:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: fastp.Out("clean_r1")},
			{Name: "r2", From: fastp.Out("clean_r2")},
		},
		Outputs: []gobble.Bind{
			{Name: "html", Spec: gobble.Literal("sample_clean_fastqc.html").WithDir(gobble.Dir("work/qc"))},
		},
	})

	join.AddTask(gobble.TaskSpec{
		Name:    "report",
		Command: []string{"report"},
		Inputs: []gobble.Bind{
			{Name: "bam", From: bwa.Out("bam")},
			{Name: "bai", From: bwa.Out("bai")},
			{Name: "html", From: fastqc.Out("html")},
		},
		Outputs: []gobble.Bind{
			{Name: "summary", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "report", Ext: ".json"}},
		},
	})
	return p
}

func addFastp(prep *gobble.Module, r1, r2 gobble.Handle) *gobble.Task {
	return prep.AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Image:   "example/fastp:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs: []gobble.Bind{
			{
				Name: "clean_r1",
				Spec: gobble.PathSpec{
					Dir:      gobble.Dir("work/prep"),
					Prefix:   "sample_S1_L001_R1_",
					Base:     "001",
					Suffixes: []string{"clean"},
					Ext:      ".fastq.gz",
				},
			},
			{
				Name: "clean_r2",
				Spec: gobble.PathSpec{
					Dir:      gobble.Dir("work/prep"),
					Prefix:   "sample_S1_L001_R2_",
					Base:     "001",
					Suffixes: []string{"clean"},
					Ext:      ".fastq.gz",
				},
			},
		},
	})
}
