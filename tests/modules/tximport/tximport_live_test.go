//go:build live

package tximport_test

import (
	"os/exec"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/modules/tximport"
)

func TestPinnedImageReceivesFirstRArgumentWithoutDelimiter(t *testing.T) {
	command := exec.Command("docker", "run", "--rm", "--network", "none", string(tximport.DefaultImage), "Rscript", "-e", `stopifnot(identical(commandArgs(trailingOnly=TRUE), c("first", "second")))`, "first", "second")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("tximport image R argv probe failed: %v\n%s", err, output)
	}
}
