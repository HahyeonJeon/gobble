//go:build !darwin && !linux

package appservice

func launchDiskSpace(string, int64) error {
	return problem("unsupported", "Run staging is not qualified on this operating system.")
}
