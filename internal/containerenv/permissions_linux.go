package containerenv

import (
	"fmt"
	"os"
	"syscall"
)

// Match a Linux bind mount's owner before creating files. Desktop normally
// exposes a root-owned mount and translates writes to the host user itself.
func useProjectOwner(project, socket string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	info, err := os.Stat(project)
	if err != nil {
		return err
	}
	owner := info.Sys().(*syscall.Stat_t)
	if owner.Uid == 0 {
		return nil
	}
	socketInfo, err := os.Stat(socket)
	if err != nil {
		return err
	}
	group := socketInfo.Sys().(*syscall.Stat_t).Gid
	if err := syscall.Setgroups([]int{int(owner.Gid), int(group)}); err != nil {
		return fmt.Errorf("set project/socket groups: %w", err)
	}
	if err := syscall.Setgid(int(owner.Gid)); err != nil {
		return err
	}
	return syscall.Setuid(int(owner.Uid))
}
