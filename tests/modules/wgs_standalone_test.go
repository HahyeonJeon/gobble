package moduleevidence

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	bcftoolssort "github.com/HahyeonJeon/gobble/assets/modules/bcftools-sort"
	bcftoolsstats "github.com/HahyeonJeon/gobble/assets/modules/bcftools-stats"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	gatk4applybqsr "github.com/HahyeonJeon/gobble/assets/modules/gatk4-applybqsr"
	gatk4baserecalibrator "github.com/HahyeonJeon/gobble/assets/modules/gatk4-baserecalibrator"
	gatk4gatherbqsrreports "github.com/HahyeonJeon/gobble/assets/modules/gatk4-gather-bqsr-reports"
	gatk4genomicsdbimport "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genomicsdbimport"
	gatk4genotypegvcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genotypegvcfs"
	gatk4haplotypecaller "github.com/HahyeonJeon/gobble/assets/modules/gatk4-haplotypecaller"
	gatk4markduplicates "github.com/HahyeonJeon/gobble/assets/modules/gatk4-markduplicates"
	gatk4mergevcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-mergevcfs"
	"github.com/HahyeonJeon/gobble/assets/modules/mosdepth"
	samtoolsflagstat "github.com/HahyeonJeon/gobble/assets/modules/samtools-flagstat"
	samtoolsidxstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-idxstats"
	samtoolsmerge "github.com/HahyeonJeon/gobble/assets/modules/samtools-merge"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestWGSCommandModulesHaveOneTaskStandaloneAdapters(t *testing.T) {
	fasta := wgsSpec("in", "genome", ".fasta")
	fai := wgsSpec("in", "genome", ".fasta.fai")
	dict := wgsSpec("in", "genome", ".dict")
	bam := wgsSpec("in", "sample", ".bam")
	bai := wgsSpec("in", "sample", ".bam.bai")
	read1 := wgsSpec("in", "reads_R1", ".fastq.gz")
	read2 := wgsSpec("in", "reads_R2", ".fastq.gz")
	interval := wgsSpec("in/intervals", "interval_001", ".bed")
	dbsnp := wgsSpec("in", "dbsnp", ".vcf.gz")
	dbsnpTBI := wgsSpec("in", "dbsnp", ".vcf.gz.tbi")
	gvcf1 := wgsSpec("in", "a", ".g.vcf.gz")
	gvcf2 := wgsSpec("in", "b", ".g.vcf.gz")
	gvcfTBI1 := wgsSpec("in", "a", ".g.vcf.gz.tbi")
	gvcfTBI2 := wgsSpec("in", "b", ".g.vcf.gz.tbi")
	prefix := gobble.PathSpec{Dir: gobble.Dir("in/bwa"), Base: "genome"}
	index := gobble.Group{}
	for _, name := range []string{"amb", "ann", "bwt", "pac", "sa"} {
		index = append(index, gobble.Member{Name: name, Spec: prefix.AppendExt("." + name)})
	}

	tests := []struct {
		name     string
		pipeline *gobble.Pipeline
	}{
		{name: "bwa index", pipeline: bwaindex.ProductPipeline(fasta, bwaindex.Options{})},
		{name: "bwa mem", pipeline: bwamem.ProductPipeline(fasta, index, read1, read2, bwamem.Options{IndexPrefix: prefix, ReadGroup: "@RG\\tID:lane\\tSM:sample"})},
		{name: "fastp", pipeline: fastp.ProductPipeline(read1, read2, fastp.Options{})},
		{name: "samtools merge", pipeline: samtoolsmerge.Pipeline([]gobble.PathSpec{bam, wgsSpec("in", "sample2", ".bam")}, []gobble.PathSpec{bai, wgsSpec("in", "sample2", ".bam.bai")}, samtoolsmerge.Options{})},
		{name: "samtools flagstat", pipeline: samtoolsflagstat.Pipeline(bam, bai, samtoolsflagstat.Options{})},
		{name: "samtools idxstats", pipeline: samtoolsidxstats.Pipeline(bam, bai, samtoolsidxstats.Options{})},
		{name: "mosdepth", pipeline: mosdepth.Pipeline(bam, bai, fasta, mosdepth.Options{})},
		{name: "mark duplicates", pipeline: gatk4markduplicates.Pipeline(bam, bai, gatk4markduplicates.Options{})},
		{name: "base recalibrator", pipeline: gatk4baserecalibrator.Pipeline(bam, bai, fasta, fai, dict, interval, []gobble.PathSpec{dbsnp}, []gobble.PathSpec{dbsnpTBI}, gatk4baserecalibrator.Options{})},
		{name: "gather BQSR", pipeline: gatk4gatherbqsrreports.Pipeline([]gobble.PathSpec{wgsSpec("in", "interval_001", ".table"), wgsSpec("in", "interval_002", ".table")}, gatk4gatherbqsrreports.Options{})},
		{name: "apply BQSR", pipeline: gatk4applybqsr.Pipeline(bam, bai, fasta, fai, dict, wgsSpec("in/tables", "interval_001", ".table"), interval, gatk4applybqsr.Options{})},
		{name: "haplotype caller", pipeline: gatk4haplotypecaller.Pipeline(bam, bai, fasta, fai, dict, dbsnp, dbsnpTBI, interval, gatk4haplotypecaller.Options{})},
		{name: "merge VCFs", pipeline: gatk4mergevcfs.Pipeline([]gobble.PathSpec{gvcf1, gvcf2}, []gobble.PathSpec{gvcfTBI1, gvcfTBI2}, dict, gatk4mergevcfs.Options{})},
		{name: "GenomicsDB import", pipeline: gatk4genomicsdbimport.Pipeline([]gobble.PathSpec{gvcf1, gvcf2}, []gobble.PathSpec{gvcfTBI1, gvcfTBI2}, interval, gatk4genomicsdbimport.Options{})},
		{name: "genotype gVCFs", pipeline: gatk4genotypegvcfs.Pipeline(gobble.DeclareTree(gobble.Dir("in/genomicsdb")), interval, fasta, fai, dict, dbsnp, dbsnpTBI, gatk4genotypegvcfs.Options{})},
		{name: "bcftools sort", pipeline: bcftoolssort.Pipeline(wgsSpec("in/intervals", "interval_001", ".vcf.gz"), wgsSpec("in/intervals", "interval_001", ".vcf.gz.tbi"), interval, bcftoolssort.Options{})},
		{name: "bcftools stats", pipeline: bcftoolsstats.Pipeline(gvcf1, gvcfTBI1, fasta, bcftoolsstats.Options{})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			graph, err := gobble.Compose(test.pipeline)
			if err != nil {
				t.Fatalf("Compose() error = %v", err)
			}
			if ids := graph.TaskIDs(); len(ids) != 1 {
				t.Fatalf("standalone task ids = %#v, want one command task", ids)
			}
		})
	}
}

func TestWGSMarkDuplicatesDefaultUsesSarekModuleImage(t *testing.T) {
	bam := wgsSpec("in", "sample", ".bam")
	bai := wgsSpec("in", "sample", ".bam.bai")
	tasks := pc.AllTasks(t, pc.MustPlanJSON(t, gatk4markduplicates.Pipeline(bam, bai, gatk4markduplicates.Options{})))
	if len(tasks) != 1 {
		t.Fatalf("MarkDuplicates task count = %d, want 1", len(tasks))
	}
	if got, want := tasks[0].Image, string(gatk4markduplicates.DefaultImage); got != want {
		t.Fatalf("MarkDuplicates default image = %q, want Sarek module image %q", got, want)
	}
}

func TestWGSMarkDuplicatesRejectsNamedFieldShortAliases(t *testing.T) {
	bam := wgsSpec("in", "sample", ".bam")
	bai := wgsSpec("in", "sample", ".bam.bai")
	for _, arg := range []string{
		"-I", "-I=other.bam", "-Iother.bam",
		"-O", "-O=other.bam", "-Oother.bam",
		"-M", "-M=other.metrics", "-Mother.metrics",
	} {
		options := gatk4markduplicates.Options{}
		options.ExtraArgs = []string{arg}
		_, err := gobble.Compose(gatk4markduplicates.Pipeline(bam, bai, options))
		var composeErr *gobble.Error
		if !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue || composeErr.Defects[0].Unit != "gatk4_markduplicates" {
			t.Errorf("ExtraArgs %q error = %#v, want structured protected-alias defect", arg, err)
		}
	}
}

func wgsSpec(dir, base, ext string) gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir(dir), Base: base, Ext: ext}
}
