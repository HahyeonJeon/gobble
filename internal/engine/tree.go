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

func stageTree(workspace, isolate string, in IO, allowSymlink bool) error {
	srcRoot := workspaceFile(workspace, treeSourceDir(in))
	dstRoot := workspaceFile(isolate, treeDir(in))
	members, err := walkTreeMembers(srcRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}
	for _, rel := range members {
		src := filepath.Join(srcRoot, filepath.FromSlash(rel))
		dst := filepath.Join(dstRoot, filepath.FromSlash(rel))
		if err := exec.StageFile(src, dst, allowSymlink); err != nil {
			return err
		}
	}
	return nil
}

func publishTree(workspace, isolate string, out IO, replace bool) (wrote []string, err error) {
	srcRoot := workspaceFile(isolate, treeDir(out))
	dstRoot := workspaceFile(workspace, treeDir(out))
	members, err := walkTreeMembers(srcRoot)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, errTreeMissing
	}
	created := false
	if !pathPresent(dstRoot) {
		if err := os.MkdirAll(dstRoot, 0o755); err != nil {
			return nil, err
		}
		created = true
	}
	old := map[string]bool{}
	if replace {
		for _, rel := range readTreeManifestMembers(workspaceFile(workspace, treeManifestPath(out))) {
			old[rel] = true
		}
	}
	rollback := func() {
		for _, p := range wrote {
			os.Remove(p)
		}
		if created {
			os.RemoveAll(dstRoot)
		}
	}
	for _, rel := range members {
		src := filepath.Join(srcRoot, filepath.FromSlash(rel))
		dst := filepath.Join(dstRoot, filepath.FromSlash(rel))
		if replace && pathPresent(dst) {
			if err := exec.StagedReplace(src, dst); err != nil {
				rollback()
				return nil, err
			}
		} else {
			if err := exec.PublishFile(src, dst); err != nil {
				rollback()
				return nil, err
			}
			wrote = append(wrote, dst)
		}
		delete(old, rel)
	}
	manPath := workspaceFile(workspace, treeManifestPath(out))
	if err := writeTreeManifest(manPath, dstRoot, members, replace); err != nil {
		rollback()
		return nil, err
	}
	if !replace || !containsString(wrote, manPath) {
		wrote = append(wrote, manPath)
	}
	if replace {
		for rel := range old {
			os.Remove(filepath.Join(dstRoot, filepath.FromSlash(rel)))
		}
	}
	return wrote, nil
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var body treeManifest
	if err := json.Unmarshal(data, &body); err != nil {
		return nil
	}
	out := make([]string, 0, len(body.Members))
	for _, m := range body.Members {
		if m.Path != "" {
			out = append(out, m.Path)
		}
	}
	return out
}

func treeDestMemberPaths(workspace string, io IO) []namedFile {
	dir := treeDir(io)
	root := workspaceFile(workspace, dir)
	rels := readTreeManifestMembers(workspaceFile(workspace, treeManifestPath(io)))
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
