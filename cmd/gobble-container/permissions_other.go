//go:build !linux && !windows

package main

func localPermissions(socket string) []string { return nil }
