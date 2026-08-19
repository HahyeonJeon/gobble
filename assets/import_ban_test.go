package assets

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bannedImport = "github.com/HahyeonJeon/gobble/assets"

func TestImportBan(t *testing.T) {
	root := moduleRoot(t)
	var hits []string
	hits = append(hits, importHits(t, filepath.Join(root), false)...)
	hits = append(hits, importHits(t, filepath.Join(root, "tests", "wgs-e2e"), true)...)
	if len(hits) > 0 {
		t.Fatalf("banned import %s in:\n%s", bannedImport, strings.Join(hits, "\n"))
	}
}

func TestDummyComposeFileOmitsAssetsImport(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "assets", "dummy_compose_test.go")
	imps := fileImports(t, path)
	for _, imp := range imps {
		if imp == bannedImport {
			t.Fatalf("%s imports %s, want gobble only", path, bannedImport)
		}
	}
	if !containsImport(imps, "github.com/HahyeonJeon/gobble") {
		t.Fatalf("%s missing gobble import, got %v", path, imps)
	}
}

func importHits(t *testing.T, dir string, recurse bool) []string {
	t.Helper()
	var hits []string
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recurse && path != dir {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		for _, imp := range fileImports(t, path) {
			if imp == bannedImport {
				hits = append(hits, path)
				break
			}
		}
		return nil
	}
	if err := filepath.WalkDir(dir, walk); err != nil {
		t.Fatalf("WalkDir(%s) error = %v", dir, err)
	}
	return hits
}

func fileImports(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s) error = %v", path, err)
	}
	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		if imp.Path != nil {
			out = append(out, strings.Trim(imp.Path.Value, `"`))
		}
	}
	return out
}

func containsImport(imps []string, want string) bool {
	for _, imp := range imps {
		if imp == want {
			return true
		}
	}
	return false
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}
