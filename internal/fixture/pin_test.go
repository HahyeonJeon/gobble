package fixture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinCheck(t *testing.T) {
	content := []byte("hello pin")
	sum := sha256.Sum256(content)
	pin := Pin{Name: "hello.txt", URL: "https://example.invalid/hello.txt", Bytes: int64(len(content)), SHA256: hex.EncodeToString(sum[:])}
	path := filepath.Join(t.TempDir(), pin.Name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pin.Check(path); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	wrongSize := pin
	wrongSize.Bytes = 1
	if err := wrongSize.Check(path); err == nil || !strings.Contains(err.Error(), "size") {
		t.Fatalf("size mismatch error = %v", err)
	}
	wrongHash := pin
	wrongHash.SHA256 = strings.Repeat("0", 64)
	if err := wrongHash.Check(path); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("hash mismatch error = %v", err)
	}
}

func TestPinCachePathSeparatesSameNames(t *testing.T) {
	left := Pin{Name: "same", SHA256: strings.Repeat("a", 64)}
	right := Pin{Name: "same", SHA256: strings.Repeat("b", 64)}
	if left.CachePath("cache") == right.CachePath("cache") {
		t.Fatal("same-name pins share cache path")
	}
}

func TestPinCachePathRejectsShortSHA(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("CachePath() panic = nil")
		}
	}()
	_ = (Pin{Name: "x", SHA256: "abcd"}).CachePath("cache")
}

func TestFetchUsesVerifiedCache(t *testing.T) {
	content := []byte("cached pin")
	sum := sha256.Sum256(content)
	pin := Pin{Name: "cached.txt", URL: "https://example.invalid/cached.txt", Bytes: int64(len(content)), SHA256: hex.EncodeToString(sum[:])}
	cache := t.TempDir()
	destination := pin.CachePath(cache)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, content, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Fetch(cache, pin)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got != destination {
		t.Fatalf("Fetch() = %q, want %q", got, destination)
	}
}

func TestFetchPublishesOnlyVerifiedBytesAndCanRetry(t *testing.T) {
	content := []byte("verified fixture")
	sum := sha256.Sum256(content)
	payload := []byte("corrupt")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payload) }))
	defer server.Close()
	pin := Pin{Name: "input.txt", URL: server.URL, Bytes: int64(len(content)), SHA256: hex.EncodeToString(sum[:])}
	cache := t.TempDir()
	if _, err := Fetch(cache, pin); err == nil {
		t.Fatal("corruption accepted")
	}
	if _, err := os.Stat(pin.CachePath(cache)); !os.IsNotExist(err) {
		t.Fatal("corrupt download published")
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(pin.CachePath(cache)), ".download-*"))
	if len(files) != 0 {
		t.Fatal("temporary downloads leaked")
	}
	payload = content
	path, err := Fetch(cache, pin)
	if err != nil {
		t.Fatal(err)
	}
	if err := pin.Check(path); err != nil {
		t.Fatal(err)
	}
	server.Close()
	if _, err := Fetch(cache, pin); err != nil {
		t.Fatalf("verified offline reuse: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := FetchContext(ctx, cache, pin); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
