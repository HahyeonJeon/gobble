//go:build !linux && !windows && !darwin

package main

func localPermissions(socket string) []string { return nil }
