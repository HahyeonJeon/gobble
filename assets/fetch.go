package assets

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// FetchPin returns p.CachePath(). When Check fails it downloads p.URL into
// that path (tmp+rename), then Check.
func FetchPin(p Pin) (string, error) {
	dest := p.CachePath()
	if err := p.Check(dest); err == nil {
		return dest, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("pin %s: mkdir: %w", p.Name, err)
	}
	if err := downloadURL(p.URL, dest); err != nil {
		return "", fmt.Errorf("pin %s: download %s: %w", p.Name, p.URL, err)
	}
	if err := p.Check(dest); err != nil {
		return "", err
	}
	return dest, nil
}

func downloadURL(rawURL, dest string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return os.ErrNotExist
	}
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dest)
}
