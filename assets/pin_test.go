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

func TestRNASeqPins(t *testing.T) {
	want := []Pin{
		{
			Name:   "SRR6357072_1.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357072_1.fastq.gz",
			Bytes:  2148269,
			SHA256: "6c92efe43dc8145951c4131cd30e3e169f9877f041fcf20c2577eeeb7ec2b6ed",
		},
		{
			Name:   "SRR6357072_2.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357072_2.fastq.gz",
			Bytes:  2167239,
			SHA256: "4501eb0062d4005cc1a4c836a86c036c15d3c38d685138d02675c5ecef84c0a3",
		},
		{
			Name:   "genome.fasta",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/reference/genome.fasta",
			Bytes:  234058,
			SHA256: "df70973809f672aa58a414fef3f01e0e465bf26f10159174a616b0dee2d458e1",
		},
		{
			Name:   "genes.gtf",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/reference/genes.gtf",
			Bytes:  204286,
			SHA256: "8d9e66311d90f14517cdb8683066d4b376547ea7c2153b05b510fb9ba7988835",
		},
	}
	if len(RNASeqPins) != len(want) {
		t.Fatalf("RNASeqPins len = %d, want %d", len(RNASeqPins), len(want))
	}
	named := []Pin{PinRNATest1FASTQ, PinRNATest2FASTQ, PinRNAGenomeFASTA, PinRNAGTF}
	for i, pin := range want {
		if RNASeqPins[i] != pin {
			t.Fatalf("RNASeqPins[%d] = %+v, want %+v", i, RNASeqPins[i], pin)
		}
		if named[i] != pin {
			t.Fatalf("named RNA pin[%d] = %+v, want %+v", i, named[i], pin)
		}
	}
	if PinRNAGenomeFASTA.CachePath() == PinWGSGenomeFASTA.CachePath() {
		t.Fatalf("RNA and WGS genome.fasta CachePath collided: %s", PinRNAGenomeFASTA.CachePath())
	}
}

func TestRNASeqDistinctPairPins(t *testing.T) {
	want := []Pin{
		{
			Name:   "SRR6357070_1.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357070_1.fastq.gz",
			Bytes:  2239317,
			SHA256: "3f50541fa9cf2bedc87e7b682ada0fccfdfcd6d27b9bb81f17be230ff140ebe7",
		},
		{
			Name:   "SRR6357070_2.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357070_2.fastq.gz",
			Bytes:  2232117,
			SHA256: "8590e1e01e568fba256aa7dced40519604cbb111ee44ab106a3dcb869660aaf4",
		},
		{
			Name:   "SRR6357071_1.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357071_1.fastq.gz",
			Bytes:  2189243,
			SHA256: "06951c97884df8975d5419a2f0d03d435b9da722564000536524b01926970c93",
		},
		{
			Name:   "SRR6357071_2.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357071_2.fastq.gz",
			Bytes:  2183711,
			SHA256: "00bbbf100ee90c5f681c3b4814637283732cd06e620171bd48718aa02de3a91d",
		},
		PinRNATest1FASTQ,
		PinRNATest2FASTQ,
		{
			Name:   "SRR6357073_1.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357073_1.fastq.gz",
			Bytes:  2253497,
			SHA256: "c564e722216fb74116b237cc88d76cd33cf198216fb01fd9febd735a4503d18f",
		},
		{
			Name:   "SRR6357073_2.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/626c8fab639062eade4b10747e919341cbf9b41a/testdata/GSE110004/SRR6357073_2.fastq.gz",
			Bytes:  2282958,
			SHA256: "599adca68abe6a6116802fb2134104cfcc545d397edb49c070e4a443831226f8",
		},
	}
	if len(RNASeqDistinctPairPins) != len(want) {
		t.Fatalf("RNASeqDistinctPairPins len = %d, want %d", len(RNASeqDistinctPairPins), len(want))
	}
	named := []Pin{
		PinRNACtrl1FASTQ1, PinRNACtrl1FASTQ2,
		PinRNACtrl2FASTQ1, PinRNACtrl2FASTQ2,
		PinRNATest1FASTQ, PinRNATest2FASTQ,
		PinRNATreat2FASTQ1, PinRNATreat2FASTQ2,
	}
	for i, pin := range want {
		if RNASeqDistinctPairPins[i] != pin {
			t.Fatalf("RNASeqDistinctPairPins[%d] = %+v, want %+v", i, RNASeqDistinctPairPins[i], pin)
		}
		if named[i] != pin {
			t.Fatalf("named distinct RNA pin[%d] = %+v, want %+v", i, named[i], pin)
		}
	}
}

func TestMethylSeqPins(t *testing.T) {
	want := []Pin{
		{
			Name:   "Ecoli_10K_methylated_R1.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R1.fastq.gz",
			Bytes:  467487,
			SHA256: "3d84b54e065f0760e830357d37bbc1ce511570b0443b6d0a7da1cf26261fe79b",
		},
		{
			Name:   "Ecoli_10K_methylated_R2.fastq.gz",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/testdata/Ecoli_10K_methylated_R2.fastq.gz",
			Bytes:  467335,
			SHA256: "2f3e6de0edf9bbc6dae46a5a43a2152d6d9d724b8b8ecd46281d47dd0606a646",
		},
		{
			Name:   "genome.fa",
			URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/e7e1fb8940fc14e2336101147a31ce8e0eda6264/reference/genome.fa",
			Bytes:  49200,
			SHA256: "52a320d932e0d873141d5a326d80a7d811653cf2d782d07f8926f6c0e1ceb21e",
		},
	}
	if len(MethylSeqPins) != len(want) {
		t.Fatalf("MethylSeqPins len = %d, want %d", len(MethylSeqPins), len(want))
	}
	named := []Pin{PinMethylTest1FASTQ, PinMethylTest2FASTQ, PinMethylGenomeFASTA}
	for i, pin := range want {
		if MethylSeqPins[i] != pin {
			t.Fatalf("MethylSeqPins[%d] = %+v, want %+v", i, MethylSeqPins[i], pin)
		}
		if named[i] != pin {
			t.Fatalf("named Methyl pin[%d] = %+v, want %+v", i, named[i], pin)
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

func TestPinCachePath(t *testing.T) {
	wgs := PinWGSTest1FASTQ
	sars := pinSARSCoV2R1
	if wgs.Name != "test_1.fastq.gz" || sars.Name != wgs.Name {
		t.Fatalf("Name = %q / %q, want both test_1.fastq.gz", wgs.Name, sars.Name)
	}
	if wgs.SHA256 == sars.SHA256 || wgs.URL == sars.URL {
		t.Fatalf("pins share SHA256 or URL, want distinct records")
	}
	wgsPath := wgs.CachePath()
	sarsPath := sars.CachePath()
	if wgsPath == sarsPath {
		t.Fatalf("CachePath() = %q for both pins", wgsPath)
	}
	if filepath.Base(wgsPath) != wgs.Name || filepath.Base(sarsPath) != sars.Name {
		t.Fatalf("CachePath bases = %q / %q, want %q", filepath.Base(wgsPath), filepath.Base(sarsPath), wgs.Name)
	}
	wantWGS := filepath.Join(CacheDir, wgs.SHA256[:16], wgs.Name)
	wantSARS := filepath.Join(CacheDir, sars.SHA256[:16], sars.Name)
	if wgsPath != wantWGS {
		t.Fatalf("CachePath(WGS) = %q, want %q", wgsPath, wantWGS)
	}
	if sarsPath != wantSARS {
		t.Fatalf("CachePath(SARS) = %q, want %q", sarsPath, wantSARS)
	}
	if wgsPath == filepath.Join(CacheDir, wgs.Name) {
		t.Fatalf("CachePath() used basename-only %q", wgsPath)
	}
}

func TestPinCachePathShortSHA(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("CachePath() panic = nil, want panic")
		}
	}()
	_ = Pin{Name: "x", SHA256: "abcd"}.CachePath()
}

func TestPinFetchCached(t *testing.T) {
	pin := seedPin(t, "cached.txt", []byte("cached pin"))
	got, err := FetchPin(pin)
	if err != nil {
		t.Fatalf("FetchPin() error = %v, want nil", err)
	}
	if got != pin.CachePath() {
		t.Fatalf("FetchPin() = %q, want %q", got, pin.CachePath())
	}
	if err := pin.Check(got); err != nil {
		t.Fatalf("Check(%s) error = %v, want nil", got, err)
	}
}

func TestPinFetchSameNameCheck(t *testing.T) {
	left := seedPin(t, "test_1.fastq.gz", []byte("wgs-like"))
	right := seedPin(t, "test_1.fastq.gz", []byte("sars-like"))
	if left.Name != right.Name {
		t.Fatalf("Name = %q / %q, want equal", left.Name, right.Name)
	}
	if left.CachePath() == right.CachePath() {
		t.Fatalf("CachePath() = %q for both", left.CachePath())
	}
	leftDest, err := FetchPin(left)
	if err != nil {
		t.Fatalf("FetchPin(left) error = %v, want nil", err)
	}
	rightDest, err := FetchPin(right)
	if err != nil {
		t.Fatalf("FetchPin(right) error = %v, want nil", err)
	}
	if leftDest == rightDest {
		t.Fatalf("FetchPin dest = %q for both", leftDest)
	}
	if err := left.Check(leftDest); err != nil {
		t.Fatalf("Check(left) error = %v, want nil", err)
	}
	if err := right.Check(rightDest); err != nil {
		t.Fatalf("Check(right) error = %v, want nil", err)
	}
}

func seedPin(t *testing.T, name string, content []byte) Pin {
	t.Helper()
	sum := sha256.Sum256(content)
	pin := Pin{
		Name:   name,
		URL:    "https://example.invalid/" + name,
		Bytes:  int64(len(content)),
		SHA256: hex.EncodeToString(sum[:]),
	}
	dest := pin.CachePath()
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(dest)) })
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(dest), err)
	}
	if err := os.WriteFile(dest, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", dest, err)
	}
	return pin
}
