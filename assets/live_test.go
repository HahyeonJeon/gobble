//go:build live

package assets

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

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

func cachePin(t *testing.T, pin Pin) string {
	t.Helper()
	dest, err := FetchPin(pin)
	if err != nil {
		t.Fatalf("download %s: %v", pin.URL, err)
	}
	return dest
}

func stageMethylPins(t *testing.T, dir string) {
	t.Helper()
	stageFile(t, dir, "in/genome.fa", cachePin(t, PinMethylGenomeFASTA))
	stageFile(t, dir, "in/Ecoli_10K_methylated_R1.fastq.gz", cachePin(t, PinMethylTest1FASTQ))
	stageFile(t, dir, "in/Ecoli_10K_methylated_R2.fastq.gz", cachePin(t, PinMethylTest2FASTQ))
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
