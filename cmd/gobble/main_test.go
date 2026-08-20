package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

type cliResult struct {
	stdout []byte
	stderr []byte
	code   int
}

func runCLI(args ...string) cliResult {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return cliResult{stdout: stdout.Bytes(), stderr: stderr.Bytes(), code: code}
}

func TestHelp(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "bare", args: nil, want: "Usage: gobble"},
		{name: "help", args: []string{"help"}, want: "Usage: gobble"},
		{name: "short", args: []string{"-h"}, want: "Usage: gobble"},
		{name: "long", args: []string{"--help"}, want: "Usage: gobble"},
		{name: "help compose", args: []string{"help", "compose"}, want: "Usage: gobble compose"},
		{name: "compose help", args: []string{"compose", "--help"}, want: "Usage: gobble compose"},
		{name: "inspect help", args: []string{"inspect", "--help"}, want: "Usage: gobble inspect"},
		{name: "run help", args: []string{"run", "--help"}, want: "Usage: gobble run"},
		{name: "release help", args: []string{"release", "--help"}, want: "Usage: gobble release"},
		{name: "inspect help skips view", args: []string{"inspect", "--help", "--workspace", t.TempDir()}, want: "Usage: gobble inspect"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runCLI(tc.args...)
			if res.code != 0 {
				t.Fatalf("exit = %d, want 0\nstderr: %s", res.code, res.stderr)
			}
			if len(res.stderr) != 0 {
				t.Fatalf("stderr = %q, want empty", res.stderr)
			}
			if !strings.Contains(string(res.stdout), tc.want) {
				t.Fatalf("stdout = %q, want substring %q", res.stdout, tc.want)
			}
			if json.Valid(bytes.TrimSpace(res.stdout)) {
				t.Fatalf("help stdout is JSON: %s", res.stdout)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"version"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := runCLI(args...)
			if res.code != 0 {
				t.Fatalf("exit = %d, want 0\nstderr: %s", res.code, res.stderr)
			}
			if len(res.stderr) != 0 {
				t.Fatalf("stderr = %q, want empty", res.stderr)
			}
			if len(res.stdout) == 0 || res.stdout[len(res.stdout)-1] != '\n' {
				t.Fatalf("stdout missing trailing newline: %q", res.stdout)
			}
			var got versionResult
			if err := json.Unmarshal(res.stdout, &got); err != nil {
				t.Fatalf("version JSON: %v\n%s", err, res.stdout)
			}
			if got.Op != "version" {
				t.Fatalf("op = %q, want version", got.Op)
			}
			if got.Module != modulePath {
				t.Fatalf("module = %q, want %s", got.Module, modulePath)
			}
			if got.Version == "" {
				t.Fatalf("version empty")
			}
			var raw map[string]any
			if err := json.Unmarshal(res.stdout, &raw); err != nil {
				t.Fatal(err)
			}
			if _, ok := raw["schema_version"]; ok {
				t.Fatalf("version JSON has schema_version: %#v", raw)
			}
		})
	}
}

func TestInvocationFailures(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		args []string
		op   string
	}{
		{name: "unknown verb", args: []string{"nope"}, op: "cli"},
		{name: "unknown flag", args: []string{"compose", "--format"}, op: "compose"},
		{name: "extra compose operands", args: []string{"compose", "a", "b"}, op: "compose"},
		{name: "extra plan operands", args: []string{"plan", "a", "b"}, op: "plan"},
		{name: "extra validate operands", args: []string{"validate", ".", "x"}, op: "validate"},
		{name: "missing view", args: []string{"inspect", "--workspace", dir}, op: "inspect"},
		{name: "missing inspect workspace", args: []string{"inspect", "run"}, op: "inspect"},
		{name: "missing inspect args", args: []string{"inspect"}, op: "inspect"},
		{name: "missing release workspace", args: []string{"release"}, op: "release"},
		{name: "missing run workspace", args: []string{"run"}, op: "run"},
		{name: "missing resume workspace", args: []string{"resume"}, op: "resume"},
		{name: "non-integer cap", args: []string{"run", "--workspace", dir, "--cap", "x"}, op: "run"},
		{name: "empty cap", args: []string{"run", "--workspace", dir, "--cap="}, op: "run"},
		{name: "repeated workspace", args: []string{"release", "--workspace", "a", "--workspace", "b"}, op: "release"},
		{name: "repeated cap", args: []string{"run", "--workspace", dir, "--cap", "1", "--cap", "2"}, op: "run"},
		{name: "repeated instance", args: []string{"inspect", "run", "--workspace", dir, "--instance", "a", "--instance", "b"}, op: "inspect"},
		{name: "help and version flags", args: []string{"--help", "--version"}, op: "cli"},
		{name: "compose help and version", args: []string{"compose", "--help", "--version"}, op: "compose"},
		{name: "run version", args: []string{"run", "--version"}, op: "run"},
		{name: "inspect version", args: []string{"inspect", "--version"}, op: "inspect"},
		{name: "version then run", args: []string{"--version", "run"}, op: "run"},
		{name: "inspect extra package", args: []string{"inspect", "run", "--workspace", dir, "pkg"}, op: "inspect"},
		{name: "release extra package", args: []string{"release", "--workspace", dir, "pkg"}, op: "release"},
		{name: "unknown --view", args: []string{"inspect", "run", "--view", "run", "--workspace", dir}, op: "inspect"},
		{name: "cap on inspect", args: []string{"inspect", "run", "--workspace", dir, "--cap", "1"}, op: "inspect"},
		{name: "instance on run", args: []string{"run", "--workspace", dir, "--instance", "x"}, op: "run"},
		{name: "workspace on compose", args: []string{"compose", "--workspace", dir}, op: "compose"},
		{name: "repeated sample", args: []string{"compose", "--sample", "a.csv", "--sample", "b.csv"}, op: "compose"},
		{name: "empty sample", args: []string{"compose", "--sample="}, op: "compose"},
		{name: "whitespace sample", args: []string{"plan", "--sample", " \t"}, op: "plan"},
		{name: "missing sample value", args: []string{"validate", "--sample"}, op: "validate"},
		{name: "unknown --samples", args: []string{"compose", "--samples", "a.csv"}, op: "compose"},
		{name: "unknown --constructor", args: []string{"compose", "--constructor", "RNASeq"}, op: "compose"},
		{name: "unknown --no-sample", args: []string{"compose", "--no-sample", "x"}, op: "compose"},
		{name: "sample on inspect", args: []string{"inspect", "run", "--workspace", dir, "--sample", "a.csv"}, op: "inspect"},
		{name: "sample on release", args: []string{"release", "--workspace", dir, "--sample", "a.csv"}, op: "release"},
		{name: "sample on help", args: []string{"help", "--sample", "a.csv"}, op: "cli"},
		{name: "sample on version", args: []string{"version", "--sample", "a.csv"}, op: "cli"},
		{name: "sample on inspect help", args: []string{"inspect", "--help", "--sample", "a.csv"}, op: "inspect"},
		{name: "sample equals form on release", args: []string{"release", "--workspace", dir, "--sample=a.csv"}, op: "release"},
		{name: "help unknown command", args: []string{"help", "nope"}, op: "cli"},
		{name: "version extra", args: []string{"version", "extra"}, op: "cli"},
		{name: "missing workspace value", args: []string{"release", "--workspace"}, op: "release"},
		{name: "empty workspace", args: []string{"release", "--workspace="}, op: "release"},
		{name: "help with version flag", args: []string{"help", "--version"}, op: "cli"},
		{name: "unknown short flag", args: []string{"-v"}, op: "cli"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireInvocationFailure(t, tc.args, tc.op)
		})
	}
}

func TestSampleHelp(t *testing.T) {
	graph := []string{"compose", "validate", "plan", "run", "resume"}
	for _, cmd := range graph {
		t.Run(cmd, func(t *testing.T) {
			res := runCLI(cmd, "--help")
			if res.code != 0 {
				t.Fatalf("exit = %d, want 0\nstderr: %s", res.code, res.stderr)
			}
			out := string(res.stdout)
			if !strings.Contains(out, "[--sample PATH]") {
				t.Fatalf("help missing --sample PATH: %s", out)
			}
			if !strings.Contains(out, "samplesheet CSV") {
				t.Fatalf("help missing samplesheet meaning: %s", out)
			}
			if !strings.Contains(out, "samplesheet.csv in the process current directory") {
				t.Fatalf("help missing process cwd default: %s", out)
			}
		})
	}
	for _, cmd := range []string{"inspect", "release"} {
		t.Run(cmd, func(t *testing.T) {
			res := runCLI(cmd, "--help")
			if res.code != 0 {
				t.Fatalf("exit = %d, want 0\nstderr: %s", res.code, res.stderr)
			}
			if strings.Contains(string(res.stdout), "--sample") {
				t.Fatalf("%s help mentions --sample: %s", cmd, res.stdout)
			}
		})
	}
}

func TestParseSampleFlag(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name   string
		args   []string
		sample string
	}{
		{name: "compose default", args: []string{"compose"}, sample: gobble.DefaultSampleSheetPath},
		{name: "validate default", args: []string{"validate", "."}, sample: gobble.DefaultSampleSheetPath},
		{name: "plan default", args: []string{"plan"}, sample: gobble.DefaultSampleSheetPath},
		{name: "run default", args: []string{"run", "--workspace", dir}, sample: gobble.DefaultSampleSheetPath},
		{name: "resume default", args: []string{"resume", "--workspace", dir}, sample: gobble.DefaultSampleSheetPath},
		{name: "compose space form", args: []string{"compose", "--sample", "foo.csv"}, sample: "foo.csv"},
		{name: "compose equals form", args: []string{"compose", "--sample=rel/sheet.csv"}, sample: "rel/sheet.csv"},
		{name: "relative not cleaned", args: []string{"plan", "--sample", "../x.csv"}, sample: "../x.csv"},
		{name: "run explicit", args: []string{"run", "--workspace", dir, "--sample", "s.csv"}, sample: "s.csv"},
		{name: "resume equals", args: []string{"resume", "--sample=keep/as.csv", "--workspace", dir}, sample: "keep/as.csv"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := parse(tc.args)
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}
			if req.sample != tc.sample {
				t.Fatalf("sample = %q, want %q", req.sample, tc.sample)
			}
		})
	}
}

func TestGraphVerbsParseWithoutCompile(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		args []string
		op   string
	}{
		{args: []string{"run", "--workspace", dir}, op: "run"},
		{args: []string{"resume", "--workspace", dir}, op: "resume"},
	}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			requireInvocationFailure(t, tc.args, tc.op)
			res := runCLI(tc.args...)
			if bytes.Contains(bytes.ToLower(res.stderr), []byte("not implemented")) {
				t.Fatalf("stderr contains not implemented: %s", res.stderr)
			}
		})
	}
}

func requireInvocationFailure(t *testing.T, args []string, wantOp string) {
	t.Helper()
	res := runCLI(args...)
	if len(res.stdout) != 0 {
		t.Fatalf("stdout = %q, want empty", res.stdout)
	}
	if res.code != 2 {
		t.Fatalf("exit = %d, want 2\nstderr: %s", res.code, res.stderr)
	}
	var ge gobble.Error
	if err := json.Unmarshal(res.stderr, &ge); err != nil {
		t.Fatalf("stderr JSON: %v\n%s", err, res.stderr)
	}
	if ge.Op != wantOp {
		t.Fatalf("op = %q, want %q", ge.Op, wantOp)
	}
	if len(ge.Defects) == 0 || ge.Defects[0].Code != gobble.DefectInvalidRequest {
		t.Fatalf("defects = %#v, want invalid-request", ge.Defects)
	}
	var raw map[string]any
	if err := json.Unmarshal(res.stderr, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["schema_version"]; ok {
		t.Fatalf("error JSON has schema_version: %#v", raw)
	}
	if _, ok := raw["op"]; !ok {
		t.Fatalf("error JSON missing op: %#v", raw)
	}
	if _, ok := raw["defects"]; !ok {
		t.Fatalf("error JSON missing defects: %#v", raw)
	}
}
