package pipelineevidence_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	"github.com/HahyeonJeon/gobble/internal/fixture"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	atacseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/atacseq"
	methylseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/methylseq"
	rnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/rnaseq"
	scrnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/scrnaseq"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

func TestFiveManifestBenchmarkPinAndDefaultImageConsistency(t *testing.T) {
	root := familyModuleRoot(t)
	products := []struct {
		name             string
		benchmarkRelease string
		cacheDir         string
		manifestPath     string
		fixtureSheet     string
		manifest         func() (fixture.Manifest, error)
		plan             func(*testing.T, string) []byte
	}{
		{name: "wgs", benchmarkRelease: wgs.BenchmarkRelease, cacheDir: wgsevidence.CacheDir, manifestPath: wgsevidence.ManifestPath, fixtureSheet: wgsevidence.FixtureSheet, manifest: wgsevidence.Manifest, plan: wgsFamilyPlan},
		{name: "rnaseq", benchmarkRelease: rnaseq.BenchmarkRelease, cacheDir: rnaseqevidence.CacheDir, manifestPath: rnaseqevidence.ManifestPath, fixtureSheet: rnaseqevidence.FixtureSheet, manifest: rnaseqevidence.Manifest, plan: rnaseqFamilyPlan},
		{name: "methylseq", benchmarkRelease: methylseq.BenchmarkRelease, cacheDir: methylseqevidence.CacheDir, manifestPath: methylseqevidence.ManifestPath, fixtureSheet: methylseqevidence.FixtureSheet, manifest: methylseqevidence.Manifest, plan: methylseqFamilyPlan},
		{name: "atacseq", benchmarkRelease: atacseq.BenchmarkRelease, cacheDir: atacseqevidence.CacheDir, manifestPath: atacseqevidence.ManifestPath, fixtureSheet: atacseqevidence.FixtureSheet, manifest: atacseqevidence.Manifest, plan: atacseqFamilyPlan},
		{name: "scrnaseq", benchmarkRelease: scrnaseq.BenchmarkRelease, cacheDir: scrnaseqevidence.CacheDir, manifestPath: scrnaseqevidence.ManifestPath, fixtureSheet: scrnaseqevidence.FixtureSheet, manifest: scrnaseqevidence.Manifest, plan: scrnaseqFamilyPlan},
	}
	seenCaches := make(map[string]bool, len(products))
	for _, product := range products {
		t.Run(product.name, func(t *testing.T) {
			manifest, err := product.manifest()
			if err != nil {
				t.Fatalf("Manifest() error = %v", err)
			}
			if got := manifest.Benchmark.Pipeline + " " + manifest.Benchmark.Release; got != product.benchmarkRelease {
				t.Fatalf("manifest benchmark = %q, product benchmark = %q", got, product.benchmarkRelease)
			}
			wantOwner := filepath.ToSlash(filepath.Join("tests", "pipelines", product.name)) + "/"
			for _, path := range []string{product.cacheDir, product.manifestPath, product.fixtureSheet} {
				if !strings.HasPrefix(path, wantOwner) {
					t.Errorf("owner path %q is outside %q", path, wantOwner)
				}
			}
			if seenCaches[product.cacheDir] {
				t.Errorf("duplicate cache authority %q", product.cacheDir)
			}
			seenCaches[product.cacheDir] = true

			images := make(map[string]bool, len(manifest.Images))
			for _, image := range manifest.Images {
				images[image.Reference+"@"+image.Digest] = true
			}
			usedImages := make(map[string]bool, len(images))
			for _, task := range pc.AllTasks(t, product.plan(t, root)) {
				if !images[task.Image] {
					t.Errorf("task %s image %q has no manifest authority", task.ID, task.Image)
				}
				usedImages[task.Image] = true
				for _, token := range append(append([]string(nil), task.Command...), task.Script) {
					if strings.Contains(token, "http://") || strings.Contains(token, "https://") {
						t.Errorf("task %s contains hidden network location %q", task.ID, token)
					}
				}
			}
			for image := range images {
				if !usedImages[image] {
					t.Errorf("manifest default image %q has no selected product task", image)
				}
			}
		})
	}
}

func wgsFamilyPlan(t *testing.T, root string) []byte {
	samples, err := wgs.Load(filepath.Join(root, filepath.FromSlash(wgsevidence.FixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	return pc.MustPlanJSON(t, wgs.Build(samples, wgs.DefaultConfig()))
}

func rnaseqFamilyPlan(t *testing.T, root string) []byte {
	samples, err := rnaseq.Load(filepath.Join(root, filepath.FromSlash(rnaseqevidence.FixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	return pc.MustPlanJSON(t, rnaseq.Build(samples, rnaseq.DefaultConfig()))
}

func methylseqFamilyPlan(t *testing.T, root string) []byte {
	samples, err := methylseq.Load(filepath.Join(root, filepath.FromSlash(methylseqevidence.FixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	return pc.MustPlanJSON(t, methylseq.Build(samples, methylseq.DefaultConfig()))
}

func atacseqFamilyPlan(t *testing.T, root string) []byte {
	samples, err := atacseq.Load(filepath.Join(root, filepath.FromSlash(atacseqevidence.FixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	return pc.MustPlanJSON(t, atacseq.Build(samples, atacseq.DefaultConfig()))
}

func scrnaseqFamilyPlan(t *testing.T, root string) []byte {
	samples, err := scrnaseq.Load(filepath.Join(root, filepath.FromSlash(scrnaseqevidence.FixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	return pc.MustPlanJSON(t, scrnaseq.Build(samples, scrnaseq.DefaultConfig()))
}

func familyModuleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
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
