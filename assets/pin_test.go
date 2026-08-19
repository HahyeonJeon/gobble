package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWGSPins(t *testing.T) {
	want := []Pin{
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
	if len(WGSPins) != len(want) {
		t.Fatalf("WGSPins len = %d, want %d", len(WGSPins), len(want))
	}
	named := []Pin{PinWGSTest1FASTQ, PinWGSTest2FASTQ, PinWGSGenomeFASTA, PinWGSGenomeFAI}
	for i, pin := range want {
		if WGSPins[i] != pin {
			t.Fatalf("WGSPins[%d] = %+v, want %+v", i, WGSPins[i], pin)
		}
		if named[i] != pin {
			t.Fatalf("named pin[%d] = %+v, want %+v", i, named[i], pin)
		}
	}
}

func TestPinCheck(t *testing.T) {
	content := []byte("hello pin")
	sum := sha256.Sum256(content)
	pin := Pin{
		Name:   "hello.txt",
		URL:    "https://example.invalid/hello.txt",
		Bytes:  int64(len(content)),
		SHA256: hex.EncodeToString(sum[:]),
	}
	dir := t.TempDir()
	path := filepath.Join(dir, pin.Name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	if err := pin.Check(path); err != nil {
		t.Fatalf("Check() error = %v, want nil", err)
	}

	if err := pin.Check(path + ".missing"); err == nil {
		t.Fatalf("Check(missing) error = nil, want error")
	}

	wrongSize := pin
	wrongSize.Bytes = 1
	if err := wrongSize.Check(path); err == nil {
		t.Fatalf("Check(size mismatch) error = nil, want error")
	} else if !strings.Contains(err.Error(), "size") {
		t.Fatalf("Check(size mismatch) error = %v, want size", err)
	}

	wrongHash := pin
	wrongHash.SHA256 = strings.Repeat("0", 64)
	if err := wrongHash.Check(path); err == nil {
		t.Fatalf("Check(sha256 mismatch) error = nil, want error")
	} else if !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("Check(sha256 mismatch) error = %v, want sha256", err)
	}
}
