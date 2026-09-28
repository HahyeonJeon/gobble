package appservice

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchCopyExactBytesAndFASTQValidation(t *testing.T) {
	good := []byte("@read\r\nACGTN\r\n+\r\nIIIII\r\n")
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, _ = gz.Write(good)
	_ = gz.Close()
	for _, tc := range []struct {
		name        string
		data        []byte
		gzip, valid bool
	}{
		{"plain", good, false, true}, {"gzip", compressed.Bytes(), true, true},
		{"truncated gzip", compressed.Bytes()[:compressed.Len()-4], true, false},
		{"empty", nil, false, false}, {"bad header", []byte("read\nAC\n+\nII\n"), false, false},
		{"quality mismatch", []byte("@r\nAC\n+\nI\n"), false, false},
		{"incomplete", []byte("@r\nAC\n+\n"), false, false},
		{"bad bases", []byte("@r\nAX\n+\nII\n"), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, e := os.Create(filepath.Join(t.TempDir(), "copy"))
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			hash, n, reads, e := copyFASTQ(context.Background(), bytes.NewReader(tc.data), f, tc.gzip, func(int64) {})
			if (e == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, e)
			}
			if tc.valid {
				saved, _ := os.ReadFile(f.Name())
				if !bytes.Equal(saved, tc.data) || hash != digest(tc.data) || n != int64(len(tc.data)) || reads != 1 {
					t.Fatal("copied bytes or identity changed")
				}
			}
		})
	}
}
func TestLaunchCopyCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f, e := os.Create(filepath.Join(t.TempDir(), "copy"))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if _, _, _, e = copyFASTQ(ctx, bytes.NewBufferString("@r\nAC\n+\nII\n"), f, false, func(int64) {}); e == nil {
		t.Fatal("cancelled copy accepted")
	}
}
