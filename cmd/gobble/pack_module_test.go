package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPackedMonitorBuildPreservesConsumerModule(t *testing.T) {
	// This consumer intentionally has no go.sum. Its dependencies were already
	// verified when preparing Gobble's cache; do not require a second network
	// lookup against the checksum database to test module-file preservation.
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(cwd, "../.."))
	consumer := t.TempDir()
	relativeRoot, err := filepath.Rel(consumer, root)
	if err != nil {
		t.Fatal(err)
	}
	mod := "module example.test/monitorconsumer\n\ngo 1.26\n\nrequire github.com/HahyeonJeon/gobble v0.0.0\nreplace github.com/HahyeonJeon/gobble => " + strconv.Quote(relativeRoot) + "\n"
	if err := os.WriteFile(filepath.Join(consumer, "go.mod"), []byte(mod), 0600); err != nil {
		t.Fatal(err)
	}
	source := "package pipeline\nimport \"github.com/HahyeonJeon/gobble\"\nfunc Pipeline()*gobble.Pipeline{return gobble.NewPipeline(\"monitor-test\")}\n"
	if err := os.WriteFile(filepath.Join(consumer, "pipeline.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := buildPackedInner(goBin, consumer, t.TempDir(), "example.test/monitorconsumer", installIdentityResult{}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(consumer, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != mod {
		t.Fatal("pack changed consumer go.mod")
	}
	if _, err := os.Stat(filepath.Join(consumer, "go.sum")); !os.IsNotExist(err) {
		t.Fatal("pack added consumer dependency lock unexpectedly")
	}
}
