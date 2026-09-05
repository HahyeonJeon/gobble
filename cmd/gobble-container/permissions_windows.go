package main

// Docker Desktop maps Linux container writes to the owning Windows user.
// This adapter is compiled here; actual Desktop permission behavior is a gate.
func localPermissions(socket string) []string { return nil }
