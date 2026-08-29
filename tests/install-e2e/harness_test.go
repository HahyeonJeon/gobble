//go:build live

package install_e2e_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	gobbleModule   = "github.com/HahyeonJeon/gobble"
	consumerModule = "example.com/gobble-install-assay"
)

type manifestEntry struct {
	Path   string
	Mode   string
	Size   int64
	SHA256 string
}

type commandResult struct {
	Stdout []byte
	Stderr []byte
	Code   int
	Err    error
}

type assay struct {
	root            string
	goBin           string
	gitBin          string
	dockerBin       string
	snapshot        string
	secondSnapshot  string
	worktreeHead    string
	manifestSHA256  string
	snapshotCommit  string
	consumer        string
	consumerCommit  string
	gobin           string
	gobble          string
	gobbleSHA256    string
	agentPATH       string
	agentEnv        []string
	neutralPATH     string
	neutralEnv      []string
	goStubMarker    string
	apiAssay        string
	stageWGS        string
	assetCache      string
	packedDir       string
	packedWorkflow  string
	packedWGS       string
	packedPrint     string
	packedWGSSHA256 string
}

func newAssay(t *testing.T) *assay {
	t.Helper()
	a := &assay{root: moduleRoot(t), dockerBin: "/usr/bin/docker"}
	var err error
	a.goBin, err = exec.LookPath("go")
	if err != nil {
		t.Fatalf("find go: %v", err)
	}
	a.gitBin, err = exec.LookPath("git")
	if err != nil {
		t.Fatalf("find git: %v", err)
	}
	if info, err := os.Stat(a.dockerBin); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("required Docker executable %s: %v", a.dockerBin, err)
	}
	docker := runCommand(a.dockerBin, a.root, replaceEnv(os.Environ(), map[string]string{"PATH": "/usr/bin:/bin"}), "info")
	if docker.Code != 0 {
		t.Fatalf("docker info failed: code=%d err=%v\nstdout: %s\nstderr: %s", docker.Code, docker.Err, docker.Stdout, docker.Stderr)
	}

	a.worktreeHead = strings.TrimSpace(string(requireCommand(t, runCommand(a.gitBin, a.root, nil, "rev-parse", "HEAD"), "git rev-parse HEAD").Stdout))
	if a.worktreeHead == "" {
		t.Fatal("worktree HEAD is empty")
	}

	paths := candidatePaths(t, a.gitBin, a.root)
	entries := readManifest(t, a.root, paths)
	a.manifestSHA256 = manifestDigest(entries)
	a.snapshot, a.snapshotCommit = createSnapshot(t, a.gitBin, a.root, paths, entries)
	a.secondSnapshot, _ = createSnapshot(t, a.gitBin, a.root, paths, entries)
	a.consumer, a.consumerCommit = createConsumer(t, a.gitBin, a.snapshot)

	a.gobin = t.TempDir()
	a.agentPATH = joinPATH(a.gobin, filepath.Dir(a.goBin), filepath.Dir(a.gitBin), filepath.Dir(a.dockerBin), "/bin")
	a.agentEnv = replaceEnv(os.Environ(), map[string]string{"GOBIN": a.gobin, "PATH": a.agentPATH})

	listed := requireCommand(t, runCommand(a.goBin, a.consumer, a.agentEnv, "list", "-m", gobbleModule), "go list -m gobble")
	if !bytes.Contains(listed.Stdout, []byte(a.snapshot)) {
		t.Fatalf("go list -m did not select snapshot %s: %s", a.snapshot, listed.Stdout)
	}
	requireCommand(t, runCommand(a.goBin, a.consumer, a.agentEnv, "install", gobbleModule+"/cmd/gobble"), "GOBIN go install gobble")
	a.gobble = filepath.Join(a.gobin, "gobble")
	requireExecutable(t, a.gobble)
	a.gobbleSHA256 = fileSHA256(t, a.gobble)
	resolved, err := lookPathIn("gobble", a.agentPATH)
	if err != nil {
		t.Fatalf("resolve installed gobble: %v", err)
	}
	if resolved != a.gobble {
		t.Fatalf("agent PATH resolves gobble to %s, want %s", resolved, a.gobble)
	}
	version := requireCommand(t, runCommand(a.gobble, a.consumer, a.agentEnv, "version"), "installed gobble version")
	var versionJSON map[string]any
	if err := json.Unmarshal(version.Stdout, &versionJSON); err != nil {
		t.Fatalf("installed version JSON: %v\n%s", err, version.Stdout)
	}
	if versionJSON["process"] != "generic-cli" || versionJSON["install_kind"] != "module" || versionJSON["identity_mode"] != "local-pin" {
		t.Fatalf("installed version identity = %#v", versionJSON)
	}
	if versionJSON["executable_sha256"] != a.gobbleSHA256 {
		t.Fatalf("installed version digest = %v, want %s", versionJSON["executable_sha256"], a.gobbleSHA256)
	}

	stubDir := t.TempDir()
	a.goStubMarker = filepath.Join(t.TempDir(), "go-called")
	writeFile(t, filepath.Join(stubDir, "go"), 0o755, "#!/bin/sh\nprintf called > \"$ASSAY_GO_MARKER\"\nexit 99\n")
	a.neutralPATH = joinPATH(stubDir, filepath.Dir(a.dockerBin), "/bin")
	a.neutralEnv = replaceEnv(os.Environ(), map[string]string{"ASSAY_GO_MARKER": a.goStubMarker, "PATH": a.neutralPATH})

	toolsDir := t.TempDir()
	a.apiAssay = filepath.Join(toolsDir, "api-assay")
	a.stageWGS = filepath.Join(toolsDir, "stage-wgs")
	requireCommand(t, runCommand(a.goBin, a.consumer, a.agentEnv, "build", "-o", a.apiAssay, "./cmd/api-assay"), "build API assay")
	requireCommand(t, runCommand(a.goBin, a.consumer, a.agentEnv, "build", "-o", a.stageWGS, "./cmd/stage-wgs"), "build WGS stager")
	requireExecutable(t, a.apiAssay)
	requireExecutable(t, a.stageWGS)
	a.assetCache = t.TempDir()
	a.packedDir = t.TempDir()
	a.packedWorkflow = filepath.Join(a.packedDir, "workflow-runner")
	a.packedWGS = filepath.Join(a.packedDir, "wgs-runner")
	a.packedPrint = filepath.Join(a.packedDir, "print-runner")
	requireCleanGit(t, a.gitBin, a.snapshot)
	requireCleanGit(t, a.gitBin, a.consumer)
	return a
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func candidatePaths(t *testing.T, gitBin, root string) []string {
	t.Helper()
	res := requireCommand(t, runCommand(gitBin, root, nil, "ls-files", "-z", "--cached", "--others", "--exclude-standard"), "git ls-files candidate")
	parts := bytes.Split(res.Stdout, []byte{0})
	paths := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, raw := range parts {
		if len(raw) == 0 {
			continue
		}
		path := string(raw)
		if seen[path] {
			continue
		}
		seen[path] = true
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			t.Fatalf("lstat candidate %s: %v", path, err)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func readManifest(t *testing.T, root string, paths []string) []manifestEntry {
	t.Helper()
	entries := make([]manifestEntry, 0, len(paths))
	for _, path := range paths {
		full := filepath.Join(root, filepath.FromSlash(path))
		info, err := os.Lstat(full)
		if err != nil {
			t.Fatalf("lstat manifest %s: %v", path, err)
		}
		var data []byte
		mode := ""
		switch {
		case info.Mode().IsRegular():
			data, err = os.ReadFile(full)
			if info.Mode().Perm()&0o111 != 0 {
				mode = "100755"
			} else {
				mode = "100644"
			}
		case info.Mode()&os.ModeSymlink != 0:
			var target string
			target, err = os.Readlink(full)
			data = []byte(target)
			mode = "120000"
		default:
			t.Fatalf("unsupported candidate file type %s: %s", path, info.Mode())
		}
		if err != nil {
			t.Fatalf("read manifest %s: %v", path, err)
		}
		sum := sha256.Sum256(data)
		entries = append(entries, manifestEntry{Path: path, Mode: mode, Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])})
	}
	return entries
}

func manifestDigest(entries []manifestEntry) string {
	h := sha256.New()
	for _, entry := range entries {
		_ = binary.Write(h, binary.BigEndian, uint64(len(entry.Path)))
		_, _ = io.WriteString(h, entry.Path)
		_ = binary.Write(h, binary.BigEndian, uint64(len(entry.Mode)))
		_, _ = io.WriteString(h, entry.Mode)
		_ = binary.Write(h, binary.BigEndian, entry.Size)
		_, _ = io.WriteString(h, entry.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func createSnapshot(t *testing.T, gitBin, root string, paths []string, want []manifestEntry) (string, string) {
	t.Helper()
	destination := t.TempDir()
	for _, path := range paths {
		source := filepath.Join(root, filepath.FromSlash(path))
		target := filepath.Join(destination, filepath.FromSlash(path))
		info, err := os.Lstat(source)
		if err != nil {
			t.Fatalf("lstat snapshot source %s: %v", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir snapshot %s: %v", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(source)
			if err != nil {
				t.Fatalf("readlink snapshot %s: %v", path, err)
			}
			if err := os.Symlink(link, target); err != nil {
				t.Fatalf("symlink snapshot %s: %v", path, err)
			}
			continue
		}
		copyFile(t, source, target, info.Mode().Perm())
	}
	got := readManifest(t, destination, paths)
	if !equalManifest(want, got) {
		t.Fatalf("candidate and snapshot manifests differ\ncandidate: %#v\nsnapshot: %#v", want, got)
	}
	initGitRepo(t, gitBin, destination, "evaluated Gobble candidate")
	requireCleanGit(t, gitBin, destination)
	commit := strings.TrimSpace(string(requireCommand(t, runCommand(gitBin, destination, nil, "rev-parse", "HEAD"), "snapshot commit").Stdout))
	if commit == "" {
		t.Fatal("snapshot commit is empty")
	}
	return destination, commit
}

func equalManifest(a, b []manifestEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func createConsumer(t *testing.T, gitBin, snapshot string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	goMod := fmt.Sprintf("module %s\n\ngo 1.26\n\nrequire %s v0.0.0\n\nreplace %s => %s\n", consumerModule, gobbleModule, gobbleModule, snapshot)
	writeFile(t, filepath.Join(dir, "go.mod"), 0o644, goMod)
	writeFile(t, filepath.Join(dir, "workflowcase", "pipe.go"), 0o644, workflowSource)
	writeFile(t, filepath.Join(dir, "wgs", "pipe.go"), 0o644, wgsSource)
	writeFile(t, filepath.Join(dir, "printpipe", "pipe.go"), 0o644, printpipeSource)
	writeFile(t, filepath.Join(dir, "internal", "hidden", "pipe.go"), 0o644, hiddenSource)
	writeFile(t, filepath.Join(dir, "markerpipe", "pipe.go"), 0o644, markerSource)
	writeFile(t, filepath.Join(dir, "cmd", "stage-wgs", "main.go"), 0o644, stageWGSSource)
	writeFile(t, filepath.Join(dir, "cmd", "api-assay", "main.go"), 0o644, apiAssaySource)
	initGitRepo(t, gitBin, dir, "external Gobble install consumer")
	requireCleanGit(t, gitBin, dir)
	commit := strings.TrimSpace(string(requireCommand(t, runCommand(gitBin, dir, nil, "rev-parse", "HEAD"), "consumer commit").Stdout))
	if commit == "" {
		t.Fatal("consumer commit is empty")
	}
	return dir, commit
}

func initGitRepo(t *testing.T, gitBin, dir, message string) {
	t.Helper()
	requireCommand(t, runCommand(gitBin, dir, nil, "init", "-q"), "git init")
	requireCommand(t, runCommand(gitBin, dir, nil, "config", "user.name", "Gobble Install Assay"), "git config user.name")
	requireCommand(t, runCommand(gitBin, dir, nil, "config", "user.email", "install-assay@example.invalid"), "git config user.email")
	requireCommand(t, runCommand(gitBin, dir, nil, "add", "-A", "-f", "--", "."), "git add snapshot")
	requireCommand(t, runCommand(gitBin, dir, nil, "commit", "-q", "-m", message), "git commit snapshot")
}

func requireCleanGit(t *testing.T, gitBin, dir string) {
	t.Helper()
	status := requireCommand(t, runCommand(gitBin, dir, nil, "status", "--porcelain"), "git status --porcelain")
	if len(status.Stdout) != 0 {
		t.Fatalf("git repository %s is dirty:\n%s", dir, status.Stdout)
	}
}

func stageWGSWorkspace(t *testing.T, a *assay, workspace string) {
	t.Helper()
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("create WGS workspace: %v", err)
	}
	requireCommand(t, runCommand(a.stageWGS, a.assetCache, a.agentEnv, workspace), "stage checked WGS pins")
}

func runCommand(executable, cwd string, env []string, args ...string) commandResult {
	cmd := exec.Command(executable, args...)
	cmd.Dir = cwd
	if env != nil {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return commandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Code: exitCode(err), Err: err}
}

func requireCommand(t *testing.T, result commandResult, operation string) commandResult {
	t.Helper()
	if result.Code != 0 || result.Err != nil || len(result.Stderr) != 0 {
		t.Fatalf("%s: code=%d err=%v\nstdout: %s\nstderr: %s", operation, result.Code, result.Err, result.Stdout, result.Stderr)
	}
	return result
}

func requireStructuredFailure(t *testing.T, result commandResult, operation string, code int, defectCode, unit string) map[string]any {
	t.Helper()
	if result.Code != code || result.Err == nil || len(result.Stdout) != 0 {
		t.Fatalf("%s: code=%d err=%v stdout=%q, want code=%d and empty stdout\nstderr: %s", operation, result.Code, result.Err, result.Stdout, code, result.Stderr)
	}
	var body struct {
		Op      string `json:"op"`
		Defects []struct {
			Code string `json:"code"`
			Unit string `json:"unit"`
		} `json:"defects"`
	}
	if err := json.Unmarshal(result.Stderr, &body); err != nil {
		t.Fatalf("%s stderr is not structured JSON: %v\n%s", operation, err, result.Stderr)
	}
	found := false
	for _, defect := range body.Defects {
		if defect.Code == defectCode && (unit == "" || defect.Unit == unit) {
			found = true
		}
	}
	if !found {
		t.Fatalf("%s defects = %#v, want code=%s unit=%q", operation, body.Defects, defectCode, unit)
	}
	var raw map[string]any
	_ = json.Unmarshal(result.Stderr, &raw)
	return raw
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func replaceEnv(base []string, values map[string]string) []string {
	out := make([]string, 0, len(base)+len(values))
	for _, entry := range base {
		name, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replace := values[name]; replace {
				continue
			}
		}
		out = append(out, entry)
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

func joinPATH(paths ...string) string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func lookPathIn(name, pathValue string) (string, error) {
	for _, dir := range filepath.SplitList(pathValue) {
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", exec.ErrNotFound
}

func writeFile(t *testing.T, path string, mode os.FileMode, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func copyFile(t *testing.T, source, destination string, mode os.FileMode) {
	t.Helper()
	in, err := os.Open(source)
	if err != nil {
		t.Fatalf("open %s: %v", source, err)
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		t.Fatalf("create %s: %v", destination, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		t.Fatalf("copy %s: %v", destination, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", destination, err)
	}
}

func requireExecutable(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("executable %s: info=%v err=%v", path, info, err)
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func workspaceInventory(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("inventory %s: %v", root, err)
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, rel := range paths {
		full := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(full)
		if err != nil {
			t.Fatalf("inventory lstat %s: %v", full, err)
		}
		_, _ = io.WriteString(h, rel+"\x00"+info.Mode().String()+"\x00")
		if info.Mode().IsRegular() {
			f, err := os.Open(full)
			if err != nil {
				t.Fatalf("inventory open %s: %v", full, err)
			}
			_, copyErr := io.Copy(h, f)
			closeErr := f.Close()
			if copyErr != nil || closeErr != nil {
				t.Fatalf("inventory read %s: copy=%v close=%v", full, copyErr, closeErr)
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(full)
			if err != nil {
				t.Fatalf("inventory readlink %s: %v", full, err)
			}
			_, _ = io.WriteString(h, target)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
