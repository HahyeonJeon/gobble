//go:build live

package assets

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker info: %v", err)
	}
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
