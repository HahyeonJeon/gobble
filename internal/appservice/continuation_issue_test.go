package appservice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContinuationReviewExplainsKnownRefusalWithoutLeakingDiagnostics(t *testing.T) {
	cases := []struct{ message, want string }{
		{"completed output is unavailable", "A completed result is missing."},
		{"completed output content changed", "A completed result has changed."},
		{"saved input content changed", "The checked input copy has changed."},
		{"accepted tool image unavailable or changed", "An analysis tool is missing or has changed."},
		{"continuation requires a settled stopped Run", "Stop has not settled."},
		{"private-path-secret", "The recorded Docker runtime could not complete this query."},
	}
	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			s, p, original, f := continuationFixture(t)
			directory := t.TempDir()
			// This shell peer simulates only a failed Docker process's stderr. The real
			// bounded process runner, domain mapping, worker and durable record are used.
			diagnostic, _ := json.Marshal(map[string]any{"op": "cli", "defects": []any{map[string]any{"code": "invalid-request", "unit": "", "message": tc.message, "paths": nil}}})
			path := filepath.Join(directory, "diagnostic.json")
			if err := os.WriteFile(path, diagnostic, 0600); err != nil {
				t.Fatal(err)
			}
			script := "#!/bin/sh\ncat '" + path + "' >&2\nexit 1\n"
			if err := os.WriteFile(filepath.Join(directory, "docker"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
			s.runtime.command = func(ctx context.Context, args, env []string) ([]byte, error) {
				if contains(args, "prepared-continuation-review") {
					return runDocker(ctx, args, env)
				}
				return f.command(ctx, args, env)
			}
			if _, err := s.checkContinuation(p.ProjectID, continuationCheckInput{"req_reason", original.Value.RequestID, "run_saved"}); err != nil {
				t.Fatal(err)
			}
			v := awaitContinuation(t, s, p.ProjectID, "req_reason")
			if v.State != "blocked" || !strings.HasPrefix(v.Issue, tc.want) || strings.Contains(v.Issue, "private-path-secret") {
				t.Fatalf("unexpected blocked review: %+v", v)
			}
		})
	}
}

func TestContinuationRefusalDoesNotExposeUntrustedShape(t *testing.T) {
	for _, raw := range []string{
		`{"op":"cli","defects":[{"code":"invalid-request","message":"completed output is unavailable","paths":["private"]}]}`,
		`{"op":"cli","defects":[{"code":"invalid-request","message":"completed output is unavailable","unit":"private"}]}`,
		`{"op":"other","defects":[{"code":"invalid-request","message":"completed output is unavailable"}]}`,
		`{"op":"cli","defects":[{"code":"invalid-request","message":"completed output is unavailable"}],"unexpected":true}`,
		`{"op":"cli","defects":[{"code":"invalid-request","message":"completed output is unavailable"}]} {}`,
		`private stderr`,
	} {
		original := &runtimeCommandError{cause: problem("runtime_unavailable", "generic"), diagnostic: []byte(raw)}
		if got := continuationReviewError(original); got != original {
			t.Fatalf("untrusted diagnostic translated: %q", raw)
		}
	}
}
