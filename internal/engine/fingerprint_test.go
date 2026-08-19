package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func TestInspectRemainingDoesNotReadFileBytes(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	inPath := filepath.Join(dir, "in", "sample.txt")
	outPath := filepath.Join(dir, "out", "sample.txt")
	if err := os.Chmod(inPath, 0); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(outPath, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(inPath, 0o644)
		os.Chmod(outPath, 0o644)
	})
	if _, err := os.ReadFile(inPath); err == nil {
		t.Fatal("input is readable, chmod 000 did not block reads")
	}
	raw, defects := Inspect(dir, viewRemaining, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining) defects %v", defects)
	}
	if len(bytes.TrimSpace(raw)) != 0 {
		t.Fatalf("remaining got %s, want empty (cheap keys must not read bytes)", raw)
	}
}

func TestCopyFallbackDestCheapKeyIsDestInode(t *testing.T) {
	orig := exec.LinkFn
	t.Cleanup(func() { exec.LinkFn = orig })
	exec.LinkFn = func(string, string) error { return syscall.EXDEV }

	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	isolateOut := filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "work", "out", "sample.txt")
	dest := filepath.Join(dir, "out", "sample.txt")
	isoKey, err := cheapKey(isolateOut)
	if err != nil {
		t.Fatal(err)
	}
	destKey, err := cheapKey(dest)
	if err != nil {
		t.Fatal(err)
	}
	if destKey.Inode == isoKey.Inode && destKey.Dev == isoKey.Dev {
		t.Fatal("copy fallback dest inode equals isolate inode")
	}
	st := taskStates(t, dir)["copy"]
	if len(st.Checksums) != 1 {
		t.Fatalf("checksums got %#v", st.Checksums)
	}
	if st.Checksums[0].Inode != destKey.Inode || st.Checksums[0].Dev != destKey.Dev {
		t.Fatalf("recorded dest inode %d/%d, want dest %d/%d not isolate %d/%d",
			st.Checksums[0].Dev, st.Checksums[0].Inode, destKey.Dev, destKey.Inode, isoKey.Dev, isoKey.Inode)
	}
}

func TestClassifyReuseCheapKeysAndImageDigest(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "out", "a.txt"), "reads")
	inRec := mustFileRecord(t, filepath.Join(dir, "in", "a.txt"), "in/a.txt")
	outRec := mustFileRecord(t, filepath.Join(dir, "out", "a.txt"), "out/a.txt")
	base := jsonTaskState{
		ID:           "copy",
		Status:       StatusSucceeded,
		Command:      []string{"cp", "in/a.txt", "out/a.txt"},
		Image:        "alpine:3.19.1",
		ImageDigest:  "sha256:deadbeef",
		Params:       []jsonParam{{Name: "mode", Value: "fast"}},
		Env:          map[string]string{"HOME": "/tmp"},
		Fingerprints: []jsonFileHash{inRec},
		Checksums:    []jsonFileHash{outRec},
		Lineage:      []jsonLineage{{Producer: "copy", Path: "out/a.txt", Checksum: outRec.SHA256}},
	}
	plan := TaskPlan{
		ID:      "copy",
		Command: []string{"cp", "in/a.txt", "out/a.txt"},
		Image:   "alpine:3.19.1",
		Params:  []ParamPlan{{Name: "mode", Value: "fast"}},
		Env:     map[string]string{"HOME": "/tmp"},
		Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
		Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
	}
	got := classifyReuse(dir, base, plan, plan)
	if got.Decision != reuseReused || got.Reason != reasonReusedIdentityMatched {
		t.Fatalf("digest-only image change must reuse, got %#v", got)
	}

	if err := os.Chtimes(filepath.Join(dir, "in", "a.txt"), time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got = classifyReuse(dir, base, plan, plan)
	if got.Reason != reasonInputFingerprintChanged {
		t.Fatalf("mtime change got %#v, want input-fingerprint-changed", got)
	}

	if err := os.Remove(filepath.Join(dir, "in", "a.txt")); err != nil {
		t.Fatal(err)
	}
	got = classifyReuse(dir, base, plan, plan)
	if got.Reason != reasonInputMissing {
		t.Fatalf("missing input got %#v, want input-missing", got)
	}
}

func TestPublishTreeManifestDestCheapKeys(t *testing.T) {
	dir := t.TempDir()
	isolate := filepath.Join(dir, "iso")
	writeCheckFile(t, filepath.Join(isolate, "idx", "SA"), "x")
	writeCheckFile(t, filepath.Join(isolate, "idx", "sub", "chrName.txt"), "y")
	out := IO{Name: "idx", Kind: ArtifactTree, Path: "idx", Manifest: "idx/" + treeManifestName}
	wrote, err := publishTree(dir, isolate, out, false)
	if err != nil {
		t.Fatalf("publishTree() error = %v", err)
	}
	if len(wrote) == 0 {
		t.Fatal("publishTree wrote nothing")
	}
	manPath := filepath.Join(dir, "idx", treeManifestName)
	raw, err := os.ReadFile(manPath)
	if err != nil {
		t.Fatal(err)
	}
	var body treeManifest
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("manifest JSON: %v", err)
	}
	if len(body.Members) != 2 {
		t.Fatalf("members got %#v", body.Members)
	}
	for _, m := range body.Members {
		if m.Path == treeManifestName {
			t.Fatal(".gobble-tree.json listed as member")
		}
		abs := filepath.Join(dir, "idx", filepath.FromSlash(m.Path))
		want, err := cheapKey(abs)
		if err != nil {
			t.Fatal(err)
		}
		if m.Inode != want.Inode || m.Dev != want.Dev || m.Size != want.Size || m.Mtime != want.Mtime {
			t.Fatalf("member %s cheap keys got %+v, want dest %+v", m.Path, m, want)
		}
		if m.SHA256 == "" {
			t.Fatalf("member %s missing content digest", m.Path)
		}
		info, err := os.Lstat(abs)
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("tree dest %s is not regular", m.Path)
		}
	}
	digests := []string{body.Members[0].SHA256, body.Members[1].SHA256}
	if body.Digest != treeManifestDigest(digests) {
		t.Fatalf("digest got %s, want %s", body.Digest, treeManifestDigest(digests))
	}
	isoSA, err := cheapKey(filepath.Join(isolate, "idx", "SA"))
	if err != nil {
		t.Fatal(err)
	}
	destSA, err := cheapKey(filepath.Join(dir, "idx", "SA"))
	if err != nil {
		t.Fatal(err)
	}
	if destSA.Inode == isoSA.Inode && destSA.Dev == isoSA.Dev {
		if body.Members[0].Path == "SA" && body.Members[0].Inode != destSA.Inode {
			t.Fatalf("hardlink dest key reused isolate stats incorrectly")
		}
	}
}

func TestPrepareIsolateDockerSkipsSymlink(t *testing.T) {
	orig := exec.LinkFn
	t.Cleanup(func() { exec.LinkFn = orig })
	exec.LinkFn = func(string, string) error { return syscall.EXDEV }

	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	isolate := filepath.Join(dir, "iso")
	task := TaskPlan{
		ID:     "copy",
		Image:  "alpine:3.19.1",
		Inputs: []IO{{Name: "in", Path: "in/sample.txt"}},
	}
	if err := prepareIsolate(dir, isolate, task); err != nil {
		t.Fatalf("prepareIsolate() error = %v", err)
	}
	dst := filepath.Join(isolate, "in", "sample.txt")
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatal("docker stage used symlink")
	}
	src := filepath.Join(dir, "in", "sample.txt")
	srcInfo, err := os.Lstat(src)
	if err != nil {
		t.Fatal(err)
	}
	if srcInfo.Mode().Perm() != 0o644 {
		t.Fatalf("source mode got %o, want 0644", srcInfo.Mode().Perm())
	}
}

func TestHardlinkStageDoesNotChmodSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in", "sample.txt")
	writeCheckFile(t, src, "reads")
	isolate := filepath.Join(dir, "iso")
	task := TaskPlan{
		ID:     "copy",
		Inputs: []IO{{Name: "in", Path: "in/sample.txt"}},
	}
	if err := prepareIsolate(dir, isolate, task); err != nil {
		t.Fatalf("prepareIsolate() error = %v", err)
	}
	dst := filepath.Join(isolate, "in", "sample.txt")
	srcKey, err := cheapKey(src)
	if err != nil {
		t.Fatal(err)
	}
	dstKey, err := cheapKey(dst)
	if err != nil {
		t.Fatal(err)
	}
	if srcKey.Inode != dstKey.Inode || srcKey.Dev != dstKey.Dev {
		t.Fatal("same-device stage did not hardlink")
	}
	info, err := os.Lstat(src)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("hardlink stage chmod source to %o", info.Mode().Perm())
	}
}

func mustFileRecord(t *testing.T, abs, plan string) jsonFileHash {
	t.Helper()
	rec, err := fileRecord(abs, plan)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}
