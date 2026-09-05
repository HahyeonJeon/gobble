// Package multiqcevidence owns MultiQC command-only fixture facts.
package multiqcevidence

import "github.com/HahyeonJeon/gobble/internal/fixture"

const CacheDir = "tests/modules/multiqc/testdata/cache"

var SARSCoV2FastQCZip = fixture.Pin{
	Name: "test_fastqc.zip", URL: "https://raw.githubusercontent.com/nf-core/test-datasets/6c82958a6f302d8471a20855023ac59f9974fa8a/data/genomics/sarscov2/illumina/fastqc/test_fastqc.zip",
	Bytes: 620149, SHA256: "3fb4ca7852b311f1ab542028cde1debd98edeac94384224b7d5bba78dd9612b6",
}
