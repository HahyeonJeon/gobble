package appservice

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestServiceHasNoEngineOrThirdPartyImports(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") {
			continue
		}
		source, err := parser.ParseFile(token.NewFileSet(), file.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range source.Imports {
			path, err := strconv.Unquote(item.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(strings.Split(path, "/")[0], ".") && path != "github.com/HahyeonJeon/gobble/internal/pipelinereview" && path != "github.com/HahyeonJeon/gobble/internal/preparation" {
				t.Errorf("%s imports %s; the native service must remain independent of engine/provider code", file.Name(), path)
			}
		}
	}
}

func TestJSONRejectsDuplicateAndNestedAmbiguity(t *testing.T) {
	for _, input := range []string{`{"schemaVersion":2,"schemaVersion":1}`, `{"schemaVersion":2,"SchemaVersion":1}`, `{"a":[{"b":1,"b":2}]}`, `{} {}`, strings.Repeat("[", 34) + strings.Repeat("]", 34)} {
		if err := validateJSON([]byte(input)); err == nil {
			t.Fatalf("accepted ambiguous JSON: %s", input)
		}
	}
	if err := validateJSON([]byte(`{"schemaVersion":1,"a":[true,null,{"b":"text"}]}`)); err != nil {
		t.Fatal(err)
	}
}
