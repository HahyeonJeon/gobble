package gatk4genomicsdbimport_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	gatk4genomicsdbimport "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genomicsdbimport"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestGenomicsDBImportRequiresTwoSamplesAndReturnsTree(t *testing.T) {
	p := gobble.NewPipeline("genomicsdb")
	variants := []gatk4genomicsdbimport.Variant{
		{GVCF: p.AddInput("a", spec("a", ".g.vcf.gz")), TBI: p.AddInput("a_tbi", spec("a", ".g.vcf.gz.tbi"))},
		{GVCF: p.AddInput("b", spec("b", ".g.vcf.gz")), TBI: p.AddInput("b_tbi", spec("b", ".g.vcf.gz.tbi"))},
	}
	intervals := p.AddInputGroup("intervals", gobble.Group{
		{Name: "interval_001", Spec: spec("interval_001", ".bed")},
		{Name: "interval_002", Spec: spec("interval_002", ".bed")},
	})
	ports, err := gatk4genomicsdbimport.Add(p.Scatter("intervals").From(intervals), variants, intervals, gatk4genomicsdbimport.Options{IntervalDir: gobble.Dir("in"), OutDir: gobble.Dir("work/genomicsdb")})
	if err != nil || ports.Database.IsZero() {
		t.Fatalf("Add() = (%#v, %v), want Tree port", ports, err)
	}
	task := cc.Task(t, p, "intervals.gatk4_genomicsdbimport")
	for _, want := range []string{"GenomicsDBImport", "--variant", "in/a.g.vcf.gz", "in/b.g.vcf.gz", "--intervals \"$interval\"", "'work/genomicsdb'/$stem"} {
		if !strings.Contains(task.Script, want) {
			t.Errorf("GenomicsDBImport script omits %q: %s", want, task.Script)
		}
	}
	pc.AssertTreeIO(t, task.Outputs, "database", "work/genomicsdb")

	p = gobble.NewPipeline("invalid")
	one := []gatk4genomicsdbimport.Variant{{GVCF: p.AddInput("a", spec("a", ".g.vcf.gz")), TBI: p.AddInput("a_tbi", spec("a", ".g.vcf.gz.tbi"))}}
	intervals = p.AddInputGroup("intervals", gobble.Group{{Name: "interval_001", Spec: spec("interval_001", ".bed")}})
	_, err = gatk4genomicsdbimport.Add(p.Scatter("intervals").From(intervals), one, intervals, gatk4genomicsdbimport.Options{IntervalDir: gobble.Dir("in")})
	if err == nil {
		t.Fatal("Add(one sample) error = nil, want cohort defect")
	}
}

func spec(base, ext string) gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: base, Ext: ext}
}

func TestGenomicsDBPreparesFreshNativeWorkspace(t *testing.T) {
	p := gatk4genomicsdbimport.Pipeline(
		[]gobble.PathSpec{spec("a", ".g.vcf.gz"), spec("b", ".g.vcf.gz")},
		[]gobble.PathSpec{spec("a", ".g.vcf.gz.tbi"), spec("b", ".g.vcf.gz.tbi")},
		spec("interval_001", ".bed"), gatk4genomicsdbimport.Options{},
	)
	task := cc.Task(t, p, "intervals.gatk4_genomicsdbimport")
	for _, existingContent := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty-tree-root", true: "existing-content"}[existingContent], func(t *testing.T) {
			dir := t.TempDir()
			workspace := filepath.Join(dir, "work/gatk4-genomicsdbimport/interval_001")
			for _, path := range []string{workspace, filepath.Join(dir, "in")} {
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "in/interval_001.bed"), []byte("chr1\t0\t10\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(workspace, "keep")
			if existingContent {
				if err := os.WriteFile(marker, []byte("existing"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			// Probe the real shell/filesystem boundary before native workspace
			// creation. The installed-assay CI separately runs actual GenomicsDB.
			probe := `gatk() { test ! -e "$workspace"; mkdir "$workspace"; printf created > "$workspace/probe"; }
`
			cmd := exec.Command("sh", "-c", probe+task.Script)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if existingContent {
				got, readErr := os.ReadFile(marker)
				if err == nil || readErr != nil || string(got) != "existing" {
					t.Fatalf("Existing workspace was not preserved: %q, %v, %v", got, err, readErr)
				}
				if _, err := os.Stat(filepath.Join(workspace, "probe")); !os.IsNotExist(err) {
					t.Fatal("Importer started with existing content")
				}
			} else if err != nil {
				t.Fatalf("Workspace preparation: %v\n%s", err, output)
			}
		})
	}
}

func TestGenomicsDBRejectsImplicitWorkspaceOverwrite(t *testing.T) {
	for _, arg := range []string{"--overwrite-existing-genomicsdb-workspace", "--overwrite=true"} {
		p := gatk4genomicsdbimport.Pipeline(
			[]gobble.PathSpec{spec("a", ".g.vcf.gz"), spec("b", ".g.vcf.gz")},
			[]gobble.PathSpec{spec("a", ".g.vcf.gz.tbi"), spec("b", ".g.vcf.gz.tbi")},
			spec("interval_001", ".bed"), gatk4genomicsdbimport.Options{Options: modules.Options{ExtraArgs: []string{arg}}},
		)
		cc.Invalid(t, p)
	}
}
