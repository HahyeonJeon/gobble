package methylseqevidence

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/internal/fixture"
)

const gobbleTreeManifestName = ".gobble-tree.json"

type readyIndexManifest struct {
	Trees []readyIndexTree `json:"trees"`
}

type readyIndexTree struct {
	Name    string             `json:"name"`
	Archive string             `json:"archive"`
	Members []readyIndexMember `json:"members"`
}

type readyIndexMember struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// PrepareReadyIndex verifies the official archive, extracts its declared
// BismarkIndex Tree, and verifies every extracted member before publication.
// treeDir must end in BismarkIndex and must not already exist.
func PrepareReadyIndex(archivePath, treeDir string) error {
	authority, err := officialReadyIndexAuthority()
	if err != nil {
		return err
	}
	return prepareReadyIndex(archivePath, treeDir, MustPin("Bowtie2_Index.tar.gz"), authority)
}

// CheckReadyIndex verifies that treeDir contains the Gobble Tree manifest and
// exactly the official BismarkIndex members, sizes, and SHA-256 identities.
func CheckReadyIndex(treeDir string) error {
	authority, err := officialReadyIndexAuthority()
	if err != nil {
		return err
	}
	return checkReadyIndex(treeDir, authority)
}

func officialReadyIndexAuthority() (readyIndexTree, error) {
	var manifest readyIndexManifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return readyIndexTree{}, fmt.Errorf("ready Bismark index manifest: %w", err)
	}
	if len(manifest.Trees) != 1 {
		return readyIndexTree{}, fmt.Errorf("ready Bismark index manifest: got %d Trees, want 1", len(manifest.Trees))
	}
	authority := manifest.Trees[0]
	if authority.Name == "" || strings.Contains(authority.Name, "/") || authority.Archive != MustPin("Bowtie2_Index.tar.gz").Name || len(authority.Members) == 0 {
		return readyIndexTree{}, errors.New("ready Bismark index manifest: invalid Tree authority")
	}
	seen := make(map[string]bool, len(authority.Members))
	for _, member := range authority.Members {
		if path.Clean(member.Path) != member.Path || !strings.HasPrefix(member.Path, authority.Name+"/") || member.Bytes <= 0 || !validMemberSHA256(member.SHA256) || seen[member.Path] {
			return readyIndexTree{}, fmt.Errorf("ready Bismark index manifest: invalid member %q", member.Path)
		}
		seen[member.Path] = true
	}
	return authority, nil
}

func prepareReadyIndex(archivePath, treeDir string, archivePin fixture.Pin, authority readyIndexTree) error {
	if err := archivePin.Check(archivePath); err != nil {
		return fmt.Errorf("ready Bismark index archive: %w", err)
	}
	if filepath.Base(filepath.Clean(treeDir)) != authority.Name {
		return fmt.Errorf("ready Bismark index destination %q must end in %s", treeDir, authority.Name)
	}
	if _, err := os.Lstat(treeDir); err == nil {
		return fmt.Errorf("ready Bismark index destination %q already exists", treeDir)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("ready Bismark index destination %q: %w", treeDir, err)
	}
	parent := filepath.Dir(treeDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("ready Bismark index destination parent: %w", err)
	}
	temp, err := os.MkdirTemp(parent, ".bismark-index-")
	if err != nil {
		return fmt.Errorf("ready Bismark index temporary directory: %w", err)
	}
	defer os.RemoveAll(temp)
	if err := extractReadyIndex(archivePath, temp, authority); err != nil {
		return err
	}
	staged := filepath.Join(temp, authority.Name)
	if err := engine.WriteTreeManifest(staged); err != nil {
		return fmt.Errorf("write ready Bismark index Tree manifest: %w", err)
	}
	if err := checkReadyIndex(staged, authority); err != nil {
		return err
	}
	if err := os.Rename(staged, treeDir); err != nil {
		return fmt.Errorf("publish ready Bismark index: %w", err)
	}
	return nil
}

func extractReadyIndex(archivePath, destination string, authority readyIndexTree) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open ready Bismark index archive: %w", err)
	}
	defer archive.Close()
	compressed, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("open ready Bismark index gzip stream: %w", err)
	}
	defer compressed.Close()

	members, directories := readyIndexSets(authority)
	seen := make(map[string]bool, len(members)+len(directories))
	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read ready Bismark index archive: %w", err)
		}
		name := strings.TrimSuffix(header.Name, "/")
		if name == "" || path.Clean(name) != name || path.IsAbs(name) || seen[name] {
			return fmt.Errorf("ready Bismark index archive has invalid or duplicate member %q", header.Name)
		}
		seen[name] = true
		target := filepath.Join(destination, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if !directories[name] {
				return fmt.Errorf("ready Bismark index archive has undeclared directory %q", name)
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create ready Bismark index directory %q: %w", name, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			member, ok := members[name]
			if !ok {
				return fmt.Errorf("ready Bismark index archive has undeclared member %q", name)
			}
			if header.Size != member.Bytes {
				return fmt.Errorf("ready Bismark index member %q size %d, want %d", name, header.Size, member.Bytes)
			}
			if err := writeReadyIndexMember(target, reader, member); err != nil {
				return err
			}
		default:
			return fmt.Errorf("ready Bismark index archive member %q is not a regular file or directory", name)
		}
	}
	for name := range members {
		if !seen[name] {
			return fmt.Errorf("ready Bismark index archive is missing member %q", name)
		}
	}
	return nil
}

func writeReadyIndexMember(target string, source io.Reader, member readyIndexMember) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create ready Bismark index member parent: %w", err)
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create ready Bismark index member %q: %w", member.Path, err)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), source)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("extract ready Bismark index member %q: %w", member.Path, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close ready Bismark index member %q: %w", member.Path, closeErr)
	}
	if written != member.Bytes {
		return fmt.Errorf("ready Bismark index member %q size %d, want %d", member.Path, written, member.Bytes)
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != member.SHA256 {
		return fmt.Errorf("ready Bismark index member %q sha256 %s, want %s", member.Path, got, member.SHA256)
	}
	return nil
}

func checkReadyIndex(treeDir string, authority readyIndexTree) error {
	if filepath.Base(filepath.Clean(treeDir)) != authority.Name {
		return fmt.Errorf("ready Bismark index Tree %q must end in %s", treeDir, authority.Name)
	}
	info, err := os.Lstat(treeDir)
	if err != nil {
		return fmt.Errorf("ready Bismark index Tree: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("ready Bismark index Tree %q is not a directory", treeDir)
	}
	members, directories := readyIndexSets(authority)
	seen := make(map[string]bool, len(members))
	manifestSeen := false
	err = filepath.WalkDir(treeDir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == treeDir {
			return nil
		}
		relative, err := filepath.Rel(filepath.Dir(treeDir), current)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if name == authority.Name+"/"+gobbleTreeManifestName {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("ready Bismark index Tree manifest is not a regular file")
			}
			manifestSeen = true
			return nil
		}
		if entry.IsDir() {
			if !directories[name] {
				return fmt.Errorf("ready Bismark index Tree has undeclared directory %q", name)
			}
			return nil
		}
		member, ok := members[name]
		if !ok {
			return fmt.Errorf("ready Bismark index Tree has undeclared member %q", name)
		}
		memberInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if !memberInfo.Mode().IsRegular() {
			return fmt.Errorf("ready Bismark index Tree member %q is not a regular file", name)
		}
		if memberInfo.Size() != member.Bytes {
			return fmt.Errorf("ready Bismark index Tree member %q size %d, want %d", name, memberInfo.Size(), member.Bytes)
		}
		got, err := fileSHA256(current)
		if err != nil {
			return err
		}
		if got != member.SHA256 {
			return fmt.Errorf("ready Bismark index Tree member %q sha256 %s, want %s", name, got, member.SHA256)
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		return err
	}
	for name := range members {
		if !seen[name] {
			return fmt.Errorf("ready Bismark index Tree is missing member %q", name)
		}
	}
	if !manifestSeen {
		return fmt.Errorf("ready Bismark index Tree is missing %s", gobbleTreeManifestName)
	}
	return nil
}

func readyIndexSets(authority readyIndexTree) (map[string]readyIndexMember, map[string]bool) {
	members := make(map[string]readyIndexMember, len(authority.Members))
	directories := map[string]bool{authority.Name: true}
	for _, member := range authority.Members {
		members[member.Path] = member
		for directory := path.Dir(member.Path); directory != "."; directory = path.Dir(directory) {
			directories[directory] = true
		}
	}
	return members, directories
}

func fileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open ready Bismark index member: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash ready Bismark index member: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validMemberSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}
