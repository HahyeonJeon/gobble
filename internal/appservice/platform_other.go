//go:build !darwin && !linux

package appservice

import "os"

func lockCatalog(*os.File) error {
	return problem("unsupported", "The Project service currently supports macOS and Linux.")
}
func rootIdentity(os.FileInfo) string { return "" }
func openResource(*os.Root, string) (*os.File, error) {
	return nil, problem("unsupported", "This host is not supported.")
}

func openReportSource(*os.Root, string) (*os.File, error) {
	return nil, problem("unsupported", "This host is not supported.")
}
