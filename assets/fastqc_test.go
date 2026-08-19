package assets

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

// Documented nf-core modules-branch sarscov2 R1. Public reuse: modules
// README lists test_{1,2}.fastq.gz as paired-end reads; test-datasets
// LICENSE is MIT, copyright 2018 nf-core.
var pinSARSCoV2R1 = Pin{
	Name:   "test_1.fastq.gz",
	URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/sarscov2/illumina/fastq/test_1.fastq.gz",
	Bytes:  9413,
	SHA256: "0515ba304cb1bf7abcdd9c156b6affad7e580273f983dfed2e8fe2d918e800ff",
}

func TestFastQCStandaloneComposeBuildPlan(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	opts := FastQCOptions{
		ExtraArgs: []string{"--kmers", "7"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := FastQCPipeline(reads, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "fastqc")
	if task.Name != "fastqc" {
		t.Fatalf("task name = %q, want fastqc", task.Name)
	}
	if task.Image != fastqcImage {
		t.Fatalf("image = %q, want %q", task.Image, fastqcImage)
	}
	if !containsAll(task.Command, "fastqc", "--outdir", "work/fastqc", "--noextract", "--threads", "2", "in/test_1.fastq.gz", "--kmers", "7") {
		t.Fatalf("command = %#v, want named flags, input, extra-args", task.Command)
	}
	if lastN := task.Command[len(task.Command)-2:]; lastN[0] != "--kmers" || lastN[1] != "7" {
		t.Fatalf("extra-args last tokens = %#v, want [--kmers 7]", lastN)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "html", "work/fastqc/test_1_fastqc.html")
	assertIOPath(t, task.Outputs, "zip", "work/fastqc/test_1_fastqc.zip")
}

func TestFastQCNestedModule(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("reads", reads)
	mod := AddModule(p, "raw")
	ports := AddFastQC(mod, h, FastQCOptions{ExtraArgs: []string{"--quiet"}})
	if ports.HTML.IsZero() || ports.Zip.IsZero() {
		t.Fatalf("ports HTML/Zip IsZero = %v/%v, want false", ports.HTML.IsZero(), ports.Zip.IsZero())
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "raw.fastqc")
	if task.Name != "fastqc" {
		t.Fatalf("nested name = %q, want fastqc", task.Name)
	}
	if task.Module != "raw" {
		t.Fatalf("nested module = %q, want raw", task.Module)
	}
	if !containsAll(task.Command, "--quiet") {
		t.Fatalf("command = %#v, want extra-args --quiet", task.Command)
	}
}

func TestFastQCStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, pinSARSCoV2R1)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_1.fastq.gz", src)
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	p := FastQCPipeline(reads, FastQCOptions{
		ExtraArgs: []string{"--quiet"},
		Resources: gobble.Resources{CPU: 1},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{"work/fastqc/test_1_fastqc.html", "work/fastqc/test_1_fastqc.zip"} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func TestFastQCExtraArgsResume(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, pinSARSCoV2R1)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_1.fastq.gz", src)
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	opts := FastQCOptions{ExtraArgs: []string{"--quiet"}, Resources: gobble.Resources{CPU: 1}}
	g, err := gobble.Compose(FastQCPipeline(reads, opts))
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	forceDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	opts.ExtraArgs = []string{"--quiet", "--kmers", "7"}
	g2, err := gobble.Compose(FastQCPipeline(reads, opts))
	if err != nil {
		t.Fatalf("Compose(changed extra-args) error = %v", err)
	}
	if err := gobble.Resume(g2, dir, 1); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	instances := inspectJSONL(t, dir, "instances")
	if len(instances) == 0 {
		t.Fatalf("Inspect(instances) empty")
	}
	if instances[0]["reuse_reason"] != "command-or-script-changed" {
		t.Fatalf("reuse_reason = %#v, want command-or-script-changed", instances[0]["reuse_reason"])
	}
}

type planTaskRec struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Module  string         `json:"module"`
	Command []string       `json:"command"`
	Image   string         `json:"image"`
	Params  []planParamRec `json:"params"`
	Inputs  []planIORec    `json:"inputs"`
	Outputs []planIORec    `json:"outputs"`
}

type planParamRec struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type planIORec struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Members []struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"members"`
}

func mustPlanJSON(t *testing.T, p *gobble.Pipeline) []byte {
	t.Helper()
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
	var buf bytes.Buffer
	if _, err := gobble.BuildPlan(g, gobble.WriteTo(&buf)); err != nil {
		t.Fatalf("BuildPlan() error = %v, want nil", err)
	}
	return buf.Bytes()
}

func planTask(t *testing.T, raw []byte, id string) planTaskRec {
	t.Helper()
	var decoded struct {
		Tasks []planTaskRec `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal plan: %v", err)
	}
	for _, task := range decoded.Tasks {
		if task.ID == id {
			return task
		}
	}
	t.Fatalf("plan missing task %q in %#v", id, decoded.Tasks)
	return planTaskRec{}
}

func assertIOPath(t *testing.T, ios []planIORec, name, path string) {
	t.Helper()
	for _, io := range ios {
		if io.Name == name {
			if io.Path != path {
				t.Fatalf("%s path = %q, want %q", name, io.Path, path)
			}
			return
		}
	}
	t.Fatalf("missing IO %q in %#v", name, ios)
}

func assertUniqueParamNames(t *testing.T, params []planParamRec) {
	t.Helper()
	seen := make(map[string]bool, len(params))
	for _, p := range params {
		if seen[p.Name] {
			t.Fatalf("duplicate Param name %q", p.Name)
		}
		seen[p.Name] = true
	}
}

func containsAll(got []string, want ...string) bool {
	have := make(map[string]int, len(got))
	for _, s := range got {
		have[s]++
	}
	for _, s := range want {
		if have[s] == 0 {
			return false
		}
		have[s]--
	}
	return true
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("docker info: %v", err)
	}
}

func cachePin(t *testing.T, pin Pin) string {
	t.Helper()
	dest, err := FetchPin(pin)
	if err != nil {
		t.Skipf("download %s: %v", pin.URL, err)
	}
	return dest
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
