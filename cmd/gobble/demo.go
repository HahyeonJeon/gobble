package main

import (
	"context"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/HahyeonJeon/gobble/internal/fixture"
)

type demoSpec struct {
	Name      string   `json:"name"`
	CPU       int      `json:"minimum_task_cpus"`
	MemoryGiB int      `json:"largest_task_memory_gib"`
	Outputs   []string `json:"expected_outputs"`
}

// These are existing product pipelines, with their existing DefaultConfig.
// Input identity/provenance remains owned by each assay's committed manifest.
var demos = []demoSpec{
	{"rnaseq", 4, 8, []string{"results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam", "results/rnaseq/matrices/gene_counts.tsv", "results/rnaseq/matrices/transcript_tpm.tsv", "results/rnaseq/multiqc/multiqc_report.html"}},
	{"wgs", 4, 6, []string{"results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam", "results/wgs/samples/patient2/testT/alignment/testT.recalibrated.bam", "results/wgs/joint/joint_germline.vcf.gz", "results/wgs/multiqc/multiqc_report.html"}},
	{"methylseq", 6, 15, []string{"results/methylseq/bismark/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bam", "results/methylseq/summary/bismark_summary_report.html", "results/methylseq/multiqc/multiqc_report.html"}},
	{"atacseq", 6, 4, []string{"results/atacseq/consensus/replicates/featurecounts/consensus.featureCounts.txt", "results/atacseq/igv/igv_session.xml", "results/atacseq/multiqc/multiqc_report.html"}},
	{"scrnaseq", 4, 8, []string{"results/scrnaseq/samples/Sample_X/qcatch/filtered_quants.h5ad", "results/scrnaseq/matrices/Sample_X/Sample_X_raw_matrix.sce.rds", "results/scrnaseq/matrices/combined_raw_matrix.h5ad", "results/scrnaseq/multiqc/multiqc_report.html"}},
}

func findDemo(name string) (demoSpec, error) {
	for _, demo := range demos {
		if demo.Name == name {
			return demo, nil
		}
	}
	return demoSpec{}, fmt.Errorf("unknown demo %q; choose rnaseq, wgs, methylseq, atacseq, or scrnaseq", name)
}

func demoFiles(source string, demo demoSpec) (map[string]string, fixture.Manifest, error) {
	dir := filepath.Join(source, "tests", "pipelines", demo.Name, "testdata")
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fixture.Manifest{}, err
	}
	manifest, err := fixture.DecodeManifest(data)
	if err != nil {
		return nil, manifest, err
	}
	sheet, err := os.ReadFile(filepath.Join(dir, demo.Name+"-samplesheet.csv"))
	if err != nil {
		return nil, manifest, err
	}
	readme := fmt.Sprintf(`# %s with official test data

This project calls Gobble's existing %s pipeline with its DefaultConfig.
Inputs are in runs/demo/in; samplesheet.csv and fixture-manifest.json record
samples and upstream data provenance. The data is for software exercises.

From this directory:

    gobble doctor
    gobble validate .
    gobble plan .
    gobble run . --workspace runs/demo --cap 1
    gobble inspect run --workspace runs/demo

Allow at least %d CPUs for a task and more than %d GiB memory in Docker's VM
(the controller and system also need memory). --cap 1 limits concurrent tasks;
it does not lower an individual task's CPU or memory request. The first run
also downloads pinned tool images, which are much larger than the test inputs.

Expected result files under runs/demo:
`, demo.Name, demo.Name, demo.CPU, demo.MemoryGiB)
	for _, output := range demo.Outputs {
		readme += "\n- " + output
	}
	readme += `

Open another terminal in this directory to monitor or stop:

    gobble watch --workspace runs/demo
    gobble stop --workspace runs/demo

Wait for Stop to report settled, then:

    gobble resume . --workspace runs/demo --cap 1

Resume retries interrupted tasks and reuses valid completed work. Keep Docker,
the pinned runtime, this project's runtime lock, and run state. If this run has
already finished, Resume should reuse its results. Keep the run terminal open.

Ask your coding agent to explain pipeline.go, the samplesheet, and the plan
before adapting anything. Inputs and reference builds must remain compatible.
For a fresh exercise, run gobble demo from the parent directory using a new
project name; never delete an active run to start over.

Full guide: https://github.com/HahyeonJeon/gobble/blob/develop/docs/tutorials.md
`
	files := map[string]string{
		"pipeline.go":     fmt.Sprintf("package pipeline\n\nimport (\n\"github.com/HahyeonJeon/gobble\"\n\"github.com/HahyeonJeon/gobble/assets/pipelines/%s\"\n)\n\nfunc Pipeline() *gobble.Pipeline { return %s.Pipeline() }\n", demo.Name, demo.Name),
		"samplesheet.csv": string(sheet), "fixture-manifest.json": string(data), "README.md": readme,
	}
	formatted, err := format.Source([]byte(files["pipeline.go"]))
	if err != nil {
		return nil, manifest, err
	}
	files["pipeline.go"] = string(formatted)
	return files, manifest, nil
}

func runDemo(req *request, stdout, stderr io.Writer) int {
	if req.pkg == "" {
		return writeJSON(stdout, stderr, "demo", map[string]any{"op": "demo", "examples": demos})
	}
	demo, err := findDemo(req.assay)
	if err != nil {
		return writeErr(stderr, invalidRequest("demo", err.Error()), 2)
	}
	if _, err := os.Lstat(req.pkg); !os.IsNotExist(err) {
		return writeErr(stderr, invalidRequest("demo", "choose a new project directory; existing files will not be replaced"), 1)
	}
	source, err := sourceCheckout()
	if err != nil {
		return writeErr(stderr, invalidRequest("demo", err.Error()), 1)
	}
	files, manifest, err := demoFiles(source, demo)
	if err != nil {
		return writeErr(stderr, invalidRequest("demo", err.Error()), 1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	// Cache outside the new project: a failed/cancelled download can be retried
	// with the same command, without exposing a half-staged runnable project.
	cache := filepath.Join(".gobble-cache", "fixtures")
	var total int64
	for i, pin := range manifest.Pins() {
		fmt.Fprintf(stderr, "Preparing %s: %d/%d %s (%d bytes)\n", demo.Name, i+1, len(manifest.Pins()), pin.Name, pin.Bytes)
		if _, err := fixture.FetchContext(ctx, cache, pin); err != nil {
			return writeErr(stderr, invalidRequest("demo", fmt.Sprintf("%v; retry the same demo command to reuse verified downloads", err)), 1)
		}
		total += pin.Bytes
	}
	err = createProject(ctx, req.pkg, source, files, func(project string) error {
		workspace := filepath.Join(project, "runs", "demo")
		if _, err := fixture.StageManifestContext(ctx, cache, workspace, manifest); err != nil {
			return err
		}
		if demo.Name == "wgs" {
			return fixture.SplitIntervals(workspace, "in/reference/genome.multi_intervals.bed", "in/reference/intervals", 2)
		}
		return nil
	})
	if err != nil {
		return writeErr(stderr, invalidRequest("demo", err.Error()), 1)
	}
	return writeJSON(stdout, stderr, "demo", map[string]any{"op": "demo", "project": req.pkg, "pipeline": demo.Name, "input_bytes": total, "workspace": "runs/demo", "expected_outputs": demo.Outputs, "next": "cd into the project, then gobble doctor, gobble plan ., and gobble run . --workspace runs/demo --cap 1"})
}
