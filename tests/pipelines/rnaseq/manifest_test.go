package rnaseqevidence_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	rnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/rnaseq"
)

type fixtureManifest struct {
	Schema    int `json:"schema"`
	Benchmark struct {
		Pipeline      string `json:"pipeline"`
		Release       string `json:"release"`
		Commit        string `json:"commit"`
		SelectedRoute string `json:"selected_route"`
		TestConfig    string `json:"test_config"`
		DatasetCommit string `json:"dataset_commit"`
	} `json:"benchmark"`
	Entries []fixtureEntry `json:"entries"`
	Images  []imageEntry   `json:"images"`
}

type fixtureEntry struct {
	LogicalName       string   `json:"logical_name"`
	Role              string   `json:"role"`
	Repository        string   `json:"repository"`
	Commit            string   `json:"commit"`
	Path              string   `json:"path"`
	URL               string   `json:"url"`
	Bytes             int64    `json:"bytes"`
	SHA256            string   `json:"sha256"`
	Provenance        string   `json:"provenance"`
	License           string   `json:"license"`
	LicenseSource     string   `json:"license_source"`
	Redistribution    string   `json:"redistribution"`
	AssayUse          []string `json:"assay_use"`
	BenchmarkRelation string   `json:"benchmark_relation"`
}

type imageEntry struct {
	Modules           []string `json:"modules"`
	Reference         string   `json:"reference"`
	Digest            string   `json:"digest"`
	Tool              string   `json:"tool"`
	Command           string   `json:"command"`
	Version           string   `json:"version"`
	BenchmarkPipeline string   `json:"benchmark_pipeline"`
	BenchmarkRelease  string   `json:"benchmark_release"`
	ModuleCommit      string   `json:"module_commit"`
	ModuleSource      string   `json:"module_source"`
	Provenance        string   `json:"provenance"`
	License           string   `json:"license"`
	LicenseSource     string   `json:"license_source"`
	Platform          string   `json:"platform"`
}

func TestManifestIsExactFixtureAndImageAuthority(t *testing.T) {
	manifest := loadManifest(t)
	if manifest.Schema != 4 || manifest.Benchmark.Pipeline != "nf-core/rnaseq" || manifest.Benchmark.Release != "3.26.0" {
		t.Fatalf("benchmark = %+v, want nf-core/rnaseq 3.26.0 schema 4", manifest.Benchmark)
	}
	if !lowerHex(manifest.Benchmark.Commit, 40) || !lowerHex(manifest.Benchmark.DatasetCommit, 40) {
		t.Fatalf("benchmark commits = %+v, want immutable pipeline and dataset commits", manifest.Benchmark)
	}
	if manifest.Benchmark.SelectedRoute == "" || !strings.Contains(manifest.Benchmark.TestConfig, manifest.Benchmark.Commit) {
		t.Fatalf("benchmark route/config = %+v, want selected route and commit URL", manifest.Benchmark)
	}
	if got, want := len(manifest.Entries), 14; got != want {
		t.Fatalf("manifest entry count = %d, want %d", got, want)
	}
	seen := make(map[string]bool, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if entry.LogicalName == "" || seen[entry.LogicalName] {
			t.Fatalf("invalid or duplicate logical name %q", entry.LogicalName)
		}
		seen[entry.LogicalName] = true
		if entry.Role == "" || entry.Repository == "" || entry.Path == "" || entry.Provenance == "" || entry.License == "" || entry.LicenseSource == "" || entry.Redistribution == "" || entry.BenchmarkRelation == "" || len(entry.AssayUse) == 0 {
			t.Fatalf("entry %q omits required authority: %+v", entry.LogicalName, entry)
		}
		if entry.Commit != manifest.Benchmark.DatasetCommit || !strings.Contains(entry.URL, entry.Commit) || entry.Bytes <= 0 || !lowerHex(entry.SHA256, 64) {
			t.Fatalf("entry %q has mutable or invalid byte identity: %+v", entry.LogicalName, entry)
		}
	}
	for _, pin := range rnaseqevidence.MustPins() {
		entry, ok := entryNamed(manifest.Entries, pin.Name)
		if !ok || entry.URL != pin.URL || entry.Bytes != pin.Bytes || entry.SHA256 != pin.SHA256 {
			t.Fatalf("typed pin %q differs from manifest authority", pin.Name)
		}
	}
	if len(manifest.Images) < 10 {
		t.Fatalf("image authority count = %d, want all selected command images", len(manifest.Images))
	}
	modules := make(map[string]bool)
	images := make(map[string]bool)
	for _, image := range manifest.Images {
		if len(image.Modules) == 0 || image.Reference == "" || strings.Contains(image.Reference, "@") || !strings.HasPrefix(image.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(image.Digest, "sha256:"), 64) || image.Tool == "" || image.Command == "" || image.Version == "" || image.BenchmarkPipeline != manifest.Benchmark.Pipeline || image.BenchmarkRelease != manifest.Benchmark.Release || !lowerHex(image.ModuleCommit, 40) || !strings.Contains(image.ModuleSource, image.ModuleCommit) || image.Provenance == "" || image.License == "" || image.LicenseSource == "" || image.Platform != "linux/amd64" {
			t.Fatalf("invalid image authority: %+v", image)
		}
		for _, module := range image.Modules {
			modules[module] = true
		}
		images[image.Reference+"@"+image.Digest] = true
	}
	for _, required := range []string{"gtf-filter", "sample-retention-trimmed", "sample-retention-mapped", "star-align", "salmon-quant", "tximport", "deseq2-qc", "multiqc", "featurecounts-biotype-qc"} {
		if !modules[required] {
			t.Errorf("image authority omits %s", required)
		}
	}
	deseq2Image, ok := imageForModule(manifest.Images, "deseq2-qc")
	if !ok || !strings.Contains(deseq2Image.Tool, "DESeq2") {
		t.Fatalf("DESeq2-QC image authority = %+v, %v, want named DESeq2 runtime", deseq2Image, ok)
	}
	tximportImage, ok := imageForModule(manifest.Images, "tximport")
	if !ok || strings.Contains(tximportImage.Tool, "DESeq2") || tximportImage.Reference == deseq2Image.Reference {
		t.Fatalf("tximport image authority = %+v, want truthful non-DESeq2 runtime distinct from DESeq2-QC", tximportImage)
	}
	for _, task := range pc.AllTasks(t, pc.MustPlanJSON(t, rnaseq.Build(loadSamples(t), rnaseq.DefaultConfig()))) {
		if task.Image != "" && !images[task.Image] {
			t.Errorf("task %s image %q is absent from manifest authority", task.ID, task.Image)
		}
	}
}

func TestLocalizedSheetConsumesOnlyManifestBytes(t *testing.T) {
	manifest := loadManifest(t)
	samples, err := rnaseq.Load(rnaFixtureSheet)
	if err != nil {
		t.Fatalf("Load fixture: %v", err)
	}
	for _, sample := range samples {
		for _, run := range sample.Runs {
			for _, fastq := range []string{run.Fastq1, run.Fastq2} {
				if fastq == "" {
					continue
				}
				if _, ok := entryNamed(manifest.Entries, filepath.Base(fastq)); !ok {
					t.Errorf("sheet input %q is absent from manifest", fastq)
				}
			}
		}
	}
	for _, source := range []string{"../../../assets/pipelines/rnaseq/build.go", "../../../assets/pipelines/rnaseq/config.go"} {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", source, err)
		}
		if strings.Contains(string(data), "http://") || strings.Contains(string(data), "https://") {
			t.Errorf("product Build/default source %s contains a network location", source)
		}
	}
}

func loadManifest(t *testing.T) fixtureManifest {
	t.Helper()
	data, err := os.ReadFile("testdata/manifest.json")
	if err != nil {
		t.Fatalf("ReadFile manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("Unmarshal manifest: %v", err)
	}
	return manifest
}

func entryNamed(entries []fixtureEntry, name string) (fixtureEntry, bool) {
	for _, entry := range entries {
		if entry.LogicalName == name {
			return entry, true
		}
	}
	return fixtureEntry{}, false
}

func imageForModule(images []imageEntry, name string) (imageEntry, bool) {
	for _, image := range images {
		for _, module := range image.Modules {
			if module == name {
				return image, true
			}
		}
	}
	return imageEntry{}, false
}

func lowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			if r < 'a' || r > 'f' {
				return false
			}
		}
	}
	return true
}
