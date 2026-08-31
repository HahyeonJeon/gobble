// Package scrnaseqevidence owns typed access to the scRNA fixture manifest's exact bytes.
package scrnaseqevidence

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
)

// CacheDir is the scRNA owner's ignored host cache.
const CacheDir = "tests/pipelines/scrnaseq/testdata/cache"

// ManifestPath is the scRNA owner's sole fixture manifest.
const ManifestPath = "tests/pipelines/scrnaseq/testdata/manifest.json"

// FixtureSheet is the localized typed scRNA samplesheet.
const FixtureSheet = "tests/pipelines/scrnaseq/testdata/scrnaseq-samplesheet.csv"

// StagedInput binds one exact manifest pin to its product workspace path.
type StagedInput struct {
	Pin         fixture.Pin
	Destination string
}

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
		return nil, fmt.Errorf("decode scRNA fixture manifest: %w", err)
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

// MustPin returns the staged pin named name or panics when the committed
// manifest omits it.
func MustPin(name string) fixture.Pin {
	for _, pin := range MustPins() {
		if pin.Name == name {
			return pin
		}
	}
	panic("scRNA fixture manifest missing staged pin " + name)
}

// StagedInputs returns the complete manifest-to-product staging plan.
func StagedInputs() []StagedInput {
	return []StagedInput{
		{Pin: MustPin("Sample_X_S1_L001_R1_001.fastq.gz"), Destination: "in/reads/Sample_X_S1_L001_R1_001.fastq.gz"},
		{Pin: MustPin("Sample_X_S1_L001_R2_001.fastq.gz"), Destination: "in/reads/Sample_X_S1_L001_R2_001.fastq.gz"},
		{Pin: MustPin("Sample_Y_S1_L001_R1_001.fastq.gz"), Destination: "in/reads/Sample_Y_S1_L001_R1_001.fastq.gz"},
		{Pin: MustPin("Sample_Y_S1_L001_R2_001.fastq.gz"), Destination: "in/reads/Sample_Y_S1_L001_R2_001.fastq.gz"},
		{Pin: MustPin("Sample_Y_S1_L002_R1_001.fastq.gz"), Destination: "in/reads/Sample_Y_S1_L002_R1_001.fastq.gz"},
		{Pin: MustPin("Sample_Y_S1_L002_R2_001.fastq.gz"), Destination: "in/reads/Sample_Y_S1_L002_R2_001.fastq.gz"},
		{Pin: MustPin("GRCm38.p6.genome.chr19.fa"), Destination: "in/reference/genome.fa"},
		{Pin: MustPin("gencode.vM19.annotation.chr19.gtf"), Destination: "in/reference/genes.gtf"},
		{Pin: MustPin("10x_V2_barcode_whitelist.txt.gz"), Destination: "in/reference/10x_V2_barcode_whitelist.txt.gz"},
	}
}

// Fetch returns a verified cached fixture for explicitly selected live evidence.
func Fetch(cacheDir string, pin fixture.Pin) (string, error) {
	return fixture.Fetch(cacheDir, pin)
}

// StageOfficial fetches every staged scRNA pin, copies it to its product input
// path, and verifies the copied byte identity before returning.
func StageOfficial(cacheDir, workspace string) ([]StagedInput, error) {
	inputs := StagedInputs()
	for _, input := range inputs {
		source, err := Fetch(cacheDir, input.Pin)
		if err != nil {
			return nil, fmt.Errorf("fetch scRNA fixture %s: %w", input.Pin.Name, err)
		}
		destination := filepath.Join(workspace, filepath.FromSlash(input.Destination))
		if err := copyFixture(source, destination); err != nil {
			return nil, fmt.Errorf("stage scRNA fixture %s: %w", input.Pin.Name, err)
		}
		if err := input.Pin.Check(destination); err != nil {
			return nil, fmt.Errorf("verify staged scRNA fixture %s: %w", input.Pin.Name, err)
		}
	}
	return inputs, nil
}

func copyFixture(source, destination string) (err error) {
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
