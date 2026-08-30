//go:build live

package dupradar_test

import (
	"os/exec"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/modules/dupradar"
)

func TestPinnedImageReceivesFirstRArgumentWithoutDelimiter(t *testing.T) {
	command := exec.Command("docker", "run", "--rm", "--network", "none", string(dupradar.DefaultImage), "Rscript", "-e", `stopifnot(identical(commandArgs(trailingOnly=TRUE), c("first", "second")))`, "first", "second")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("dupRadar image R argv probe failed: %v\n%s", err, output)
	}
}
