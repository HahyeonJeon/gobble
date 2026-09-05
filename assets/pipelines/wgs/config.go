package wgs

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bcftoolssort "github.com/HahyeonJeon/gobble/assets/modules/bcftools-sort"
	bcftoolsstats "github.com/HahyeonJeon/gobble/assets/modules/bcftools-stats"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	gatk4applybqsr "github.com/HahyeonJeon/gobble/assets/modules/gatk4-applybqsr"
	gatk4baserecalibrator "github.com/HahyeonJeon/gobble/assets/modules/gatk4-baserecalibrator"
	gatk4gatherbqsrreports "github.com/HahyeonJeon/gobble/assets/modules/gatk4-gather-bqsr-reports"
	gatk4genomicsdbimport "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genomicsdbimport"
	gatk4genotypegvcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genotypegvcfs"
	gatk4haplotypecaller "github.com/HahyeonJeon/gobble/assets/modules/gatk4-haplotypecaller"
	gatk4markduplicates "github.com/HahyeonJeon/gobble/assets/modules/gatk4-markduplicates"
	gatk4mergevcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-mergevcfs"
	"github.com/HahyeonJeon/gobble/assets/modules/mosdepth"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	samtoolsflagstat "github.com/HahyeonJeon/gobble/assets/modules/samtools-flagstat"
	samtoolsidxstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-idxstats"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolsmerge "github.com/HahyeonJeon/gobble/assets/modules/samtools-merge"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
)

const (
	wgsFastQCImage   modules.Image = "quay.io/biocontainers/fastqc:0.12.1--hdfd78af_0@sha256:e194048df39c3145d9b4e0a14f4da20b59d59250465b6f2a9cb698445fd45900"
	wgsSamtoolsImage modules.Image = "community.wave.seqera.io/library/htslib_samtools:1.24--d697cfb9dce007cd@sha256:a55ddea590e567a91df592300a960aa534cfc1bd16e7623e3938ec21f4f3df15"
	// The upstream image exposes multiqc directly; the selected Wave build does not.
	wgsMultiQCImage modules.Image = "ghcr.io/multiqc/multiqc:v1.35@sha256:a0c146fb9ec0207a88627da3aaa36c10014c5ef8dc841c55a01c0345d8b5cae5"
)

// DefaultConfig returns a fresh Sarek 3.10.0-derived joint-germline config. It
// names caller-staged workspace inputs and performs no I/O or download.
func DefaultConfig() Config {
	intervalDir := gobble.Dir("in/reference/intervals")
	return Config{
		Reference: ReferenceConfig{
			FASTA:      gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genome", Ext: ".fasta"},
			FAI:        gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genome", Ext: ".fasta.fai"},
			Dictionary: gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genome", Ext: ".dict"},
			KnownSites: []KnownSite{
				{Name: "dbsnp", VCF: gobble.PathSpec{Dir: gobble.Dir("in/reference/known-sites"), Base: "dbsnp_146.hg38", Ext: ".vcf.gz"}, Index: gobble.PathSpec{Dir: gobble.Dir("in/reference/known-sites"), Base: "dbsnp_146.hg38", Ext: ".vcf.gz.tbi"}},
				{Name: "mills", VCF: gobble.PathSpec{Dir: gobble.Dir("in/reference/known-sites"), Base: "mills_and_1000G.indels", Ext: ".vcf.gz"}, Index: gobble.PathSpec{Dir: gobble.Dir("in/reference/known-sites"), Base: "mills_and_1000G.indels", Ext: ".vcf.gz.tbi"}},
			},
			DBSNP: "dbsnp",
			Intervals: gobble.Group{
				{Name: "interval_001", Spec: gobble.PathSpec{Dir: intervalDir, Base: "interval_001", Ext: ".bed"}},
				{Name: "interval_002", Spec: gobble.PathSpec{Dir: intervalDir, Base: "interval_002", Ext: ".bed"}},
			},
		},
		Results: gobble.Dir("results/wgs"),
		Format:  OutputBAM,
		Publication: PublicationPolicy{
			RecalibratedAlignments: true,
			SampleGVCFs:            true,
			JointCallset:           true,
			Reports:                true,
		},
		BWAIndex:          bwaindex.Options{Options: wgsBase(1, "6g")},
		FastQC:            fastqc.Options{Options: wgsImageBase(wgsFastQCImage, 2, "1g")},
		FastP:             fastp.Options{Options: wgsBase(4, "4g")},
		BWAMem:            bwamem.Options{Options: wgsBase(4, "4g")},
		SamtoolsSort:      samtoolssort.Options{Options: wgsImageBase(wgsSamtoolsImage, 2, "2g")},
		SamtoolsMerge:     samtoolsmerge.Options{Options: wgsImageBase(wgsSamtoolsImage, 2, "2g")},
		MarkDuplicates:    gatk4markduplicates.Options{Options: wgsImageBase(gatk4markduplicates.DefaultImage, 2, "6g")},
		BaseRecalibrator:  gatk4baserecalibrator.Options{Options: wgsBase(2, "4g")},
		GatherBQSRReports: gatk4gatherbqsrreports.Options{Options: wgsBase(1, "3g")},
		ApplyBQSR:         gatk4applybqsr.Options{Options: wgsBase(2, "4g")},
		GatherBAM:         samtoolsmerge.Options{Options: wgsImageBase(wgsSamtoolsImage, 1, "4g")},
		SamtoolsIndex:     samtoolsindex.Options{Options: wgsImageBase(wgsSamtoolsImage, 1, "1g")},
		SamtoolsStats:     samtoolsstats.Options{Options: wgsImageBase(wgsSamtoolsImage, 1, "1g")},
		SamtoolsFlagstat:  samtoolsflagstat.Options{Options: wgsImageBase(wgsSamtoolsImage, 1, "1g")},
		SamtoolsIdxstats:  samtoolsidxstats.Options{Options: wgsImageBase(wgsSamtoolsImage, 1, "1g")},
		Mosdepth:          mosdepth.Options{Options: wgsBase(2, "2g")},
		HaplotypeCaller:   gatk4haplotypecaller.Options{Options: wgsBase(2, "4g")},
		MergeGVCFs:        gatk4mergevcfs.Options{Options: wgsBase(1, "4g")},
		GenomicsDBImport:  gatk4genomicsdbimport.Options{Options: wgsBase(2, "6g")},
		GenotypeGVCFs:     gatk4genotypegvcfs.Options{Options: wgsBase(1, "4g")},
		BCFToolsSort:      bcftoolssort.Options{Options: wgsBase(1, "2g")},
		MergeJointVCFs:    gatk4mergevcfs.Options{Options: wgsBase(1, "4g")},
		BCFToolsStats:     bcftoolsstats.Options{Options: wgsBase(1, "2g")},
		MultiQC:           multiqc.Options{Options: wgsImageBase(wgsMultiQCImage, 1, "2g")},
	}
}

func wgsBase(cpu float64, memory string) modules.Options {
	return modules.Options{Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}

func wgsImageBase(image modules.Image, cpu float64, memory string) modules.Options {
	return modules.Options{Image: image, Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}
