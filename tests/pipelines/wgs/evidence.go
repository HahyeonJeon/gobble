// Package wgsevidence owns typed access to the WGS fixture manifest.
package wgsevidence

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
)

const (
	CacheDir     = "tests/pipelines/wgs/testdata/cache"
	ManifestPath = "tests/pipelines/wgs/testdata/manifest.json"
	FixtureSheet = "tests/pipelines/wgs/testdata/wgs-samplesheet.csv"
)

//go:embed testdata/manifest.json
var manifestJSON []byte

func Manifest() (fixture.Manifest, error) { return fixture.DecodeManifest(manifestJSON) }

func MustManifest() fixture.Manifest {
	manifest, err := Manifest()
	if err != nil {
		panic(err)
	}
	return manifest
}

func Pins() ([]fixture.Pin, error) {
	manifest, err := Manifest()
	if err != nil {
		return nil, fmt.Errorf("decode WGS fixture manifest: %w", err)
	}
	return manifest.Pins(), nil
}

func MustPins() []fixture.Pin {
	pins, err := Pins()
	if err != nil {
		panic(err)
	}
	return pins
}

func MustPin(name string) fixture.Pin {
	for _, pin := range MustPins() {
		if pin.Name == name {
			return pin
		}
	}
	panic("WGS fixture manifest missing staged pin " + name)
}

func StagedInputs() []fixture.StagedInput { return MustManifest().StagedInputs() }

func Fetch(cacheDir string, pin fixture.Pin) (string, error) { return fixture.Fetch(cacheDir, pin) }

// StageOfficial stages the default WGS product inputs and materializes the two
// declared interval members from the manifest-owned source byte.
func StageOfficial(cacheDir, workspace string) ([]fixture.StagedInput, error) {
	inputs, err := fixture.StageManifest(cacheDir, workspace, MustManifest())
	if err != nil {
		return nil, err
	}
	source := filepath.Join(workspace, "in", "reference", "genome.multi_intervals.bed")
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("read WGS interval source: %w", err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != 2 {
		return nil, fmt.Errorf("WGS interval source has %d members, want 2", len(lines))
	}
	for i, line := range lines {
		path := filepath.Join(workspace, "in", "reference", "intervals", fmt.Sprintf("interval_%03d.bed", i+1))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, append(append([]byte(nil), line...), '\n'), 0o644); err != nil {
			return nil, err
		}
	}
	return inputs, nil
}
