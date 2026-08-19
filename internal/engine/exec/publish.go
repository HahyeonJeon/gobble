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

// OpenReadFile opens a regular file without following symlinks.
func OpenReadFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrNotRegular
	}
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
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

// StagedReplace copies src over dst through a same-directory temp file.
func StagedReplace(src, dst string) error {
	in, err := OpenReadFile(src)
	if err != nil {
		return err
	}
	defer in.Close()
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".gobble-replace-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := io.Copy(f, in)
	cerr := f.Close()
	if werr != nil {
		os.Remove(tmp)
		return werr
	}
	if cerr != nil {
		os.Remove(tmp)
		return cerr
	}
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(dst); err == nil && info.Mode().IsRegular() {
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(tmp, mode); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
