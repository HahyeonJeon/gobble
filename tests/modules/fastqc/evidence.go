// Package fastqcevidence owns FastQC command-only fixture facts.
package fastqcevidence

import "github.com/HahyeonJeon/gobble/tests/internal/fixture"

const CacheDir = "tests/modules/fastqc/testdata/cache"

var SARSCoV2R1 = fixture.Pin{
	Name: "test_1.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/sarscov2/illumina/fastq/test_1.fastq.gz",
	Bytes: 9413, SHA256: "0515ba304cb1bf7abcdd9c156b6affad7e580273f983dfed2e8fe2d918e800ff",
}
