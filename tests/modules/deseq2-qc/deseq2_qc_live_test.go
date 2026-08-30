//go:build live

package deseq2qc_test

import (
	"os/exec"
	"strings"
	"testing"

	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
)

func TestPinnedImageContainsDeclaredDESeq2AndReceivesFirstArgument(t *testing.T) {
	script := `stopifnot(identical(commandArgs(trailingOnly=TRUE), c("first", "second"))); suppressPackageStartupMessages(library(DESeq2)); cat(as.character(packageVersion("DESeq2")))`
	command := exec.Command("docker", "run", "--rm", "--network", "none", string(deseq2qc.DefaultImage), "Rscript", "-e", script, "first", "second")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("DESeq2-QC image package/argv probe failed: %v\n%s", err, output)
	}
	if !strings.HasSuffix(strings.TrimSpace(string(output)), "1.46.0") {
		t.Fatalf("DESeq2 version = %q, want manifest-proven 1.46.0", output)
	}
}
