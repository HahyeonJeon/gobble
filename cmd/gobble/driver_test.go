package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/cmd/gobble/testdata/badbackend"
	"github.com/HahyeonJeon/gobble/cmd/gobble/testdata/badpipe"
	"github.com/HahyeonJeon/gobble/cmd/gobble/testdata/nilpipe"
	"github.com/HahyeonJeon/gobble/cmd/gobble/testdata/okpipe"
)

func TestResolveImportIgnoresListStderr(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "go")
	script := "#!/bin/sh\necho 'go: downloading example.com/x v1.0.0' >&2\necho 'example.com/pipe'\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveImport(stub, dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	if got != "example.com/pipe" {
		t.Fatalf("import path = %q, want example.com/pipe", got)
	}
}

func TestDriverWaitCodeSignaled(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	code := driverWaitCode(err)
	want := 128 + int(syscall.SIGKILL)
	if code != want {
		t.Fatalf("wait code = %d, want %d", code, want)
	}
}

func TestMissingGo(t *testing.T) {
	watchDriverTemps(t)
	t.Setenv("PATH", t.TempDir())
	res := runCLI("compose", "./testdata/okpipe")
	requireCompileFailure(t, res, "compose")
}

func TestMissingPipeline(t *testing.T) {
	watchDriverTemps(t)
	res := runCLI("compose", "./testdata/nopipe")
	requireCompileFailure(t, res, "compose")
	if !strings.Contains(string(res.stderr), "Pipeline") {
		t.Fatalf("message missing compiler text: %s", res.stderr)
	}
}

func TestPackageMainUserPackage(t *testing.T) {
	watchDriverTemps(t)
	res := runCLI("compose", "./testdata/mainpipe")
	requireCompileFailure(t, res, "compose")
	if !strings.Contains(strings.ToLower(string(res.stderr)), "import") {
		t.Fatalf("message missing compiler text: %s", res.stderr)
	}
}

func TestPipelinePanicForwardsAbort(t *testing.T) {
	watchDriverTemps(t)
	res := runCLI("compose", "./testdata/panicpipe")
	if len(res.stdout) != 0 {
		t.Fatalf("stdout = %q, want empty", res.stdout)
	}
	if res.code == 0 {
		t.Fatalf("exit = 0, want driver abort")
	}
	if bytes.Contains(res.stderr, []byte(`"code":"invalid-request"`)) && json.Valid(bytes.TrimSpace(res.stderr)) {
		t.Fatalf("panic wrapped as invalid-request JSON: %s", res.stderr)
	}
	if !bytes.Contains(res.stderr, []byte("panic:")) {
		t.Fatalf("stderr missing panic trace: %s", res.stderr)
	}
}

func TestComposeSuccess(t *testing.T) {
	watchDriverTemps(t)
	res := runCLI("compose", "./testdata/okpipe")
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	if len(res.stderr) != 0 {
		t.Fatalf("stderr = %q, want empty", res.stderr)
	}
	want := "{\"op\":\"compose\",\"pipeline\":\"" + okpipe.Pipeline().Name() + "\"}\n"
	if string(res.stdout) != want {
		t.Fatalf("stdout = %q, want %q", res.stdout, want)
	}
	var raw map[string]any
	if err := json.Unmarshal(res.stdout, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["schema_version"]; ok {
		t.Fatalf("compose JSON has schema_version: %#v", raw)
	}
}

func TestComposeWritesNoGraphFile(t *testing.T) {
	watchDriverTemps(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(cwd, "testdata", "okpipe")
	beforeCWD := namesIn(t, cwd)
	beforeFix := namesIn(t, fixture)
	res := runCLI("compose", "./testdata/okpipe")
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	afterCWD := namesIn(t, cwd)
	afterFix := namesIn(t, fixture)
	if afterCWD != beforeCWD {
		t.Fatalf("compose changed cwd files:\nbefore %s\nafter %s", beforeCWD, afterCWD)
	}
	if afterFix != beforeFix {
		t.Fatalf("compose changed fixture files:\nbefore %s\nafter %s", beforeFix, afterFix)
	}
	for _, name := range []string{"graph.json", "compose.json"} {
		if _, err := os.Stat(filepath.Join(cwd, name)); err == nil {
			t.Fatalf("compose wrote %s", name)
		}
	}
}

func TestNilPipelineDomainError(t *testing.T) {
	watchDriverTemps(t)
	_, libErr := gobble.Compose(nilpipe.Pipeline())
	res := runCLI("compose", "./testdata/nilpipe")
	requireDomainError(t, res, libErr)
}

func TestComposeDefectEmptyStdout(t *testing.T) {
	watchDriverTemps(t)
	_, libErr := gobble.Compose(badpipe.Pipeline())
	res := runCLI("compose", "./testdata/badpipe")
	requireDomainError(t, res, libErr)
}

func TestValidateSuccess(t *testing.T) {
	watchDriverTemps(t)
	res := runCLI("validate", "./testdata/okpipe")
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	if len(res.stderr) != 0 {
		t.Fatalf("stderr = %q, want empty", res.stderr)
	}
	if string(res.stdout) != "{\"op\":\"validate\"}\n" {
		t.Fatalf("stdout = %q, want {\"op\":\"validate\"}\\n", res.stdout)
	}
	var raw map[string]any
	if err := json.Unmarshal(res.stdout, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["schema_version"]; ok {
		t.Fatalf("validate JSON has schema_version: %#v", raw)
	}
}

func TestValidateDefect(t *testing.T) {
	watchDriverTemps(t)
	g, err := gobble.Compose(badbackend.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	libErr := gobble.Validate(g)
	res := runCLI("validate", "./testdata/badbackend")
	requireDomainError(t, res, libErr)
}

func TestPlanWriteJSON(t *testing.T) {
	watchDriverTemps(t)
	g, err := gobble.Compose(okpipe.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	p, err := gobble.BuildPlan(g)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	var want bytes.Buffer
	if err := p.WriteJSON(&want); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	res := runCLI("plan", "./testdata/okpipe")
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	if len(res.stderr) != 0 {
		t.Fatalf("stderr = %q, want empty", res.stderr)
	}
	if !bytes.Equal(res.stdout, want.Bytes()) {
		t.Fatalf("stdout %q != WriteJSON %q", res.stdout, want.Bytes())
	}
}

func TestPlanFailureEmptyStdout(t *testing.T) {
	watchDriverTemps(t)
	g, err := gobble.Compose(badbackend.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	_, libErr := gobble.BuildPlan(g)
	res := runCLI("plan", "./testdata/badbackend")
	requireDomainError(t, res, libErr)
}

func requireCompileFailure(t *testing.T, res cliResult, wantOp string) {
	t.Helper()
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
	if ge.Defects[0].Message == "" {
		t.Fatalf("empty compile message: %#v", ge.Defects)
	}
}

func watchDriverTemps(t *testing.T) {
	t.Helper()
	pattern := filepath.Join(os.TempDir(), driverTempPrefix+"*")
	before, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]struct{}, len(before))
	for _, p := range before {
		seen[p] = struct{}{}
	}
	t.Cleanup(func() {
		after, err := filepath.Glob(pattern)
		if err != nil {
			t.Errorf("glob driver temps: %v", err)
			return
		}
		for _, p := range after {
			if _, ok := seen[p]; !ok {
				t.Errorf("leftover driver temp %s", p)
			}
		}
	})
}

func namesIn(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return strings.Join(names, "\n")
}
