package main

import (
	"bytes"
	"debug/buildinfo"
	"debug/elf"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestMakePackedIdentity(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(*gobble.Identity)
		wantErr string
	}{
		{name: "local pin"},
		{name: "exact tag", edit: func(id *gobble.Identity) {
			id.IdentityMode = "exact-tag"
			id.GobbleVCSRevision = ""
		}},
		{name: "dirty gobble", edit: func(id *gobble.Identity) {
			id.GobbleVCSModified = true
			id.GobbleSourceRealpath = "/gobble"
		}, wantErr: "dirty Gobble source"},
		{name: "dirty pipeline", edit: func(id *gobble.Identity) {
			id.PipelineVCSModified = true
			id.PipelineSourceRealpath = "/pipeline"
		}, wantErr: "dirty pipeline source"},
		{name: "empty pipeline revision", edit: func(id *gobble.Identity) {
			id.PipelineVCSRevision = ""
		}, wantErr: "empty pipeline revision"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			install := testPackInstallIdentity()
			if tc.edit != nil {
				tc.edit(&install.Identity)
			}
			got, err := makePackedIdentity(install)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("makePackedIdentity() error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("makePackedIdentity() error = %v", err)
			}
			if got.Identity.InstallKind != "packed-runner" {
				t.Fatalf("install kind = %q, want packed-runner", got.Identity.InstallKind)
			}
			if got.Identity.GobbleExecutableSHA256 != "" {
				t.Fatalf("executable digest = %q, want empty", got.Identity.GobbleExecutableSHA256)
			}
			if got.Identity.IdentityMode != install.Identity.IdentityMode || got.Identity.GobbleVCSRevision != install.Identity.GobbleVCSRevision {
				t.Fatalf("packed identity changed family fields: got %#v, input %#v", got.Identity, install.Identity)
			}
			if got.Identity.GobbleSourceRealpath != "" || got.Identity.PipelineSourceRealpath != "" {
				t.Fatalf("packed identity retained realpath: %#v", got.Identity)
			}
		})
	}
}

func TestPackInnerSourceContract(t *testing.T) {
	install, err := makePackedIdentity(testPackInstallIdentity())
	if err != nil {
		t.Fatal(err)
	}
	src, err := packInnerSource("example.test/pipeline", install)
	if err != nil {
		t.Fatalf("packInnerSource() error = %v", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated inner parse: %v\n%s", err, src)
	}
	imports := map[string]bool{}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		imports[path] = true
		if strings.Contains(path, "/internal/") || strings.HasSuffix(path, "/internal") {
			t.Fatalf("generated inner imports internal package %q", path)
		}
	}
	if !imports["example.test/pipeline"] || !imports[modulePath] {
		t.Fatalf("generated inner imports = %#v", imports)
	}
	for path := range imports {
		if strings.HasPrefix(path, modulePath+"/") {
			t.Fatalf("generated inner imports Gobble child package %q", path)
		}
	}
	withIdentity := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "gobble" {
			return true
		}
		switch sel.Sel.Name {
		case "Run", "Resume", "Inspect", "Release":
			for _, arg := range call.Args {
				option, ok := arg.(*ast.CallExpr)
				if !ok {
					continue
				}
				optionSel, ok := option.Fun.(*ast.SelectorExpr)
				if ok && optionSel.Sel.Name == "WithIdentity" {
					withIdentity[sel.Sel.Name] = true
				}
			}
		}
		return true
	})
	for _, name := range []string{"Run", "Resume", "Inspect", "Release"} {
		if !withIdentity[name] {
			t.Fatalf("generated inner %s omits WithIdentity", name)
		}
	}
	text := string(src)
	if strings.Contains(text, `case "pack"`) || strings.Contains(text, `"output"`) || strings.Contains(text, `--output`) {
		t.Fatalf("generated inner accepts pack/output:\n%s", src)
	}
}

func TestPackFailuresPreserveDestination(t *testing.T) {
	t.Run("missing go", func(t *testing.T) {
		watchPackTemps(t)
		destination := filepath.Join(t.TempDir(), "runner")
		old := []byte("existing destination\n")
		if err := os.WriteFile(destination, old, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", t.TempDir())
		res := runCLI("pack", "./testdata/printpipe", "--output", destination)
		requirePackFailure(t, res)
		requireFileBytes(t, destination, old)
	})

	t.Run("directory before go lookup", func(t *testing.T) {
		watchPackTemps(t)
		destination := t.TempDir()
		t.Setenv("PATH", t.TempDir())
		res := runCLI("pack", "./testdata/printpipe", "--output", destination)
		requirePackFailure(t, res)
		if !strings.Contains(string(res.stderr), "output path is a directory") {
			t.Fatalf("stderr = %s, want directory refusal", res.stderr)
		}
		if info, err := os.Stat(destination); err != nil || !info.IsDir() {
			t.Fatalf("destination directory changed: info=%v err=%v", info, err)
		}
	})

	t.Run("internal package", func(t *testing.T) {
		watchPackTemps(t)
		destination := filepath.Join(t.TempDir(), "runner")
		old := []byte("keep internal refusal\n")
		if err := os.WriteFile(destination, old, 0o755); err != nil {
			t.Fatal(err)
		}
		res := runCLI("pack", "../../internal/engine", "--output", destination)
		requirePackFailure(t, res)
		if !strings.Contains(string(res.stderr), "consumer internal/ packages are unsupported") {
			t.Fatalf("stderr = %s, want internal-package refusal", res.stderr)
		}
		requireFileBytes(t, destination, old)
	})

	t.Run("dirty identity", func(t *testing.T) {
		watchPackTemps(t)
		destination := filepath.Join(t.TempDir(), "runner")
		old := []byte("keep dirty refusal\n")
		if err := os.WriteFile(destination, old, 0o755); err != nil {
			t.Fatal(err)
		}
		previous := resolvePackInstallIdentity
		resolvePackInstallIdentity = func(_, _, _, _ string) (installIdentityResult, error) {
			install := testPackInstallIdentity()
			install.Identity.PipelineVCSModified = true
			install.Identity.PipelineSourceRealpath = "/dirty/pipeline"
			return install, nil
		}
		t.Cleanup(func() { resolvePackInstallIdentity = previous })
		res := runCLI("pack", "./testdata/printpipe", "--output", destination)
		requirePackFailure(t, res)
		if !strings.Contains(string(res.stderr), "dirty pipeline source") {
			t.Fatalf("stderr = %s, want dirty-pipeline refusal", res.stderr)
		}
		requireFileBytes(t, destination, old)
	})
}

func TestPackPrintpipeArtifact(t *testing.T) {
	watchPackTemps(t)
	expected := useCleanPackIdentity(t, "./testdata/printpipe")
	destinationDir := t.TempDir()
	destination := filepath.Join(destinationDir, "gobble-printpipe")
	if err := os.WriteFile(destination, []byte("replace me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := runCLI("pack", "./testdata/printpipe", "--output", destination)
	if res.code != 0 || len(res.stdout) != 0 || len(res.stderr) != 0 {
		t.Fatalf("pack result = code %d stdout %q stderr %s", res.code, res.stdout, res.stderr)
	}
	entries, err := os.ReadDir(destinationDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("destination entries = %#v, want only %q", entries, filepath.Base(destination))
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("runner mode = %o, want 755", info.Mode().Perm())
	}
	elfFile, err := elf.Open(destination)
	if err != nil {
		t.Fatalf("open packed ELF: %v", err)
	}
	if elfFile.Machine != elf.EM_X86_64 {
		t.Fatalf("ELF machine = %v, want x86-64", elfFile.Machine)
	}
	if err := elfFile.Close(); err != nil {
		t.Fatal(err)
	}
	build, err := buildinfo.ReadFile(destination)
	if err != nil {
		t.Fatalf("read packed build info: %v", err)
	}
	settings := map[string]string{}
	for _, setting := range build.Settings {
		settings[setting.Key] = setting.Value
	}
	for key, want := range map[string]string{"GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"} {
		if settings[key] != want {
			t.Fatalf("build setting %s = %q, want %q", key, settings[key], want)
		}
	}

	compose := runPacked(destination, "compose")
	if compose.code != 0 || len(compose.stderr) != 0 {
		t.Fatalf("packed compose = code %d stderr %s", compose.code, compose.stderr)
	}
	wantCompose := "{\"op\":\"compose\",\"pipeline\":\"printed\"}\n"
	if string(compose.stdout) != wantCompose {
		t.Fatalf("packed compose stdout = %q, want %q", compose.stdout, wantCompose)
	}
	for _, leaked := range []string{"user init output", "user Pipeline output"} {
		if bytes.Contains(compose.stdout, []byte(leaked)) {
			t.Fatalf("packed compose leaked %q: %s", leaked, compose.stdout)
		}
	}

	unknown := runPacked(destination, "pack")
	if unknown.code != 2 || len(unknown.stdout) != 0 {
		t.Fatalf("packed pack = code %d stdout %q stderr %s", unknown.code, unknown.stdout, unknown.stderr)
	}
	var unknownErr gobble.Error
	if err := json.Unmarshal(unknown.stderr, &unknownErr); err != nil || unknownErr.Op != "cli" {
		t.Fatalf("packed pack stderr = %s, error %v", unknown.stderr, err)
	}

	version := runPacked(destination, "version")
	if version.code != 0 || len(version.stderr) != 0 {
		t.Fatalf("packed version = code %d stderr %s", version.code, version.stderr)
	}
	var versionJSON map[string]any
	if err := json.Unmarshal(version.stdout, &versionJSON); err != nil {
		t.Fatalf("packed version JSON: %v\n%s", err, version.stdout)
	}
	identity := expected.Identity
	for key, want := range map[string]any{
		"process":               "packed-runner",
		"install_kind":          "packed-runner",
		"identity_mode":         identity.IdentityMode,
		"executable_sha256":     "",
		"pipeline_module":       identity.PipelineModule,
		"pipeline_import":       identity.PipelineImport,
		"pipeline_version":      identity.PipelineVersion,
		"pipeline_vcs_revision": identity.PipelineVCSRevision,
		"pipeline_vcs_modified": false,
		"goos":                  "linux",
		"goarch":                "amd64",
	} {
		if got := versionJSON[key]; got != want {
			t.Fatalf("packed version %s = %#v, want %#v", key, got, want)
		}
	}
	if _, ok := versionJSON["schema_version"]; ok {
		t.Fatalf("packed version has schema_version: %#v", versionJSON)
	}

	help := runPacked(destination)
	if help.code != 0 || len(help.stderr) != 0 {
		t.Fatalf("packed help = code %d stderr %s", help.code, help.stderr)
	}
	helpText := strings.ToLower(strings.Join(strings.Fields(string(help.stdout)), " "))
	for _, want := range []string{"linux/amd64", "one embedded pipeline", "docker", "gobble portions", "different license", "no go", "first-horizon", "copyright (c) 2026 hahyeonjeon", "permission is hereby granted"} {
		if !strings.Contains(helpText, want) {
			t.Fatalf("packed help missing %q: %s", want, help.stdout)
		}
	}
	for _, forbidden := range []string{"go on path", "[package]", "--output", "license is unset", "runner is licensed under mit", "it is licensed under mit"} {
		if strings.Contains(helpText, forbidden) {
			t.Fatalf("packed help contains %q: %s", forbidden, help.stdout)
		}
	}
	licenseText, err := os.ReadFile("../../LICENSE")
	if err != nil {
		t.Fatalf("read LICENSE: %v", err)
	}
	if !bytes.Contains(help.stdout, licenseText) {
		t.Fatalf("packed root help omits Gobble's exact MIT notice:\n%s", help.stdout)
	}
	commandHelp := runPacked(destination, "help", "compose")
	if commandHelp.code != 0 || len(commandHelp.stderr) != 0 {
		t.Fatalf("packed command help = code %d stderr %s", commandHelp.code, commandHelp.stderr)
	}
	commandText := strings.ToLower(strings.Join(strings.Fields(string(commandHelp.stdout)), " "))
	for _, want := range []string{"gobble portions", "mit", "embedded pipeline", "different license", "root help"} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("packed command help missing %q: %s", want, commandHelp.stdout)
		}
	}
}

func TestPackHostpipeEmptyInspectProtocol(t *testing.T) {
	watchPackTemps(t)
	useCleanPackIdentity(t, "./testdata/hostpipe")
	runner := filepath.Join(t.TempDir(), "gobble-hostpipe")
	pack := runCLI("pack", "./testdata/hostpipe", "--output", runner)
	if pack.code != 0 || len(pack.stdout) != 0 || len(pack.stderr) != 0 {
		t.Fatalf("pack result = code %d stdout %q stderr %s", pack.code, pack.stdout, pack.stderr)
	}

	workspace := t.TempDir()
	requireOpSuccess(t, runPacked(runner, "run", "--workspace", workspace), "run")
	requireOccupied(t, workspace)
	requirePackedEmptyInspect(t, runner, workspace, "remaining")
	requirePackedEmptyInspect(t, runner, workspace, "reuse")

	requireOpSuccess(t, runPacked(runner, "release", "--workspace", workspace), "release")
	requirePackedEmptyInspect(t, runner, workspace, "remaining")
	requirePackedEmptyInspect(t, runner, workspace, "reuse")
	requireOpSuccess(t, runPacked(runner, "resume", "--workspace", workspace), "resume")
	requireOccupied(t, workspace)
	requirePackedEmptyInspect(t, runner, workspace, "remaining")
	reuse := runPacked(runner, "inspect", "reuse", "--workspace", workspace)
	if reuse.code != 0 || len(reuse.stdout) == 0 || len(reuse.stderr) != 0 {
		t.Fatalf("packed inspect reuse after Resume = code %d stdout %q stderr %s", reuse.code, reuse.stdout, reuse.stderr)
	}
	for _, line := range bytes.Split(bytes.TrimSpace(reuse.stdout), []byte{'\n'}) {
		if !json.Valid(line) {
			t.Fatalf("packed inspect reuse after Resume has invalid JSONL record %q", line)
		}
	}
}

func TestPackForcedTempFallback(t *testing.T) {
	watchPackTemps(t)
	install := useCleanPackIdentity(t, "./testdata/printpipe")
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	innerDir := t.TempDir()
	innerSource, err := packInnerSource(install.Identity.PipelineImport, install)
	if err != nil {
		t.Fatal(err)
	}
	innerMain := filepath.Join(innerDir, "main.go")
	if err := os.WriteFile(innerMain, []byte(innerSource), 0o600); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(innerDir, packInnerName)
	if err := buildPacked(goBin, cwd, innerMain, inner); err != nil {
		t.Fatal(err)
	}
	trampolineDir := t.TempDir()
	if err := copyPackFile(inner, filepath.Join(trampolineDir, packInnerName), 0o600); err != nil {
		t.Fatal(err)
	}
	trampolineSource, err := packTrampolineSource(true)
	if err != nil {
		t.Fatal(err)
	}
	trampolineMain := filepath.Join(trampolineDir, "main.go")
	if err := os.WriteFile(trampolineMain, []byte(trampolineSource), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := filepath.Join(trampolineDir, "runner")
	if err := buildPacked(goBin, cwd, trampolineMain, runner); err != nil {
		t.Fatal(err)
	}
	compose := runPacked(runner, "compose")
	if compose.code != 0 || len(compose.stderr) != 0 || string(compose.stdout) != "{\"op\":\"compose\",\"pipeline\":\"printed\"}\n" {
		t.Fatalf("forced fallback compose = code %d stdout %q stderr %s", compose.code, compose.stdout, compose.stderr)
	}
}

func TestPackedProtocolValidator(t *testing.T) {
	watchPackTemps(t)
	innerDir := t.TempDir()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	innerSource := filepath.Join(innerDir, "main.go")
	source := `package main

import "os"

func main() {
	protocol := os.NewFile(3, "protocol")
	if len(os.Args) < 2 {
		return
	}
	switch os.Args[1] {
	case "jsonl":
		_, _ = protocol.WriteString("{\"n\":1}\n{\"n\":2}\n")
	case "inspect":
		for _, arg := range os.Args[2:] {
			if arg == "--workspace=whitespace" {
				_, _ = protocol.WriteString(" \n")
			}
		}
	case "invalid":
		_, _ = protocol.WriteString("not protocol")
	}
}
`
	if err := os.WriteFile(innerSource, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(innerDir, packInnerName)
	if err := buildPacked(goBin, cwd, innerSource, inner); err != nil {
		t.Fatalf("build protocol-test inner: %v", err)
	}
	runner, err := buildPackedTrampoline(goBin, cwd, t.TempDir(), inner, true)
	if err != nil {
		t.Fatalf("build protocol-test trampoline: %v", err)
	}
	jsonl := runPacked(runner, "jsonl")
	if jsonl.code != 0 || string(jsonl.stdout) != "{\"n\":1}\n{\"n\":2}\n" || len(jsonl.stderr) != 0 {
		t.Fatalf("JSONL protocol = code %d stdout %q stderr %s", jsonl.code, jsonl.stdout, jsonl.stderr)
	}
	invalid := runPacked(runner, "invalid")
	if invalid.code != 1 || len(invalid.stdout) != 0 || !bytes.Contains(invalid.stderr, []byte("invalid child protocol")) {
		t.Fatalf("invalid protocol = code %d stdout %q stderr %s", invalid.code, invalid.stdout, invalid.stderr)
	}
	for _, args := range [][]string{
		{"inspect", "remaining", "--workspace", "/tmp/workspace"},
		{"--workspace=/tmp/workspace", "inspect", "reuse"},
		{"watch", "--workspace", "/tmp/workspace"},
		{"--workspace=/tmp/workspace", "watch"},
	} {
		empty := runPacked(runner, args...)
		if empty.code != 0 || len(empty.stdout) != 0 || len(empty.stderr) != 0 {
			t.Fatalf("empty Inspect protocol %q = code %d stdout %q stderr %s", args, empty.code, empty.stdout, empty.stderr)
		}
	}
	for _, args := range [][]string{
		{"compose"},
		{"run", "--workspace", "/tmp/workspace"},
		{"inspect", "remaining"},
		{"inspect", "reuse", "--unknown=value"},
		{"inspect", "remaining", "--workspace=whitespace"},
		{"watch"},
		{"watch", "--workspace=/tmp/workspace", "--instance=a"},
		{"watch", "extra", "--workspace=/tmp/workspace"},
	} {
		empty := runPacked(runner, args...)
		if empty.code != 1 || len(empty.stdout) != 0 || !bytes.Contains(empty.stderr, []byte("invalid child protocol")) {
			t.Fatalf("invalid empty protocol %q = code %d stdout %q stderr %s", args, empty.code, empty.stdout, empty.stderr)
		}
	}
}

func useCleanPackIdentity(t *testing.T, pkg string) installIdentityResult {
	t.Helper()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	importPath, err := resolveImport(goBin, cwd, pkg)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := resolveInstallIdentity(goBin, cwd, importPath, "pack")
	if err != nil {
		t.Fatal(err)
	}
	parent.Identity.GobbleVCSModified = false
	parent.Identity.GobbleSourceRealpath = ""
	parent.Identity.PipelineVCSModified = false
	parent.Identity.PipelineSourceRealpath = ""
	packed, err := makePackedIdentity(parent)
	if err != nil {
		t.Fatal(err)
	}
	previous := resolvePackInstallIdentity
	resolvePackInstallIdentity = func(_, _, _, _ string) (installIdentityResult, error) {
		return parent, nil
	}
	t.Cleanup(func() { resolvePackInstallIdentity = previous })
	return packed
}

func requirePackFailure(t *testing.T, result cliResult) {
	t.Helper()
	if result.code != 2 || len(result.stdout) != 0 {
		t.Fatalf("pack failure = code %d stdout %q stderr %s", result.code, result.stdout, result.stderr)
	}
	var ge gobble.Error
	if err := json.Unmarshal(result.stderr, &ge); err != nil {
		t.Fatalf("pack stderr JSON: %v\n%s", err, result.stderr)
	}
	if ge.Op != "pack" || len(ge.Defects) == 0 || ge.Defects[0].Code != gobble.DefectInvalidRequest {
		t.Fatalf("pack error = %#v", ge)
	}
}

func requireFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s bytes = %q, want %q", path, got, want)
	}
}

func runPacked(path string, args ...string) cliResult {
	var stdout, stderr bytes.Buffer
	command := exec.Command(path, args...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}
	return cliResult{stdout: stdout.Bytes(), stderr: stderr.Bytes(), code: code}
}

func requirePackedEmptyInspect(t *testing.T, runner, workspace, view string) {
	t.Helper()
	args := []string{"inspect", view, "--workspace", workspace}
	result := runPacked(runner, args...)
	if result.code != 0 || len(result.stdout) != 0 || len(result.stderr) != 0 {
		t.Fatalf("packed inspect %s = code %d stdout %q stderr %s", view, result.code, result.stdout, result.stderr)
	}
}

func watchPackTemps(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("TMPDIR", root)
	t.Cleanup(func() {
		patterns := []string{packTempPrefix + "*", "gobble-packed-inner-*", "gobble-packed-protocol-*"}
		for _, pattern := range patterns {
			leftovers, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				t.Errorf("glob pack temps: %v", err)
				continue
			}
			for _, path := range leftovers {
				t.Errorf("leftover pack temp %s", path)
			}
		}
	})
}

func TestPackedHelpLicenseBoundary(t *testing.T) {
	licenseText, err := os.ReadFile("../../LICENSE")
	if err != nil {
		t.Fatalf("read LICENSE: %v", err)
	}
	if !strings.Contains(packedRootHelp, string(licenseText)) {
		t.Fatalf("packed root help omits exact LICENSE text: %q", packedRootHelp)
	}
	for name, text := range packedCommandHelp {
		lower := strings.ToLower(strings.Join(strings.Fields(text), " "))
		for _, want := range []string{"gobble portions", "mit", "embedded pipeline", "different license", "root help"} {
			if !strings.Contains(lower, want) {
				t.Fatalf("packed %s help omits %q: %q", name, want, text)
			}
		}
	}
	for name, text := range map[string]string{"generic pack": commandHelp["pack"], "packed root": packedRootHelp} {
		lower := strings.ToLower(strings.Join(strings.Fields(text), " "))
		for _, want := range []string{"gobble portions", "mit", "embedded pipeline", "different license"} {
			if !strings.Contains(lower, want) {
				t.Fatalf("%s help omits %q: %q", name, want, text)
			}
		}
		for _, forbidden := range []string{"runner is licensed under mit", "it is licensed under mit"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s help licenses the whole runner with %q: %q", name, forbidden, text)
			}
		}
	}
}

func TestPackTrampolineSourceContract(t *testing.T) {
	src, err := packTrampolineSource(false)
	if err != nil {
		t.Fatalf("packTrampolineSource() error = %v", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated trampoline parse: %v\n%s", err, src)
	}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if path == modulePath || strings.Contains(path, "/internal/") || strings.Contains(path, "golang.org/x/sys") || strings.Contains(path, "example.test/pipeline") {
			t.Fatalf("generated trampoline imports forbidden package %q", path)
		}
	}
	for _, required := range []string{
		"syscall.Syscall(319",
		"cmd.ExtraFiles = []*os.File{protocol}",
		"cmd.Stdout = io.Discard",
		"cmd.Stderr = os.Stderr",
		"protocol.Seek(0, io.SeekStart)",
		"cmd.Wait()",
	} {
		if !strings.Contains(src, required) {
			t.Fatalf("generated trampoline missing %q:\n%s", required, src)
		}
	}
	if strings.Contains(src, "os.Pipe") {
		t.Fatalf("generated trampoline uses a protocol pipe:\n%s", src)
	}
	fallback, err := packTrampolineSource(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fallback, "const forceFallback = true") {
		t.Fatalf("forced fallback source does not set forceFallback:\n%s", fallback)
	}
}

func testPackInstallIdentity() installIdentityResult {
	return installIdentityResult{
		Identity: gobble.Identity{
			GobbleModule:           modulePath,
			GobbleVersion:          "v0.0.0",
			GobbleVCSRevision:      "gobble-revision",
			GobbleExecutableSHA256: strings.Repeat("a", 64),
			PipelineModule:         "example.test/pipeline",
			PipelineImport:         "example.test/pipeline",
			PipelineVersion:        "(devel)",
			PipelineVCSRevision:    "pipeline-revision",
			GOOS:                   "linux",
			GOARCH:                 "amd64",
			InstallKind:            "module",
			IdentityMode:           "local-pin",
		},
		HasReplace:  true,
		ReplacePath: "/source/gobble",
	}
}
