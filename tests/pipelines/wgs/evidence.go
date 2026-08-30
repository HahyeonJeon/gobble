// Package wgsevidence owns checkpoint WGS fixture facts.
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

//go:embed testdata/manifest.json
var manifestJSON []byte

// Pins returns copied pin records in manifest order.
func Pins() ([]fixture.Pin, error) {
	var pins []fixture.Pin
	if err := json.Unmarshal(manifestJSON, &pins); err != nil {
		return nil, fmt.Errorf("decode WGS fixture manifest: %w", err)
	}
	return append([]fixture.Pin(nil), pins...), nil
}

// MustPins returns Pins or panics for an invalid committed manifest.
func MustPins() []fixture.Pin {
	pins, err := Pins()
	if err != nil {
		panic(err)
	}
	return pins
}

// Fetch returns a verified cached fixture for live evidence.
func Fetch(cacheDir string, pin fixture.Pin) (string, error) {
	return fixture.Fetch(cacheDir, pin)
}
