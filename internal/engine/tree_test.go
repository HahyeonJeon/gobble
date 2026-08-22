package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWalkTreeMembers(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "SA"), "x")
	writeCheckFile(t, filepath.Join(dir, "sub", "chrName.txt"), "y")
	writeCheckFile(t, filepath.Join(dir, treeManifestName), "{}")
	got, err := walkTreeMembers(dir)
	if err != nil {
		t.Fatalf("walkTreeMembers() error = %v", err)
	}
	if len(got) != 2 || got[0] != "SA" || got[1] != "sub/chrName.txt" {
		t.Fatalf("members = %#v, want [SA sub/chrName.txt]", got)
	}
}

func TestWalkTreeMembersRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "SA"), "x")
	if err := os.Symlink("SA", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	_, err := walkTreeMembers(dir)
	if err != errTreeInvalid {
		t.Fatalf("walkTreeMembers() error = %v, want invalid path", err)
	}
}

func TestWalkTreeMembersEmpty(t *testing.T) {
	dir := t.TempDir()
	got, err := walkTreeMembers(dir)
	if err != nil {
		t.Fatalf("empty dir walk error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty members = %#v", got)
	}
}

func TestWalkTreeMembersMissing(t *testing.T) {
	_, err := walkTreeMembers(filepath.Join(t.TempDir(), "absent"))
	if err != errTreeMissing {
		t.Fatalf("missing dir walk error = %v, want missing output", err)
	}
}

func TestPublishTreeReplaceIsOneGeneration(t *testing.T) {
	dir := t.TempDir()
	oldIso := filepath.Join(dir, "old-iso")
	writeCheckFile(t, filepath.Join(oldIso, "idx", "a"), "old-a")
	writeCheckFile(t, filepath.Join(oldIso, "idx", "b"), "old-b")
	if _, err := publishTree(dir, oldIso, IO{Name: "idx", Kind: ArtifactTree, Path: "idx", Manifest: "idx/" + treeManifestName}, false); err != nil {
		t.Fatalf("old publishTree() error = %v", err)
	}
	newIso := filepath.Join(dir, "new-iso")
	writeCheckFile(t, filepath.Join(newIso, "idx", "a"), "new-a")
	writeCheckFile(t, filepath.Join(newIso, "idx", "c"), "new-c")
	if _, err := publishTree(dir, newIso, IO{Name: "idx", Kind: ArtifactTree, Path: "idx", Manifest: "idx/" + treeManifestName}, true); err != nil {
		t.Fatalf("replace publishTree() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "idx", "b")); !os.IsNotExist(err) {
		t.Fatal("mixed tree kept old member b")
	}
	gotA, err := os.ReadFile(filepath.Join(dir, "idx", "a"))
	if err != nil || string(gotA) != "new-a" {
		t.Fatalf("member a got %q, want new-a", gotA)
	}
	gotC, err := os.ReadFile(filepath.Join(dir, "idx", "c"))
	if err != nil || string(gotC) != "new-c" {
		t.Fatalf("member c got %q, want new-c", gotC)
	}
	man, err := os.ReadFile(filepath.Join(dir, "idx", treeManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(man, []byte(`"path": "c"`)) || bytes.Contains(man, []byte(`"path": "b"`)) {
		t.Fatalf("manifest got %s, want new generation", man)
	}
}

func TestPublishTreeDoesNotDeleteEscapedManifest(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "idx")
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dst, "a"), "old")
	escape := filepath.Join("..", filepath.Base(outside), "sentinel")
	if err := os.WriteFile(filepath.Join(dst, treeManifestName), []byte(`{"members":[{"path":"`+escape+`"}]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	iso := filepath.Join(dir, "iso")
	writeCheckFile(t, filepath.Join(iso, "idx", "a"), "new")
	writeCheckFile(t, filepath.Join(iso, "idx", "c"), "c")
	if _, err := publishTree(dir, iso, IO{Name: "idx", Kind: ArtifactTree, Path: "idx", Manifest: "idx/" + treeManifestName}, true); err != nil {
		t.Fatalf("replace with escaped old manifest error = %v", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep" {
		t.Fatalf("escaped sentinel got %q, want keep", got)
	}
}
