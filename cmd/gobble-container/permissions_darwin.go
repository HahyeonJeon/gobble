package main

// Docker Desktop mediates permissions for host bind mounts. The Linux socket
// belongs to Desktop's VM, so host UID/GID and socket groups do not apply.
func localPermissions(socket string) []string { return nil }
