package pipelinereview

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestPortableReviewCannotImportRuntimeOrProvider(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		tree, err := parser.ParseFile(token.NewFileSet(), f.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range tree.Imports {
			p, err := strconv.Unquote(v.Path.Value)
			if err != nil || strings.Contains(strings.Split(p, "/")[0], ".") {
				t.Fatalf("portable review imported runtime dependency: %s", v.Path.Value)
			}
		}
	}
}
func TestComparisonOwnsValuesAndRejectsUnknownBehavior(t *testing.T) {
	before, after := 25, 30
	base := Definition{SchemaVersion: 1, Context: Digest("inputs"), Steps: []Step{{ID: "trim", Recipe: "trim-galore-v1", Fingerprint: Digest(25), Residual: Digest("other fields"), Settings: []Setting{{"quality", "Quality threshold", "Phred", &before}}}}, Edges: []Edge{}}
	next := base
	next.Steps = []Step{base.Steps[0]}
	next.Steps[0].Fingerprint = Digest(30)
	next.Steps[0].Settings = []Setting{{"quality", "Quality threshold", "Phred", &after}}
	if err := Validate(base); err != nil {
		t.Fatal(err)
	}
	out := Compare(base, next)
	if len(out.Gaps) > 0 || len(out.Changes) != 1 {
		t.Fatal(out)
	}
	before = 1
	after = 2
	if *out.Changes[0].Before != 25 || *out.Changes[0].After != 30 {
		t.Fatal("comparison aliases source values")
	}
	next.Steps[0].Residual = Digest("changed environment")
	if len(Compare(base, next).Gaps) == 0 {
		t.Fatal("hidden behavior accepted")
	}
	next.Steps[0].Fingerprint = "invalid"
	if Validate(next) == nil {
		t.Fatal("invalid digest accepted")
	}
}
