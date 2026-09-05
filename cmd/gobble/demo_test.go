package main

import (
	"bytes"
	"github.com/HahyeonJeon/gobble/internal/fixture"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDemoProjectsComposeExistingAssaysWithoutNetwork(t *testing.T) {
	source := moduleRoot(t)
	for _, demo := range demos {
		t.Run(demo.Name, func(t *testing.T) {
			files, manifest, err := demoFiles(source, demo)
			if err != nil {
				t.Fatal(err)
			}
			if len(manifest.Pins()) == 0 {
				t.Fatal("empty official fixture")
			}
			// Execute each generated entry point and validate its real product graph.
			// No tool boundary is replaced; this is preparation/plan evidence only.
			files["pipeline_test.go"] = `package pipeline
import("testing"; "github.com/HahyeonJeon/gobble")
func TestPlan(t *testing.T){g,e:=gobble.Compose(Pipeline());if e!=nil{t.Fatal(e)};if _,e=gobble.BuildPlan(g);e!=nil{t.Fatal(e)}}
`
			project := filepath.Join(t.TempDir(), "assay 한글")
			if err := createProject(t.Context(), project, source, files, nil); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "test", "-count=1", "./...")
			cmd.Dir = project
			if data, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("generated %s: %v\n%s", demo.Name, err, data)
			}
			if err := createProject(t.Context(), project, source, files, nil); err == nil {
				t.Fatal("existing project overwritten")
			}
		})
	}
}

func TestDemoRejectsInvalidNameAndExistingDirectoryBeforeDownloading(t *testing.T) {
	t.Setenv("GOBBLE_SOURCE", "missing")
	for _, args := range [][]string{{"demo", "../rnaseq", "new"}, {"demo", "rnaseq", t.TempDir()}} {
		var out, stderr bytes.Buffer
		if code := run(args, &out, &stderr); code == 0 {
			t.Fatal("invalid demo accepted")
		}
		if bytes.Contains(stderr.Bytes(), []byte("GOBBLE_SOURCE")) {
			t.Fatal("source accessed before request validation")
		}
	}
	var out, stderr bytes.Buffer
	if code := run([]string{"demo"}, &out, &stderr); code != 0 || !bytes.Contains(out.Bytes(), []byte("rnaseq")) {
		t.Fatalf("catalog: %s %s", out.String(), stderr.String())
	}
}

func TestWGSIntervalMembersMatchPinnedSource(t *testing.T) {
	workspace := t.TempDir()
	source := filepath.Join(workspace, "in", "reference", "genome.multi_intervals.bed")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("chr1\t1\t20\nchr1\t21\t40\n")
	if err := os.WriteFile(source, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fixture.SplitIntervals(workspace, "in/reference/genome.multi_intervals.bed", "in/reference/intervals", 2); err != nil {
		t.Fatal(err)
	}
	var combined []byte
	for _, name := range []string{"interval_001.bed", "interval_002.bed"} {
		b, err := os.ReadFile(filepath.Join(filepath.Dir(source), "intervals", name))
		if err != nil {
			t.Fatal(err)
		}
		combined = append(combined, b...)
	}
	if !bytes.Equal(data, combined) {
		t.Fatalf("intervals changed: %s", combined)
	}
}
