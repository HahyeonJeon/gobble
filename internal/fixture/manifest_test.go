package fixture

import (
	"strings"
	"testing"
)

func TestManifestRejectsMutableOrOwnerlessAuthority(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{name: "mutable byte URL", mutate: func(m *Manifest) { m.Entries[0].URL = "https://example.invalid/main/reads.fastq.gz" }},
		{name: "mutable image", mutate: func(m *Manifest) { m.Images[0].Reference = "registry.invalid/tool:latest" }},
		{name: "missing stage destination", mutate: func(m *Manifest) { m.Entries[0].Destination = "" }},
		{name: "missing license", mutate: func(m *Manifest) { m.Entries[0].License = "" }},
		{name: "missing license source", mutate: func(m *Manifest) { m.Entries[0].LicenseSource = "" }},
		{name: "missing redistribution", mutate: func(m *Manifest) { m.Entries[0].Redistribution = "" }},
		{name: "missing benchmark relation", mutate: func(m *Manifest) { m.Entries[0].BenchmarkRelation = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validManifest()
			test.mutate(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestManifestPinsReturnCopiedAuthority(t *testing.T) {
	manifest := validManifest()
	pins := manifest.Pins()
	inputs := manifest.StagedInputs()
	pins[0].Name = "changed"
	inputs[0].Destination = "changed"
	if manifest.Entries[0].Name != "reads.fastq.gz" || manifest.Entries[0].Destination != "in/reads/reads.fastq.gz" {
		t.Fatalf("manifest retained caller mutation: %+v", manifest.Entries[0])
	}
}

func validManifest() Manifest {
	commit := strings.Repeat("a", 40)
	return Manifest{
		Schema: 4,
		Benchmark: ManifestBenchmark{
			Pipeline:      "nf-core/example",
			Release:       "1.2.3",
			Commit:        commit,
			SelectedRoute: "selected path",
			DatasetCommit: strings.Repeat("b", 40),
		},
		Entries: []ManifestEntry{{
			LogicalName:       "READS",
			Name:              "reads.fastq.gz",
			Role:              "read",
			Repository:        "nf-core/test-datasets",
			Commit:            commit,
			Path:              "reads.fastq.gz",
			URL:               "https://example.invalid/" + commit + "/reads.fastq.gz",
			Bytes:             1,
			SHA256:            strings.Repeat("c", 64),
			Provenance:        "engineering fixture",
			License:           "MIT",
			LicenseSource:     "https://example.invalid/" + commit + "/LICENSE",
			LicenseAuthority:  "LICENSE",
			Redistribution:    "preserve attribution",
			AssayUse:          []string{"input"},
			BenchmarkRelation: "direct benchmark input",
			Staged:            true,
			Destination:       "in/reads/reads.fastq.gz",
		}},
		StageTrace: map[string][]string{"input": {"READS"}},
		Limits:     []string{"engineering evidence only"},
		Images: []ManifestImage{{
			Module:    "example",
			TaskName:  "example",
			Reference: "registry.invalid/tool:1.2.3",
			Digest:    "sha256:" + strings.Repeat("d", 64),
			Tool:      "example",
			Command:   "example",
			Version:   "1.2.3",
			Source:    "immutable source",
			License:   "MIT",
			Platform:  "linux/amd64",
		}},
	}
}
