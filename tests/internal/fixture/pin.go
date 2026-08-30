// Package fixture provides byte verification and live fetch mechanics without
// owning any module or assay fixture fact.
package fixture

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Pin identifies one remote fixture byte.
type Pin struct {
	Name   string
	URL    string
	Bytes  int64
	SHA256 string
}

// CachePath returns cacheDir/<sha256[:16]>/<name>.
func (p Pin) CachePath(cacheDir string) string {
	if len(p.SHA256) < 16 {
		panic(fmt.Sprintf("pin %s: sha256 too short", p.Name))
	}
	return filepath.Join(cacheDir, p.SHA256[:16], p.Name)
}

// Check verifies that path is a regular file with the declared byte identity.
func (p Pin) Check(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("pin %s: %w", p.Name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("pin %s: %s is not a regular file", p.Name, path)
	}
	if info.Size() != p.Bytes {
		return fmt.Errorf("pin %s: size %d, want %d", p.Name, info.Size(), p.Bytes)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("pin %s: %w", p.Name, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("pin %s: hash %s: %w", p.Name, path, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != p.SHA256 {
		return fmt.Errorf("pin %s: sha256 %s, want %s", p.Name, got, p.SHA256)
	}
	return nil
}

// Fetch returns the verified cache path for p, downloading it when absent.
func Fetch(cacheDir string, p Pin) (string, error) {
	dest := p.CachePath(cacheDir)
	if err := p.Check(dest); err == nil {
		return dest, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("pin %s: mkdir: %w", p.Name, err)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	response, err := client.Get(p.URL)
	if err != nil {
		return "", fmt.Errorf("pin %s: download %s: %w", p.Name, p.URL, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pin %s: download %s: %w", p.Name, p.URL, os.ErrNotExist)
	}
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(f, response.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return "", closeErr
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	if err := p.Check(dest); err != nil {
		return "", err
	}
	return dest, nil
}
