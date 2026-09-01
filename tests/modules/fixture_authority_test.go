package moduleevidence

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
	fastpevidence "github.com/HahyeonJeon/gobble/tests/modules/fastp"
	fastqcevidence "github.com/HahyeonJeon/gobble/tests/modules/fastqc"
	multiqcevidence "github.com/HahyeonJeon/gobble/tests/modules/multiqc"
)

func TestCommandFixtureURLsUseExactDatasetCommit(t *testing.T) {
	const commit = "6c82958a6f302d8471a20855023ac59f9974fa8a"
	for name, pin := range map[string]fixture.Pin{
		"fastp":   fastpevidence.SARSCoV2R2,
		"fastqc":  fastqcevidence.SARSCoV2R1,
		"multiqc": multiqcevidence.SARSCoV2FastQCZip,
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(pin.URL, "/"+commit+"/") || strings.Contains(pin.URL, "/modules/") {
				t.Fatalf("fixture URL = %q, want exact nf-core/test-datasets commit %s", pin.URL, commit)
			}
		})
	}
}
