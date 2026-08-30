// Package fastpevidence owns fastp command-only fixture facts.
package fastpevidence

import "github.com/HahyeonJeon/gobble/tests/internal/fixture"

const CacheDir = "tests/modules/fastp/testdata/cache"

var SARSCoV2R2 = fixture.Pin{
	Name: "test_2.fastq.gz", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/sarscov2/illumina/fastq/test_2.fastq.gz",
	Bytes: 9395, SHA256: "0080f40cab58c7e7b85443e37de22775e3ed6b7afdef9a3271ac3147576f3027",
}
