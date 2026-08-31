package atacseq

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	atacconsensuspeaks "github.com/HahyeonJeon/gobble/assets/modules/atac-consensus-peaks"
	atacfripscore "github.com/HahyeonJeon/gobble/assets/modules/atac-frip-score"
	atacreferenceintervals "github.com/HahyeonJeon/gobble/assets/modules/atac-reference-intervals"
	"github.com/HahyeonJeon/gobble/assets/modules/ataqv"
	ataqvmkarv "github.com/HahyeonJeon/gobble/assets/modules/ataqv-mkarv"
	bedgraphscale "github.com/HahyeonJeon/gobble/assets/modules/bedgraph-scale"
	bedtoolsgenomecov "github.com/HahyeonJeon/gobble/assets/modules/bedtools-genomecov"
	bedtoolsintersect "github.com/HahyeonJeon/gobble/assets/modules/bedtools-intersect"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	deeptoolscomputematrix "github.com/HahyeonJeon/gobble/assets/modules/deeptools-compute-matrix"
	deeptoolsplotfingerprint "github.com/HahyeonJeon/gobble/assets/modules/deeptools-plot-fingerprint"
	deeptoolsplotprofile "github.com/HahyeonJeon/gobble/assets/modules/deeptools-plot-profile"
	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	featurecountsmergematrices "github.com/HahyeonJeon/gobble/assets/modules/featurecounts-merge-matrices"
	homerannotatepeaks "github.com/HahyeonJeon/gobble/assets/modules/homer-annotate-peaks"
	igvsession "github.com/HahyeonJeon/gobble/assets/modules/igv-session"
	macs2callpeak "github.com/HahyeonJeon/gobble/assets/modules/macs2-callpeak"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	picardcollectmultiplemetrics "github.com/HahyeonJeon/gobble/assets/modules/picard-collect-multiple-metrics"
	picardmarkduplicates "github.com/HahyeonJeon/gobble/assets/modules/picard-markduplicates"
	picardmergesamfiles "github.com/HahyeonJeon/gobble/assets/modules/picard-merge-sam-files"
	plothomerannotatepeaks "github.com/HahyeonJeon/gobble/assets/modules/plot-homer-annotatepeaks"
	plotmacs2qc "github.com/HahyeonJeon/gobble/assets/modules/plot-macs2-qc"
	samtoolsfaidx "github.com/HahyeonJeon/gobble/assets/modules/samtools-faidx"
	samtoolsflagstat "github.com/HahyeonJeon/gobble/assets/modules/samtools-flagstat"
	samtoolsidxstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-idxstats"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
	samtoolsview "github.com/HahyeonJeon/gobble/assets/modules/samtools-view"
	samtoolsviewcount "github.com/HahyeonJeon/gobble/assets/modules/samtools-view-count"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	ucscbedgraphtobigwig "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedgraphtobigwig"
	wclines "github.com/HahyeonJeon/gobble/assets/modules/wc-lines"
)

const (
	atacFastQCImage   modules.Image = "quay.io/biocontainers/fastqc:0.11.9--0@sha256:70de12400206b9c1784c8dfd019cfe4e42eed9a42eabf6c61eb68342843bdaab"
	atacTrimImage     modules.Image = "quay.io/biocontainers/trim-galore:0.6.7--hdfd78af_0@sha256:3c986513543ace0d0456d51f4a5e4c254065fa665b47f7ed2fe01ed23e406608"
	atacBWAIndexImage modules.Image = "quay.io/biocontainers/bwa:0.7.17--hed695b0_7@sha256:c3a708bea7947a44288e675fd9791c7aaf0c97dba0710addba336ed193821f8a"
	atacBWAMemImage   modules.Image = "quay.io/biocontainers/mulled-v2-fe8faa35dbf6dc65a0f7f5d4ea12e31a79f73e40:219b6c272b25e7e642ae3ff0bf0c5c81a5135ab4-0@sha256:9548dc56bdc0b734cd3767f9eea0f9d0ea1c44b35ef5fb35b0f746807cacbeea"
	atacSamtoolsImage modules.Image = "quay.io/biocontainers/samtools:1.17--h00cdaf9_0@sha256:6f88956b747a67b2a39a3ff72c4de30e665239ee11db610624dd4298e30db1bf"
	atacPicardImage   modules.Image = "quay.io/biocontainers/picard:3.0.0--hdfd78af_1@sha256:1807618ee8ac1af18a2a4656dd8b4d4a0a6f679b6a1e554a6603ac7a6d732d95"
	atacBedtoolsImage modules.Image = "quay.io/biocontainers/bedtools:2.30.0--hc088bd4_0@sha256:b0018bd0a10853e19ee92f6d46d8d12f1c41e516845105e1f02c91b4d7b961b1"
	atacUCSCImage     modules.Image = "quay.io/biocontainers/ucsc-bedgraphtobigwig:445--h954228d_0@sha256:b1b69d9f17f5de643c5dca055a9f2d55dbdeae5c554513c323a2a8010a838b00"
	atacMultiQCImage  modules.Image = "quay.io/biocontainers/multiqc:1.13--pyhdfd78af_0@sha256:db913f47894a386040d76ae5f46b19149a71377f5107327d15ec89d3a1ec3f5f"
)

// DefaultConfig returns a fresh atacseq 2.1.2-selected BWA config. It names
// caller-staged workspace inputs and performs no filesystem or network access.
func DefaultConfig() Config {
	return Config{
		Reference: ReferenceConfig{
			FASTA:               gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genome", Ext: ".fa"},
			Annotation:          gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genes", Ext: ".gtf"},
			MitoName:            "MT",
			Organism:            "NA",
			ReadLength:          50,
			EffectiveGenomeSize: "12157105",
		},
		Results:          gobble.Dir("results/atacseq"),
		PeakMode:         PeakBroad,
		Filters:          FilterPolicy{MinimumMAPQ: 20, RemoveOrphans: true, RemoveMito: true},
		ConsensusMinimum: 1,
		Publication: PublicationPolicy{
			FilteredAlignments: true,
			Tracks:             true,
			ReplicatePeaks:     true,
			ConsensusMatrix:    true,
			QC:                 true,
			Reports:            true,
			IGVSession:         true,
		},
		BWAIndex:               bwaindex.Options{Options: atacBase(atacBWAIndexImage, 1, "4g")},
		SamtoolsFAIDX:          samtoolsfaidx.Options{Options: atacBase(atacSamtoolsImage, 1, "1g")},
		ReferenceIntervals:     atacreferenceintervals.Options{Options: atacResources(1, "1g")},
		FastQC:                 fastqc.Options{Options: atacBase(atacFastQCImage, 2, "2g")},
		TrimGalore:             trimgalore.Options{Options: atacBase(atacTrimImage, 6, "4g"), Quality: 20},
		BWAMem:                 bwamem.Options{Options: atacBase(atacBWAMemImage, 2, "4g")},
		SamtoolsSort:           samtoolssort.Options{Options: atacBase(atacSamtoolsImage, 2, "3g")},
		SamtoolsIndex:          samtoolsindex.Options{Options: atacBase(atacSamtoolsImage, 1, "1g")},
		SamtoolsStats:          samtoolsstats.Options{Options: atacBase(atacSamtoolsImage, 1, "1g")},
		SamtoolsFlagstat:       samtoolsflagstat.Options{Options: atacBase(atacSamtoolsImage, 1, "1g")},
		SamtoolsIdxstats:       samtoolsidxstats.Options{Options: atacBase(atacSamtoolsImage, 1, "1g")},
		MergeRuns:              picardmergesamfiles.Options{Options: atacBase(atacPicardImage, 2, "4g")},
		MergeReplicates:        picardmergesamfiles.Options{Options: atacBase(atacPicardImage, 2, "4g")},
		MarkDuplicates:         picardmarkduplicates.Options{Options: atacBase(atacPicardImage, 2, "4g")},
		FilterBAM:              samtoolsview.Options{Options: atacBase(atacSamtoolsImage, 2, "2g")},
		BlacklistFilter:        bedtoolsintersect.Options{Options: atacBase(atacBedtoolsImage, 1, "2g"), Invert: true},
		CollectMultipleMetrics: picardcollectmultiplemetrics.Options{Options: atacBase(atacPicardImage, 1, "4g")},
		GenomeCoverage:         bedtoolsgenomecov.Options{Options: atacBase(atacBedtoolsImage, 1, "2g")},
		ScaleCoverage:          bedgraphscale.Options{Options: atacBase(atacBedtoolsImage, 1, "1g")},
		BedGraphToBigWig:       ucscbedgraphtobigwig.Options{Options: atacBase(atacUCSCImage, 1, "1g")},
		ComputeMatrix:          deeptoolscomputematrix.Options{Options: atacResources(2, "4g"), Upstream: 3000, Downstream: 3000},
		PlotProfile:            deeptoolsplotprofile.Options{Options: atacResources(1, "2g")},
		PlotFingerprint:        deeptoolsplotfingerprint.Options{Options: atacResources(2, "4g")},
		MACS2:                  macs2callpeak.Options{Options: atacResources(2, "4g"), QValue: 0.05},
		HOMER:                  homerannotatepeaks.Options{Options: atacResources(2, "4g")},
		PlotMACS2QC:            plotmacs2qc.Options{Options: atacResources(2, "4g")},
		PlotHOMERAnnotatePeaks: plothomerannotatepeaks.Options{Options: atacResources(2, "4g")},
		PeakCount:              wclines.Options{Options: atacResources(1, "256m")},
		PeakIntersect:          bedtoolsintersect.Options{Options: atacBase(atacBedtoolsImage, 1, "2g")},
		ReadCount:              samtoolsviewcount.Options{Options: atacBase(atacSamtoolsImage, 1, "1g")},
		FRiP:                   atacfripscore.Options{Options: atacResources(1, "256m")},
		Consensus:              atacconsensuspeaks.Options{Options: atacResources(1, "2g")},
		FeatureCounts:          featurecounts.ATACOptions{Options: atacResources(2, "4g")},
		FeatureCountsMerge:     featurecountsmergematrices.Options{Options: atacResources(1, "1g")},
		DESeq2QC:               deseq2qc.Options{Options: atacResources(2, "4g")},
		Ataqv:                  ataqv.Options{Options: atacResources(2, "4g")},
		Mkarv:                  ataqvmkarv.Options{Options: atacResources(2, "4g")},
		IGV:                    igvsession.Options{Options: atacResources(1, "1g")},
		MultiQC:                multiqc.Options{Options: atacBase(atacMultiQCImage, 1, "2g")},
	}
}

func atacResources(cpu float64, memory string) modules.Options {
	return modules.Options{Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}

func atacBase(image modules.Image, cpu float64, memory string) modules.Options {
	return modules.Options{Image: image, Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}
