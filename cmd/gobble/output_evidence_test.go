package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestOutputEvidenceCLIContract(t *testing.T) {
	req, err := parse([]string{"output-evidence", "--workspace", "/saved", "run-id", "qc", "3", "html"})
	if err != nil || req.outputRequest.Attempt != 3 || req.outputRequest.RunID != "run-id" || req.workspace != "/saved" {
		t.Fatal(req, err)
	}
	for _, args := range [][]string{
		{"output-evidence", "--workspace", "/saved", "run", "qc", "0", "html"},
		{"output-evidence", "run", "qc", "1", "html"},
		{"output-evidence", "--workspace", "/saved", "run", "qc", "1", "html", "--sample", "data.csv"},
		{"output-capabilities", "--workspace", "/saved"},
	} {
		if _, err := parse(args); err == nil {
			t.Fatal("accepted", args)
		}
	}
	for _, verb := range []string{"output-capabilities", "prepared-capabilities"} {
		var out, diagnostic bytes.Buffer
		if code := run([]string{verb}, &out, &diagnostic); code != 0 {
			t.Fatal(code, diagnostic.String())
		}
		var actual map[string]int
		if json.Unmarshal(out.Bytes(), &actual) != nil {
			t.Fatal(out.String())
		}
		if verb == "output-capabilities" && (len(actual) != 2 || actual["outputEvidenceVersion"] != 1) {
			t.Fatal(actual)
		}
		if verb == "prepared-capabilities" && (len(actual) != 3 || actual["launchSchemaVersion"] != 2 || actual["continuationVersion"] != 1) {
			t.Fatal("prepared contract changed", actual)
		}
	}
}
