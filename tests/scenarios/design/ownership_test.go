package design

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/HahyeonJeon/gobble"

func TestProductSourceDependencyDirection(t *testing.T) {
	root := moduleRoot(t)
	zones := []struct {
		directory    string
		rejectExact  []string
		rejectPrefix []string
	}{
		{
			directory: filepath.Join(root, "assets", "modules"),
			rejectExact: []string{
				modulePath + "/assets",
			},
			rejectPrefix: []string{
				modulePath + "/assets/modules/",
				modulePath + "/assets/pipelines/",
				modulePath + "/tests/",
			},
		},
		{
			directory: filepath.Join(root, "assets", "pipelines"),
			rejectExact: []string{
				modulePath + "/assets",
			},
			rejectPrefix: []string{
				modulePath + "/assets/pipelines/",
				modulePath + "/tests/",
			},
		},
	}
	for _, zone := range zones {
		checkImports(t, root, zone.directory, zone.rejectExact, zone.rejectPrefix)
	}
}

func checkImports(t *testing.T, root, directory string, rejectedExact, rejectedPrefixes []string) {
	t.Helper()
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			for _, rejected := range rejectedExact {
				if importPath == rejected {
					t.Errorf("%s imports reverse dependency %s", relative, importPath)
				}
			}
			for _, prefix := range rejectedPrefixes {
				if strings.HasPrefix(importPath, prefix) {
					t.Errorf("%s imports reverse dependency %s", relative, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s) error = %v", directory, err)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatalf("go.mod not found from %s", directory)
		}
		directory = parent
	}
}
