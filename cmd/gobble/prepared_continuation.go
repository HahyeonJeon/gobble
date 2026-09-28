package main

import (
	"context"
	"encoding/json"
	"io"

	"github.com/HahyeonJeon/gobble"
)

func runPreparedContinuation(req *request, stdout, stderr io.Writer) int {
	var original gobble.LaunchIntent
	if err := readLaunchFile("intent.json", 16384, &original); err != nil {
		return writeErr(stderr, invalidRequest(req.command, "invalid original launch"), 2)
	}
	if req.command == "prepared-continuation-receipt" {
		var intent gobble.ContinuationIntent
		if err := readLaunchFile("continuation.json", 4096, &intent); err != nil {
			return writeErr(stderr, invalidRequest(req.command, "invalid continuation intent"), 2)
		}
		receipt, exists, err := gobble.ReadPreparedContinuation(req.workspace, original, intent)
		if err != nil {
			return writeLibraryErr(stderr, err)
		}
		var value *gobble.ContinuationAdmission
		if exists {
			value = &receipt
		}
		return writeJSON(stdout, stderr, req.command, struct {
			SchemaVersion int                           `json:"schemaVersion"`
			Found         bool                          `json:"found"`
			Receipt       *gobble.ContinuationAdmission `json:"receipt,omitempty"`
		}{1, exists, value})
	}
	var payload json.RawMessage
	if err := readLaunchFile("payload.json", 3<<20, &payload); err != nil {
		return writeErr(stderr, invalidRequest(req.command, "invalid prepared payload"), 2)
	}
	if req.command == "prepared-continuation-review" {
		review, err := gobble.ReviewPreparedContinuation(context.Background(), req.workspace, payload, original)
		if err != nil {
			return writeLibraryErr(stderr, err)
		}
		return writeJSON(stdout, stderr, req.command, review)
	}
	var intent gobble.ContinuationIntent
	if err := readLaunchFile("continuation.json", 4096, &intent); err != nil {
		return writeErr(stderr, invalidRequest(req.command, "invalid continuation intent"), 2)
	}
	receipt, err := gobble.ContinuePreparedPipeline(context.Background(), req.workspace, payload, original, intent)
	// Receipt lookup is authoritative even if this long-running command exits with
	// an execution error or loses stdout. Never infer absence from its exit code.
	if err != nil {
		return writeLibraryErr(stderr, err)
	}
	return writeJSON(stdout, stderr, req.command, receipt)
}
