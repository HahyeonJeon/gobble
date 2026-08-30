package pipelineevidence

import (
	"fmt"
	"strings"
)

// Benchmark records the exact upstream main path selected for one product.
// Sources identify the versioned config, workflow, or test that establishes
// the selected route.
type Benchmark struct {
	Pipeline      string
	Release       string
	SelectedRoute string
	Sources       []string
}

// Upstream identifies one remote file at an immutable repository commit.
type Upstream struct {
	Repository string
	Commit     string
	Path       string
	URL        string
}

// ManifestEntry records one directly or transitively consumed fixture byte.
// ArchiveMember is empty for a downloaded file and names the member when an
// extracted member becomes its own bound input.
type ManifestEntry struct {
	LogicalName       string
	Role              string
	Source            Upstream
	ArchiveMember     string
	ByteCount         int64
	SHA256            string
	Provenance        string
	License           string
	LicenseSource     string
	Redistribution    string
	AssayUse          []string
	BenchmarkRelation string
}

// Manifest is the sole fixture and benchmark authority for one assay.
type Manifest struct {
	Benchmark Benchmark
	Entries   []ManifestEntry
}

// Validate checks whether m declares the required immutable authority. It does
// not fetch or inspect fixture bytes.
func (m Manifest) Validate() error {
	if strings.TrimSpace(m.Benchmark.Pipeline) == "" {
		return fmt.Errorf("manifest benchmark pipeline is empty")
	}
	if strings.TrimSpace(m.Benchmark.Release) == "" {
		return fmt.Errorf("manifest benchmark release is empty")
	}
	if strings.TrimSpace(m.Benchmark.SelectedRoute) == "" {
		return fmt.Errorf("manifest benchmark selected route is empty")
	}
	if len(m.Benchmark.Sources) == 0 {
		return fmt.Errorf("manifest benchmark sources are empty")
	}
	for i, source := range m.Benchmark.Sources {
		if strings.TrimSpace(source) == "" {
			return fmt.Errorf("manifest benchmark source %d is empty", i)
		}
	}
	if len(m.Entries) == 0 {
		return fmt.Errorf("manifest entries are empty")
	}
	seen := make(map[string]struct{}, len(m.Entries))
	for i, entry := range m.Entries {
		if err := validateEntry(entry); err != nil {
			return fmt.Errorf("manifest entry %d: %w", i, err)
		}
		if _, ok := seen[entry.LogicalName]; ok {
			return fmt.Errorf("manifest entry %d: duplicate logical name %q", i, entry.LogicalName)
		}
		seen[entry.LogicalName] = struct{}{}
	}
	return nil
}

func validateEntry(entry ManifestEntry) error {
	fields := []struct {
		name  string
		value string
	}{
		{name: "logical name", value: entry.LogicalName},
		{name: "role", value: entry.Role},
		{name: "source repository", value: entry.Source.Repository},
		{name: "source path", value: entry.Source.Path},
		{name: "source URL", value: entry.Source.URL},
		{name: "provenance", value: entry.Provenance},
		{name: "license", value: entry.License},
		{name: "license source", value: entry.LicenseSource},
		{name: "redistribution", value: entry.Redistribution},
		{name: "benchmark relation", value: entry.BenchmarkRelation},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is empty", field.name)
		}
	}
	if strings.EqualFold(strings.TrimSpace(entry.License), "unknown") {
		return fmt.Errorf("license is unknown")
	}
	if !validCommit(entry.Source.Commit) {
		return fmt.Errorf("source commit is not a full hexadecimal object id")
	}
	if entry.ByteCount < 0 {
		return fmt.Errorf("byte count must not be negative")
	}
	if !validHex(entry.SHA256, 64) {
		return fmt.Errorf("SHA-256 must be 64 lowercase hexadecimal characters")
	}
	if len(entry.AssayUse) == 0 {
		return fmt.Errorf("assay use is empty")
	}
	for i, use := range entry.AssayUse {
		if strings.TrimSpace(use) == "" {
			return fmt.Errorf("assay use %d is empty", i)
		}
	}
	return nil
}

func validCommit(commit string) bool {
	return validHex(commit, 40) || validHex(commit, 64)
}

func validHex(value string, size int) bool {
	if len(value) != size {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
