// Package sourcecheck provides narrow source-ownership assertions.
package sourcecheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// AssertNoCall fails when path contains a call with any supplied name.
func AssertNoCall(t *testing.T, path string, names ...string) {
	t.Helper()
	calls := callNames(t, path)
	for _, name := range names {
		if calls[name] {
			t.Fatalf("%s calls %s, want no raw %s", path, name, name)
		}
	}
}

// AssertCalls fails when path omits a call with any supplied name.
func AssertCalls(t *testing.T, path string, names ...string) {
	t.Helper()
	calls := callNames(t, path)
	for _, name := range names {
		if !calls[name] {
			t.Fatalf("%s missing call %s", path, name)
		}
	}
}

func callNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s) error = %v", path, err)
	}
	calls := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch function := call.Fun.(type) {
		case *ast.Ident:
			calls[function.Name] = true
		case *ast.SelectorExpr:
			calls[function.Sel.Name] = true
		}
		return true
	})
	return calls
}
