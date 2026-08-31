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
	if m.Schema != 4 || m.Benchmark.Pipeline != "nf-core/atacseq" || m.Benchmark.Release != "2.1.2" || m.Benchmark.Commit != "1a1dbe52ffbd82256c941a032b0e22abbd925b8a" || m.Benchmark.DatasetCommit != "cd022b097372b078a68d8afadb172ad7342fd91f" {
		t.Fatalf("benchmark = %+v, want exact atacseq and dataset commits", m.Benchmark)
	}
	if !strings.Contains(m.Benchmark.SelectedRoute, "no alternate aligner") || !strings.Contains(m.Benchmark.SelectedRoute, "IDR") {
		t.Fatalf("selected route is not bounded: %s", m.Benchmark.SelectedRoute)
	}
	expected := expectedBytes()
	if len(m.Entries) != len(expected) {
		t.Fatalf("entries = %d, want %d exact authorities", len(m.Entries), len(expected))
	}
	staged := make(map[string]entry)
	seen := make(map[string]bool, len(m.Entries))
	for _, entry := range m.Entries {
		want, ok := expected[entry.LogicalName]
		if !ok || seen[entry.LogicalName] || entry.Name == "" || entry.Role == "" || entry.Repository == "" || entry.Path == "" || entry.Commit == "" || !strings.Contains(entry.URL, entry.Commit) || entry.Bytes != want.bytes || entry.SHA256 != want.sha256 || !lowerHex(entry.SHA256, 64) || entry.LicenseAuthority == "" || entry.Provenance == "" || len(entry.AssayUse) == 0 {
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

type byteIdentity struct {
	bytes  int64
	sha256 string
}

func expectedBytes() map[string]byteIdentity {
	return map[string]byteIdentity{
		"AT-FQ-2153-R1":           {5272547, "e94bb94aa3524dff446f405b8979b95304085323a851a33b436fa6511bec9b58"},
		"AT-FQ-2153-R2":           {5254354, "a7c9455e792e095f5db0f2e27edf912b26bd28398976d8a3ac11ff4a463ada1f"},
		"AT-FQ-2154-R1":           {5253787, "977e96d46afff3af01527ae8db385c844c9501ed6d239390aa091265a1063159"},
		"AT-FQ-2154-R2":           {5315718, "95fc1034e236f790a820f5977feb42c0ed895f8fe538ffb3b1643197f6aa7534"},
		"AT-FQ-2157-R1":           {5382195, "2c08804f7e8f4abf57ed928da041e22f442c033d4e8993946b8d95137d969a60"},
		"AT-FQ-2157-R2":           {5380339, "d1f8df0027c48ff74b68245e3def9aa50c3c042498e8bd2f073ea12bba2d3d7b"},
		"AT-FQ-2158-R1":           {5350303, "557434ecf97ed576642689a98622410a7903b777d6b8e1cd613f717ec78d1687"},
		"AT-FQ-2158-R2":           {5401912, "32b00634ea4e3dfc64385d837addb5f981b473aee2a3a7a5f79f8a5bc228f4e9"},
		"AT-REF-FASTA":            {12359807, "c0b7305c230b550c3d8ccc692df52338afc7a297b43d965868c285b98aa64ae1"},
		"AT-REF-GTF":              {12037571, "3a1e64b8f290127562612b47d6014bc6e4c130399da3e06ad062b268fd6d08fb"},
		"AT-DATASET-SHEET":        {1371, "52a21d927287e0b39c40243cf9c2ce7134eef83ca1582d3c89b9eeb278e67218"},
		"AT-DATASET-DESIGN":       {874, "f8f38d25705527598e7d506ed28b10bf3caca7ecd2deeff046bde1faf8215e2d"},
		"A1":                      {1064, "1fa2a62f20d23902c7aad04f3d728fbfc9d153df472e8f2ef163322a8c5557c8"},
		"A2":                      {3552, "1ade3b4da0f9c7aebd505eaeb6847eb641fd187ec7a6f7e73a50b52143802e25"},
		"AT-PIPELINE-TEST-CONFIG": {1275, "2add69ec6ac85ac7f64c958be38cd1faee115410644846f80ece5b494b74764c"},
		"AT-PIPELINE-WORKFLOW":    {33000, "4c1ed05343026f339c02cf6efeef837dc8fdfc69481bb480af13c00b78c026e1"},
		"A3":                      {1098, "d44c17cdbb17478f7529066ea6838eda09654aa8e8f3822a7188a298c620d961"},
	}
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
