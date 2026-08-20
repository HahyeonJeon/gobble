package local_e2e_test

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	wgsE2EManifestPath = "testdata/manifest.json"
	wgsE2ECacheDir     = "testdata/cache"
	wgsE2EPairCount    = 266736
	wgsE2EStagedR1     = "in/test_1.fastq.gz"
	wgsE2EStagedR2     = "in/test_2.fastq.gz"
	wgsE2EStagedFASTA  = "in/genome.fasta"
)

// FAI is cached for hash check only. Do not stage in/genome.fasta.fai.
var wgsE2EStaged = []string{
	wgsE2EStagedR1,
	wgsE2EStagedR2,
	wgsE2EStagedFASTA,
}

var wantWGSFixturePins = []wgsE2EFile{
	{
		Name:   "test_1.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/illumina/fastq/test_1.fastq.gz",
		Bytes:  16013759,
		SHA256: "bae22d1ab233d9a746d7f6413dbc99f3c4a24b278eba97286940b3460f4dac97",
	},
	{
		Name:   "test_2.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/illumina/fastq/test_2.fastq.gz",
		Bytes:  16698858,
		SHA256: "d89ff7b7009e29c31f7722646dfec7fe6dd1ac90c2a82e631e59685065c9dddf",
	},
	{
		Name:   "genome.fasta",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/genome/genome.fasta",
		Bytes:  40675,
		SHA256: "48d0bbb875d37e529640d43f2751ec2a25e0ba1144f1994773e9c643d3cf9d05",
	},
	{
		Name:   "genome.fasta.fai",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/genome/genome.fasta.fai",
		Bytes:  20,
		SHA256: "aa684078e5bc8ec77af619fb6c3e6e9e51e143d997d84ab213b47eff844757d1",
	},
}

type wgsE2EFile struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func TestWGSFixtureManifestPins(t *testing.T) {
	files := wgsE2EManifestFiles(t)
	if len(files) != len(wantWGSFixturePins) {
		t.Fatalf("manifest files got %d, want %d", len(files), len(wantWGSFixturePins))
	}
	for i, got := range files {
		want := wantWGSFixturePins[i]
		if got != want {
			t.Fatalf("manifest[%d] got %+v, want %+v", i, got, want)
		}
	}
}

func fillWGSFixtureCache(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(wgsE2ECacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", wgsE2ECacheDir, err)
	}
	for _, file := range wgsE2EManifestFiles(t) {
		dest := filepath.Join(wgsE2ECacheDir, file.Name)
		if wgsE2ECacheMatch(dest, file) {
			continue
		}
		if err := downloadWGSFixture(file.URL, dest); err != nil {
			t.Fatalf("download %s: %v", file.URL, err)
		}
	}
	verifyWGSFixtureCache(t)
}

func verifyWGSFixtureCache(t *testing.T) {
	t.Helper()
	for _, file := range wgsE2EManifestFiles(t) {
		path := filepath.Join(wgsE2ECacheDir, file.Name)
		if !wgsE2ECacheMatch(path, file) {
			sum, size := wgsE2EHashFile(t, path)
			t.Fatalf("cache %s size %d sha256 %s, want %d %s", path, size, sum, file.Bytes, file.SHA256)
		}
	}
}

func stageWGSFixture(t *testing.T, workspace string) {
	t.Helper()
	fillWGSFixtureCache(t)
	byName := wgsE2EFileByName(t)
	for _, rel := range wgsE2EStaged {
		name := filepath.Base(rel)
		file, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing staged file %s", name)
		}
		src := filepath.Join(wgsE2ECacheDir, name)
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
	data, err := os.ReadFile(wgsE2EManifestPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", wgsE2EManifestPath, err)
	}
	var files []wgsE2EFile
	if err := json.Unmarshal(data, &files); err != nil {
		t.Fatalf("%s is not complete JSON: %v\n%s", wgsE2EManifestPath, err, data)
	}
	if len(files) == 0 {
		t.Fatalf("%s has no files", wgsE2EManifestPath)
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

func wgsE2ECacheMatch(path string, file wgsE2EFile) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != file.Bytes {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == file.SHA256
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

func downloadWGSFixture(rawURL, dest string) error {
	tmp := dest + ".partial"
	if err := os.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func wgsE2ENetworkUnavailable(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	var dns *net.DNSError
	return errors.As(err, &dns)
}
