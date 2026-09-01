package scrnaseq

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	anndatarconvert "github.com/HahyeonJeon/gobble/assets/modules/anndatar-convert"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	gffreadtranscriptome "github.com/HahyeonJeon/gobble/assets/modules/gffread-transcriptome"
	gtfgenefilter "github.com/HahyeonJeon/gobble/assets/modules/gtf-gene-filter"
	gtftot2g "github.com/HahyeonJeon/gobble/assets/modules/gtf-to-t2g"
	h5adconcat "github.com/HahyeonJeon/gobble/assets/modules/h5ad-concat"
	matrixtoh5ad "github.com/HahyeonJeon/gobble/assets/modules/matrix-to-h5ad"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	"github.com/HahyeonJeon/gobble/assets/modules/qcatch"
	simpleafindex "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-index"
	simpleafquant "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-quant"
)

const (
	// catFastqImage is the Gobble-owned 2026-08-30 image tuple for the
	// product-required run consolidation. Its container bytes were previously
	// recorded for nf-core/rnaseq 3.26.0; cat_fastq is not an
	// nf-core/scrnaseq 4.2.0 module.
	catFastqImage modules.Image = "community.wave.seqera.io/library/coreutils_grep_gzip_lbzip2_pruned:838ba80435a629f8@sha256:63c2c6b22e83b2f656e88fbb1553e595da4e9e58794e3bfcb98b20b3837f328a"
	multiQCImage  modules.Image = "community.wave.seqera.io/library/multiqc:1.34--db7c73dae76bc9e6@sha256:22eb821173e8b85e1632263d98447a13cf3eef1803e895aa712b984630e2d793"
)

// DefaultConfig returns a fresh scrnaseq 4.2.0-selected Simpleaf config. It
// names caller-staged workspace inputs and performs no filesystem or network
// access. V2 is explicit because the official fixture is 10x V2.
func DefaultConfig() Config {
	return Config{
		Protocol: Protocol10xV2,
		Reference: ReferenceConfig{
			FASTA:      gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genome", Ext: ".fa"},
			Annotation: gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genes", Ext: ".gtf"},
			BarcodeWhitelist: WhitelistConfig{
				Protocol: Protocol10xV2,
				Path:     gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "10x_V2_barcode_whitelist", Ext: ".txt.gz"},
			},
		},
		Results:       gobble.Dir("results/scrnaseq"),
		UMIResolution: ResolutionCRLike,
		Publication: PublicationPolicy{
			Index: true, Quantification: true, QCatch: true, RawH5AD: true,
			RawRDS: true, CombinedH5AD: true, MultiQC: true,
		},
		Consolidate:      catfastq.Options{Options: scrnaBase(catFastqImage, 1, "512m")},
		FastQC:           fastqc.Options{Options: scrnaBase(fastqc.DefaultImage, 2, "2g")},
		GTFFilter:        gtfgenefilter.Options{Options: scrnaBase(gtfgenefilter.DefaultImage, 1, "1g")},
		Transcriptome:    gffreadtranscriptome.Options{Options: scrnaBase(gffreadtranscriptome.DefaultImage, 1, "1g")},
		TranscriptToGene: gtftot2g.Options{Options: scrnaBase(gtftot2g.DefaultImage, 1, "1g")},
		SimpleafIndex:    simpleafindex.Options{Options: scrnaBase(simpleafindex.DefaultImage, 4, "8g")},
		SimpleafQuant:    simpleafquant.Options{Options: scrnaBase(simpleafquant.DefaultImage, 4, "8g")},
		QCatch:           qcatch.Options{Options: scrnaBase(qcatch.DefaultImage, 2, "6g"), Chemistry: "10X_3p_v2"},
		MatrixToH5AD:     matrixtoh5ad.Options{Options: scrnaBase(matrixtoh5ad.DefaultImage, 2, "6g")},
		AnnDataR:         anndatarconvert.Options{Options: scrnaBase(anndatarconvert.DefaultImage, 2, "6g")},
		H5ADConcat:       h5adconcat.Options{Options: scrnaBase(h5adconcat.DefaultImage, 2, "6g")},
		MultiQC:          multiqc.Options{Options: scrnaBase(multiQCImage, 1, "2g")},
	}
}

func scrnaBase(image modules.Image, cpu float64, memory string) modules.Options {
	return modules.Options{Image: image, Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}
