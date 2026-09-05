package fixture

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Manifest is the common executable shape of an assay-owned fixture manifest.
// Assay-specific fields remain in the manifest and its owner tests.
type Manifest struct {
	Schema     int                 `json:"schema"`
	Benchmark  ManifestBenchmark   `json:"benchmark"`
	Entries    []ManifestEntry     `json:"entries"`
	StageTrace map[string][]string `json:"stage_trace"`
	Limits     []string            `json:"limits"`
	Images     []ManifestImage     `json:"images"`
	Trees      json.RawMessage     `json:"trees"`
}

// ManifestBenchmark identifies one immutable upstream product benchmark.
type ManifestBenchmark struct {
	Pipeline          string            `json:"pipeline"`
	Release           string            `json:"release"`
	Commit            string            `json:"commit"`
	SelectedRoute     string            `json:"selected_route"`
	DatasetCommit     string            `json:"dataset_commit"`
	TestConfig        string            `json:"test_config"`
	CoverageScenarios map[string]string `json:"coverage_scenarios"`
}

// ManifestEntry identifies one immutable upstream byte. Staged entries also
// name the exact workspace-relative destination used by live evidence.
type ManifestEntry struct {
	LogicalName       string   `json:"logical_name"`
	Name              string   `json:"name"`
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
	LicenseAuthority  string   `json:"license_authority"`
	Redistribution    string   `json:"redistribution"`
	AssayUse          []string `json:"assay_use"`
	BenchmarkRelation string   `json:"benchmark_relation"`
	Staged            bool     `json:"staged"`
	Destination       string   `json:"destination"`
}

// ManifestImage records one immutable default image and its provenance.
type ManifestImage struct {
	Module            string   `json:"module"`
	Modules           []string `json:"modules"`
	TaskName          string   `json:"task_name"`
	Reference         string   `json:"reference"`
	Digest            string   `json:"digest"`
	Tool              string   `json:"tool"`
	Command           string   `json:"command"`
	Version           string   `json:"version"`
	Source            string   `json:"source"`
	ModuleCommit      string   `json:"module_commit"`
	ModuleSource      string   `json:"module_source"`
	Provenance        string   `json:"provenance"`
	License           string   `json:"license"`
	LicenseSource     string   `json:"license_source"`
	Platform          string   `json:"platform"`
	BenchmarkPipeline string   `json:"benchmark_pipeline"`
	BenchmarkRelease  string   `json:"benchmark_release"`
}

// StagedInput binds one manifest pin to its product workspace destination.
type StagedInput struct {
	Pin         Pin
	Destination string
}

// DecodeManifest decodes and validates one committed assay manifest.
func DecodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Manifest{}, fmt.Errorf("manifest has trailing JSON")
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Validate checks the family-wide immutable byte, image, benchmark, and staging
// contract. It performs no network or filesystem access.
func (m Manifest) Validate() error {
	if m.Schema != 4 {
		return fmt.Errorf("manifest schema = %d, want 4", m.Schema)
	}
	if m.Benchmark.Pipeline == "" || m.Benchmark.Release == "" || m.Benchmark.SelectedRoute == "" {
		return fmt.Errorf("manifest benchmark is incomplete")
	}
	if !lowerHex(m.Benchmark.Commit, 40) || !lowerHex(m.Benchmark.DatasetCommit, 40) {
		return fmt.Errorf("manifest benchmark commits are not immutable object ids")
	}
	if len(m.Entries) == 0 || len(m.Images) == 0 || len(m.StageTrace) == 0 || len(m.Limits) == 0 {
		return fmt.Errorf("manifest entries, images, stage trace, and limits are required")
	}
	seenEntries := make(map[string]bool, len(m.Entries))
	seenNames := make(map[string]bool)
	seenDestinations := make(map[string]bool)
	for _, entry := range m.Entries {
		if entry.LogicalName == "" || seenEntries[entry.LogicalName] || entry.Role == "" || entry.Repository == "" || entry.Path == "" || entry.URL == "" || entry.Bytes <= 0 || entry.Provenance == "" || len(entry.AssayUse) == 0 {
			return fmt.Errorf("manifest entry %q is incomplete or duplicated", entry.LogicalName)
		}
		seenEntries[entry.LogicalName] = true
		if !lowerHex(entry.Commit, 40) || !lowerHex(entry.SHA256, 64) || !strings.Contains(entry.URL, entry.Commit) || mutableURL(entry.URL) {
			return fmt.Errorf("manifest entry %q has mutable or invalid byte identity", entry.LogicalName)
		}
		if entry.License == "" || entry.LicenseSource == "" || entry.Redistribution == "" || entry.BenchmarkRelation == "" {
			return fmt.Errorf("manifest entry %q has incomplete license or benchmark authority", entry.LogicalName)
		}
		if entry.Staged {
			if entry.Name == "" || seenNames[entry.Name] || !validRelativePath(entry.Destination) || seenDestinations[entry.Destination] {
				return fmt.Errorf("manifest entry %q has invalid or duplicate staging destination", entry.LogicalName)
			}
			seenNames[entry.Name] = true
			seenDestinations[entry.Destination] = true
		}
	}
	for stage, authorities := range m.StageTrace {
		if stage == "" || len(authorities) == 0 {
			return fmt.Errorf("manifest stage trace contains an empty stage")
		}
	}
	for _, limit := range m.Limits {
		if strings.TrimSpace(limit) == "" {
			return fmt.Errorf("manifest contains an empty support limit")
		}
	}
	seenModules := make(map[string]bool)
	for _, image := range m.Images {
		modules := image.Modules
		if image.Module != "" {
			modules = append([]string{image.Module}, modules...)
		}
		if len(modules) == 0 || image.Reference == "" || image.Tool == "" || image.Command == "" || image.Version == "" || image.License == "" || image.Platform != "linux/amd64" {
			return fmt.Errorf("manifest image %q is incomplete", image.Reference)
		}
		if strings.Contains(image.Reference, "@") || strings.Contains(strings.ToLower(image.Reference), ":latest") || !strings.HasPrefix(image.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(image.Digest, "sha256:"), 64) {
			return fmt.Errorf("manifest image %q has mutable or invalid identity", image.Reference)
		}
		if image.Source == "" && image.ModuleSource == "" {
			return fmt.Errorf("manifest image %q has no source provenance", image.Reference)
		}
		if mutableURL(image.Source) || mutableURL(image.ModuleSource) || mutableURL(image.LicenseSource) {
			return fmt.Errorf("manifest image %q uses mutable source authority", image.Reference)
		}
		if (image.BenchmarkPipeline != "" && image.BenchmarkPipeline != m.Benchmark.Pipeline) || (image.BenchmarkRelease != "" && image.BenchmarkRelease != m.Benchmark.Release) {
			return fmt.Errorf("manifest image %q benchmark differs from its owner", image.Reference)
		}
		for _, module := range modules {
			if module == "" || seenModules[module] {
				return fmt.Errorf("manifest image module %q is empty or duplicated", module)
			}
			seenModules[module] = true
		}
	}
	return nil
}

// Pins returns copied staged pin records in manifest order.
func (m Manifest) Pins() []Pin {
	pins := make([]Pin, 0, len(m.Entries))
	for _, entry := range m.Entries {
		if entry.Staged {
			pins = append(pins, Pin{Name: entry.Name, URL: entry.URL, Bytes: entry.Bytes, SHA256: entry.SHA256})
		}
	}
	return pins
}

// StagedInputs returns copied manifest-to-workspace staging records.
func (m Manifest) StagedInputs() []StagedInput {
	inputs := make([]StagedInput, 0, len(m.Entries))
	for _, entry := range m.Entries {
		if entry.Staged {
			inputs = append(inputs, StagedInput{
				Pin:         Pin{Name: entry.Name, URL: entry.URL, Bytes: entry.Bytes, SHA256: entry.SHA256},
				Destination: entry.Destination,
			})
		}
	}
	return inputs
}

// StageManifest fetches, copies, and rechecks every staged manifest byte.
func StageManifest(cacheDir, workspace string, manifest Manifest) ([]StagedInput, error) {
	return StageManifestContext(context.Background(), cacheDir, workspace, manifest)
}

// StageManifestContext stages inputs into a caller-owned fresh workspace.
func StageManifestContext(ctx context.Context, cacheDir, workspace string, manifest Manifest) ([]StagedInput, error) {
	inputs := manifest.StagedInputs()
	for _, input := range inputs {
		source, err := FetchContext(ctx, cacheDir, input.Pin)
		if err != nil {
			return nil, fmt.Errorf("fetch fixture %s: %w", input.Pin.Name, err)
		}
		destination := filepath.Join(workspace, filepath.FromSlash(input.Destination))
		if err := copyFile(source, destination); err != nil {
			return nil, fmt.Errorf("stage fixture %s: %w", input.Pin.Name, err)
		}
		if err := input.Pin.Check(destination); err != nil {
			return nil, fmt.Errorf("verify staged fixture %s: %w", input.Pin.Name, err)
		}
	}
	return inputs, nil
}

func copyFile(source, destination string) (err error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := in.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(destination)
		return err
	}
	if err = out.Close(); err != nil {
		_ = os.Remove(destination)
	}
	return err
}

func validRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.Clean(value) == "." {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	return clean == value && clean != ".." && !strings.HasPrefix(clean, "../")
}

func mutableURL(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "/main/") || strings.Contains(lower, "/master/") || strings.Contains(lower, "/refs/heads/") || strings.Contains(lower, "/latest/")
}

func lowerHex(value string, size int) bool {
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
