package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CacheDir is the uncommitted download cache relative to this package.
const CacheDir = "testdata/cache"

// Pin identifies one downloaded proof file.
type Pin struct {
	Name   string
	URL    string
	Bytes  int64
	SHA256 string
}

// WGS homo_sapiens pins copied from tests/wgs-e2e/testdata/manifest.json.
// These are copies, not an import of that package.
var (
	PinWGSTest1FASTQ = Pin{
		Name:   "test_1.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/illumina/fastq/test_1.fastq.gz",
		Bytes:  16013759,
		SHA256: "bae22d1ab233d9a746d7f6413dbc99f3c4a24b278eba97286940b3460f4dac97",
	}
	PinWGSTest2FASTQ = Pin{
		Name:   "test_2.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/illumina/fastq/test_2.fastq.gz",
		Bytes:  16698858,
		SHA256: "d89ff7b7009e29c31f7722646dfec7fe6dd1ac90c2a82e631e59685065c9dddf",
	}
	PinWGSGenomeFASTA = Pin{
		Name:   "genome.fasta",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/genome/genome.fasta",
		Bytes:  40675,
		SHA256: "48d0bbb875d37e529640d43f2751ec2a25e0ba1144f1994773e9c643d3cf9d05",
	}
	PinWGSGenomeFAI = Pin{
		Name:   "genome.fasta.fai",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/homo_sapiens/genome/genome.fasta.fai",
		Bytes:  20,
		SHA256: "aa684078e5bc8ec77af619fb6c3e6e9e51e143d997d84ab213b47eff844757d1",
	}
)

// WGSPins is the four WGS homo_sapiens pin records in manifest order.
var WGSPins = []Pin{
	PinWGSTest1FASTQ,
	PinWGSTest2FASTQ,
	PinWGSGenomeFASTA,
	PinWGSGenomeFAI,
}

// CachePath is CacheDir/<sha256[:16]>/<Name>. Name stays the workspace
// basename. It panics if SHA256 is shorter than 16 hex characters.
func (p Pin) CachePath() string {
	if len(p.SHA256) < 16 {
		panic(fmt.Sprintf("pin %s: sha256 too short", p.Name))
	}
	return filepath.Join(CacheDir, p.SHA256[:16], p.Name)
}

// Check reports whether path is a regular file whose size and sha256 match
// the pin. A mismatch must fail before execute.
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
	sum, err := hashFile(f)
	if err != nil {
		return fmt.Errorf("pin %s: hash %s: %w", p.Name, path, err)
	}
	if sum != p.SHA256 {
		return fmt.Errorf("pin %s: sha256 %s, want %s", p.Name, sum, p.SHA256)
	}
	return nil
}

func hashFile(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
