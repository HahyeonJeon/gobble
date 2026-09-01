package atacseqevidence_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	atacseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/atacseq"
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
	if m.Schema != 4 || m.Benchmark.Pipeline != "nf-core/atacseq" || m.Benchmark.Release != "2.1.2" || !lowerHex(m.Benchmark.Commit, 40) || !lowerHex(m.Benchmark.DatasetCommit, 40) {
		t.Fatalf("benchmark = %+v, want immutable atacseq and dataset commits", m.Benchmark)
	}
	if !strings.Contains(m.Benchmark.SelectedRoute, "no alternate aligner") || !strings.Contains(m.Benchmark.SelectedRoute, "IDR") {
		t.Fatalf("selected route is not bounded: %s", m.Benchmark.SelectedRoute)
	}
	if len(m.Entries) != 17 {
		t.Fatalf("entries = %d, want 17 exact authorities", len(m.Entries))
	}
	staged := make(map[string]entry)
	seen := make(map[string]bool, len(m.Entries))
	for _, entry := range m.Entries {
		if seen[entry.LogicalName] || entry.LogicalName == "" || entry.Name == "" || entry.Role == "" || entry.Repository == "" || entry.Path == "" || entry.Commit == "" || !strings.Contains(entry.URL, entry.Commit) || entry.Bytes <= 0 || !lowerHex(entry.SHA256, 64) || entry.LicenseAuthority == "" || entry.Provenance == "" || len(entry.AssayUse) == 0 {
			t.Fatalf("invalid or substituted entry: %+v", entry)
		}
		seen[entry.LogicalName] = true
		if entry.Staged {
			staged[entry.Name] = entry
		}
	}
	if len(staged) != 10 {
		t.Fatalf("staged entries = %d, want 10 official FASTA/GTF/FASTQ bytes", len(staged))
	}
	for _, pin := range atacseqevidence.MustPins() {
		entry, ok := staged[pin.Name]
		if !ok || pin.URL != entry.URL || pin.Bytes != entry.Bytes || pin.SHA256 != entry.SHA256 {
			t.Errorf("typed staged pin %q differs from manifest staging authority", pin.Name)
		}
	}
	for _, sample := range loadSamples(t) {
		for _, replicate := range sample.Replicates {
			for _, run := range replicate.Runs {
				for _, read := range []string{run.Fastq1, run.Fastq2} {
					if _, ok := staged[filepath.Base(read)]; read != "" && !ok {
						t.Errorf("fixture read %q is absent from staged manifest authority", read)
					}
				}
			}
		}
	}
	for _, stage := range []string{"reference_preparation", "raw_qc_and_trimming", "bwa_alignment_and_run_merge", "duplicate_filter_alignment_qc_tracks", "macs2_annotation_peak_qc", "consensus_featurecounts_deseq2", "ataqv_replicate_aggregation", "multiqc_igv"} {
		if len(m.StageTrace[stage]) == 0 {
			t.Errorf("stage trace omits %s", stage)
		}
	}
	limits := strings.Join(m.Limits, " ")
	for _, required := range []string{"never scientific thresholds", "cohort QC only", "does not claim reproducible peaks", "appropriate controls"} {
		if !strings.Contains(limits, required) {
			t.Errorf("manifest limits omit %q: %s", required, limits)
		}
	}

	imagesByTask := make(map[string]image, len(m.Images))
	for _, authority := range m.Images {
		if authority.Module == "" || authority.TaskName == "" || authority.Reference == "" || strings.Contains(authority.Reference, "@") || !strings.HasPrefix(authority.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(authority.Digest, "sha256:"), 64) || authority.Tool == "" || authority.Command == "" || authority.Version == "" || authority.Source == "" || authority.License == "" || authority.Platform != "linux/amd64" || imagesByTask[authority.TaskName].Module != "" {
			t.Fatalf("invalid image authority: %+v", authority)
		}
		imagesByTask[authority.TaskName] = authority
	}
	raw := pc.MustPlanJSON(t, atacseq.Build(loadSamples(t), atacseq.DefaultConfig()))
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
