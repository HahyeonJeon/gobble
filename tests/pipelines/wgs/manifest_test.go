package wgsevidence_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

type wgsManifest struct {
	Schema    int `json:"schema"`
	Benchmark struct {
		Pipeline          string            `json:"pipeline"`
		Release           string            `json:"release"`
		Commit            string            `json:"commit"`
		SelectedRoute     string            `json:"selected_route"`
		DatasetCommit     string            `json:"dataset_commit"`
		CoverageScenarios map[string]string `json:"coverage_scenarios"`
	} `json:"benchmark"`
	Entries    []wgsEntry          `json:"entries"`
	StageTrace map[string][]string `json:"stage_trace"`
	Limits     []string            `json:"limits"`
	Images     []wgsImage          `json:"images"`
}

type wgsEntry struct {
	LogicalName      string   `json:"logical_name"`
	Name             string   `json:"name"`
	Role             string   `json:"role"`
	Repository       string   `json:"repository"`
	Commit           string   `json:"commit"`
	Path             string   `json:"path"`
	URL              string   `json:"url"`
	Bytes            int64    `json:"bytes"`
	SHA256           string   `json:"sha256"`
	LicenseAuthority string   `json:"license_authority"`
	Provenance       string   `json:"provenance"`
	AssayUse         []string `json:"assay_use"`
	Staged           bool     `json:"staged"`
	Destination      string   `json:"destination"`
}

type wgsImage struct {
	Module       string `json:"module"`
	TaskName     string `json:"task_name"`
	Reference    string `json:"reference"`
	Digest       string `json:"digest"`
	Tool         string `json:"tool"`
	Command      string `json:"command"`
	Version      string `json:"version"`
	ModuleCommit string `json:"module_commit"`
	ModuleSource string `json:"module_source"`
	License      string `json:"license"`
	Platform     string `json:"platform"`
}

func TestManifestIsExactPlanningByteAndImageAuthority(t *testing.T) {
	manifest := loadManifest(t)
	if manifest.Schema != 4 || manifest.Benchmark.Pipeline != "nf-core/sarek" || manifest.Benchmark.Release != "3.10.0" || !lowerHex(manifest.Benchmark.Commit, 40) || !lowerHex(manifest.Benchmark.DatasetCommit, 40) {
		t.Fatalf("benchmark = %+v, want immutable Sarek and dataset authority", manifest.Benchmark)
	}
	if !strings.Contains(manifest.Benchmark.SelectedRoute, "without VQSR") || manifest.Benchmark.CoverageScenarios["F"] == "" || manifest.Benchmark.CoverageScenarios["J"] == "" {
		t.Fatalf("benchmark route/scenarios = %+v", manifest.Benchmark)
	}
	if len(manifest.Entries) != 24 {
		t.Fatalf("manifest entries = %d, want 24 Planning-bound bytes", len(manifest.Entries))
	}
	staged := make(map[string]wgsEntry)
	seen := make(map[string]bool, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if seen[entry.LogicalName] || entry.LogicalName == "" || entry.Name == "" || entry.Role == "" || entry.Repository == "" || entry.Path == "" || !strings.Contains(entry.URL, entry.Commit) || entry.Bytes <= 0 || !lowerHex(entry.SHA256, 64) || entry.LicenseAuthority == "" || entry.Provenance == "" || len(entry.AssayUse) == 0 {
			t.Fatalf("invalid or substituted manifest entry: %+v", entry)
		}
		seen[entry.LogicalName] = true
		if entry.Staged {
			staged[entry.Name] = entry
		}
	}
	if len(staged) != 12 {
		t.Fatalf("staged entries = %d, want 12 default-product bytes", len(staged))
	}
	for _, pin := range wgsevidence.MustPins() {
		entry, ok := staged[pin.Name]
		if !ok || pin.URL != entry.URL || pin.Bytes != entry.Bytes || pin.SHA256 != entry.SHA256 {
			t.Fatalf("typed pin %q differs from manifest authority", pin.Name)
		}
	}

	for _, stage := range []string{"reference_readiness", "raw_read_qc", "read_preparation", "alignment", "alignment_normalization", "duplicate_marking", "bqsr", "alignment_qc", "haplotypecaller", "sample_gvcf_gather", "genomicsdb", "joint_genotype", "joint_gather_and_qc"} {
		if len(manifest.StageTrace[stage]) == 0 {
			t.Errorf("stage trace omits %s", stage)
		}
	}

	imagesByTask := make(map[string]wgsImage, len(manifest.Images))
	for _, image := range manifest.Images {
		if image.Module == "" || image.TaskName == "" || imagesByTask[image.TaskName].Module != "" || strings.Contains(image.Reference, "@") || !strings.HasPrefix(image.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(image.Digest, "sha256:"), 64) || !lowerHex(image.ModuleCommit, 40) || !strings.Contains(image.ModuleSource, image.ModuleCommit) || image.Tool == "" || image.Command == "" || image.Version == "" || image.License == "" || image.Platform != "linux/amd64" {
			t.Fatalf("invalid image authority: %+v", image)
		}
		imagesByTask[image.TaskName] = image
	}
	seenTasks := make(map[string]bool, len(imagesByTask))
	for _, task := range pc.AllTasks(t, pc.MustPlanJSON(t, wgs.Build(loadSamples(t), wgs.DefaultConfig()))) {
		authority, ok := imagesByTask[task.Name]
		if !ok {
			t.Errorf("task %s command %q has no one-to-one manifest authority", task.ID, task.Name)
			continue
		}
		if want := authority.Reference + "@" + authority.Digest; task.Image != want {
			t.Errorf("task %s image = %q, want its exact module authority %q", task.ID, task.Image, want)
		}
		seenTasks[task.Name] = true
	}
	for taskName := range imagesByTask {
		if !seenTasks[taskName] {
			t.Errorf("manifest authority for command %q has no WGS task", taskName)
		}
	}
}

func TestFixtureSheetUsesOnlyManifestFASTQBytes(t *testing.T) {
	staged := make(map[string]bool)
	for _, pin := range wgsevidence.MustPins() {
		staged[pin.Name] = true
	}
	for _, sample := range loadSamples(t) {
		for _, lane := range sample.Lanes {
			for _, fastq := range []string{lane.Fastq1, lane.Fastq2} {
				if !staged[filepath.Base(fastq)] {
					t.Errorf("samplesheet FASTQ %q is absent from staged manifest bytes", fastq)
				}
			}
		}
	}
}

func loadManifest(t *testing.T) wgsManifest {
	t.Helper()
	data, err := os.ReadFile("testdata/manifest.json")
	if err != nil {
		t.Fatalf("ReadFile manifest: %v", err)
	}
	var manifest wgsManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode strict manifest: %v", err)
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
