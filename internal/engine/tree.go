package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	intpath "github.com/HahyeonJeon/gobble/internal/path"
)

var (
	errTreeMissing = errors.New("missing output")
	errTreeInvalid = errors.New("invalid path")
)

type treeManifest struct {
	Members []treeManifestMember `json:"members"`
	Digest  string               `json:"digest"`
}

type treeManifestMember struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Mtime  int64  `json:"mtime"`
	Dev    uint64 `json:"dev"`
	Inode  uint64 `json:"inode"`
	SHA256 string `json:"sha256"`
}

func isTreeIO(io IO) bool {
	return ioKind(io) == ArtifactTree
}

func treeDir(io IO) string {
	return io.Path
}

func treeManifestPath(io IO) string {
	if io.Manifest != "" {
		return io.Manifest
	}
	dir := strings.ReplaceAll(io.Path, `\`, "/")
	if dir == "" {
		return treeManifestName
	}
	return strings.TrimSuffix(dir, "/") + "/" + treeManifestName
}

func treeSourceDir(io IO) string {
	if io.Source != "" {
		return io.Source
	}
	return io.Path
}

func isDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

func treeInputReady(root string) bool {
	return isDir(root) && regularFile(filepath.Join(root, treeManifestName))
}

func walkTreeMembers(root string) ([]string, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		return nil, errTreeMissing
	}
	var members []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		mode := d.Type()
		if mode&os.ModeSymlink != 0 {
			return errTreeInvalid
		}
		if d.IsDir() {
			return nil
		}
		if !mode.IsRegular() {
			return errTreeInvalid
		}
		if d.Name() == treeManifestName {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		members = append(members, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		if errors.Is(err, errTreeInvalid) || errors.Is(err, errTreeMissing) {
			return nil, err
		}
		return nil, errTreeInvalid
	}
	sort.Strings(members)
	return members, nil
}

func checkTreeOutput(isolate string, out IO) error {
	members, err := walkTreeMembers(workspaceFile(isolate, treeDir(out)))
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return errTreeMissing
	}
	return nil
}

// WriteTreeManifest writes the engine metadata that marks an existing,
// non-empty directory as a complete Gobble Tree. It does not replace an
// existing manifest.
func WriteTreeManifest(root string) error {
	members, err := walkTreeMembers(root)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return errTreeMissing
	}
	for _, rel := range members {
		if err := validateTreeMember(rel); err != nil {
			return err
		}
	}
	return writeTreeManifest(filepath.Join(root, treeManifestName), root, members, false)
}

func stageTree(workspace, isolate string, in IO) error {
	srcRel := treeSourceDir(in)
	srcRoot, err := containedFile(workspace, srcRel)
	if err != nil {
		return err
	}
	if !treeInputReady(srcRoot) {
		return errTreeMissing
	}
	dstRoot := workspaceFile(isolate, treeDir(in))
	members, err := walkTreeMembers(srcRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}
	for _, rel := range members {
		if err := validateTreeMember(rel); err != nil {
			return err
		}
		src := filepath.Join(srcRoot, filepath.FromSlash(rel))
		if err := proveInside(srcRoot, src); err != nil {
			return errTreeInvalid
		}
		dst := filepath.Join(dstRoot, filepath.FromSlash(rel))
		if err := exec.StageFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func publishTree(workspace, isolate string, out IO, replace bool) (wrote []string, err error) {
	srcRoot := workspaceFile(isolate, treeDir(out))
	members, err := walkTreeMembers(srcRoot)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, errTreeMissing
	}
	for _, rel := range members {
		if err := validateTreeMember(rel); err != nil {
			return nil, err
		}
	}
	dstRel := treeDir(out)
	dstRoot, present, err := containedRel(workspace, dstRel, true)
	if err != nil {
		return nil, err
	}
	if !replace && present {
		return nil, os.ErrExist
	}
	parent := filepath.Dir(dstRoot)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	gen, err := os.MkdirTemp(parent, ".gobble-tree-")
	if err != nil {
		return nil, err
	}
	cleanupGen := true
	defer func() {
		if cleanupGen {
			os.RemoveAll(gen)
		}
	}()
	for _, rel := range members {
		src := filepath.Join(srcRoot, filepath.FromSlash(rel))
		if err := proveInside(srcRoot, src); err != nil {
			return nil, errTreeInvalid
		}
		dst := filepath.Join(gen, filepath.FromSlash(rel))
		if err := proveInside(gen, dst); err != nil {
			return nil, errTreeInvalid
		}
		if err := exec.PublishFile(src, dst); err != nil {
			return nil, err
		}
	}
	manRel := treeManifestName
	if out.Manifest != "" {
		manAbs, _, manErr := containedRel(workspace, out.Manifest, true)
		if manErr != nil {
			return nil, manErr
		}
		rel, relErr := filepath.Rel(dstRoot, manAbs)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, errTreeInvalid
		}
		manRel = filepath.ToSlash(rel)
	}
	manPath := filepath.Join(gen, filepath.FromSlash(manRel))
	if err := writeTreeManifest(manPath, gen, members, false); err != nil {
		return nil, err
	}
	if !replace || !present {
		if err := os.Rename(gen, dstRoot); err != nil {
			return nil, err
		}
		cleanupGen = false
		return []string{dstRoot}, nil
	}
	backup, err := os.MkdirTemp(parent, ".gobble-tree-old-")
	if err != nil {
		return nil, err
	}
	os.Remove(backup)
	if err := os.Rename(dstRoot, backup); err != nil {
		os.RemoveAll(backup)
		return nil, err
	}
	if err := os.Rename(gen, dstRoot); err != nil {
		if restoreErr := os.Rename(backup, dstRoot); restoreErr != nil {
			return nil, err
		}
		return nil, err
	}
	cleanupGen = false
	os.RemoveAll(backup)
	return []string{dstRoot}, nil
}

func validateTreeMember(rel string) error {
	if rel == "" || rel == treeManifestName || strings.Contains(rel, "\x00") {
		return errTreeInvalid
	}
	normalized := strings.ReplaceAll(rel, `\`, "/")
	if filepath.IsAbs(rel) || strings.HasPrefix(normalized, "/") {
		return errTreeInvalid
	}
	cleaned, escaped := intpath.Clean(normalized)
	if escaped || cleaned == "" || cleaned == "." || cleaned == treeManifestName {
		return errTreeInvalid
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == "" || part == "." || part == ".." || part == treeManifestName {
			return errTreeInvalid
		}
	}
	return nil
}

func writeTreeManifest(path, destRoot string, members []string, replace bool) error {
	body := treeManifest{Members: make([]treeManifestMember, 0, len(members))}
	digests := make([]string, 0, len(members))
	for _, rel := range members {
		abs := filepath.Join(destRoot, filepath.FromSlash(rel))
		rec, err := fileRecord(abs, rel)
		if err != nil {
			return err
		}
		body.Members = append(body.Members, treeManifestMember{
			Path:   rel,
			Size:   rec.Size,
			Mtime:  rec.Mtime,
			Dev:    rec.Dev,
			Inode:  rec.Inode,
			SHA256: rec.SHA256,
		})
		digests = append(digests, rec.SHA256)
	}
	body.Digest = treeManifestDigest(digests)
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if replace && pathPresent(path) {
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			os.Remove(tmp)
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil {
		os.Remove(path)
		return werr
	}
	if cerr != nil {
		os.Remove(path)
		return cerr
	}
	return nil
}

func treeManifestDigest(digests []string) string {
	sorted := append([]string(nil), digests...)
	sort.Strings(sorted)
	h := sha256.New()
	for _, d := range sorted {
		h.Write([]byte(d))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func readTreeManifestMembers(path string) []string {
	rels, err := readTreeManifestMembersStrict(path)
	if err != nil {
		return nil
	}
	return rels
}

func readTreeManifestMembersStrict(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var body treeManifest
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(body.Members))
	out := make([]string, 0, len(body.Members))
	for _, m := range body.Members {
		if err := validateTreeMember(m.Path); err != nil {
			return nil, err
		}
		if seen[m.Path] {
			return nil, errTreeInvalid
		}
		seen[m.Path] = true
		out = append(out, m.Path)
	}
	return out, nil
}

func treeDestMemberPaths(workspace string, io IO) []namedFile {
	return treeMemberPaths(workspace, treeDir(io), treeManifestPath(io))
}

func treeSourceMemberPaths(workspace string, io IO) []namedFile {
	src := treeSourceDir(io)
	if !treeInputReady(workspaceFile(workspace, src)) {
		return nil
	}
	man := treeManifestName
	if src != "" {
		man = strings.TrimSuffix(strings.ReplaceAll(src, `\`, "/"), "/") + "/" + treeManifestName
	}
	return treeMemberPaths(workspace, src, man)
}

func treeMemberPaths(workspace, dir, manifest string) []namedFile {
	root := workspaceFile(workspace, dir)
	rels := readTreeManifestMembers(workspaceFile(workspace, manifest))
	if len(rels) == 0 {
		walked, err := walkTreeMembers(root)
		if err != nil {
			return nil
		}
		rels = walked
	}
	out := make([]namedFile, 0, len(rels))
	prefix := strings.TrimSuffix(strings.ReplaceAll(dir, `\`, "/"), "/")
	for _, rel := range rels {
		plan := rel
		if prefix != "" {
			plan = prefix + "/" + rel
		}
		out = append(out, namedFile{name: rel, path: plan})
	}
	return out
}

func containsString(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}
