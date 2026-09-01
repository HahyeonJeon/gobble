package scrnaseqevidence_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	scrnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/scrnaseq"
)

type manifest struct {
	Schema    int `json:"schema"`
	Benchmark struct {
		Pipeline      string `json:"pipeline"`
		Release       string `json:"release"`
		Commit        string `json:"commit"`
		SelectedRoute string `json:"selected_route"`
		DatasetCommit string `json:"dataset_commit"`
	} `json:"benchmark"`
	Entries    []entry             `json:"entries"`
	StageTrace map[string][]string `json:"stage_trace"`
	Limits     []string            `json:"limits"`
	Images     []image             `json:"images"`
}

type entry struct {
	LogicalName       string   `json:"logical_name"`
	Name              string   `json:"name"`
	Role              string   `json:"role"`
	Repository        string   `json:"repository"`
	Commit            string   `json:"commit"`
	Path              string   `json:"path"`
	URL               string   `json:"url"`
	Bytes             int64    `json:"bytes"`
	SHA256            string   `json:"sha256"`
	License           string   `json:"license"`
	LicenseSource     string   `json:"license_source"`
	LicenseAuthority  string   `json:"license_authority"`
	Redistribution    string   `json:"redistribution"`
	Provenance        string   `json:"provenance"`
	AssayUse          []string `json:"assay_use"`
	BenchmarkRelation string   `json:"benchmark_relation"`
	Staged            bool     `json:"staged"`
	Destination       string   `json:"destination"`
}

type image struct {
	Module    string `json:"module"`
	TaskName  string `json:"task_name"`
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
	Tool      string `json:"tool"`
	Command   string `json:"command"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	License   string `json:"license"`
	Platform  string `json:"platform"`
}

func TestManifestIsExactFixtureAndCommandAuthority(t *testing.T) {
	m := loadManifest(t)
	if m.Schema != 4 || m.Benchmark.Pipeline != "nf-core/scrnaseq" || m.Benchmark.Release != "4.2.0" || !lowerHex(m.Benchmark.Commit, 40) || !lowerHex(m.Benchmark.DatasetCommit, 40) {
		t.Fatalf("benchmark = %+v, want immutable scrnaseq and dataset commits", m.Benchmark)
	}
	for _, excluded := range []string{"no CellBender", "Cell Ranger", "alternate aligner", "custom chemistry", "downstream single-cell analysis"} {
		if !strings.Contains(m.Benchmark.SelectedRoute, excluded) {
			t.Errorf("selected route omits boundary %q: %s", excluded, m.Benchmark.SelectedRoute)
		}
	}
	if len(m.Entries) != 17 {
		t.Fatalf("entries = %d, want 17 exact authorities", len(m.Entries))
	}
	staged := make(map[string]entry)
	seen := make(map[string]bool, len(m.Entries))
	for _, entry := range m.Entries {
		if seen[entry.LogicalName] || entry.LogicalName == "" || entry.Name == "" || entry.Role == "" || entry.Repository == "" || entry.Path == "" || entry.Commit == "" || !strings.Contains(entry.URL, entry.Commit) || entry.Bytes <= 0 || !lowerHex(entry.SHA256, 64) || entry.License == "" || entry.LicenseSource == "" || entry.Redistribution == "" || entry.Provenance == "" || len(entry.AssayUse) == 0 || entry.BenchmarkRelation == "" {
			t.Fatalf("invalid or substituted entry: %+v", entry)
		}
		seen[entry.LogicalName] = true
		if entry.Staged {
			staged[entry.Name] = entry
		}
	}
	if len(staged) != 9 {
		t.Fatalf("staged entries = %d, want 9 official reads/reference/whitelist bytes", len(staged))
	}
	for _, pin := range scrnaseqevidence.MustPins() {
		entry, ok := staged[pin.Name]
		if !ok || pin.URL != entry.URL || pin.Bytes != entry.Bytes || pin.SHA256 != entry.SHA256 {
			t.Errorf("typed staged pin %q differs from manifest authority", pin.Name)
		}
	}
	for _, sample := range loadSamples(t) {
		for _, run := range sample.Runs {
			for _, read := range []string{run.Fastq1, run.Fastq2} {
				if _, ok := staged[filepath.Base(read)]; !ok {
					t.Errorf("fixture read %q is absent from staged manifest authority", read)
				}
			}
		}
	}
	for _, stage := range []string{"run_consolidation_raw_qc", "reference_normalization", "simpleaf_index", "simpleaf_quantification", "qcatch", "raw_matrix_conversion", "combined_raw_h5ad", "multiqc"} {
		if len(m.StageTrace[stage]) == 0 {
			t.Errorf("stage trace omits %s", stage)
		}
	}
	limits := strings.Join(m.Limits, " ")
	for _, required := range []string{"never scientific thresholds", "barcode validity", "suitable filtering", "normalization", "integration", "annotation", "scientific interpretation"} {
		if !strings.Contains(limits, required) {
			t.Errorf("manifest limits omit %q: %s", required, limits)
		}
	}

	imagesByTask := make(map[string]image, len(m.Images))
	for _, authority := range m.Images {
		if authority.Module == "" || authority.TaskName == "" || authority.Reference == "" || strings.Contains(authority.Reference, "@") || !strings.HasPrefix(authority.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(authority.Digest, "sha256:"), 64) || authority.Tool == "" || authority.Command == "" || authority.Version == "" || authority.Source == "" || authority.License == "" || authority.Platform != "linux/amd64" || imagesByTask[authority.TaskName].Module != "" {
			t.Fatalf("invalid image authority: %+v", authority)
		}
		if authority.TaskName == "cat_fastq" {
			for _, provenance := range []string{"Gobble-owned", "2026-08-30", "nf-core/rnaseq 3.26.0", "not an nf-core/scrnaseq 4.2.0 module"} {
				if !strings.Contains(authority.Source, provenance) {
					t.Errorf("cat_fastq source omits %q: %s", provenance, authority.Source)
				}
			}
		}
		imagesByTask[authority.TaskName] = authority
	}
	config := scrnaseq.DefaultConfig()
	catAuthority := imagesByTask["cat_fastq"]
	wantCatImage := catAuthority.Reference + "@" + catAuthority.Digest
	if got := string(config.Consolidate.Image); got != wantCatImage {
		t.Errorf("Gobble-owned scRNA cat_fastq image = %q, want dated manifest tuple %q", got, wantCatImage)
	}
	raw := pc.MustPlanJSON(t, scrnaseq.Build(loadSamples(t), config))
	seenTasks := make(map[string]bool)
	for _, task := range pc.AllTasks(t, raw) {
		authority, ok := imagesByTask[task.Name]
		if !ok {
			t.Errorf("task %s command %q has no manifest image authority", task.ID, task.Name)
			continue
		}
		if want := authority.Reference + "@" + authority.Digest; task.Image != want {
			t.Errorf("task %s image = %q, want %q", task.ID, task.Image, want)
		}
		seenTasks[task.Name] = true
	}
	for taskName := range imagesByTask {
		if !seenTasks[taskName] {
			t.Errorf("image authority for %q has no product task", taskName)
		}
	}
}

func loadManifest(t *testing.T) manifest {
	t.Helper()
	data, err := os.ReadFile("testdata/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		t.Fatalf("decode strict manifest: %v", err)
	}
	return m
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
