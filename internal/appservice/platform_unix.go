//go:build darwin || linux

package appservice

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

func lockCatalog(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
func rootIdentity(info os.FileInfo) string {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
}

// O_NONBLOCK prevents an attacker-swapped FIFO from stalling a regular-file read.
func openResource(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}

// openReportSource pins each directory with O_NOFOLLOW, then verifies that its
// child Root names that exact directory. No link can substitute a different root.
func openReportSource(root *os.Root, name string) (*os.File, error) {
	if !validRelative(name) || name == "." {
		return nil, problem("outside_project", "Invalid report source path.")
	}
	parent, err := root.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	defer func() { parent.Close() }()
	parts := strings.Split(name, "/")
	for _, part := range parts[:len(parts)-1] {
		pinned, err := parent.OpenFile(part, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return nil, err
		}
		before, err := pinned.Stat()
		if err != nil {
			pinned.Close()
			return nil, err
		}
		child, err := parent.OpenRoot(part)
		if err != nil {
			pinned.Close()
			return nil, err
		}
		after, err := child.Stat(".")
		pinned.Close()
		if err != nil || !os.SameFile(before, after) {
			child.Close()
			return nil, problem("stale_revision", "The report directory changed while opening.")
		}
		parent.Close()
		parent = child
	}
	return parent.OpenFile(parts[len(parts)-1], os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
}
