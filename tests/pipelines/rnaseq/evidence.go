// Package rnaseqevidence owns typed access to the RNA-seq fixture manifest.
package rnaseqevidence

import (
	_ "embed"
	"fmt"

	"github.com/HahyeonJeon/gobble/internal/fixture"
)

const (
	CacheDir         = "tests/pipelines/rnaseq/testdata/cache"
	FixtureSheet     = "tests/pipelines/rnaseq/testdata/rnaseq-samplesheet.csv"
	LiveFixtureSheet = "tests/pipelines/rnaseq/testdata/rnaseq-live-samplesheet.csv"
	ManifestPath     = "tests/pipelines/rnaseq/testdata/manifest.json"
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
		return nil, fmt.Errorf("decode RNA-seq fixture manifest: %w", err)
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
	panic("RNA-seq fixture manifest missing staged pin " + name)
}

func StagedInputs() []fixture.StagedInput { return MustManifest().StagedInputs() }

func Fetch(cacheDir string, pin fixture.Pin) (string, error) { return fixture.Fetch(cacheDir, pin) }

func StageOfficial(cacheDir, workspace string) ([]fixture.StagedInput, error) {
	return fixture.StageManifest(cacheDir, workspace, MustManifest())
}
