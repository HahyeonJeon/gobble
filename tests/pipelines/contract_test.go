package pipelineevidence

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	product "github.com/HahyeonJeon/gobble/assets/pipelines"
)

type contractSample struct {
	Name string
}

type contractConfig struct {
	Labels map[string]string
}

var contractShape = product.Contract[contractSample, contractConfig]{
	Parse:         contractParse,
	Load:          contractLoad,
	DefaultConfig: contractDefaultConfig,
	Build:         contractBuild,
	Pipeline:      contractPipeline,
}

func TestTypedPipelineContractShape(t *testing.T) {
	parsed, err := contractShape.Parse(strings.NewReader("name\nsample\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(parsed) != 1 || parsed[0].Name != "sample" {
		t.Fatalf("Parse() = %#v, want sample", parsed)
	}

	path := filepath.Join(t.TempDir(), "samples.csv")
	if err := os.WriteFile(path, []byte("name\nloaded\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
	loaded, err := contractShape.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(loaded) != 1 || loaded[0].Name != "loaded" {
		t.Fatalf("Load() = %#v, want loaded", loaded)
	}

	config := contractShape.DefaultConfig()
	config.Labels["owner"] = "changed"
	if got := contractShape.DefaultConfig().Labels["owner"]; got != "default" {
		t.Fatalf("DefaultConfig() retained caller mutation: got %q, want default", got)
	}
	p := contractShape.Build([]contractSample{{Name: "sample"}}, contractShape.DefaultConfig())
	if _, err := gobble.Compose(p); err != nil {
		t.Fatalf("Build() Compose() error = %v, want nil", err)
	}
	if _, err := gobble.Compose(contractShape.Pipeline()); err != nil {
		t.Fatalf("Pipeline() Compose() error = %v, want nil", err)
	}
}

func TestTypedPipelineContractRejectsInvalidCSVAndPath(t *testing.T) {
	t.Run("Parse malformed CSV", func(t *testing.T) {
		_, err := contractShape.Parse(strings.NewReader("name\n\"unterminated"))
		requireContractDefect(t, err, gobble.DefectInvalidSampleSheet, "<reader>")
	})

	t.Run("Load malformed CSV", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "malformed.csv")
		if err := os.WriteFile(path, []byte("name\n\"unterminated"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v, want nil", err)
		}
		_, err := contractShape.Load(path)
		requireContractDefect(t, err, gobble.DefectInvalidSampleSheet, path)
	})

	t.Run("Load unreadable path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.csv")
		_, err := contractShape.Load(path)
		requireContractDefect(t, err, gobble.DefectInvalidPath, path)
	})
}

func TestRecordComposeDefectsCopiesStructuredFailure(t *testing.T) {
	p := gobble.NewPipeline("contract")
	paths := []string{"references/missing.fa"}
	product.RecordComposeDefects(p, gobble.Defect{
		Code:    gobble.DefectInvalidPath,
		Unit:    "reference",
		Message: "required reference member is missing",
		Paths:   paths,
	})
	paths[0] = "changed"

	_, err := gobble.Compose(p)
	var composeErr *gobble.Error
	if !errors.As(err, &composeErr) {
		t.Fatalf("Compose() error = %T %v, want *gobble.Error", err, err)
	}
	if composeErr.Op != "compose" || len(composeErr.Defects) != 1 {
		t.Fatalf("Compose() error = %#v, want one compose defect", composeErr)
	}
	defect := composeErr.Defects[0]
	if defect.Code != gobble.DefectInvalidPath || defect.Paths[0] != "references/missing.fa" {
		t.Fatalf("Compose() defect = %#v, want copied invalid-path defect", defect)
	}
}

func TestTypedPipelineBuildRejectsInvalidSamplesWithoutPanic(t *testing.T) {
	p := contractShape.Build(nil, contractShape.DefaultConfig())
	_, err := gobble.Compose(p)
	var composeErr *gobble.Error
	if !errors.As(err, &composeErr) {
		t.Fatalf("Build(nil) Compose() error = %T %v, want *gobble.Error", err, err)
	}
	if composeErr.Op != "compose" || len(composeErr.Defects) != 1 {
		t.Fatalf("Build(nil) Compose() error = %#v, want one compose defect", composeErr)
	}
	defect := composeErr.Defects[0]
	if defect.Code != gobble.DefectInvalidSampleSheet || defect.Unit != "samples" {
		t.Fatalf("Build(nil) Compose() defect = %#v, want invalid-samplesheet for samples", defect)
	}
}

func TestCopyHelpersDoNotAliasContainers(t *testing.T) {
	samples := []contractSample{{Name: "sample"}}
	labels := map[string]string{"owner": "default"}
	copiedSamples := product.CopySlice(samples)
	copiedLabels := product.CopyMap(labels)
	copiedSamples[0].Name = "changed"
	copiedLabels["owner"] = "changed"
	if samples[0].Name != "sample" || labels["owner"] != "default" {
		t.Fatalf("copy helpers aliased inputs: samples=%#v labels=%#v", samples, labels)
	}
}

func TestManifestValidate(t *testing.T) {
	manifest := validManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Manifest.Validate() error = %v, want nil", err)
	}

	manifest.Entries[0].SHA256 = strings.Repeat("0", 63)
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("Manifest.Validate() error = %v, want SHA-256 failure", err)
	}
}

func TestManifestValidateRejectsBlankAuthority(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{name: "benchmark pipeline", mutate: func(m *Manifest) { m.Benchmark.Pipeline = " \t\n" }, want: "benchmark pipeline is empty"},
		{name: "benchmark release", mutate: func(m *Manifest) { m.Benchmark.Release = " \t\n" }, want: "benchmark release is empty"},
		{name: "benchmark selected route", mutate: func(m *Manifest) { m.Benchmark.SelectedRoute = " \t\n" }, want: "benchmark selected route is empty"},
		{name: "benchmark source", mutate: func(m *Manifest) { m.Benchmark.Sources[0] = " \t\n" }, want: "benchmark source 0 is empty"},
		{name: "logical name", mutate: func(m *Manifest) { m.Entries[0].LogicalName = " \t\n" }, want: "logical name is empty"},
		{name: "role", mutate: func(m *Manifest) { m.Entries[0].Role = " \t\n" }, want: "role is empty"},
		{name: "source repository", mutate: func(m *Manifest) { m.Entries[0].Source.Repository = " \t\n" }, want: "source repository is empty"},
		{name: "source path", mutate: func(m *Manifest) { m.Entries[0].Source.Path = " \t\n" }, want: "source path is empty"},
		{name: "source URL", mutate: func(m *Manifest) { m.Entries[0].Source.URL = " \t\n" }, want: "source URL is empty"},
		{name: "provenance", mutate: func(m *Manifest) { m.Entries[0].Provenance = " \t\n" }, want: "provenance is empty"},
		{name: "license", mutate: func(m *Manifest) { m.Entries[0].License = " \t\n" }, want: "license is empty"},
		{name: "license source", mutate: func(m *Manifest) { m.Entries[0].LicenseSource = " \t\n" }, want: "license source is empty"},
		{name: "redistribution", mutate: func(m *Manifest) { m.Entries[0].Redistribution = " \t\n" }, want: "redistribution is empty"},
		{name: "assay use", mutate: func(m *Manifest) { m.Entries[0].AssayUse[0] = " \t\n" }, want: "assay use 0 is empty"},
		{name: "benchmark relation", mutate: func(m *Manifest) { m.Entries[0].BenchmarkRelation = " \t\n" }, want: "benchmark relation is empty"},
		{name: "unknown license", mutate: func(m *Manifest) { m.Entries[0].License = " unknown " }, want: "license is unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := validManifest()
			tt.mutate(&manifest)
			err := manifest.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Manifest.Validate() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func contractDefaultConfig() contractConfig {
	return contractConfig{Labels: map[string]string{"owner": "default"}}
}

func contractParse(r io.Reader) ([]contractSample, error) {
	return contractParseAt(r, "<reader>")
}

func contractLoad(path string) ([]contractSample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, contractInputError(gobble.DefectInvalidPath, "samplesheet path is not readable", path)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, contractInputError(gobble.DefectInvalidPath, "samplesheet path is not a regular file", path)
	}
	return contractParseAt(f, path)
}

func contractParseAt(r io.Reader, path string) ([]contractSample, error) {
	if r == nil {
		return nil, contractInputError(gobble.DefectInvalidSampleSheet, "samplesheet is malformed", path)
	}
	records, err := csv.NewReader(r).ReadAll()
	if err != nil || len(records) < 2 || len(records[0]) != 1 || records[0][0] != "name" {
		return nil, contractInputError(gobble.DefectInvalidSampleSheet, "samplesheet is malformed", path)
	}
	samples := make([]contractSample, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) != 1 || strings.TrimSpace(record[0]) == "" {
			return nil, contractInputError(gobble.DefectInvalidSampleSheet, "sample name is empty", path)
		}
		samples = append(samples, contractSample{Name: record[0]})
	}
	if len(samples) == 0 {
		return nil, contractInputError(gobble.DefectInvalidSampleSheet, "samplesheet has no samples", path)
	}
	return samples, nil
}

func contractInputError(code gobble.DefectCode, message string, path string) *gobble.Error {
	return &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
		Code:    code,
		Unit:    "samplesheet",
		Message: message,
		Paths:   []string{path},
	}}}
}

func requireContractDefect(t *testing.T, err error, code gobble.DefectCode, path string) {
	t.Helper()
	var contractErr *gobble.Error
	if !errors.As(err, &contractErr) {
		t.Fatalf("error = %T %v, want *gobble.Error", err, err)
	}
	if contractErr.Op != "compose" || len(contractErr.Defects) != 1 {
		t.Fatalf("error = %#v, want one compose defect", contractErr)
	}
	defect := contractErr.Defects[0]
	if defect.Code != code || defect.Unit != "samplesheet" || len(defect.Paths) != 1 || defect.Paths[0] != path {
		t.Fatalf("defect = %#v, want code %q, samplesheet unit, path %q", defect, code, path)
	}
}

func contractBuild(samples []contractSample, config contractConfig) *gobble.Pipeline {
	samples = product.CopySlice(samples)
	config.Labels = product.CopyMap(config.Labels)
	p := gobble.NewPipeline("contract")
	if len(samples) == 0 {
		product.RecordComposeDefects(p, gobble.Defect{
			Code:    gobble.DefectInvalidSampleSheet,
			Unit:    "samples",
			Message: "at least one sample is required",
		})
		return p
	}
	input := p.AddInput("input", gobble.PathSpec{Dir: gobble.Dir("inputs"), Base: samples[0].Name, Ext: ".txt"})
	task := p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "inputs/sample.txt", "outputs/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "input", From: input}},
		Outputs: []gobble.Bind{{Name: "output", Spec: gobble.PathSpec{Dir: gobble.Dir("outputs"), Base: samples[0].Name, Ext: ".txt"}}},
		Params:  []gobble.Param{{Name: "owner", Value: config.Labels["owner"]}},
	})
	task.Out("output")
	return p
}

func contractPipeline() *gobble.Pipeline {
	return contractBuild([]contractSample{{Name: "sample"}}, contractDefaultConfig())
}

func validManifest() Manifest {
	return Manifest{
		Benchmark: Benchmark{
			Pipeline:      "nf-core/example",
			Release:       "1.2.3",
			SelectedRoute: "default",
			Sources:       []string{"https://example.invalid/nf-core/example/1.2.3/test"},
		},
		Entries: []ManifestEntry{{
			LogicalName: "reads",
			Role:        "read",
			Source: Upstream{
				Repository: "nf-core/test-datasets",
				Commit:     "0123456789abcdef0123456789abcdef01234567",
				Path:       "data/reads.fastq.gz",
				URL:        "https://example.invalid/reads.fastq.gz",
			},
			ByteCount:         1,
			SHA256:            strings.Repeat("0", 64),
			Provenance:        "synthetic contract specimen",
			License:           "CC0-1.0",
			LicenseSource:     "https://example.invalid/license",
			Redistribution:    "allowed with attribution",
			AssayUse:          []string{"input"},
			BenchmarkRelation: "selected by the versioned default test",
		}},
	}
}
