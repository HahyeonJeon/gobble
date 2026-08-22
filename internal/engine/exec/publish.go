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

// LinkFn is os.Link. Tests may replace it.
var LinkFn = os.Link

// SymlinkFn is os.Symlink. Tests may replace it.
var SymlinkFn = os.Symlink

// OpenReadFile opens a regular file without following symlinks.
func OpenReadFile(path string) (*os.File, error) {
	if err := requireRegular(path); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}

// StageFile copies src into isolate dest as an independent regular file
// with mode 0444. It does not hardlink or symlink.
func StageFile(src, dst string) error {
	if err := requireRegular(src); err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := copyToTemp(src, dir)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		os.Remove(tmp)
		return os.ErrExist
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	if sameInode(src, dst) {
		os.Remove(dst)
		return errors.New("staged dest shares source inode")
	}
	return os.Chmod(dst, 0o444)
}

// PublishFile copies src to dest through a complete temporary, then
// installs exclusively. Dest must not already exist.
func PublishFile(src, dst string) error {
	if err := requireRegular(src); err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := copyToTemp(src, dir)
	if err != nil {
		return err
	}
	if err := InstallExclusive(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// CopyFile copies src to dst using exclusive create.
func CopyFile(src, dst string) error {
	return PublishFile(src, dst)
}

// StagedReplace publishes src over dst through a complete same-directory
// temporary, then renames onto dest.
func StagedReplace(src, dst string) error {
	if err := requireRegular(src); err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := copyToTemp(src, dir)
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// CopyToTemp copies src into a new exclusive regular file in dir.
func CopyToTemp(src, dir string) (string, error) {
	if err := requireRegular(src); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return copyToTemp(src, dir)
}

// InstallExclusive installs a complete temporary onto dst only when dst
// does not exist. tmp is removed after a successful install.
func InstallExclusive(tmp, dst string) error {
	if err := os.Link(tmp, dst); err != nil {
		return err
	}
	os.Remove(tmp)
	return nil
}

func copyToTemp(src, dir string) (string, error) {
	in, err := OpenReadFile(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.CreateTemp(dir, ".gobble-pub-*")
	if err != nil {
		return "", err
	}
	tmp := out.Name()
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return "", copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return "", closeErr
	}
	return tmp, nil
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
