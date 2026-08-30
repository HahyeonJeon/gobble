// Package methylseqevidence owns checkpoint Methyl-seq fixture facts.
package methylseqevidence

import "github.com/HahyeonJeon/gobble/tests/internal/fixture"

// CacheDir is the Methyl-seq owner's ignored host cache.
const CacheDir = "tests/pipelines/methylseq/testdata/cache"

const FixtureSheet = "tests/pipelines/methylseq/testdata/methylseq-samplesheet.csv"

var (
	Test1FASTQ  = fixture.Pin{Name: "Ecoli_10K_methylated_R1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R1.fastq.gz", Bytes: 467487, SHA256: "3d84b54e065f0760e830357d37bbc1ce511570b0443b6d0a7da1cf26261fe79b"}
	Test2FASTQ  = fixture.Pin{Name: "Ecoli_10K_methylated_R2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R2.fastq.gz", Bytes: 467335, SHA256: "2f3e6de0edf9bbc6dae46a5a43a2152d6d9d724b8b8ecd46281d47dd0606a646"}
	GenomeFASTA = fixture.Pin{Name: "genome.fa", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/reference/genome.fa", Bytes: 49200, SHA256: "52a320d932e0d873141d5a326d80a7d811653cf2d782d07f8926f6c0e1ceb21e"}
)

var Pins = []fixture.Pin{Test1FASTQ, Test2FASTQ, GenomeFASTA}
