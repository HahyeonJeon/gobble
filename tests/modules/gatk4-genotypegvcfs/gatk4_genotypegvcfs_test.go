package gatk4genotypegvcfs_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	gatk4genotypegvcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genotypegvcfs"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestGenotypeGVCFsMapsEveryIntervalToOneTree(t *testing.T) {
	p := gobble.NewPipeline("genotype")
	intervals := p.AddInputGroup("intervals", gobble.Group{{Name: "interval_001", Spec: path("in/intervals", "interval_001", ".bed")}, {Name: "interval_002", Spec: path("in/intervals", "interval_002", ".bed")}})
	database := p.AddInputTree("database", gobble.DeclareTree(gobble.Dir("in/db")))
	fasta := p.AddInput("fasta", path("in", "genome", ".fasta"))
	fai := p.AddInput("fai", path("in", "genome", ".fasta.fai"))
	dict := p.AddInput("dict", path("in", "genome", ".dict"))
	dbsnp := p.AddInput("dbsnp", path("in", "dbsnp", ".vcf.gz"))
	dbsnpTBI := p.AddInput("dbsnp_tbi", path("in", "dbsnp", ".vcf.gz.tbi"))
	_, err := gatk4genotypegvcfs.Add(p.Scatter("intervals").From(intervals), database, intervals, fasta, fai, dict, dbsnp, dbsnpTBI, gatk4genotypegvcfs.Options{IntervalDir: gobble.Dir("in/intervals"), OutDir: gobble.Dir("work/genotype")})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	task := cc.Task(t, p, "intervals.gatk4_genotypegvcfs")
	for _, want := range []string{"'in/db'/$stem", "gendb://$database", "--intervals \"$interval\""} {
		if !strings.Contains(task.Script, want) {
			t.Errorf("GenotypeGVCFs script omits %q: %s", want, task.Script)
		}
	}
	if len(task.Outputs) != 2 || task.Outputs[0].Name != "vcf" || task.Outputs[1].Name != "tbi" {
		t.Fatalf("GenotypeGVCFs outputs = %#v, want indexed VCF", task.Outputs)
	}
}

func path(dir, base, ext string) gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir(dir), Base: base, Ext: ext}
}
