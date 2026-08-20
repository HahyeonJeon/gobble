package main

import (
	"encoding/json"
	"errors"
	"io"
	"runtime/debug"

	"github.com/HahyeonJeon/gobble"
)

const modulePath = "github.com/HahyeonJeon/gobble"

type versionResult struct {
	Op      string `json:"op"`
	Module  string `json:"module"`
	Version string `json:"version"`
}

func invalidRequest(op, message string) *gobble.Error {
	return &gobble.Error{
		Op: op,
		Defects: []gobble.Defect{{
			Code:    gobble.DefectInvalidRequest,
			Message: message,
		}},
	}
}

func writeErr(stderr io.Writer, ge *gobble.Error, code int) int {
	if ge == nil {
		return code
	}
	data, err := json.Marshal(ge)
	if err != nil {
		return code
	}
	_, _ = stderr.Write(append(data, '\n'))
	return code
}

func writeLibraryErr(stderr io.Writer, err error) int {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return writeErr(stderr, invalidRequest("cli", err.Error()), 1)
	}
	return writeErr(stderr, ge, 1)
}

func writeJSON(stdout, stderr io.Writer, op string, v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return writeErr(stderr, invalidRequest(op, "invalid request"), 1)
	}
	if _, err := stdout.Write(append(data, '\n')); err != nil {
		return writeErr(stderr, invalidRequest(op, "stdout write failed"), 1)
	}
	return 0
}

func writeVersion(stdout, stderr io.Writer) int {
	return writeJSON(stdout, stderr, "version", versionResult{
		Op:      "version",
		Module:  modulePath,
		Version: buildVersion(),
	})
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}
