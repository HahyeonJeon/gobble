package main

import (
	"io"
	"os"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

func runCreationScaffold(stdout, stderr io.Writer) int {
	mod, err := os.ReadFile("/opt/gobble/go.mod")
	if err != nil {
		return writeErr(stderr, invalidRequest("creation-scaffold", "The installed creation scaffold is unavailable."), 1)
	}
	sum, err := os.ReadFile("/opt/gobble/go.sum")
	if err != nil {
		return writeErr(stderr, invalidRequest("creation-scaffold", "The installed dependency checksums are unavailable."), 1)
	}
	value, err := pipelinereview.ExportCreationScaffold(string(mod), string(sum))
	if err != nil {
		return writeErr(stderr, invalidRequest("creation-scaffold", err.Error()), 1)
	}
	return writeJSON(stdout, stderr, "creation-scaffold", value)
}
