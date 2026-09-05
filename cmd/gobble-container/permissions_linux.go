package main

import (
	"os"
	"strconv"
	"syscall"
)

func localPermissions(socket string) []string {
	args := []string{"--user", strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())}
	if info, err := os.Stat(socket); err == nil {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			args = append(args, "--group-add", strconv.FormatUint(uint64(stat.Gid), 10))
		}
	}
	return args
}
