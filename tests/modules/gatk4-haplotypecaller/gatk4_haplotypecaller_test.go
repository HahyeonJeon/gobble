package gatk4haplotypecaller_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	gatk4haplotypecaller "github.com/HahyeonJeon/gobble/assets/modules/gatk4-haplotypecaller"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestHaplotypeCallerOwnsOneIndexedGVCFCommand(t *testing.T) {
	pipeline := haplotypePipeline(nil)
	task := cc.Task(t, pipeline, "intervals.gatk4_haplotypecaller")
	if !strings.Contains(task.Script, "'gatk' 'HaplotypeCaller'") || !strings.Contains(task.Script, "'--emit-ref-confidence' 'GVCF'") || !strings.Contains(task.Script, "--intervals \"$interval\"") || !strings.Contains(task.Script, "--output \"$output\"") {
		t.Fatalf("HaplotypeCaller script = %q, want one interval gVCF command", task.Script)
	}
	if len(task.Outputs) != 2 || task.Outputs[0].Name != "gvcf" || task.Outputs[1].Name != "tbi" {
		t.Fatalf("HaplotypeCaller outputs = %#v, want gvcf and tbi", task.Outputs)
	}
}

func TestHaplotypeCallerRejectsEveryOutputPrefix(t *testing.T) {
	for _, arg := range []string{"--o", "--out", "--output", "--OUTPUT=value"} {
		_, err := gobble.Compose(haplotypePipeline([]string{arg}))
		var composeErr *gobble.Error
		if !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue || composeErr.Defects[0].Unit != "gatk4_haplotypecaller" {
			t.Fatalf("ExtraArgs %q error = %#v, want structured protected-option defect", arg, err)
		}
	}
}

func haplotypePipeline(extra []string) *gobble.Pipeline {
	p := gobble.NewPipeline("haplotype")
	bam := p.AddInput("bam", file("in", "sample", ".bam"))
	bai := p.AddInput("bai", file("in", "sample", ".bam.bai"))
	fasta := p.AddInput("fasta", file("in", "genome", ".fasta"))
	fai := p.AddInput("fai", file("in", "genome", ".fasta.fai"))
	dict := p.AddInput("dict", file("in", "genome", ".dict"))
	dbsnp := p.AddInput("dbsnp", file("in", "dbsnp", ".vcf.gz"))
	dbsnpTBI := p.AddInput("dbsnp_tbi", file("in", "dbsnp", ".vcf.gz.tbi"))
	intervals := p.AddInputGroup("intervals", gobble.Group{{Name: "interval_001", Spec: file("in/intervals", "interval_001", ".bed")}, {Name: "interval_002", Spec: file("in/intervals", "interval_002", ".bed")}})
	_, err := gatk4haplotypecaller.Add(p.Scatter("intervals").From(intervals), bam, bai, fasta, fai, dict, dbsnp, dbsnpTBI, intervals, gatk4haplotypecaller.Options{IntervalDir: gobble.Dir("in/intervals"), OutDir: gobble.Dir("work/calls"), Options: moduleOptions(extra)})
	if err != nil {
		p.RecordComposeError(err)
	}
	return p
}

func file(dir, base, ext string) gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir(dir), Base: base, Ext: ext}
}
