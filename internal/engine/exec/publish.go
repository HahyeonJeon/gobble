package exec

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// ErrNotRegular means the path exists but is not a regular file.
var ErrNotRegular = errors.New("not a regular file")

// LinkFn is os.Link. Tests replace it to force copy fallback.
var LinkFn = os.Link

// SymlinkFn is os.Symlink. Tests replace it.
var SymlinkFn = os.Symlink

// OpenReadFile opens a regular file without following symlinks.
func OpenReadFile(path string) (*os.File, error) {
	if err := requireRegular(path); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}

// StageFile stages src into isolate dest: hardlink, then optional
// symlink, then copy and chmod 0444. It does not chmod a dest that
// shares an inode with src.
func StageFile(src, dst string, allowSymlink bool) error {
	if err := requireRegular(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := LinkFn(src, dst); err == nil {
		return nil
	}
	if allowSymlink {
		if err := SymlinkFn(src, dst); err == nil {
			return nil
		}
	}
	if err := CopyFile(src, dst); err != nil {
		return err
	}
	return chmodUnshared(src, dst, 0o444)
}

// PublishFile publishes src to dest: hardlink then copy. Never symlink.
func PublishFile(src, dst string) error {
	if err := requireRegular(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := LinkFn(src, dst); err == nil {
		return nil
	}
	return CopyFile(src, dst)
}

// CopyFile copies src to dst using exclusive create.
func CopyFile(src, dst string) error {
	in, err := OpenReadFile(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(dst)
		return closeErr
	}
	return nil
}

// StagedReplace publishes src over dst through a same-directory temp
// file: hardlink then copy. It does not chmod a dest that shares an
// inode with src.
func StagedReplace(src, dst string) error {
	if err := requireRegular(src); err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".gobble-replace-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	os.Remove(tmp)
	if err := LinkFn(src, tmp); err == nil {
		if err := os.Rename(tmp, dst); err != nil {
			os.Remove(tmp)
			return err
		}
		return nil
	}
	if err := copyOver(src, tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(dst); err == nil && info.Mode().IsRegular() {
		mode = info.Mode().Perm()
	}
	if err := chmodUnshared(src, tmp, mode); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func copyOver(src, dst string) error {
	in, err := OpenReadFile(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(dst)
		return closeErr
	}
	return nil
}

func requireRegular(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return ErrNotRegular
	}
	return nil
}

func chmodUnshared(src, dst string, mode os.FileMode) error {
	if sameInode(src, dst) {
		return nil
	}
	return os.Chmod(dst, mode)
}

func sameInode(a, b string) bool {
	ad, ai, err := inodeOf(a)
	if err != nil {
		return false
	}
	bd, bi, err := inodeOf(b)
	if err != nil {
		return false
	}
	return ad == bd && ai == bi
}

func inodeOf(path string) (dev, ino uint64, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, 0, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("stat_t unavailable")
	}
	return uint64(st.Dev), uint64(st.Ino), nil
}
