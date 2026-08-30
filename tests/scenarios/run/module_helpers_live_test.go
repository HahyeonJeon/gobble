//go:build live

package run

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	methylseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/methylseq"
)

var identityOnce sync.Once
var identity gobble.Identity
var identityErr error

func testOccupyOption(t *testing.T) gobble.OccupyOption {
	t.Helper()
	identityOnce.Do(func() {
		identity, identityErr = gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/tests/scenarios/run")
	})
	if identityErr != nil {
		t.Fatalf("IdentityFromBuildInfo() error = %v", identityErr)
	}
	return gobble.WithIdentity(identity)
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker info: %v", err)
	}
}

func fatalAPIError(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	t.Fatal(formatAPIError(name, err))
}

func formatAPIError(name string, err error) string {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return name + " error = " + err.Error()
	}
	var b strings.Builder
	b.WriteString(name)
	b.WriteString(" op=")
	b.WriteString(ge.Op)
	b.WriteString(" error = ")
	b.WriteString(err.Error())
	for _, d := range ge.Defects {
		b.WriteString("\n  code=")
		b.WriteString(string(d.Code))
		b.WriteString(" unit=")
		b.WriteString(d.Unit)
		b.WriteString(" path=")
		b.WriteString(strings.Join(d.Paths, ","))
		b.WriteString(" message=")
		b.WriteString(d.Message)
	}
	return b.String()
}

func cachePin(t *testing.T, cacheDir string, pin fixture.Pin) string {
	t.Helper()
	if !filepath.IsAbs(cacheDir) {
		cacheDir = filepath.Join(moduleRoot(t), filepath.FromSlash(cacheDir))
	}
	dest, err := fixture.Fetch(cacheDir, pin)
	if err != nil {
		t.Fatalf("download %s: %v", pin.URL, err)
	}
	return dest
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatalf("go.mod not found from %s", directory)
		}
		directory = parent
	}
}

func stageMethylPins(t *testing.T, dir string) {
	t.Helper()
	pc.StageFile(t, dir, "in/genome.fa", cachePin(t, methylseqevidence.CacheDir, methylseqevidence.GenomeFASTA))
	pc.StageFile(t, dir, "in/Ecoli_10K_methylated_R1.fastq.gz", cachePin(t, methylseqevidence.CacheDir, methylseqevidence.Test1FASTQ))
	pc.StageFile(t, dir, "in/Ecoli_10K_methylated_R2.fastq.gz", cachePin(t, methylseqevidence.CacheDir, methylseqevidence.Test2FASTQ))
}

func inspectJSONL(t *testing.T, workspace, view string) []map[string]any {
	t.Helper()
	data, err := gobble.Inspect(workspace, gobble.View(view), "")
	if err != nil {
		t.Fatalf("Inspect(%s) error = %v", view, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	var out []map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("Inspect(%s) JSONL: %v\n%s", view, err, data)
		}
		out = append(out, rec)
	}
	return out
}
