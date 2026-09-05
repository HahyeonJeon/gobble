package methylseqevidence

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/internal/fixture"
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
	files := map[string]string{
		"BismarkIndex/genome.fa":                     "one",
		"BismarkIndex/Bisulfite_Genome/CT/index.bt2": "two",
	}
	archive := writeReadyIndexArchive(t, authority, files)
	destination := filepath.Join(t.TempDir(), "reference", "BismarkIndex")
	if err := prepareReadyIndex(archive.path, destination, archive.pin, authority); err != nil {
		t.Fatalf("prepareReadyIndex() error = %v", err)
	}
	if err := checkReadyIndex(destination, authority); err != nil {
		t.Fatalf("checkReadyIndex() error = %v", err)
	}
	assertGobbleTreeManifest(t, destination, files)
	if _, err := os.Stat(filepath.Join(destination, "Bisulfite_Genome", "CT", "index.bt2")); err != nil {
		t.Fatalf("staged ready-index member: %v", err)
	}
}

func TestCheckReadyIndexRejectsMissingTreeManifest(t *testing.T) {
	authority := testReadyIndexAuthority()
	root := writeReadyIndexTree(t)
	if err := os.Remove(filepath.Join(root, gobbleTreeManifestName)); err != nil {
		t.Fatal(err)
	}

	err := checkReadyIndex(root, authority)
	if err == nil || !strings.Contains(err.Error(), "missing "+gobbleTreeManifestName) {
		t.Fatalf("checkReadyIndex() error = %v, want missing %s", err, gobbleTreeManifestName)
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
	if err := engine.WriteTreeManifest(root); err != nil {
		t.Fatal(err)
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

type testTreeManifest struct {
	Members []testTreeManifestMember `json:"members"`
	Digest  string                   `json:"digest"`
}

type testTreeManifestMember struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Mtime  int64  `json:"mtime"`
	Dev    uint64 `json:"dev"`
	Inode  uint64 `json:"inode"`
	SHA256 string `json:"sha256"`
}

func assertGobbleTreeManifest(t *testing.T, root string, files map[string]string) {
	t.Helper()
	manifestPath := filepath.Join(root, gobbleTreeManifestName)
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		t.Fatalf("ready-index Tree manifest: %v", err)
	}
	if !manifestInfo.Mode().IsRegular() {
		t.Fatalf("ready-index Tree manifest mode = %v, want regular file", manifestInfo.Mode())
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest testTreeManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("ready-index Tree manifest JSON: %v", err)
	}
	wantPaths := make([]string, 0, len(files))
	for member := range files {
		relative, err := filepath.Rel(filepath.Base(root), filepath.FromSlash(member))
		if err != nil {
			t.Fatal(err)
		}
		wantPaths = append(wantPaths, filepath.ToSlash(relative))
	}
	sort.Strings(wantPaths)
	if len(manifest.Members) != len(wantPaths) {
		t.Fatalf("ready-index Tree manifest members = %#v, want %d", manifest.Members, len(wantPaths))
	}
	digests := make([]string, 0, len(wantPaths))
	for i, wantPath := range wantPaths {
		member := manifest.Members[i]
		if member.Path != wantPath || member.Path == gobbleTreeManifestName {
			t.Fatalf("ready-index Tree manifest member %d path = %q, want %q", i, member.Path, wantPath)
		}
		absolute := filepath.Join(root, filepath.FromSlash(wantPath))
		info, err := os.Lstat(absolute)
		if err != nil {
			t.Fatal(err)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatalf("ready-index Tree member %q stat_t unavailable", wantPath)
		}
		wantSHA := sumString(files[filepath.ToSlash(filepath.Join(filepath.Base(root), wantPath))])
		if member.Size != info.Size() || member.Mtime != info.ModTime().UnixNano() || member.Dev != uint64(stat.Dev) || member.Inode != uint64(stat.Ino) || member.SHA256 != wantSHA {
			t.Fatalf("ready-index Tree manifest member %q = %+v, want destination stats and sha256 %s", wantPath, member, wantSHA)
		}
		digests = append(digests, member.SHA256)
	}
	sort.Strings(digests)
	digest := sha256.New()
	for _, value := range digests {
		digest.Write([]byte(value))
		digest.Write([]byte{'\n'})
	}
	wantDigest := hex.EncodeToString(digest.Sum(nil))
	if manifest.Digest != wantDigest {
		t.Fatalf("ready-index Tree manifest digest = %q, want %q", manifest.Digest, wantDigest)
	}
}
