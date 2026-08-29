package engine

import (
	"bytes"
	"context"
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
		Identity:  testInstallIdentity(),
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
	raw, defects := Inspect(dir, viewRemaining, "", testInstallIdentity())
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining, testInstallIdentity()) defects %v", defects)
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
		Identity:  testInstallIdentity(),
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
	stubImageID(t, "sha256:deadbeef")
	base := jsonTaskState{
		ID:           "copy",
		Status:       StatusSucceeded,
		Command:      []string{"cp", "in/a.txt", "out/a.txt"},
		Image:        "alpine:3.19.1",
		ImageDigest:  "sha256:deadbeef",
		Params:       []jsonParam{{Name: "mode", Value: "fast"}},
		EnvDigest:    envDigest(map[string]string{"HOME": "/tmp"}),
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
		t.Fatalf("matching digest must reuse, got %#v", got)
	}

	if err := os.Chtimes(filepath.Join(dir, "in", "a.txt"), time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got = classifyReuse(dir, base, plan, plan)
	if got.Decision != reuseReused {
		t.Fatalf("mtime-only change got %#v, want reuse", got)
	}

	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "READS")
	if err := os.Chtimes(filepath.Join(dir, "in", "a.txt"), time.Unix(inRec.Mtime/1e9, inRec.Mtime%1e9), time.Unix(inRec.Mtime/1e9, inRec.Mtime%1e9)); err != nil {
		t.Fatal(err)
	}
	got = classifyReuse(dir, base, plan, plan)
	if got.Reason != reasonInputFingerprintChanged {
		t.Fatalf("content mismatch got %#v, want input-fingerprint-changed", got)
	}

	stubImageID(t, "sha256:other")
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "reads")
	got = classifyReuse(dir, base, plan, plan)
	if got.Reason != reasonImageChanged {
		t.Fatalf("digest mismatch got %#v, want image-changed", got)
	}

	stubImageID(t, "")
	got = classifyReuse(dir, base, plan, plan)
	if got.Reason != reasonImageChanged {
		t.Fatalf("missing digest got %#v, want image-changed", got)
	}

	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "reads")
	stubImageID(t, "sha256:deadbeef")
	if err := os.Remove(filepath.Join(dir, "in", "a.txt")); err != nil {
		t.Fatal(err)
	}
	got = classifyReuse(dir, base, plan, plan)
	if got.Reason != reasonInputMissing {
		t.Fatalf("missing input got %#v, want input-missing", got)
	}
}

func TestClassifyReuseEmptyExecIdentityMiss(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "out", "a.txt"), "reads")
	inRec := mustFileRecord(t, filepath.Join(dir, "in", "a.txt"), "in/a.txt")
	outRec := mustFileRecord(t, filepath.Join(dir, "out", "a.txt"), "out/a.txt")
	base := jsonTaskState{
		ID:           "copy",
		Status:       StatusSucceeded,
		Command:      []string{"true"},
		Fingerprints: []jsonFileHash{inRec},
		Checksums:    []jsonFileHash{outRec},
	}
	plan := TaskPlan{
		ID:      "copy",
		Command: []string{"true"},
		Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
		Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
	}
	got := classifyReuse(dir, base, plan, plan)
	if got.Decision != reuseRerun || got.Reason != reasonImageChanged {
		t.Fatalf("empty exec identity got %#v, want image-changed miss", got)
	}
}

func stubImageID(t *testing.T, id string) {
	t.Helper()
	orig := lookupImageID
	t.Cleanup(func() { lookupImageID = orig })
	lookupImageID = func(string) string { return id }
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
	src := filepath.Join(dir, "in", "sample.txt")
	writeCheckFile(t, src, "reads")
	wantPerm := filePerm(t, src)
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
	srcKey, err := cheapKey(src)
	if err != nil {
		t.Fatal(err)
	}
	dstKey, err := cheapKey(dst)
	if err != nil {
		t.Fatal(err)
	}
	if srcKey.Inode == dstKey.Inode && srcKey.Dev == dstKey.Dev {
		t.Fatal("docker stage dest inode equals source")
	}
	if got := filePerm(t, src); got != wantPerm {
		t.Fatalf("source mode got %o, want %o", got, wantPerm)
	}
}

func TestStageCopyIndependentFromSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in", "sample.txt")
	writeCheckFile(t, src, "reads")
	wantPerm := filePerm(t, src)
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
	if srcKey.Inode == dstKey.Inode && srcKey.Dev == dstKey.Dev {
		t.Fatal("staged dest inode equals source")
	}
	if got := filePerm(t, src); got != wantPerm {
		t.Fatalf("stage chmod source from %o to %o", wantPerm, got)
	}
	if err := os.Chmod(dst, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(src)
	if err != nil || string(got) != "reads" {
		t.Fatalf("source bytes got %q, want reads", got)
	}
}

func TestRunFingerprintsStagedBytesBeforeSubmit(t *testing.T) {
	enteredSubmit := make(chan struct{})
	continueSubmit := make(chan struct{})
	var writeErr error
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			close(enteredSubmit)
			<-continueSubmit
			if err := os.MkdirAll(filepath.Join(job.Isolate, "out"), 0o755); err != nil {
				writeErr = err
			} else {
				writeErr = os.WriteFile(filepath.Join(job.Isolate, "out", "sample.txt"), []byte("result"), 0o644)
			}
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "1"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "1", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
		},
	})
	dir := t.TempDir()
	source := filepath.Join(dir, "in", "sample.txt")
	writeCheckFile(t, source, "staged")
	done := make(chan []Defect, 1)
	go func() {
		done <- Run(t.Context(), Request{
			Identity:  testInstallIdentity(),
			Workspace: dir,
			Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
		})
	}()
	select {
	case <-enteredSubmit:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for staged Submit")
	}
	if err := os.WriteFile(source, []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	close(continueSubmit)
	if defects := <-done; len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	if writeErr != nil {
		t.Fatalf("write isolate output: %v", writeErr)
	}
	state := taskStates(t, dir)["copy"]
	if len(state.Fingerprints) != 1 {
		t.Fatalf("fingerprints got %#v, want one", state.Fingerprints)
	}
	stagedPath := filepath.Join(dir, ControlDir, "tasks", "copy", emptyInstanceSeg, "0", "1", "work", "in", "sample.txt")
	stagedSHA, err := sha256File(stagedPath)
	if err != nil {
		t.Fatal(err)
	}
	mutatedSHA, err := sha256File(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := state.Fingerprints[0].SHA256; got != stagedSHA || got == mutatedSHA {
		t.Fatalf("fingerprint got %q, staged %q mutated %q", got, stagedSHA, mutatedSHA)
	}
}

func TestStatusSucceededEmptyChecksumsIsReuseMiss(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	if defects := Run(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  doc,
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	state := taskStates(t, dir)["copy"]
	state.Checksums = nil
	decision := classifyReuse(dir, state, doc.Tasks[0], doc.Tasks[0])
	if decision.Decision != reuseRerun || decision.Reason != reasonOutputMissing {
		t.Fatalf("empty checksums decision got %#v, want rerun output-missing", decision)
	}
	patchAttempt(t, dir, func(st *jsonTaskState) {
		st.Checksums = nil
		st.Lineage = nil
	})
	forceDeadOwner(t, dir)
	if defects := Release(dir, testInstallIdentity()); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	if defects := Resume(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  doc,
	}); len(defects) != 0 {
		t.Fatalf("Resume() empty-checksum defects %v, want rerun", defects)
	}
	after := taskStates(t, dir)["copy"]
	if after.Status != StatusSucceeded || after.Attempt != state.Attempt+1 {
		t.Fatalf("empty-checksum Resume state got status=%q attempt=%d, want succeeded attempt %d", after.Status, after.Attempt, state.Attempt+1)
	}
}

func mustAttachExec(t *testing.T, st *jsonTaskState, argv0 string) {
	t.Helper()
	path, err := exec.ResolveArgv0(argv0, nil)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	st.ExecutablePath = path
	st.ExecutableSHA256 = sum
}

func mustExecState(t *testing.T, st jsonTaskState, argv0 string) jsonTaskState {
	t.Helper()
	mustAttachExec(t, &st, argv0)
	return st
}

func mustFileRecord(t *testing.T, abs, plan string) jsonFileHash {
	t.Helper()
	rec, err := fileRecord(abs, plan)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func filePerm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
