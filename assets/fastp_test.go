package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

// Documented nf-core modules-branch sarscov2 R2. Same README and MIT
// public-reuse basis as pinSARSCoV2R1.
var pinSARSCoV2R2 = Pin{
	Name:   "test_2.fastq.gz",
	URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/sarscov2/illumina/fastq/test_2.fastq.gz",
	Bytes:  9395,
	SHA256: "0080f40cab58c7e7b85443e37de22775e3ed6b7afdef9a3271ac3147576f3027",
}

func TestFastpStandaloneComposeBuildPlan(t *testing.T) {
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	opts := FastpOptions{
		ExtraArgs: []string{"--qualified_quality_phred", "15"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := FastpPipeline(r1, r2, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "fastp")
	if task.Name != "fastp" {
		t.Fatalf("task name = %q, want fastp", task.Name)
	}
	if task.Image != fastpImage {
		t.Fatalf("image = %q, want %q", task.Image, fastpImage)
	}
	if !containsAll(task.Command,
		"fastp",
		"--in1", "in/test_1.fastq.gz",
		"--in2", "in/test_2.fastq.gz",
		"--out1", "work/fastp/test_1.clean.fastq.gz",
		"--out2", "work/fastp/test_2.clean.fastq.gz",
		"--json", "work/fastp/test_1.fastp.json",
		"--html", "work/fastp/test_1.fastp.html",
		"--detect_adapter_for_pe",
		"--thread", "2",
		"--qualified_quality_phred", "15",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--qualified_quality_phred" || task.Command[n-1] != "15" {
		t.Fatalf("extra-args last tokens = %#v, want [--qualified_quality_phred 15]", task.Command[n-2:])
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "clean_r1", "work/fastp/test_1.clean.fastq.gz")
	assertIOPath(t, task.Outputs, "clean_r2", "work/fastp/test_2.clean.fastq.gz")
	assertIOPath(t, task.Outputs, "json", "work/fastp/test_1.fastp.json")
	assertIOPath(t, task.Outputs, "html", "work/fastp/test_1.fastp.html")
}

func TestFastpNestedModule(t *testing.T) {
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "prep")
	ports := AddFastp(mod, h1, h2, FastpOptions{ExtraArgs: []string{"--disable_quality_filtering"}})
	if ports.CleanR1.IsZero() || ports.JSON.IsZero() {
		t.Fatalf("ports IsZero, want handles")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "prep.fastp")
	if task.Name != "fastp" || task.Module != "prep" {
		t.Fatalf("nested task = %+v, want name fastp module prep", task)
	}
	if !containsAll(task.Command, "--disable_quality_filtering") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
}

func TestFastpStandaloneRun(t *testing.T) {
	requireDocker(t)
	src1 := cachePin(t, pinSARSCoV2R1)
	src2 := cachePin(t, pinSARSCoV2R2)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_1.fastq.gz", src1)
	stageFile(t, dir, "in/test_2.fastq.gz", src2)
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := FastpPipeline(r1, r2, FastpOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{
		"work/fastp/test_1.clean.fastq.gz",
		"work/fastp/test_2.clean.fastq.gz",
		"work/fastp/test_1.fastp.json",
		"work/fastp/test_1.fastp.html",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}
