package main

import (
	"encoding/json"
	"github.com/HahyeonJeon/gobble"
	"io"
)

func runOutputEvidence(req *request, stdout, stderr io.Writer) int {
	var launch gobble.LaunchIntent
	var payload json.RawMessage
	if readLaunchFile("intent.json", 16384, &launch) != nil || readLaunchFile("payload.json", 3<<20, &payload) != nil {
		return writeErr(stderr, invalidRequest(req.command, "invalid output launch bundle"), 2)
	}
	value, err := gobble.ReadOutputEvidence(req.workspace, payload, launch, req.outputRequest)
	if err != nil {
		return writeLibraryErr(stderr, err)
	}
	return writeJSON(stdout, stderr, req.command, value)
}
