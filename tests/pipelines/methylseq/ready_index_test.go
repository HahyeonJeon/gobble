package methylseqevidence

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
)

func TestOfficialReadyIndexArchiveIdentity(t *testing.T) {
	archive := os.Getenv("GOBBLE_METHYL_READY_INDEX_ARCHIVE")
	if archive == "" {
		t.Skip("set GOBBLE_METHYL_READY_INDEX_ARCHIVE to the pinned Bowtie2_Index.tar.gz")
	}
	destination := filepath.Join(t.TempDir(), "reference", "BismarkIndex")
	if err := PrepareReadyIndex(archive, destination); err != nil {
		t.Fatalf("PrepareReadyIndex(official archive) error = %v", err)
	}
	if err := CheckReadyIndex(destination); err != nil {
		t.Fatalf("CheckReadyIndex(official archive) error = %v", err)
	}
}

func TestStageReadyIndexVerifiesArchiveMembersBeforePublication(t *testing.T) {
	authority := testReadyIndexAuthority()
	archive := writeReadyIndexArchive(t, authority, map[string]string{
		"BismarkIndex/genome.fa":                     "one",
		"BismarkIndex/Bisulfite_Genome/CT/index.bt2": "two",
	})
	destination := filepath.Join(t.TempDir(), "reference", "BismarkIndex")
	if err := prepareReadyIndex(archive.path, destination, archive.pin, authority); err != nil {
		t.Fatalf("prepareReadyIndex() error = %v", err)
	}
	if err := checkReadyIndex(destination, authority); err != nil {
		t.Fatalf("checkReadyIndex() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "Bisulfite_Genome", "CT", "index.bt2")); err != nil {
		t.Fatalf("staged ready-index member: %v", err)
	}
}

func TestReadyIndexRejectsMissingExtraWrongSizeAndWrongHashMembers(t *testing.T) {
	authority := testReadyIndexAuthority()
	tests := []struct {
		name   string
		mutate func(t *testing.T, root string)
		want   string
	}{
		{name: "missing", mutate: func(t *testing.T, root string) {
			t.Helper()
			if err := os.Remove(filepath.Join(root, "genome.fa")); err != nil {
				t.Fatal(err)
			}
		}, want: "missing member"},
		{name: "extra", mutate: func(t *testing.T, root string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(root, "extra.bt2"), []byte("extra"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: "undeclared member"},
		{name: "wrong size", mutate: func(t *testing.T, root string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(root, "genome.fa"), []byte("longer"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: "size"},
		{name: "wrong hash", mutate: func(t *testing.T, root string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(root, "genome.fa"), []byte("ONE"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: "sha256"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writeReadyIndexTree(t)
			test.mutate(t, root)
			if err := checkReadyIndex(root, authority); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("checkReadyIndex() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestStageReadyIndexRejectsUndeclaredArchiveMember(t *testing.T) {
	authority := testReadyIndexAuthority()
	archive := writeReadyIndexArchive(t, authority, map[string]string{
		"BismarkIndex/genome.fa":                     "one",
		"BismarkIndex/Bisulfite_Genome/CT/index.bt2": "two",
		"BismarkIndex/extra.bt2":                     "extra",
	})
	destination := filepath.Join(t.TempDir(), "BismarkIndex")
	err := prepareReadyIndex(archive.path, destination, archive.pin, authority)
	if err == nil || !strings.Contains(err.Error(), "undeclared member") {
		t.Fatalf("prepareReadyIndex() error = %v, want undeclared member", err)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("invalid archive published destination: %v", statErr)
	}
}

func testReadyIndexAuthority() readyIndexTree {
	return readyIndexTree{
		Name:    "BismarkIndex",
		Archive: "index.tar.gz",
		Members: []readyIndexMember{
			{Path: "BismarkIndex/genome.fa", Bytes: 3, SHA256: sumString("one")},
			{Path: "BismarkIndex/Bisulfite_Genome/CT/index.bt2", Bytes: 3, SHA256: sumString("two")},
		},
	}
}

func writeReadyIndexTree(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "BismarkIndex")
	if err := os.MkdirAll(filepath.Join(root, "Bisulfite_Genome", "CT"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"genome.fa":                     "one",
		"Bisulfite_Genome/CT/index.bt2": "two",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

type testArchive struct {
	path string
	pin  fixture.Pin
}

func writeReadyIndexArchive(t *testing.T, authority readyIndexTree, files map[string]string) testArchive {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), authority.Archive)
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	writer := tar.NewWriter(compressed)
	for _, directory := range []string{"BismarkIndex/", "BismarkIndex/Bisulfite_Genome/", "BismarkIndex/Bisulfite_Genome/CT/"} {
		if err := writer.WriteHeader(&tar.Header{Name: directory, Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range files {
		if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := fileSHA256(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	return testArchive{
		path: archivePath,
		pin:  fixture.Pin{Name: authority.Archive, Bytes: info.Size(), SHA256: hash},
	}
}

func sumString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
