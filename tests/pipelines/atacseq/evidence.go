// Package atacseqevidence owns typed access to the ATAC fixture manifest's exact bytes.
package atacseqevidence

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
)

// CacheDir is the ATAC owner's ignored host cache.
const CacheDir = "tests/pipelines/atacseq/testdata/cache"

// ManifestPath is the ATAC owner's sole fixture manifest.
const ManifestPath = "tests/pipelines/atacseq/testdata/manifest.json"

// FixtureSheet is the localized typed ATAC samplesheet.
const FixtureSheet = "tests/pipelines/atacseq/testdata/atacseq-samplesheet.csv"

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
// selectors and license/provenance bytes remain authority but are not inputs.
func Pins() ([]fixture.Pin, error) {
	var decoded manifest
	if err := json.Unmarshal(manifestJSON, &decoded); err != nil {
		return nil, fmt.Errorf("decode ATAC fixture manifest: %w", err)
	}
	pins := make([]fixture.Pin, 0, len(decoded.Entries))
	for _, entry := range decoded.Entries {
		if entry.Staged {
			pins = append(pins, fixture.Pin{Name: entry.Name, URL: entry.URL, Bytes: entry.Bytes, SHA256: entry.SHA256})
		}
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

// Fetch returns a verified cached fixture for explicitly selected live evidence.
func Fetch(cacheDir string, pin fixture.Pin) (string, error) {
	return fixture.Fetch(cacheDir, pin)
}
