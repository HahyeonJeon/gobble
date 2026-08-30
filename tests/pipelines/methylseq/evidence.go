// Package methylseqevidence owns typed access to the Methyl-seq fixture
// manifest's directly staged bytes.
package methylseqevidence

import "github.com/HahyeonJeon/gobble/tests/internal/fixture"

// CacheDir is the Methyl-seq owner's ignored host cache.
const CacheDir = "tests/pipelines/methylseq/testdata/cache"

const FixtureSheet = "tests/pipelines/methylseq/testdata/methylseq-samplesheet.csv"

var (
	Single1FASTQ      = fixture.Pin{Name: "SRR389222_sub1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/SRR389222_sub1.fastq.gz", Bytes: 3302006, SHA256: "cf64dcfa6399e1f872a2c8cad0568db96d9fcc4c210ef00c9f5db06e689e1238"}
	Single2FASTQ      = fixture.Pin{Name: "SRR389222_sub2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/SRR389222_sub2.fastq.gz", Bytes: 3947922, SHA256: "287d636a1676bd1f2e1905cf2796d542d39e6e80ce0ae24144cb13088ed7d435"}
	Single3FASTQ      = fixture.Pin{Name: "SRR389222_sub3.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/SRR389222_sub3.fastq.gz", Bytes: 2646597, SHA256: "13abeb23edef24ea58f7d4683ba321cfc0321274864299e1c81e3dc6d25843f7"}
	Test1FASTQ        = fixture.Pin{Name: "Ecoli_10K_methylated_R1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R1.fastq.gz", Bytes: 467487, SHA256: "3d84b54e065f0760e830357d37bbc1ce511570b0443b6d0a7da1cf26261fe79b"}
	Test2FASTQ        = fixture.Pin{Name: "Ecoli_10K_methylated_R2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R2.fastq.gz", Bytes: 467335, SHA256: "2f3e6de0edf9bbc6dae46a5a43a2152d6d9d724b8b8ecd46281d47dd0606a646"}
	GenomeFASTA       = fixture.Pin{Name: "genome.fa", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/reference/genome.fa", Bytes: 49200, SHA256: "52a320d932e0d873141d5a326d80a7d811653cf2d782d07f8926f6c0e1ceb21e"}
	ReadyIndexArchive = fixture.Pin{Name: "Bowtie2_Index.tar.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/reference/Bowtie2_Index.tar.gz", Bytes: 375571, SHA256: "6a97bd6fbd9167c8fd4ba773f729425cda86ae04c229a3a3e36442a0fc17fa28"}
)

var Pins = []fixture.Pin{Single1FASTQ, Single2FASTQ, Single3FASTQ, Test1FASTQ, Test2FASTQ, GenomeFASTA, ReadyIndexArchive}
