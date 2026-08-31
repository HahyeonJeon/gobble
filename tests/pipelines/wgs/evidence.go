// Package wgsevidence owns typed access to the WGS fixture manifest's exact
// Planning-bound bytes.
package wgsevidence

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
)

// CacheDir is the WGS owner's ignored host cache.
const CacheDir = "tests/pipelines/wgs/testdata/cache"

// ManifestPath is the WGS owner's sole fixture manifest.
const ManifestPath = "tests/pipelines/wgs/testdata/manifest.json"

// FixtureSheet is the localized typed WGS samplesheet.
const FixtureSheet = "tests/pipelines/wgs/testdata/wgs-samplesheet.csv"

type manifest struct {
	Entries []struct {
		Name   string `json:"name"`
		URL    string `json:"url"`
		Bytes  int64  `json:"bytes"`
		SHA256 string `json:"sha256"`
		Staged bool   `json:"staged"`
	} `json:"entries"`
}

//go:embed testdata/manifest.json
var manifestJSON []byte

// Pins returns copied directly staged pin records in manifest order. Benchmark
// selectors and license/provenance bytes remain manifest authority but are not
// fetched as product inputs.
func Pins() ([]fixture.Pin, error) {
	var decoded manifest
	if err := json.Unmarshal(manifestJSON, &decoded); err != nil {
		return nil, fmt.Errorf("decode WGS fixture manifest: %w", err)
	}
	pins := make([]fixture.Pin, 0, len(decoded.Entries))
	for _, entry := range decoded.Entries {
		if !entry.Staged {
			continue
		}
		pins = append(pins, fixture.Pin{Name: entry.Name, URL: entry.URL, Bytes: entry.Bytes, SHA256: entry.SHA256})
	}
	return pins, nil
}

// MustPins returns Pins or panics for an invalid committed manifest.
func MustPins() []fixture.Pin {
	pins, err := Pins()
	if err != nil {
		panic(err)
	}
	return pins
}

// MustPin returns the staged pin named name or panics when the committed
// manifest omits it.
func MustPin(name string) fixture.Pin {
	for _, pin := range MustPins() {
		if pin.Name == name {
			return pin
		}
	}
	panic("WGS fixture manifest missing staged pin " + name)
}

// Fetch returns a verified cached fixture for live evidence.
func Fetch(cacheDir string, pin fixture.Pin) (string, error) {
	return fixture.Fetch(cacheDir, pin)
}
