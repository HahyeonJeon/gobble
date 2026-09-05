package samtoolsmerge_test

import (
	"reflect"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	samtoolsmerge "github.com/HahyeonJeon/gobble/assets/modules/samtools-merge"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func mergePipeline(extra []string) *gobble.Pipeline {
	var bams, indexes []gobble.PathSpec
	for _, name := range []string{"lane1", "lane2"} {
		bams = append(bams, gobble.PathSpec{Dir: gobble.Dir("in"), Base: name, Ext: ".bam"})
		indexes = append(indexes, gobble.PathSpec{Dir: gobble.Dir("in"), Base: name, Ext: ".bam.bai"})
	}
	return samtoolsmerge.Pipeline(bams, indexes, samtoolsmerge.Options{Options: modules.Options{ExtraArgs: extra}})
}

func TestMergeUsesSupportedSamtoolsOptions(t *testing.T) {
	task := cc.Task(t, mergePipeline([]string{"--no-PG"}), "samtools_merge")
	// samtools 1.24 has only short -f / -o options; --force / --output fail.
	want := []string{"samtools", "merge", "-f", "-o", "work/samtools-merge/merged.bam", "--threads", "1", "--no-PG", "in/lane1.bam", "in/lane2.bam"}
	if !reflect.DeepEqual(task.Command, want) {
		t.Fatalf("merge argv = %q, want %q", task.Command, want)
	}
}

func TestMergeProtectsOutputAndResourceOptions(t *testing.T) {
	for _, arg := range []string{"-f", "-o", "-oother.bam", "-@", "-@8", "--threads=8", "--th=8", "--write-index"} {
		t.Run(arg, func(t *testing.T) { cc.Invalid(t, mergePipeline([]string{arg})) })
	}
}
