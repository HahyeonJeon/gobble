package fixture

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// SplitIntervals materializes stable scatter members from a pinned BED file.
func SplitIntervals(workspace, source, destination string, count int) error {
	if !validRelativePath(source) || !validRelativePath(destination) || count < 1 {
		return fmt.Errorf("invalid interval staging request")
	}
	data, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(source)))
	if err != nil {
		return err
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != count {
		return fmt.Errorf("interval source has %d members, want %d", len(lines), count)
	}
	dir := filepath.Join(workspace, filepath.FromSlash(destination))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i, line := range lines {
		path := filepath.Join(dir, fmt.Sprintf("interval_%03d.bed", i+1))
		if err := os.WriteFile(path, append(append([]byte(nil), line...), '\n'), 0o644); err != nil {
			return err
		}
	}
	return nil
}
