package bismarkalign_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestLiftedBismarkAlignSupportsSingleAndPairedTrees(t *testing.T) {
	index := gobble.DeclareTree(gobble.Dir("in/reference/bismark-index"))
	paired := cc.Task(t, bismarkalign.Pipeline(index, gobble.Literal("in/r1.fastq.gz"), gobble.Literal("in/r2.fastq.gz"), bismarkalign.Options{Prefix: "sample"}), "bismark_align")
	if !pc.ContainsAll(paired.Command, "--bowtie2", "--genome", "in/reference/bismark-index", "-1", "in/r1.fastq.gz", "-2", "in/r2.fastq.gz") {
		t.Fatalf("paired command = %#v", paired.Command)
	}
	pc.AssertTreeIO(t, paired.Inputs, "index", "in/reference/bismark-index")
	pc.AssertIOPath(t, paired.Outputs, "bam", "work/bismark-align/sample_pe.bam")
	pc.AssertIOPath(t, paired.Outputs, "report", "work/bismark-align/sample_PE_report.txt")

	single := cc.Task(t, bismarkalign.Pipeline(index, gobble.Literal("in/read.fastq.gz"), gobble.PathSpec{}, bismarkalign.Options{Prefix: "single"}), "bismark_align")
	if pc.ContainsAll(single.Command, "-1") || pc.ContainsAll(single.Command, "-2") {
		t.Fatalf("single command = %#v, want one positional read", single.Command)
	}
	pc.AssertIOPath(t, single.Outputs, "bam", "work/bismark-align/single.bam")
	pc.AssertIOPath(t, single.Outputs, "report", "work/bismark-align/single_SE_report.txt")

	for _, extra := range [][]string{
		{"--non_directional"},
		{"--slam"},
		{"--sla"},
		{"--pba"},
		{"--minimap2"},
		{"--mm2"},
		{"--genome_folder", "in/other-index"},
		{"--genome_folder=in/other-index"},
		{"--output_dir", "work/other"},
		{"--output_dir=work/other"},
		{"--outp", "work/other"},
		{"--outpu=work/other"},
		{"-o", "work/other"},
		{"-owork/other"},
		{"--basename", "other"},
		{"--prefix=other"},
		{"-Bother"},
		{"--single_end", "in/other.fastq.gz"},
		{"--se=in/other.fastq.gz"},
		{"-f"},
		{"-un"},
		{"--ambiguous"},
		{"--ambig_bam"},
		{"--parallel", "2"},
		{"-I42"},
		{"-X=400"},
	} {
		t.Run(extra[0], func(t *testing.T) {
			cc.Invalid(t, bismarkalign.Pipeline(index, gobble.Literal("in/read.fastq.gz"), gobble.PathSpec{}, bismarkalign.Options{Options: modules.Options{ExtraArgs: extra}}))
		})
	}
}
