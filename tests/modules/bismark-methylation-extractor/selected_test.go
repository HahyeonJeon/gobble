package bismarkmethylationextractor_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkmethylationextractor "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedBismarkMethylationExtractorStandalone(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".bam"}
	p := bismarkmethylationextractor.Pipeline(bam, true, bismarkmethylationextractor.Options{
		Options:        modules.Options{Resources: gobble.Resources{CPU: 6}},
		ExcludeOverlap: true,
		IgnoreR1:       2,
		IgnoreR2:       3,
		CoverageCutoff: 1,
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "bismark_methylation_extractor")
	if task.Image != string(bismarkmethylationextractor.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, bismarkmethylationextractor.DefaultImage)
	}
	if !pc.ContainsAll(task.Command,
		"bismark_methylation_extractor", "in/sample.bam", "--comprehensive", "--paired-end", "--no_overlap",
		"--ignore", "2", "--ignore_r2", "3", "--cutoff", "1", "--multicore", "2",
	) {
		t.Fatalf("command = %#v, want selected methylation extractor argv", task.Command)
	}
	for name, path := range map[string]string{
		"cpg": "work/bismark-methylation-extractor/CpG_context_sample.txt.gz",
		"chg": "work/bismark-methylation-extractor/CHG_context_sample.txt.gz",
		"chh": "work/bismark-methylation-extractor/CHH_context_sample.txt.gz",
	} {
		pc.AssertIOPath(t, task.Outputs, name, path)
	}
}

func TestSelectedBismarkMethylationExtractorNestedModule(t *testing.T) {
	p := gobble.NewPipeline("assay")
	h := p.AddInput("bam", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".bam"})
	ports, err := bismarkmethylationextractor.Add(p.AddModule("methylation"), h, true, bismarkmethylationextractor.Options{})
	if err != nil || ports.CpG.IsZero() || ports.CHG.IsZero() || ports.CHH.IsZero() {
		t.Fatalf("Add selected methylation extractor = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "methylation.bismark_methylation_extractor")
	if task.Module != "methylation" || task.Image != string(bismarkmethylationextractor.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
