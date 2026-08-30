package local_e2e_test

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

const (
	wgsE2EPairCount   = 266736
	wgsE2EStagedR1    = "in/test_1.fastq.gz"
	wgsE2EStagedR2    = "in/test_2.fastq.gz"
	wgsE2EStagedFASTA = "in/genome.fasta"
)

// FAI is cached for hash check only. Do not stage in/genome.fasta.fai.
var wgsE2EStaged = []string{
	wgsE2EStagedR1,
	wgsE2EStagedR2,
	wgsE2EStagedFASTA,
}

type wgsE2EFile struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func TestWGSFixtureManifestPins(t *testing.T) {
	files := wgsE2EManifestFiles(t)
	if len(files) != 4 {
		t.Fatalf("manifest files got %d, want 4", len(files))
	}
	for i, name := range []string{"test_1.fastq.gz", "test_2.fastq.gz", "genome.fasta", "genome.fasta.fai"} {
		if files[i].Name != name {
			t.Fatalf("manifest[%d] name = %q, want %q", i, files[i].Name, name)
		}
	}
}

func fillWGSFixtureCache(t *testing.T) {
	t.Helper()
	cacheDir := wgsE2ECacheDir(t)
	for _, file := range wgsE2EManifestFiles(t) {
		if _, err := fixture.Fetch(cacheDir, file.pin()); err != nil {
			t.Fatalf("download %s: %v", file.URL, err)
		}
	}
	verifyWGSFixtureCache(t)
}

func verifyWGSFixtureCache(t *testing.T) {
	t.Helper()
	cacheDir := wgsE2ECacheDir(t)
	for _, file := range wgsE2EManifestFiles(t) {
		path := file.pin().CachePath(cacheDir)
		if err := file.pin().Check(path); err != nil {
			t.Fatalf("cache %s: %v", path, err)
		}
	}
}

func stageWGSFixture(t *testing.T, workspace string) {
	t.Helper()
	fillWGSFixtureCache(t)
	cacheDir := wgsE2ECacheDir(t)
	byName := wgsE2EFileByName(t)
	for _, rel := range wgsE2EStaged {
		name := filepath.Base(rel)
		file, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing staged file %s", name)
		}
		src := file.pin().CachePath(cacheDir)
		dst := filepath.Join(workspace, rel)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(dst), err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", dst, err)
		}
		sum, size := wgsE2EHashFile(t, dst)
		if size != file.Bytes || sum != file.SHA256 {
			t.Fatalf("staged %s size %d sha256 %s, want %d %s", dst, size, sum, file.Bytes, file.SHA256)
		}
	}
}

func (file wgsE2EFile) pin() fixture.Pin {
	return fixture.Pin{Name: file.Name, URL: file.URL, Bytes: file.Bytes, SHA256: file.SHA256}
}

func wgsE2ECacheDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(moduleRoot(t), filepath.FromSlash(wgsevidence.CacheDir))
}

func countFASTQRecords(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", path, err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader(%s) error = %v", path, err)
	}
	defer zr.Close()
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	if n%4 != 0 {
		t.Fatalf("FASTQ %s line count %d is not a multiple of 4", path, n)
	}
	return n / 4
}

func wgsE2EManifestFiles(t *testing.T) []wgsE2EFile {
	t.Helper()
	pins, err := wgsevidence.Pins()
	if err != nil {
		t.Fatal(err)
	}
	files := make([]wgsE2EFile, len(pins))
	for i, pin := range pins {
		files[i] = wgsE2EFile{Name: pin.Name, URL: pin.URL, Bytes: pin.Bytes, SHA256: pin.SHA256}
	}
	return files
}

func wgsE2EFileByName(t *testing.T) map[string]wgsE2EFile {
	t.Helper()
	out := make(map[string]wgsE2EFile)
	for _, file := range wgsE2EManifestFiles(t) {
		out[file.Name] = file
	}
	return out
}

func wgsE2EHashFile(t *testing.T, path string) (sum string, size int64) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", path, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), info.Size()
}
