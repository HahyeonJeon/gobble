//go:build darwin || linux

package appservice

import "syscall"

func launchDiskSpace(path string, size int64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return err
	}
	if uint64(stat.Bavail)*uint64(stat.Bsize) < uint64(size)+(64<<20) {
		return problem("runtime_unavailable", "There is not enough available space for the checked data copy. Free space and check again.")
	}
	return nil
}
