//go:build live

package wgs_e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestWGSSuccessInspectReleaseResume(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageWGSPins(t, dir)
	g, err := gobble.Compose(assets.WGS())
	if err != nil {
		t.Fatalf("Compose(assets.WGS()) error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 2); err != nil {
		t.Fatalf("Run(assets.WGS()) error = %v", err)
	}
	assertOccupied(t, dir)
	if remaining := inspectJSONL(t, dir, gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("success remaining got %#v, want empty", remaining)
	}
	for _, rel := range []string{
		"work/multiqc/multiqc_report.html",
		"work/sample1/samtools-sort/aligned.bam",
		"work/sample2/samtools-sort/aligned.bam",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
	err = gobble.Run(t.Context(), g, dir, 0)
	requireRunError(t, "second Run", err, gobble.DefectOccupiedWorkspace, "")

	forceDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	run := inspectObject(t, dir, gobble.ViewRun)
	occ, _ := run["occupancy"].(map[string]any)
	if occ["active"] != false {
		t.Fatalf("released occupancy got %#v", occ)
	}

	if err := gobble.Resume(t.Context(), g, dir, 2); err != nil {
		t.Fatalf("Resume(assets.WGS()) error = %v", err)
	}
	assertOccupied(t, dir)
	if remaining := inspectJSONL(t, dir, gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("resume remaining got %#v, want empty", remaining)
	}
	reuse := inspectJSONL(t, dir, gobble.ViewReuse)
	if len(reuse) == 0 {
		t.Fatalf("resume reuse empty, want reused identities")
	}
	for _, rec := range reuse {
		if rec["decision"] != "reused" || rec["reason"] != "reused-identity-matched" {
			t.Fatalf("resume reuse got %#v, want reused-identity-matched", rec)
		}
	}
}

func TestThinFailFixtureInspectReleaseResume(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	g, err := gobble.Compose(thinFailFixture())
	if err != nil {
		t.Fatalf("Compose(thin fail fixture) error = %v", err)
	}
	err = gobble.Run(t.Context(), g, dir, 2)
	requireRunError(t, "contained failure", err, gobble.DefectFailed, "fail")
	assertOccupied(t, dir)
	if _, statErr := os.Stat(filepath.Join(dir, "out", "fail.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("failed dest was published")
	}
	got, readErr := os.ReadFile(filepath.Join(dir, "out", "ok.txt"))
	if readErr != nil {
		t.Fatalf("independent dest: %v", readErr)
	}
	if string(got) != "ok\n" && string(got) != "ok" {
		t.Fatalf("independent dest got %q, want ok", got)
	}

	remaining := instanceByID(inspectJSONL(t, dir, gobble.ViewRemaining))
	if remaining["fail"] == nil || remaining["fail"]["remaining"] != true {
		t.Fatalf("fail remaining got %#v", remaining["fail"])
	}

	forceDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	err = gobble.Resume(t.Context(), g, dir, 2)
	requireRunError(t, "resume remaining", err, gobble.DefectFailed, "fail")
	assertOccupied(t, dir)
	reuse := instanceByID(inspectJSONL(t, dir, gobble.ViewReuse))
	if reuse["fail"] == nil || reuse["fail"]["decision"] != "rerun" {
		t.Fatalf("fail resume reuse got %#v", reuse["fail"])
	}
	if remaining := instanceByID(inspectJSONL(t, dir, gobble.ViewRemaining)); remaining["fail"] == nil {
		t.Fatalf("resume remaining missing fail: %#v", remaining)
	}
}

func thinFailFixture() *gobble.Pipeline {
	p := gobble.NewPipeline("thin-fail")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "fail",
		Command: []string{"sh", "-c", "exit 1"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "fail", Ext: ".txt"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "ok",
		Command: []string{"sh", "-c", "echo ok > out/ok.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "ok", Ext: ".txt"},
		}},
	})
	return p
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker info: %v", err)
	}
}

func stageWGSPins(t *testing.T, dir string) {
	t.Helper()
	pins := []struct {
		pin assets.Pin
		rel string
	}{
		{assets.PinWGSTest1FASTQ, "in/test_1.fastq.gz"},
		{assets.PinWGSTest2FASTQ, "in/test_2.fastq.gz"},
		{assets.PinWGSGenomeFASTA, "in/genome.fasta"},
		{assets.PinWGSGenomeFAI, "in/genome.fasta.fai"},
	}
	for _, p := range pins {
		src, err := assets.FetchPin(p.pin)
		if err != nil {
			t.Fatalf("download %s: %v", p.pin.URL, err)
		}
		stageFile(t, dir, p.rel, src)
	}
}

func stageFile(t *testing.T, workspace, rel, src string) {
	t.Helper()
	dst := filepath.Join(workspace, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dst, err)
	}
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatalf("copy %s: %v", dst, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("Close(%s) error = %v", dst, err)
	}
}

func assertOccupied(t *testing.T, dir string) {
	t.Helper()
	run := inspectObject(t, dir, gobble.ViewRun)
	occ, _ := run["occupancy"].(map[string]any)
	if occ["active"] != true {
		t.Fatalf("occupancy got %#v, want active", occ)
	}
}

func forceDeadOwner(t *testing.T, workspace string) {
	t.Helper()
	path := filepath.Join(workspace, engine.ControlDir, engine.RunIdentityFile)
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

func inspectObject(t *testing.T, workspace string, view gobble.View) map[string]any {
	t.Helper()
	data, err := gobble.Inspect(workspace, view, "")
	if err != nil {
		t.Fatalf("Inspect(%s) error = %v", view, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Inspect(%s) JSON: %v\n%s", view, err, data)
	}
	return out
}

func inspectJSONL(t *testing.T, workspace string, view gobble.View) []map[string]any {
	t.Helper()
	data, err := gobble.Inspect(workspace, view, "")
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

func instanceByID(recs []map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(recs))
	for _, rec := range recs {
		id, _ := rec["identity"].(string)
		out[id] = rec
	}
	return out
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func requireRunError(t *testing.T, name string, err error, code gobble.DefectCode, unit string) *gobble.Error {
	t.Helper()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("case %s: error = %v, want *Error", name, err)
	}
	found := false
	for _, d := range ge.Defects {
		if d.Code == code && (unit == "" || d.Unit == unit) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("case %s: defects %#v, want code %s unit %q", name, ge.Defects, code, unit)
	}
	return ge
}
