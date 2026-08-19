package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/HahyeonJeon/gobble"
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

// RNA-seq pins from nf-core/rnaseq 3.26.0 test.config snapshot
// 626c8fab639062eade4b10747e919341cbf9b41a. Not the WGS modules pins.
var (
	PinRNATest1FASTQ = Pin{
		Name:   "SRR6357072_1.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357072_1.fastq.gz",
		Bytes:  2148269,
		SHA256: "6c92efe43dc8145951c4131cd30e3e169f9877f041fcf20c2577eeeb7ec2b6ed",
	}
	PinRNATest2FASTQ = Pin{
		Name:   "SRR6357072_2.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357072_2.fastq.gz",
		Bytes:  2167239,
		SHA256: "4501eb0062d4005cc1a4c836a86c036c15d3c38d685138d02675c5ecef84c0a3",
	}
	PinRNAGenomeFASTA = Pin{
		Name:   "genome.fasta",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/reference/genome.fasta",
		Bytes:  234058,
		SHA256: "df70973809f672aa58a414fef3f01e0e465bf26f10159174a616b0dee2d458e1",
	}
	PinRNAGTF = Pin{
		Name:   "genes.gtf",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/reference/genes.gtf",
		Bytes:  204286,
		SHA256: "8d9e66311d90f14517cdb8683066d4b376547ea7c2153b05b510fb9ba7988835",
	}
)

// RNASeqPins is the four official RNA-seq pin records.
var RNASeqPins = []Pin{
	PinRNATest1FASTQ,
	PinRNATest2FASTQ,
	PinRNAGenomeFASTA,
	PinRNAGTF,
}

// Methyl-seq pins from nf-core/test-datasets methylseq
// e7e1fb8940fc14e2336101147a31ce8e0eda6264. Not the WGS modules pins.
var (
	PinMethylTest1FASTQ = Pin{
		Name:   "Ecoli_10K_methylated_R1.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R1.fastq.gz",
		Bytes:  467487,
		SHA256: "3d84b54e065f0760e830357d37bbc1ce511570b0443b6d0a7da1cf26261fe79b",
	}
	PinMethylTest2FASTQ = Pin{
		Name:   "Ecoli_10K_methylated_R2.fastq.gz",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R2.fastq.gz",
		Bytes:  467335,
		SHA256: "2f3e6de0edf9bbc6dae46a5a43a2152d6d9d724b8b8ecd46281d47dd0606a646",
	}
	PinMethylGenomeFASTA = Pin{
		Name:   "genome.fa",
		URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/reference/genome.fa",
		Bytes:  49200,
		SHA256: "52a320d932e0d873141d5a326d80a7d811653cf2d782d07f8926f6c0e1ceb21e",
	}
)

// MethylSeqPins is the three official Methyl-seq pin records.
var MethylSeqPins = []Pin{
	PinMethylTest1FASTQ,
	PinMethylTest2FASTQ,
	PinMethylGenomeFASTA,
}

func pinnedRNAFASTQ1() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Name: "SRR6357072_1", Ext: ".fastq.gz"}
}

func pinnedRNAFASTQ2() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Name: "SRR6357072_2", Ext: ".fastq.gz"}
}

func pinnedRNAFASTA() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
}

func pinnedRNAGTF() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genes", Ext: ".gtf"}
}

func pinnedMethylFASTQ1() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Name: "Ecoli_10K_methylated_R1", Ext: ".fastq.gz"}
}

func pinnedMethylFASTQ2() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Name: "Ecoli_10K_methylated_R2", Ext: ".fastq.gz"}
}

func pinnedMethylFASTA() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fa"}
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
