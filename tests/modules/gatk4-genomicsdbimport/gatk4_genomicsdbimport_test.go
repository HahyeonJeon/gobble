package gatk4genomicsdbimport_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	gatk4genomicsdbimport "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genomicsdbimport"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestGenomicsDBImportRequiresTwoSamplesAndReturnsTree(t *testing.T) {
	p := gobble.NewPipeline("genomicsdb")
	variants := []gatk4genomicsdbimport.Variant{
		{GVCF: p.AddInput("a", spec("a", ".g.vcf.gz")), TBI: p.AddInput("a_tbi", spec("a", ".g.vcf.gz.tbi"))},
		{GVCF: p.AddInput("b", spec("b", ".g.vcf.gz")), TBI: p.AddInput("b_tbi", spec("b", ".g.vcf.gz.tbi"))},
	}
	intervals := p.AddInputGroup("intervals", gobble.Group{
		{Name: "interval_001", Spec: spec("interval_001", ".bed")},
		{Name: "interval_002", Spec: spec("interval_002", ".bed")},
	})
	ports, err := gatk4genomicsdbimport.Add(p.Scatter("intervals").From(intervals), variants, intervals, gatk4genomicsdbimport.Options{IntervalDir: gobble.Dir("in"), OutDir: gobble.Dir("work/genomicsdb")})
	if err != nil || ports.Database.IsZero() {
		t.Fatalf("Add() = (%#v, %v), want Tree port", ports, err)
	}
	task := cc.Task(t, p, "intervals.gatk4_genomicsdbimport")
	for _, want := range []string{"GenomicsDBImport", "--variant", "in/a.g.vcf.gz", "in/b.g.vcf.gz", "--intervals \"$interval\"", "'work/genomicsdb'/$stem"} {
		if !strings.Contains(task.Script, want) {
			t.Errorf("GenomicsDBImport script omits %q: %s", want, task.Script)
		}
	}
	pc.AssertTreeIO(t, task.Outputs, "database", "work/genomicsdb")

	p = gobble.NewPipeline("invalid")
	one := []gatk4genomicsdbimport.Variant{{GVCF: p.AddInput("a", spec("a", ".g.vcf.gz")), TBI: p.AddInput("a_tbi", spec("a", ".g.vcf.gz.tbi"))}}
	intervals = p.AddInputGroup("intervals", gobble.Group{{Name: "interval_001", Spec: spec("interval_001", ".bed")}})
	_, err = gatk4genomicsdbimport.Add(p.Scatter("intervals").From(intervals), one, intervals, gatk4genomicsdbimport.Options{IntervalDir: gobble.Dir("in")})
	if err == nil {
		t.Fatal("Add(one sample) error = nil, want cohort defect")
	}
}

func spec(base, ext string) gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: base, Ext: ext}
}
