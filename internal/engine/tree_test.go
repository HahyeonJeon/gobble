package engine

import (
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
