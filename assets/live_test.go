//go:build live

package assets

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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

func forceDeadOwner(t *testing.T, workspace string) {
	t.Helper()
	path := filepath.Join(workspace, ".gobble", "run.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var run map[string]any
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("Unmarshal run.json: %v", err)
	}
	occ, _ := run["occupancy"].(map[string]any)
	if occ == nil {
		occ = map[string]any{"active": true}
		run["occupancy"] = occ
	}
	host, err := os.Hostname()
	if err != nil {
		t.Fatalf("Hostname() error = %v", err)
	}
	occ["active"] = true
	occ["host"] = host
	occ["pid"] = deadPID(t)
	out, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func deadPID(t *testing.T) int {
	t.Helper()
	for pid := 1 << 22; pid > 2; pid-- {
		if err := syscall.Kill(pid, 0); err != nil && err != syscall.EPERM {
			return pid
		}
	}
	t.Fatal("no dead pid")
	return 0
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
