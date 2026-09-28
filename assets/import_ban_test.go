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
	hits := importHits(t, moduleRoot(t))
	if len(hits) > 0 {
		t.Fatalf("banned import %s in:\n%s", bannedImport, strings.Join(hits, "\n"))
	}
}

func importHits(t *testing.T, root string) []string {
	t.Helper()
	var hits []string
	assetsDir := filepath.Join(root, "assets")
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Nested modules and generated dependencies are not this module's product source.
			if path != root {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return fs.SkipDir
				}
				if d.Name() == "node_modules" || d.Name() == "vendor" || path == filepath.Join(root, "app", "test-results") {
					return fs.SkipDir
				}
			}
			if path == assetsDir {
				return fs.SkipDir
			}
			if path == filepath.Join(root, "tests") {
				return fs.SkipDir
			}
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for _, imp := range fileImports(t, path) {
			if imp == bannedImport || strings.HasPrefix(imp, bannedImport+"/") {
				hits = append(hits, path)
				break
			}
		}
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		t.Fatalf("WalkDir(%s) error = %v", root, err)
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

func TestImportBanProductionScope(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"core.go":                              "package core\nimport _ \"" + bannedImport + "/modules\"\n",
		"core_test.go":                         "package core_test\nimport _ \"" + bannedImport + "/modules\"\n",
		"fixture/go.mod":                       "module fixture\n",
		"fixture/pipeline.go":                  "package main\nimport _ \"" + bannedImport + "/modules\"\n",
		"app/test-results/project/pipeline.go": "package main\nimport _ \"" + bannedImport + "/modules\"\n",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	hits := importHits(t, root)
	if len(hits) != 1 || hits[0] != filepath.Join(root, "core.go") {
		t.Fatalf("production reverse import must remain rejected, got %v", hits)
	}
}
