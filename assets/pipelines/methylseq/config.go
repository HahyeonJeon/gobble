package methylseq

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkdeduplicate "github.com/HahyeonJeon/gobble/assets/modules/bismark-deduplicate"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	bismarkmethylationextractor "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	bismarkreport "github.com/HahyeonJeon/gobble/assets/modules/bismark-report"
	bismarksummary "github.com/HahyeonJeon/gobble/assets/modules/bismark-summary"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
)

const (
	methylTrimGaloreImage modules.Image = "community.wave.seqera.io/library/cutadapt_trim-galore_pigz:a98edd405b34582d@sha256:4e56e5205f5a69d002e1d088bf05ca962ed65c55b97bb529e397b28c61d6dc24"
	methylMultiQCImage    modules.Image = "community.wave.seqera.io/library/multiqc:1.32--d58f60e4deb769bf@sha256:677f4c8e38cfd741926e5bd1e80d96b756540bc6a9e9c5ed520aa7a98358d11d"
)

// DefaultConfig returns a fresh nf-core/methylseq 4.2.0 directional Bismark
// config. It names caller-staged workspace inputs and performs no I/O or
// download.
func DefaultConfig() Config {
	return Config{
		Reference:   ReferenceConfig{FASTA: gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genome", Ext: ".fa"}},
		LibraryMode: LibraryModeDirectional,
		Results:     gobble.Dir("results/methylseq"),
		Publication: PublicationPolicy{
			DeduplicatedBAMs: true,
			MethylationCalls: true,
			Reports:          true,
		},
		CatFASTQ:      catfastq.Options{Options: methylBase(1, "512m")},
		FastQC:        fastqc.Options{Options: methylBase(2, "1g")},
		TrimGalore:    trimgalore.Options{Options: methylImageBase(methylTrimGaloreImage, 4, "2g")},
		BismarkGenome: bismarkgenome.Options{Options: methylBase(4, "15g")},
		BismarkAlign:  bismarkalign.Options{Options: methylBase(4, "15g")},
		Deduplicate:   bismarkdeduplicate.Options{Options: methylBase(1, "4g")},
		Extractor: bismarkmethylationextractor.Options{
			Options: methylBase(6, "15g"), ExcludeOverlap: true, IgnoreR2: 2, CoverageCutoff: 1,
		},
		Report:  bismarkreport.Options{Options: methylBase(1, "2g")},
		Summary: bismarksummary.Options{Options: methylBase(1, "2g")},
		MultiQC: multiqc.Options{Options: methylImageBase(methylMultiQCImage, 1, "2g")},
	}
}

func methylBase(cpu float64, memory string) modules.Options {
	return modules.Options{Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}

func methylImageBase(image modules.Image, cpu float64, memory string) modules.Options {
	return modules.Options{Image: image, Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}
