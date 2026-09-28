package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPreparedContinuationCommandsNeverLoadProject(t *testing.T) {
	for _, verb := range []string{"prepared-continuation-review", "prepared-continue", "prepared-continuation-receipt"} {
		req, err := parse([]string{verb, "--workspace", "/saved/run"})
		if err != nil || req.command != verb || req.workspace != "/saved/run" {
			t.Fatal(verb, err)
		}
		if _, err := parse([]string{verb, "--workspace", "/saved/run", "--sample", "in.csv"}); err == nil {
			t.Fatal("source flag accepted", verb)
		}
		if _, err := parse([]string{verb}); err == nil {
			t.Fatal("missing Run accepted", verb)
		}
	}
	var out, diagnostic bytes.Buffer
	if code := run([]string{"prepared-capabilities"}, &out, &diagnostic); code != 0 || !strings.Contains(out.String(), `"continuationVersion":1`) {
		t.Fatal(code, out.String(), diagnostic.String())
	}
	if _, err := parse([]string{"prepared-capabilities", "--workspace", "/unnecessary"}); err == nil {
		t.Fatal("capability probe accepted workspace")
	}
}
