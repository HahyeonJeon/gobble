package moduleevidence

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestModulePackagesAreSelectedPipelineDependencies(t *testing.T) {
	root := reachabilityModuleRoot(t)
	modules := goList(t, root, "./assets/modules/...")
	dependencies := goList(t, root, "-deps",
		"./assets/pipelines/wgs",
		"./assets/pipelines/rnaseq",
		"./assets/pipelines/methylseq",
		"./assets/pipelines/atacseq",
		"./assets/pipelines/scrnaseq",
	)

	reachable := make(map[string]bool, len(dependencies))
	for _, dependency := range dependencies {
		reachable[dependency] = true
	}

	var unreachable []string
	for _, module := range modules {
		if !reachable[module] {
			unreachable = append(unreachable, module)
		}
	}
	if len(unreachable) != 0 {
		sort.Strings(unreachable)
		t.Fatalf("unreachable module packages =\n%s\nwant every module package in a selected pipeline dependency graph", strings.Join(unreachable, "\n"))
	}
}

func goList(t *testing.T, root string, args ...string) []string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(cmd.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.Fields(string(out))
}

func reachabilityModuleRoot(t *testing.T) string {
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
