package methylseqevidence_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	methylseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/methylseq"
)

type methylManifest struct {
	Schema    int `json:"schema"`
	Benchmark struct {
		Pipeline      string `json:"pipeline"`
		Release       string `json:"release"`
		Commit        string `json:"commit"`
		SelectedRoute string `json:"selected_route"`
		TestConfig    string `json:"test_config"`
		DatasetCommit string `json:"dataset_commit"`
	} `json:"benchmark"`
	Entries []methylEntry `json:"entries"`
	Trees   []struct {
		Name    string       `json:"name"`
		Archive string       `json:"archive"`
		Members []treeMember `json:"members"`
	} `json:"trees"`
	Images []methylImage `json:"images"`
}

type methylEntry struct {
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

type treeMember struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type methylImage struct {
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

func TestManifestIsExactFixtureTreeAndImageAuthority(t *testing.T) {
	manifest := loadMethylManifest(t)
	if manifest.Schema != 4 || manifest.Benchmark.Pipeline != "nf-core/methylseq" || manifest.Benchmark.Release != "4.2.0" || !lowerHex(manifest.Benchmark.Commit, 40) || !lowerHex(manifest.Benchmark.DatasetCommit, 40) {
		t.Fatalf("benchmark = %+v, want immutable methylseq 4.2.0 authority", manifest.Benchmark)
	}
	if manifest.Benchmark.SelectedRoute == "" || !strings.Contains(manifest.Benchmark.TestConfig, manifest.Benchmark.Commit) {
		t.Fatalf("benchmark route/config = %+v", manifest.Benchmark)
	}
	entries := make(map[string]methylEntry, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if entry.LogicalName == "" || entries[entry.LogicalName].LogicalName != "" || entry.Role == "" || entry.Repository == "" || entry.Path == "" || !strings.Contains(entry.URL, entry.Commit) || entry.Bytes <= 0 || !lowerHex(entry.SHA256, 64) || entry.Provenance == "" || entry.License == "" || entry.LicenseSource == "" || entry.Redistribution == "" || entry.BenchmarkRelation == "" || len(entry.AssayUse) == 0 {
			t.Fatalf("invalid manifest entry: %+v", entry)
		}
		entries[entry.LogicalName] = entry
	}
	for _, pin := range methylseqevidence.MustPins() {
		entry := entries[pin.Name]
		if entry.URL != pin.URL || entry.Bytes != pin.Bytes || entry.SHA256 != pin.SHA256 {
			t.Fatalf("typed pin %q differs from manifest authority", pin.Name)
		}
	}
	if len(manifest.Trees) != 1 || manifest.Trees[0].Name != "BismarkIndex" || manifest.Trees[0].Archive != "Bowtie2_Index.tar.gz" || len(manifest.Trees[0].Members) != 15 {
		t.Fatalf("Tree authority = %+v, want one 15-member BismarkIndex", manifest.Trees)
	}
	seenTree := make(map[string]bool, 15)
	for _, member := range manifest.Trees[0].Members {
		if !strings.HasPrefix(member.Path, "BismarkIndex/") || seenTree[member.Path] || member.Bytes <= 0 || !lowerHex(member.SHA256, 64) {
			t.Fatalf("invalid Tree member: %+v", member)
		}
		seenTree[member.Path] = true
	}
	for _, required := range []string{"BismarkIndex/genome.fa", "BismarkIndex/Bisulfite_Genome/CT_conversion/BS_CT.1.bt2", "BismarkIndex/Bisulfite_Genome/GA_conversion/BS_GA.rev.2.bt2"} {
		if !seenTree[required] {
			t.Errorf("Tree authority omits %s", required)
		}
	}

	imageSet := make(map[string]bool)
	moduleSet := make(map[string]bool)
	for _, image := range manifest.Images {
		if len(image.Modules) == 0 || image.Reference == "" || strings.Contains(image.Reference, "@") || !strings.HasPrefix(image.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(image.Digest, "sha256:"), 64) || image.Tool == "" || image.Command == "" || image.Version == "" || image.BenchmarkPipeline != manifest.Benchmark.Pipeline || image.BenchmarkRelease != manifest.Benchmark.Release || !lowerHex(image.ModuleCommit, 40) || !strings.Contains(image.ModuleSource, image.ModuleCommit) || image.Provenance == "" || image.License == "" || image.LicenseSource == "" || image.Platform != "linux/amd64" {
			t.Fatalf("invalid image authority: %+v", image)
		}
		imageSet[image.Reference+"@"+image.Digest] = true
		for _, module := range image.Modules {
			moduleSet[module] = true
		}
	}
	for _, required := range []string{"cat-fastq", "fastqc", "trim-galore", "bismark-genome-preparation", "bismark-align", "bismark-deduplicate", "bismark-methylation-extractor", "bismark-report", "bismark-summary", "multiqc"} {
		if !moduleSet[required] {
			t.Errorf("image authority omits %s", required)
		}
	}
	for _, task := range pc.AllTasks(t, pc.MustPlanJSON(t, methylseq.Build(loadSamples(t), methylseq.DefaultConfig()))) {
		if task.Image != "" && !imageSet[task.Image] {
			t.Errorf("task %s image %q is absent from manifest", task.ID, task.Image)
		}
	}
}

func TestLocalizedSheetConsumesOnlyManifestBytes(t *testing.T) {
	manifest := loadMethylManifest(t)
	entries := make(map[string]bool, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		entries[entry.LogicalName] = true
	}
	for _, sample := range loadSamples(t) {
		for _, run := range sample.Runs {
			for _, fastq := range []string{run.Fastq1, run.Fastq2} {
				if fastq != "" && !entries[filepath.Base(fastq)] {
					t.Errorf("sheet input %q is absent from manifest", fastq)
				}
			}
		}
	}
}

func loadMethylManifest(t *testing.T) methylManifest {
	t.Helper()
	data, err := os.ReadFile("testdata/manifest.json")
	if err != nil {
		t.Fatalf("ReadFile manifest: %v", err)
	}
	var manifest methylManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("Unmarshal manifest: %v", err)
	}
	return manifest
}

func lowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
