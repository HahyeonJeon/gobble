package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdGobbleImportBan(t *testing.T) {
	root := moduleRoot(t)
	cmdDir := filepath.Join(root, "cmd", "gobble")
	engineDir := filepath.Join(root, "internal", "engine")
	internalDir := filepath.Join(root, "internal")
	cmdImport := "github.com/HahyeonJeon/gobble/cmd"
	bannedInCmd := []string{
		"github.com/HahyeonJeon/gobble/internal/engine",
		"github.com/HahyeonJeon/gobble/assets",
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel := path
		if r, err := filepath.Rel(root, path); err == nil {
			rel = r
		}
		inCmd := inDir(path, cmdDir)
		inEngine := inDir(path, engineDir)
		inInternal := inDir(path, internalDir)
		name := f.Name.Name
		if !inInternal && (name == "cli" || name == "cli_test") {
			t.Errorf("%s is public package %s", rel, name)
		}
		for _, imp := range f.Imports {
			got := strings.Trim(imp.Path.Value, `"`)
			if inCmd {
				for _, ban := range bannedInCmd {
					if got == ban || strings.HasPrefix(got, ban+"/") {
						t.Errorf("%s imports %s", rel, got)
					}
				}
			}
			if inEngine && (got == cmdImport || strings.HasPrefix(got, cmdImport+"/")) {
				t.Errorf("%s imports %s", rel, got)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func inDir(path, dir string) bool {
	if path == dir {
		return true
	}
	prefix := dir + string(os.PathSeparator)
	return strings.HasPrefix(path, prefix)
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
