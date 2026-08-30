// Package rnaseqevidence owns checkpoint RNA-seq fixture facts.
package rnaseqevidence

import "github.com/HahyeonJeon/gobble/tests/internal/fixture"

// CacheDir is the RNA-seq owner's ignored host cache.
const CacheDir = "tests/pipelines/rnaseq/testdata/cache"

const (
	FixtureSheet     = "tests/pipelines/rnaseq/testdata/rnaseq-samplesheet.csv"
	LiveFixtureSheet = "tests/pipelines/rnaseq/testdata/rnaseq-live-samplesheet.csv"
)

var (
	Test1FASTQ   = fixture.Pin{Name: "SRR6357072_1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357072_1.fastq.gz", Bytes: 2148269, SHA256: "6c92efe43dc8145951c4131cd30e3e169f9877f041fcf20c2577eeeb7ec2b6ed"}
	Test2FASTQ   = fixture.Pin{Name: "SRR6357072_2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357072_2.fastq.gz", Bytes: 2167239, SHA256: "4501eb0062d4005cc1a4c836a86c036c15d3c38d685138d02675c5ecef84c0a3"}
	GenomeFASTA  = fixture.Pin{Name: "genome.fasta", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/reference/genome.fasta", Bytes: 234058, SHA256: "df70973809f672aa58a414fef3f01e0e465bf26f10159174a616b0dee2d458e1"}
	GTF          = fixture.Pin{Name: "genes.gtf", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/reference/genes.gtf", Bytes: 204286, SHA256: "8d9e66311d90f14517cdb8683066d4b376547ea7c2153b05b510fb9ba7988835"}
	Ctrl1FASTQ1  = fixture.Pin{Name: "SRR6357070_1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357070_1.fastq.gz", Bytes: 2239317, SHA256: "3f50541fa9cf2bedc87e7b682ada0fccfdfcd6d27b9bb81f17be230ff140ebe7"}
	Ctrl1FASTQ2  = fixture.Pin{Name: "SRR6357070_2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357070_2.fastq.gz", Bytes: 2232117, SHA256: "8590e1e01e568fba256aa7dced40519604cbb111ee44ab106a3dcb869660aaf4"}
	Ctrl2FASTQ1  = fixture.Pin{Name: "SRR6357071_1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357071_1.fastq.gz", Bytes: 2189243, SHA256: "06951c97884df8975d5419a2f0d03d435b9da722564000536524b01926970c93"}
	Ctrl2FASTQ2  = fixture.Pin{Name: "SRR6357071_2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357071_2.fastq.gz", Bytes: 2183711, SHA256: "00bbbf100ee90c5f681c3b4814637283732cd06e620171bd48718aa02de3a91d"}
	Treat2FASTQ1 = fixture.Pin{Name: "SRR6357073_1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357073_1.fastq.gz", Bytes: 2253497, SHA256: "c564e722216fb74116b237cc88d76cd33cf198216fb01fd9febd735a4503d18f"}
	Treat2FASTQ2 = fixture.Pin{Name: "SRR6357073_2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357073_2.fastq.gz", Bytes: 2282958, SHA256: "599adca68abe6a6116802fb2134104cfcc545d397edb49c070e4a443831226f8"}
)

var Pins = []fixture.Pin{Test1FASTQ, Test2FASTQ, GenomeFASTA, GTF}

var DistinctPairPins = []fixture.Pin{
	Ctrl1FASTQ1, Ctrl1FASTQ2, Ctrl2FASTQ1, Ctrl2FASTQ2,
	Test1FASTQ, Test2FASTQ, Treat2FASTQ1, Treat2FASTQ2,
}
