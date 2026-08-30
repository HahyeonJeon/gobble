package trimgalore_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestActualTrimGalore0610Filenames(t *testing.T) {
	r1, r2 := gobble.Literal("in/run_1.fastq.gz"), gobble.Literal("in/run_2.fastq.gz")
	paired := cc.Task(t, trimgalore.Pipeline(r1, r2, trimgalore.Options{Prefix: "sample"}), "trim_galore")
	pc.AssertIOPath(t, paired.Outputs, "trimmed_read1", "work/trim-galore/sample_val_1.fq.gz")
	pc.AssertIOPath(t, paired.Outputs, "trimmed_read2", "work/trim-galore/sample_val_2.fq.gz")
	pc.AssertIOPath(t, paired.Outputs, "report1", "work/trim-galore/run_1.fastq.gz_trimming_report.txt")
	pc.AssertIOPath(t, paired.Outputs, "report2", "work/trim-galore/run_2.fastq.gz_trimming_report.txt")
	single := cc.Task(t, trimgalore.Pipeline(r1, gobble.PathSpec{}, trimgalore.Options{Prefix: "sample"}), "trim_galore")
	pc.AssertIOPath(t, single.Outputs, "trimmed_read1", "work/trim-galore/sample_trimmed.fq.gz")
	pc.AssertIOPath(t, single.Outputs, "report1", "work/trim-galore/run_1.fastq.gz_trimming_report.txt")
	cc.Invalid(t, trimgalore.Pipeline(r1, gobble.PathSpec{}, trimgalore.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
