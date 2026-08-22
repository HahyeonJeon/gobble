package exec

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCopyFileRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "secret")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "link")
	if err := os.Symlink(target, src); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out.txt")
	if err := CopyFile(src, dst); err == nil {
		t.Fatal("CopyFile followed symlink")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("copied through symlink")
	}
}

func TestStageHardlinkDoesNotChmodSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantPerm := filePerm(t, src)
	dst := filepath.Join(dir, "iso", "src.txt")
	if err := StageFile(src, dst, true); err != nil {
		t.Fatalf("StageFile() error = %v", err)
	}
	if !sameInode(src, dst) {
		t.Fatal("same-device stage did not hardlink")
	}
	if got := filePerm(t, src); got != wantPerm {
		t.Fatalf("source mode got %o, want %o", got, wantPerm)
	}
	dstInfo, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !dstInfo.Mode().IsRegular() {
		t.Fatal("hardlink dest is not regular")
	}
}

func TestStageCopyFallbackDestInodeAndChmod(t *testing.T) {
	orig := LinkFn
	t.Cleanup(func() { LinkFn = orig })
	LinkFn = func(string, string) error { return syscall.EXDEV }

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantPerm := filePerm(t, src)
	dst := filepath.Join(dir, "iso", "src.txt")
	if err := StageFile(src, dst, false); err != nil {
		t.Fatalf("StageFile() error = %v", err)
	}
	if sameInode(src, dst) {
		t.Fatal("copy fallback dest inode equals source")
	}
	if got := filePerm(t, src); got != wantPerm {
		t.Fatalf("source mode after copy got %o, want %o", got, wantPerm)
	}
	dstInfo, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !dstInfo.Mode().IsRegular() {
		t.Fatal("copy dest is not regular")
	}
	if dstInfo.Mode().Perm() != 0o444 {
		t.Fatalf("copy dest mode got %o, want 0444", dstInfo.Mode().Perm())
	}
}

func TestStageProcessSymlinkFallback(t *testing.T) {
	orig := LinkFn
	t.Cleanup(func() { LinkFn = orig })
	LinkFn = func(string, string) error { return syscall.EXDEV }

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "iso", "src.txt")
	if err := StageFile(src, dst, true); err != nil {
		t.Fatalf("StageFile() error = %v", err)
	}
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("process stage fallback is not a symlink")
	}
}

func TestPublishCopyFallbackDestInodeDiffers(t *testing.T) {
	orig := LinkFn
	t.Cleanup(func() { LinkFn = orig })
	LinkFn = func(string, string) error { return syscall.EXDEV }

	dir := t.TempDir()
	src := filepath.Join(dir, "iso", "out.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("out"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out.txt")
	if err := PublishFile(src, dst); err != nil {
		t.Fatalf("PublishFile() error = %v", err)
	}
	if sameInode(src, dst) {
		t.Fatal("copy fallback dest inode equals isolate inode")
	}
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatal("published dest is not regular")
	}
}

func TestPublishNeverSymlink(t *testing.T) {
	origLink, origSym := LinkFn, SymlinkFn
	t.Cleanup(func() {
		LinkFn = origLink
		SymlinkFn = origSym
	})
	LinkFn = func(string, string) error { return syscall.EXDEV }
	SymlinkFn = func(string, string) error {
		t.Fatal("PublishFile called symlink")
		return nil
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("out"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst.txt")
	if err := PublishFile(src, dst); err != nil {
		t.Fatalf("PublishFile() error = %v", err)
	}
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatal("published dest is not regular")
	}
}

func TestCopyFileDestInodeDiffersFromSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst.txt")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}
	if sameInode(src, dst) {
		t.Fatal("CopyFile dest inode equals source")
	}
}

func TestStagedReplaceHardlinkDoesNotChmodSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out", "sample.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("prior"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "work", "sample.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("next"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantPerm := filePerm(t, src)
	if err := StagedReplace(src, dst); err != nil {
		t.Fatalf("StagedReplace() error = %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "next" {
		t.Fatalf("dest got %q, want next", got)
	}
	if !sameInode(src, dst) {
		t.Fatal("StagedReplace did not hardlink")
	}
	if srcPerm := filePerm(t, src); srcPerm != wantPerm {
		t.Fatalf("source mode got %o, want %o", srcPerm, wantPerm)
	}
}

func filePerm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
