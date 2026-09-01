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
	if m.Schema != 4 || m.Benchmark.Pipeline != "nf-core/scrnaseq" || m.Benchmark.Release != "4.2.0" || m.Benchmark.Commit != "3fc17b4f971a89e47c88337de71d0e777ffad8cc" || m.Benchmark.DatasetCommit != "d934d6e8367fe2626184496b1889671cf2b02dab" {
		t.Fatalf("benchmark = %+v, want exact scrnaseq and dataset commits", m.Benchmark)
	}
	for _, excluded := range []string{"no CellBender", "Cell Ranger", "alternate aligner", "custom chemistry", "downstream single-cell analysis"} {
		if !strings.Contains(m.Benchmark.SelectedRoute, excluded) {
			t.Errorf("selected route omits boundary %q: %s", excluded, m.Benchmark.SelectedRoute)
		}
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
	benchmarkReferences := map[string]string{
		"gtf_gene_filter":       "docker.io/biocontainers/python:3.9--1",
		"gffread_transcriptome": "docker.io/biocontainers/gffread:0.12.7--hd03093a_1",
		"gtf_to_t2g":            "docker.io/biocontainers/python:3.9--1",
	}
	for _, authority := range m.Images {
		if authority.Module == "" || authority.TaskName == "" || authority.Reference == "" || strings.Contains(authority.Reference, "@") || !strings.HasPrefix(authority.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(authority.Digest, "sha256:"), 64) || authority.Tool == "" || authority.Command == "" || authority.Version == "" || authority.Source == "" || authority.License == "" || authority.Platform != "linux/amd64" || imagesByTask[authority.TaskName].Module != "" {
			t.Fatalf("invalid image authority: %+v", authority)
		}
		if want := benchmarkReferences[authority.TaskName]; want != "" && authority.Reference != want {
			t.Errorf("%s benchmark image reference = %q, want exact nf-core/scrnaseq 4.2.0 reference %q", authority.TaskName, authority.Reference, want)
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

type byteIdentity struct {
	bytes  int64
	sha256 string
}

func expectedBytes() map[string]byteIdentity {
	return map[string]byteIdentity{
		"SC-FQ-X-R1":              {1727503, "a318e85709d690b25c763fb3f156fc98d0ce46360115f3de3d9b1b2ecc55a1f7"},
		"SC-FQ-X-R2":              {4259756, "1514b26ce8e972ac7ba1860d591fa72e8c1fb6f716825651c2eb24b59c1a38d2"},
		"SC-FQ-Y-L1-R1":           {1727503, "a318e85709d690b25c763fb3f156fc98d0ce46360115f3de3d9b1b2ecc55a1f7"},
		"SC-FQ-Y-L1-R2":           {4259756, "1514b26ce8e972ac7ba1860d591fa72e8c1fb6f716825651c2eb24b59c1a38d2"},
		"SC-FQ-Y-L2-R1":           {737653, "f33ae3d31e78020843b78198508b130506b2e7899eb2ecc658d2640b845a1114"},
		"SC-FQ-Y-L2-R2":           {1981608, "6270bd60af5341463fd425079cfc067990221dfe78437880240b5894fa30477f"},
		"SC-REF-FASTA":            {62455433, "b03ea6d17e5e02cd092dabb58358a30a70bfe639f19c71b296c1109ad0b0b931"},
		"SC-REF-GTF":              {20994920, "2270e0de93df1e12cdcb91cb9bd29640d318cd676d8e538c52a45bebef6c1247"},
		"SC-REF-WHITELIST":        {2238617, "4101687b6cbb947b8ace340c38eecf872a1a59f230eab23becacd038a46c6fb5"},
		"SC-DATASET-SHEET":        {854, "ca95a44b760f6f35e3bac66088433d3d6cdf6e413c291af2d620e51b10825396"},
		"SC-A2":                   {3120, "0e8ee703e9140206199317b8b50ed6c9de161e86224765c09e62ebfe6c999d5c"},
		"SC-A1":                   {1064, "1fa2a62f20d23902c7aad04f3d728fbfc9d153df472e8f2ef163322a8c5557c8"},
		"SC-PIPELINE-TEST-CONFIG": {1590, "bb46cb92576f8819573d57dae767fa119dce15d736f386f8e049cbdbd57b5934"},
		"SC-PIPELINE-WORKFLOW":    {18050, "0579d006efc478106ef15735985e9b3d6f92746e4e36ec19571403a2e50d3eac"},
		"SC-PIPELINE-SIMPLEAF":    {5259, "461196697e3a27c0379c95dc0e32327a695bd28a9d5fb24bd63b5d1cbc62b83b"},
		"SC-PIPELINE-PROTOCOLS":   {2545, "33d0653366c45a10ce23941a6a7af7ebd6863d400f919ee065b362bf00cdd422"},
		"SC-A3":                   {1077, "caf95a0a699bf82a6fe4b1c68095d3bdfdfe4b1b98997113aa6a692b3232a076"},
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
