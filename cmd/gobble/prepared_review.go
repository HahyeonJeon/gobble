package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/HahyeonJeon/gobble"
)

// Trusted binary path: no driver, Go compiler, import or invocation of Project code.
// Its working folder contains only the service-retained private payload and intent.
func runPreparedReview(stdout, stderr io.Writer) int {
	var intent struct {
		Digest  string                    `json:"digest"`
		Binding gobble.PreparationBinding `json:"binding"`
	}
	f, err := os.Open("intent.json")
	if err != nil {
		return writeErr(stderr, invalidRequest("prepared-review", "missing preparation intent"), 2)
	}
	raw, err := io.ReadAll(io.LimitReader(f, 4097))
	f.Close()
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err != nil || len(raw) > 4096 || d.Decode(&intent) != nil || d.Decode(new(any)) != io.EOF {
		return writeErr(stderr, invalidRequest("prepared-review", "invalid preparation intent"), 2)
	}
	f, err = os.Open("payload.json")
	if err != nil {
		return writeErr(stderr, invalidRequest("prepared-review", "missing private payload"), 2)
	}
	raw, err = io.ReadAll(io.LimitReader(f, (3<<20)+1))
	f.Close()
	if err != nil || len(raw) > 3<<20 {
		return writeErr(stderr, invalidRequest("prepared-review", "invalid private payload"), 2)
	}
	value, err := gobble.InspectPreparedPipeline(raw, intent.Digest, intent.Binding)
	if err != nil {
		return writeErr(stderr, invalidRequest("prepared-review", "the sealed plan failed validation; review required"), 1)
	}
	return writeJSON(stdout, stderr, "prepared-review", value)
}
