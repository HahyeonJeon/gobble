package scrnaseqevidence_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	engineexec "github.com/HahyeonJeon/gobble/internal/engine/exec"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

const fixtureSheet = "testdata/scrnaseq-samplesheet.csv"

func TestParseTypedPairedRepeatedRunsAndMetadata(t *testing.T) {
	samples := loadSamples(t)
	if len(samples) != 2 || len(samples[0].Runs) != 1 || len(samples[1].Runs) != 2 {
		t.Fatalf("Load() = %#v, want Sample_X and repeated-run Sample_Y", samples)
	}
	if samples[1].Runs[0].ID != "run_001" || samples[1].Runs[1].ID != "run_002" || samples[1].ExpectedCells != 5000 || samples[1].SeqCenter != "CRG Barcelona" {
		t.Fatalf("typed repeated-run metadata = %#v", samples[1])
	}
}

func TestParserRejectsMissingMatesMetadataConflictsAndUnknownColumns(t *testing.T) {
	tests := []struct {
		name  string
		sheet string
		unit  string
	}{
		{name: "missing mate", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,\n", unit: "fastq_2"},
		{name: "metadata conflict", sheet: "sample,fastq_1,fastq_2,expected_cells\nA,in/a1.fastq.gz,in/a2.fastq.gz,10\nA,in/b1.fastq.gz,in/b2.fastq.gz,20\n", unit: "A"},
		{name: "duplicate pair", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,in/b.fastq.gz\nA,in/a.fastq.gz,in/b.fastq.gz\n", unit: "A"},
		{name: "same path for both mates", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,in/a.fastq.gz\n", unit: "fastq_2"},
		{name: "rendered path alias between mates", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,in/./a.fastq.gz\n", unit: "fastq_2"},
		{name: "path reused across runs", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,in/b.fastq.gz\nA,in/a.fastq.gz,in/c.fastq.gz\n", unit: "fastq_1"},
		{name: "rendered path alias across runs", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,in/b.fastq.gz\nA,in/./a.fastq.gz,in/c.fastq.gz\n", unit: "fastq_1"},
		{name: "path reused across mate roles", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,in/b.fastq.gz\nA,in/c.fastq.gz,in/a.fastq.gz\n", unit: "fastq_2"},
		{name: "rendered path alias across samples", sheet: "sample,fastq_1,fastq_2\nA,in/a.fastq.gz,in/b.fastq.gz\nB,in/c.fastq.gz,in/./b.fastq.gz\n", unit: "fastq_2"},
		{name: "unknown column", sheet: "sample,fastq_1,fastq_2,chemistry\nA,in/a.fastq.gz,in/b.fastq.gz,custom\n", unit: "samplesheet"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := scrnaseq.Parse(strings.NewReader(test.sheet))
			requireDefect(t, err, gobble.DefectInvalidSampleSheet, test.unit)
		})
	}
}

func TestBuildRejectsMateAndRunPathAliasing(t *testing.T) {
	tests := []struct {
		name    string
		unit    string
		samples []scrnaseq.Sample
	}{
		{
			name: "same path for both mates",
			unit: "A",
			samples: []scrnaseq.Sample{{Name: "A", Runs: []scrnaseq.Run{
				{ID: "run_001", Fastq1: "in/a.fastq.gz", Fastq2: "in/a.fastq.gz"},
			}}},
		},
		{
			name: "rendered path alias between mates",
			unit: "A",
			samples: []scrnaseq.Sample{{Name: "A", Runs: []scrnaseq.Run{
				{ID: "run_001", Fastq1: "in/a.fastq.gz", Fastq2: "in/./a.fastq.gz"},
			}}},
		},
		{
			name: "path reused across runs",
			unit: "A",
			samples: []scrnaseq.Sample{{Name: "A", Runs: []scrnaseq.Run{
				{ID: "run_001", Fastq1: "in/a.fastq.gz", Fastq2: "in/b.fastq.gz"},
				{ID: "run_002", Fastq1: "in/a.fastq.gz", Fastq2: "in/c.fastq.gz"},
			}}},
		},
		{
			name: "rendered path alias across runs",
			unit: "A",
			samples: []scrnaseq.Sample{{Name: "A", Runs: []scrnaseq.Run{
				{ID: "run_001", Fastq1: "in/a.fastq.gz", Fastq2: "in/b.fastq.gz"},
				{ID: "run_002", Fastq1: "in/./a.fastq.gz", Fastq2: "in/c.fastq.gz"},
			}}},
		},
		{
			name: "path reused across mate roles",
			unit: "A",
			samples: []scrnaseq.Sample{{Name: "A", Runs: []scrnaseq.Run{
				{ID: "run_001", Fastq1: "in/a.fastq.gz", Fastq2: "in/b.fastq.gz"},
				{ID: "run_002", Fastq1: "in/c.fastq.gz", Fastq2: "in/a.fastq.gz"},
			}}},
		},
		{
			name: "path reused across samples",
			unit: "B",
			samples: []scrnaseq.Sample{
				{Name: "A", Runs: []scrnaseq.Run{{ID: "run_001", Fastq1: "in/a.fastq.gz", Fastq2: "in/b.fastq.gz"}}},
				{Name: "B", Runs: []scrnaseq.Run{{ID: "run_001", Fastq1: "in/c.fastq.gz", Fastq2: "in/b.fastq.gz"}}},
			},
		},
		{
			name: "rendered path alias across samples",
			unit: "B",
			samples: []scrnaseq.Sample{
				{Name: "A", Runs: []scrnaseq.Run{{ID: "run_001", Fastq1: "in/a.fastq.gz", Fastq2: "in/b.fastq.gz"}}},
				{Name: "B", Runs: []scrnaseq.Run{{ID: "run_001", Fastq1: "in/c.fastq.gz", Fastq2: "in/./b.fastq.gz"}}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			graph, err := gobble.Compose(scrnaseq.Build(test.samples, scrnaseq.DefaultConfig()))
			if graph != nil {
				t.Fatalf("Compose() graph = %v, want nil", graph)
			}
			requireDefect(t, err, gobble.DefectInvalidSampleSheet, test.unit)
		})
	}
}

func TestBuildRejectsCrossRoleRenderedPathAliasing(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		mutate func([]scrnaseq.Sample, *scrnaseq.Config)
	}{
		{
			name: "FASTA and GTF",
			unit: "reference.annotation",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.Annotation = gobble.PathSpec{Dir: gobble.Dir("in/reference/."), Base: "genome", Ext: ".fa"}
			},
		},
		{
			name: "FASTA and whitelist",
			unit: "reference.whitelist",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.BarcodeWhitelist.Path = gobble.PathSpec{Dir: gobble.Dir("in/reference/."), Base: "genome", Ext: ".fa"}
			},
		},
		{
			name: "FASTA and read",
			unit: "reference.fasta",
			mutate: func(samples []scrnaseq.Sample, _ *scrnaseq.Config) {
				samples[0].Runs[0].Fastq1 = "in/reference/./genome.fa"
			},
		},
		{
			name: "GTF and whitelist",
			unit: "reference.whitelist",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.BarcodeWhitelist.Path = gobble.PathSpec{Dir: gobble.Dir("in/reference/."), Base: "genes", Ext: ".gtf"}
			},
		},
		{
			name: "GTF and read",
			unit: "reference.annotation",
			mutate: func(samples []scrnaseq.Sample, _ *scrnaseq.Config) {
				samples[0].Runs[0].Fastq1 = "in/reference/./genes.gtf"
			},
		},
		{
			name: "whitelist and read",
			unit: "reference.whitelist",
			mutate: func(samples []scrnaseq.Sample, _ *scrnaseq.Config) {
				samples[0].Runs[0].Fastq1 = "in/reference/./10x_V2_barcode_whitelist.txt.gz"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			samples := loadSamples(t)
			config := scrnaseq.DefaultConfig()
			test.mutate(samples, &config)
			graph, err := gobble.Compose(scrnaseq.Build(samples, config))
			if graph != nil {
				t.Fatalf("Compose() graph = %v, want nil", graph)
			}
			requireDefect(t, err, gobble.DefectInvalidValue, test.unit)
		})
	}
}

func TestBuildRejectsReadyReferencePathAliasing(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		mutate func([]scrnaseq.Sample, *scrnaseq.Config)
	}{
		{
			name: "transcript-to-gene and whitelist",
			unit: "reference.transcript_to_gene",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.TranscriptToGene = config.Reference.BarcodeWhitelist.Path
			},
		},
		{
			name: "index Tree root and whitelist",
			unit: "reference.simpleaf_index",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.SimpleafIndex = gobble.DeclareTree(gobble.Dir("in/reference/./10x_V2_barcode_whitelist.txt.gz"))
			},
		},
		{
			name: "literal transcript-to-gene aliases literal whitelist",
			unit: "reference.transcript_to_gene",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.BarcodeWhitelist.Path = gobble.Literal("shared.txt")
				config.Reference.TranscriptToGene = gobble.Literal("shared.txt")
			},
		},
		{
			name: "index Tree root aliases literal whitelist",
			unit: "reference.simpleaf_index",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.BarcodeWhitelist.Path = gobble.Literal("shared")
				config.Reference.SimpleafIndex = gobble.DeclareTree(gobble.Dir("shared"))
			},
		},
		{
			name: "literal transcript-to-gene aliases read",
			unit: "reference.transcript_to_gene",
			mutate: func(samples []scrnaseq.Sample, config *scrnaseq.Config) {
				samples[0].Runs[0].Fastq1 = "shared.fastq.gz"
				config.Reference.TranscriptToGene = gobble.Literal("shared.fastq.gz")
			},
		},
		{
			name: "ready form mixes literal FASTA",
			unit: "reference",
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.FASTA = gobble.Literal("genome.fa")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			samples := loadSamples(t)
			config := scrnaseq.DefaultConfig()
			config.Reference.FASTA = gobble.PathSpec{}
			config.Reference.Annotation = gobble.PathSpec{}
			config.Reference.SimpleafIndex = gobble.DeclareTree(gobble.Dir("in/reference/simpleaf-index"))
			config.Reference.TranscriptToGene = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "t2g", Ext: ".tsv"}
			test.mutate(samples, &config)
			graph, err := gobble.Compose(scrnaseq.Build(samples, config))
			if graph != nil {
				t.Fatalf("Compose() graph = %v, want nil", graph)
			}
			requireDefect(t, err, gobble.DefectInvalidValue, test.unit)
		})
	}
}

func TestBuildRejectsReadyIndexTreeFilePathOverlap(t *testing.T) {
	tests := []struct {
		name   string
		tree   gobble.Directory
		mutate func([]scrnaseq.Sample, *scrnaseq.Config)
	}{
		{
			name: "index Tree contains transcript-to-gene",
			tree: gobble.Dir("in/ready-index"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.TranscriptToGene = gobble.PathSpec{Dir: gobble.Dir("in/ready-index"), Base: "t2g", Ext: ".tsv"}
			},
		},
		{
			name: "transcript-to-gene contains index Tree",
			tree: gobble.Dir("in/ready-index/tree"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.TranscriptToGene = gobble.PathSpec{Dir: gobble.Dir("in"), Base: "ready-index"}
			},
		},
		{
			name: "index Tree contains whitelist",
			tree: gobble.Dir("in/reference/index"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.BarcodeWhitelist.Path = gobble.PathSpec{Dir: gobble.Dir("in/reference/index"), Base: "whitelist", Ext: ".txt.gz"}
			},
		},
		{
			name: "whitelist contains index Tree",
			tree: gobble.Dir("in/reference/index/tree"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.BarcodeWhitelist.Path = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "index"}
			},
		},
		{
			name: "index Tree contains FASTA",
			tree: gobble.Dir("in/ready-index"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.FASTA = gobble.PathSpec{Dir: gobble.Dir("in/ready-index"), Base: "genome", Ext: ".fa"}
			},
		},
		{
			name: "FASTA contains index Tree",
			tree: gobble.Dir("in/ready-index/tree"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.FASTA = gobble.PathSpec{Dir: gobble.Dir("in"), Base: "ready-index"}
			},
		},
		{
			name: "index Tree contains GTF",
			tree: gobble.Dir("in/ready-index"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.Annotation = gobble.PathSpec{Dir: gobble.Dir("in/ready-index"), Base: "genes", Ext: ".gtf"}
			},
		},
		{
			name: "GTF contains index Tree",
			tree: gobble.Dir("in/ready-index/tree"),
			mutate: func(_ []scrnaseq.Sample, config *scrnaseq.Config) {
				config.Reference.Annotation = gobble.PathSpec{Dir: gobble.Dir("in"), Base: "ready-index"}
			},
		},
		{
			name: "index Tree contains read",
			tree: gobble.Dir("in/ready-index"),
			mutate: func(samples []scrnaseq.Sample, _ *scrnaseq.Config) {
				samples[0].Runs[0].Fastq1 = "in/ready-index/read.fastq.gz"
			},
		},
		{
			name: "read contains index Tree",
			tree: gobble.Dir("in/ready-index/tree"),
			mutate: func(samples []scrnaseq.Sample, _ *scrnaseq.Config) {
				samples[0].Runs[0].Fastq1 = "in/ready-index"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			samples := loadSamples(t)
			config := scrnaseq.DefaultConfig()
			config.Reference.FASTA = gobble.PathSpec{}
			config.Reference.Annotation = gobble.PathSpec{}
			config.Reference.SimpleafIndex = gobble.DeclareTree(test.tree)
			config.Reference.TranscriptToGene = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "t2g", Ext: ".tsv"}
			test.mutate(samples, &config)
			graph, err := gobble.Compose(scrnaseq.Build(samples, config))
			if graph != nil {
				t.Fatalf("Compose() graph = %v, want nil", graph)
			}
			requireDefect(t, err, gobble.DefectInvalidValue, "reference.simpleaf_index")
		})
	}
}

func TestBuildRealizesSelectedSimpleafVerticalWithoutRuntimeScatter(t *testing.T) {
	pipeline := scrnaseq.Build(loadSamples(t), scrnaseq.DefaultConfig())
	if _, err := gobble.Compose(pipeline); err != nil {
		var structured *gobble.Error
		if errors.As(err, &structured) {
			t.Fatalf("Compose defects = %#v", structured.Defects)
		}
		t.Fatal(err)
	}
	raw := pc.MustPlanJSON(t, pipeline)
	tasks := pc.AllTasks(t, raw)
	for name, want := range map[string]int{
		"gtf_gene_filter": 1, "gffread_transcriptome": 1, "gtf_to_t2g": 1,
		"simpleaf_index": 1, "fastqc": 6, "cat_fastq": 2,
		"simpleaf_quant": 2, "qcatch": 2, "matrix_to_h5ad": 2,
		"anndatar_convert": 2, "h5ad_concat": 1, "multiqc": 1,
	} {
		if got := pc.CountTasksNamed(tasks, name); got != want {
			t.Errorf("%s count = %d, want %d", name, got, want)
		}
	}
	for _, id := range []string{
		"reference.gtf_gene_filter", "reference.gffread_transcriptome",
		"reference.gtf_to_t2g", "reference.simpleaf_index",
		"Sample_X.run_001.raw_fastqc_r1.fastqc",
		"Sample_Y.consolidate_r1.cat_fastq", "Sample_Y.consolidate_r2.cat_fastq",
		"Sample_X.simpleaf_quant", "Sample_X.qcatch", "Sample_X.matrix_to_h5ad",
		"Sample_X.anndatar_convert", "cohort.h5ad_concat", "multiqc",
	} {
		pc.MustHaveTaskID(t, tasks, id)
	}
	pc.AssertTreeIO(t, pc.TaskByID(t, raw, "reference.simpleaf_index").Outputs, "index", "results/scrnaseq/reference/simpleaf_index/index")
	quant := pc.TaskByID(t, raw, "Sample_X.simpleaf_quant")
	pc.AssertTreeIO(t, quant.Inputs, "index", "results/scrnaseq/reference/simpleaf_index/index")
	pc.AssertTreeIO(t, quant.Outputs, "map", "results/scrnaseq/samples/Sample_X/simpleaf/af_map")
	pc.AssertTreeIO(t, quant.Outputs, "quant", "results/scrnaseq/samples/Sample_X/simpleaf/af_quant")
	if !strings.Contains(quant.Script, "'--chemistry' '10xv2'") || !strings.Contains(quant.Script, "'--unfiltered-pl'") || strings.Contains(quant.Script, "cellranger") {
		t.Fatalf("Simpleaf V2 raw-quant command = %q", quant.Script)
	}
	qcatch := pc.TaskByID(t, raw, "Sample_X.qcatch")
	pc.AssertTreeIO(t, qcatch.Inputs, "quant", "results/scrnaseq/samples/Sample_X/simpleaf/af_quant")
	if !pc.ContainsAll(qcatch.Command, "--chemistry", "10X_3p_v2", "--save_filtered_h5ad", "--export_summary_table") {
		t.Fatalf("QCatch command = %#v", qcatch.Command)
	}
	pc.AssertIOPath(t, qcatch.Outputs, "filtered_h5ad", "results/scrnaseq/samples/Sample_X/qcatch/filtered_quants.h5ad")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "Sample_X.matrix_to_h5ad").Outputs, "h5ad", "results/scrnaseq/matrices/Sample_X/Sample_X_raw_matrix.h5ad")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "Sample_X.anndatar_convert").Outputs, "seurat_rds", "results/scrnaseq/matrices/Sample_X/Sample_X_raw_matrix.seurat.rds")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "cohort.h5ad_concat").Outputs, "h5ad", "results/scrnaseq/matrices/combined_raw_matrix.h5ad")
	for _, task := range tasks {
		if task.Scatter != "" {
			t.Errorf("scRNA task %s has runtime Scatter %q; cells and samples are not runtime members", task.ID, task.Scatter)
		}
	}
	pc.AssertNoTaskName(t, tasks, "cellbender", "cellranger", "star", "kallisto")
	multiQC := pc.TaskByID(t, raw, "multiqc")
	treeInputs := 0
	for _, input := range multiQC.Inputs {
		if input.Kind == "tree" {
			treeInputs++
		}
	}
	if treeInputs != 2 {
		t.Fatalf("MultiQC quantification Tree inputs = %d, want both samples", treeInputs)
	}
}

func TestProtocolReadyTreeAndQCatchSettingsAreExplicit(t *testing.T) {
	config := scrnaseq.DefaultConfig()
	config.Protocol = scrnaseq.Protocol10xV4
	config.Reference.BarcodeWhitelist.Protocol = scrnaseq.Protocol10xV4
	config.Reference.BarcodeWhitelist.Path = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "v4", Ext: ".txt.gz"}
	config.QCatch.Chemistry = "10X_3p_v4"
	raw := pc.MustPlanJSON(t, scrnaseq.Build(loadSamples(t), config))
	if script := pc.TaskByID(t, raw, "Sample_X.simpleaf_quant").Script; !strings.Contains(script, "'--chemistry' '10xv4-3p'") {
		t.Fatalf("typed V4 protocol absent from Simpleaf command: %q", script)
	}

	ready := scrnaseq.DefaultConfig()
	ready.Reference.FASTA = gobble.PathSpec{}
	ready.Reference.Annotation = gobble.PathSpec{}
	ready.Reference.SimpleafIndex = gobble.DeclareTree(gobble.Dir("in/reference/simpleaf-index"))
	ready.Reference.TranscriptToGene = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "t2g", Ext: ".tsv"}
	readyRaw := pc.MustPlanJSON(t, scrnaseq.Build(loadSamples(t), ready))
	readyTasks := pc.AllTasks(t, readyRaw)
	if pc.CountTasksNamed(readyTasks, "simpleaf_index") != 0 || pc.CountTasksNamed(readyTasks, "gtf_gene_filter") != 0 {
		t.Fatal("ready Simpleaf Tree did not suppress source reference normalization")
	}
	pc.AssertTreeIO(t, pc.TaskByID(t, readyRaw, "Sample_X.simpleaf_quant").Inputs, "index", "in/reference/simpleaf-index")

	v1 := scrnaseq.DefaultConfig()
	v1.Protocol = scrnaseq.Protocol10xV1
	v1.Reference.BarcodeWhitelist.Protocol = scrnaseq.Protocol10xV1
	v1.Reference.BarcodeWhitelist.Path.Base = "10x_V1_barcode_whitelist"
	v1.QCatch.Chemistry = ""
	v1.QCatch.NPartitions = 100000
	v1Raw := pc.MustPlanJSON(t, scrnaseq.Build(loadSamples(t), v1))
	if command := pc.TaskByID(t, v1Raw, "Sample_X.qcatch").Command; !pc.ContainsAll(command, "--n_partitions", "100000") || contains(command, "--chemistry") {
		t.Fatalf("typed V1 QCatch partition command = %#v", command)
	}
}

func TestUnsupportedProtocolsIncompleteReferenceAndProtectedAliasesFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		mutate func(*scrnaseq.Config)
	}{
		{name: "custom chemistry", unit: "protocol", mutate: func(c *scrnaseq.Config) { c.Protocol = "custom"; c.Reference.BarcodeWhitelist.Protocol = "custom" }},
		{name: "whitelist mismatch", unit: "reference.whitelist", mutate: func(c *scrnaseq.Config) { c.Reference.BarcodeWhitelist.Protocol = scrnaseq.Protocol10xV3 }},
		{name: "mixed ready tree", unit: "reference", mutate: func(c *scrnaseq.Config) {
			c.Reference.SimpleafIndex = gobble.DeclareTree(gobble.Dir("in/index"))
			c.Reference.TranscriptToGene = gobble.PathSpec{Base: "t2g", Ext: ".tsv"}
		}},
		{name: "V1 missing partitions", unit: "qcatch", mutate: func(c *scrnaseq.Config) {
			c.Protocol = scrnaseq.Protocol10xV1
			c.Reference.BarcodeWhitelist.Protocol = scrnaseq.Protocol10xV1
			c.QCatch.Chemistry = ""
		}},
		{name: "V2 partition override", unit: "qcatch", mutate: func(c *scrnaseq.Config) { c.QCatch.NPartitions = 10 }},
		{name: "quant index long", unit: "simpleaf_quant", mutate: func(c *scrnaseq.Config) { c.SimpleafQuant.ExtraArgs = []string{"--index=elsewhere"} }},
		{name: "quant output short", unit: "simpleaf_quant", mutate: func(c *scrnaseq.Config) { c.SimpleafQuant.ExtraArgs = []string{"-oelsewhere"} }},
		{name: "index no piscem route", unit: "simpleaf_index", mutate: func(c *scrnaseq.Config) { c.SimpleafIndex.ExtraArgs = []string{"--no-piscem"} }},
		{name: "index selective alignment route", unit: "simpleaf_index", mutate: func(c *scrnaseq.Config) { c.SimpleafIndex.ExtraArgs = []string{"--use-selective-alignment"} }},
		{name: "quant no piscem route", unit: "simpleaf_quant", mutate: func(c *scrnaseq.Config) { c.SimpleafQuant.ExtraArgs = []string{"--no-piscem"} }},
		{name: "quant selective alignment route", unit: "simpleaf_quant", mutate: func(c *scrnaseq.Config) { c.SimpleafQuant.ExtraArgs = []string{"--use-selective-alignment"} }},
		{name: "quant generic aligner route", unit: "simpleaf_quant", mutate: func(c *scrnaseq.Config) { c.SimpleafQuant.ExtraArgs = []string{"--aligner=star"} }},
		{name: "qcatch input short", unit: "qcatch", mutate: func(c *scrnaseq.Config) { c.QCatch.ExtraArgs = []string{"-iother"} }},
		{name: "cat extra input", unit: "cat_fastq", mutate: func(c *scrnaseq.Config) { c.Consolidate.ExtraArgs = []string{"in/other.fastq.gz"} }},
		{name: "publication", unit: "publication", mutate: func(c *scrnaseq.Config) { c.Publication.CombinedH5AD = false }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := scrnaseq.DefaultConfig()
			test.mutate(&config)
			graph, err := gobble.Compose(scrnaseq.Build(loadSamples(t), config))
			if graph != nil {
				t.Fatalf("Compose() graph = %v, want nil", graph)
			}
			requireDefect(t, err, gobble.DefectInvalidValue, test.unit)
		})
	}
}

func TestBuildRejectsEverySimpleafIndexOwnedOptionAlias(t *testing.T) {
	tests := []struct {
		name  string
		extra string
	}{
		{name: "threads long", extra: "--threads=1"},
		{name: "threads short", extra: "-t1"},
		{name: "direct reference long", extra: "--ref-seq=other.fa"},
		{name: "direct reference long alias", extra: "--refseq=other.fa"},
		{name: "FASTA long", extra: "--fasta=other.fa"},
		{name: "FASTA short", extra: "-fother.fa"},
		{name: "GTF long", extra: "--gtf=other.gtf"},
		{name: "GTF short", extra: "-gother.gtf"},
		{name: "GFF3 format", extra: "--gff3-format"},
		{name: "read length long", extra: "--rlen=100"},
		{name: "read length short", extra: "-r100"},
		{name: "deduplicate", extra: "--dedup"},
		{name: "spliced sequence", extra: "--spliced=other.fa"},
		{name: "unspliced sequence", extra: "--unspliced=other.fa"},
		{name: "feature CSV", extra: "--feature-csv=features.csv"},
		{name: "probe CSV", extra: "--probe-csv=probes.csv"},
		{name: "output long", extra: "--output=elsewhere"},
		{name: "output short", extra: "-oelsewhere"},
		{name: "disable piscem", extra: "--no-piscem"},
		{name: "select piscem", extra: "--use-piscem"},
		{name: "sparse salmon index long", extra: "--sparse"},
		{name: "sparse salmon index short", extra: "-p"},
		{name: "selective alignment route", extra: "--use-selective-alignment"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := scrnaseq.DefaultConfig()
			config.SimpleafIndex.ExtraArgs = []string{test.extra}
			graph, err := gobble.Compose(scrnaseq.Build(loadSamples(t), config))
			if graph != nil {
				t.Fatalf("Compose() with ExtraArgs %q returned a graph, want nil", test.extra)
			}
			requireDefect(t, err, gobble.DefectInvalidValue, "simpleaf_index")
		})
	}
}

func TestBuildRejectsEverySimpleafQuantOwnedOptionAlias(t *testing.T) {
	tests := []struct {
		name  string
		extra string
	}{
		{name: "mapped directory", extra: "--map-dir=other"},
		{name: "index long", extra: "--index=other"},
		{name: "index short", extra: "-iother"},
		{name: "transcript relation long", extra: "--t2g-map=other.tsv"},
		{name: "transcript relation short", extra: "-mother.tsv"},
		{name: "chemistry long", extra: "--chemistry=10xv4-3p"},
		{name: "chemistry short", extra: "-c10xv4-3p"},
		{name: "read one long", extra: "--reads1=other.fastq.gz"},
		{name: "read one short", extra: "-1other.fastq.gz"},
		{name: "read two long", extra: "--reads2=other.fastq.gz"},
		{name: "read two short", extra: "-2other.fastq.gz"},
		{name: "resolution long", extra: "--resolution=parsimony"},
		{name: "resolution short", extra: "-rparsimony"},
		{name: "output long", extra: "--output=elsewhere"},
		{name: "output short", extra: "-oelsewhere"},
		{name: "threads long", extra: "--threads=1"},
		{name: "threads short", extra: "-t1"},
		{name: "anndata output", extra: "--anndata-out"},
		{name: "knee filtering long", extra: "--knee"},
		{name: "knee filtering short", extra: "-k"},
		{name: "forced cells long", extra: "--forced-cells=100"},
		{name: "forced cells short", extra: "-f100"},
		{name: "explicit permit list long", extra: "--explicit-pl=other.txt"},
		{name: "explicit permit list short", extra: "-xother.txt"},
		{name: "expected cells long", extra: "--expect-cells=100"},
		{name: "expected cells short", extra: "-e100"},
		{name: "unfiltered permit list long", extra: "--unfiltered-pl=other.txt"},
		{name: "unfiltered permit list short", extra: "-uother.txt"},
		{name: "expected orientation long", extra: "--expected-ori=both"},
		{name: "expected orientation short", extra: "-dboth"},
		{name: "minimum reads", extra: "--min-reads=1"},
		{name: "disable piscem", extra: "--no-piscem"},
		{name: "select piscem", extra: "--use-piscem"},
		{name: "selective alignment long", extra: "--use-selective-alignment"},
		{name: "selective alignment short", extra: "-s"},
		{name: "generic aligner route", extra: "--aligner=star"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := scrnaseq.DefaultConfig()
			config.SimpleafQuant.ExtraArgs = []string{test.extra}
			graph, err := gobble.Compose(scrnaseq.Build(loadSamples(t), config))
			if graph != nil {
				t.Fatalf("Compose() with ExtraArgs %q returned a graph, want nil", test.extra)
			}
			requireDefect(t, err, gobble.DefectInvalidValue, "simpleaf_quant")
		})
	}
}

func TestIncompleteReadyTreeFailsBeforeWorkspaceMutation(t *testing.T) {
	workspace := t.TempDir()
	for _, rel := range []string{"in/reference/simpleaf-index/member", "in/reference/t2g.tsv", "in/reference/10x_V2_barcode_whitelist.txt.gz", "in/reads/r1.fastq.gz", "in/reads/r2.fastq.gz"} {
		path := filepath.Join(workspace, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(rel+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	config := scrnaseq.DefaultConfig()
	config.Reference.FASTA = gobble.PathSpec{}
	config.Reference.Annotation = gobble.PathSpec{}
	config.Reference.SimpleafIndex = gobble.DeclareTree(gobble.Dir("in/reference/simpleaf-index"))
	config.Reference.TranscriptToGene = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "t2g", Ext: ".tsv"}
	graph, err := gobble.Compose(scrnaseq.Build([]scrnaseq.Sample{{Name: "sample", Runs: []scrnaseq.Run{{ID: "run_001", Fastq1: "in/reads/r1.fastq.gz", Fastq2: "in/reads/r2.fastq.gz"}}}}, config))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/tests/pipelines/scrnaseq")
	if err != nil {
		t.Fatal(err)
	}
	original := engineexec.DockerCLI
	dockerCalls := 0
	engineexec.DockerCLI = func(context.Context, []string, []string, io.Writer, io.Writer) (int, error) {
		dockerCalls++
		return 1, errors.New("unexpected Docker execution")
	}
	t.Cleanup(func() { engineexec.DockerCLI = original })
	err = gobble.Run(t.Context(), graph, workspace, 1, gobble.WithIdentity(identity))
	if !hasDefect(err, gobble.DefectMissingInput, "simpleaf_quant.index") {
		t.Fatalf("Run() error = %v, want incomplete ready Tree input", err)
	}
	if dockerCalls != 0 {
		t.Fatalf("Run() made %d Docker calls before rejecting incomplete Tree", dockerCalls)
	}
	if _, statErr := os.Lstat(filepath.Join(workspace, ".gobble")); !os.IsNotExist(statErr) {
		t.Fatalf("Run() created engine state before rejecting incomplete Tree: %v", statErr)
	}
}

func TestBuildCopiesCallerDataDefaultsAreFreshAndAdapterMatches(t *testing.T) {
	samples := loadSamples(t)
	config := scrnaseq.DefaultConfig()
	pipeline := scrnaseq.Build(samples, config)
	want := pc.MustPlanJSON(t, pipeline)
	samples[0].Runs[0].Fastq1 = "changed.fastq.gz"
	config.SimpleafQuant.ExtraArgs = append(config.SimpleafQuant.ExtraArgs, "--changed")
	config.H5ADConcat.Labels = append(config.H5ADConcat.Labels, "changed")
	if got := pc.MustPlanJSON(t, pipeline); !bytes.Equal(got, want) {
		t.Fatal("Build retained caller-owned mutable data")
	}
	first := scrnaseq.DefaultConfig()
	first.QCatch.ExtraArgs = append(first.QCatch.ExtraArgs, "--verbose")
	if second := scrnaseq.DefaultConfig(); len(second.QCatch.ExtraArgs) != 0 || second.Protocol != scrnaseq.Protocol10xV2 || second.UMIResolution != scrnaseq.ResolutionCRLike {
		t.Fatalf("DefaultConfig retained mutation or lost explicit defaults: %#v", second)
	}
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/scrnaseq/build.go", "SampleSheetPath", "Load", "Open", "ReadFile", "Stat", "Lstat", "Getwd")

	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(fixtureSheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	if got := pc.MustPlanJSON(t, scrnaseq.Pipeline()); !bytes.Equal(got, want) {
		t.Fatal("Pipeline adapter differs from typed Build")
	}
	lifecycle := scrnaseq.Lifecycle
	if lifecycle.GraphGeneration != scrnaseq.GraphGeneration || !lifecycle.Design || !lifecycle.Build || !lifecycle.Customize || !lifecycle.Run || !lifecycle.Resume || !lifecycle.Stop || !lifecycle.Failure || lifecycle.PreLiftResumable {
		t.Fatalf("Lifecycle = %#v, want complete first-generation participation", lifecycle)
	}
}

func loadSamples(t *testing.T) []scrnaseq.Sample {
	t.Helper()
	samples, err := scrnaseq.Load(fixtureSheet)
	if err != nil {
		t.Fatalf("Load(%s): %v", fixtureSheet, err)
	}
	return samples
}

func requireDefect(t *testing.T, err error, code gobble.DefectCode, unit string) {
	t.Helper()
	if !hasDefect(err, code, unit) {
		t.Fatalf("error = %v, want defect code %s unit containing %q", err, code, unit)
	}
}

func hasDefect(err error, code gobble.DefectCode, unit string) bool {
	var structured *gobble.Error
	if !errors.As(err, &structured) {
		return false
	}
	for _, defect := range structured.Defects {
		if defect.Code == code && strings.Contains(defect.Unit, unit) {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
