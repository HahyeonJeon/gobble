package wgs_e2e_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

const (
	wgsE2ESpineStrelka  = "quay.io/biocontainers/strelka:2.9.10--h9ee0642_1"
	wgsE2ESpineBCFTools = "quay.io/biocontainers/bcftools:1.24--h118bc1c_2"
	wgsE2ESpineFAI      = "in/genome.fasta.fai"
	wgsE2ESpineVCF      = "work/sample.variants.vcf.gz"
	wgsE2ESpinePileup   = "work/pileup.vcf"
	wgsE2ESpineCalls    = "work/calls.vcf"
)

// nf-core strelka/germline move-to-regular-file. Run dir is scratch, not a bind.
const wgsE2EStrelkaScript = `set -e
configureStrelkaGermlineWorkflow.py --bam work/aligned.bam --referenceFasta in/genome.fasta --runDir strelka
sed -i s/"isEmail = isLocalSmtp()"/"isEmail = False"/g strelka/runWorkflow.py
python strelka/runWorkflow.py -m local -j 1
mv strelka/results/variants/variants.vcf.gz work/sample.variants.vcf.gz`

type wgsSpineAlign struct {
	fasta gobble.Handle
	faidx *gobble.Task
	sort  *gobble.Task
	bai   *gobble.Task
}

func addWGSSpineAlign(p *gobble.Pipeline) wgsSpineAlign {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	inFASTA := p.AddInput("fasta", fasta)
	inR1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"})
	inR2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"})

	index := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Image:   wgsE2EThinBWA,
		Command: []string{"bwa", "index", wgsE2EStagedFASTA},
		Inputs:  []gobble.Bind{{Name: "fasta", From: inFASTA}},
		Outputs: []gobble.Bind{
			{Name: "amb", Spec: fasta.Append(".amb")},
			{Name: "ann", Spec: fasta.Append(".ann")},
			{Name: "bwt", Spec: fasta.Append(".bwt")},
			{Name: "pac", Spec: fasta.Append(".pac")},
			{Name: "sa", Spec: fasta.Append(".sa")},
		},
	})

	sam := gobble.PathSpec{Dir: gobble.Dir("work"), Name: "aligned", Ext: ".sam"}
	mem := p.AddTask(gobble.TaskSpec{
		Name:  "mem",
		Image: wgsE2EThinBWA,
		Command: []string{
			"bwa", "mem", "-o", wgsE2EThinSAM,
			wgsE2EStagedFASTA, wgsE2EStagedR1, wgsE2EStagedR2,
		},
		Inputs: []gobble.Bind{
			{Name: "fasta", From: inFASTA},
			{Name: "amb", From: index.Out("amb")},
			{Name: "ann", From: index.Out("ann")},
			{Name: "bwt", From: index.Out("bwt")},
			{Name: "pac", From: index.Out("pac")},
			{Name: "sa", From: index.Out("sa")},
			{Name: "r1", From: inR1},
			{Name: "r2", From: inR2},
		},
		Outputs: []gobble.Bind{{Name: "sam", Spec: sam}},
	})

	bam := gobble.PathSpec{Dir: gobble.Dir("work"), Name: "aligned", Ext: ".bam"}
	sort := p.AddTask(gobble.TaskSpec{
		Name:    "sort",
		Image:   wgsE2EThinSamtools,
		Command: []string{"samtools", "sort", "-o", wgsE2EThinBAM, wgsE2EThinSAM},
		Inputs:  []gobble.Bind{{Name: "sam", From: mem.Out("sam")}},
		Outputs: []gobble.Bind{{Name: "bam", Spec: bam}},
	})

	// BAI is a second-task related-file From. Same-task From is a cycle.
	bai := p.AddTask(gobble.TaskSpec{
		Name:    "bai",
		Image:   wgsE2EThinSamtools,
		Command: []string{"samtools", "index", wgsE2EThinBAM, wgsE2EThinBAI},
		Inputs:  []gobble.Bind{{Name: "bam", From: sort.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "bai",
			Spec: gobble.PathSpec{Ext: ".bai"},
			From: sort.Out("bam"),
		}},
	})

	faidx := p.AddTask(gobble.TaskSpec{
		Name:    "faidx",
		Image:   wgsE2EThinSamtools,
		Command: []string{"samtools", "faidx", wgsE2EStagedFASTA},
		Inputs:  []gobble.Bind{{Name: "fasta", From: inFASTA}},
		Outputs: []gobble.Bind{{Name: "fai", Spec: fasta.Append(".fai")}},
	})

	return wgsSpineAlign{fasta: inFASTA, faidx: faidx, sort: sort, bai: bai}
}

func callerBinds(a wgsSpineAlign) []gobble.Bind {
	return []gobble.Bind{
		{Name: "fasta", From: a.fasta},
		{Name: "fai", From: a.faidx.Out("fai")},
		{Name: "bam", From: a.sort.Out("bam")},
		{Name: "bai", From: a.bai.Out("bai")},
	}
}

func wgsE2ESpineStrelkaPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("wgs-e2e-spine")
	a := addWGSSpineAlign(p)
	vcf := gobble.PathSpec{Dir: gobble.Dir("work"), Name: "sample", Ext: ".variants.vcf.gz"}
	p.AddTask(gobble.TaskSpec{
		Name:    "strelka",
		Image:   wgsE2ESpineStrelka,
		Command: []string{"sh", "-c", wgsE2EStrelkaScript},
		Inputs:  callerBinds(a),
		Outputs: []gobble.Bind{{Name: "vcf", Spec: vcf}},
	})
	return p
}

func wgsE2ESpineBCFToolsPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("wgs-e2e-spine-bcftools")
	a := addWGSSpineAlign(p)
	pileup := gobble.PathSpec{Dir: gobble.Dir("work"), Name: "pileup", Ext: ".vcf"}
	mpileup := p.AddTask(gobble.TaskSpec{
		Name:    "mpileup",
		Image:   wgsE2ESpineBCFTools,
		Command: []string{"bcftools", "mpileup", "-f", wgsE2EStagedFASTA, "-o", wgsE2ESpinePileup, wgsE2EThinBAM},
		Inputs:  callerBinds(a),
		Outputs: []gobble.Bind{{Name: "pileup", Spec: pileup}},
	})
	calls := gobble.PathSpec{Dir: gobble.Dir("work"), Name: "calls", Ext: ".vcf"}
	p.AddTask(gobble.TaskSpec{
		Name:    "call",
		Image:   wgsE2ESpineBCFTools,
		Command: []string{"bcftools", "call", "-mv", "-o", wgsE2ESpineCalls, wgsE2ESpinePileup},
		Inputs:  []gobble.Bind{{Name: "pileup", From: mpileup.Out("pileup")}},
		Outputs: []gobble.Bind{{Name: "calls", Spec: calls}},
	})
	return p
}

func TestWGSSpinePlan(t *testing.T) {
	g, err := gobble.Compose(wgsE2ESpineStrelkaPipeline())
	if err != nil {
		t.Fatalf("Compose(strelka) error = %v, want nil", err)
	}
	if err := gobble.Validate(g); err != nil {
		t.Fatalf("Validate(strelka) error = %v, want nil", err)
	}
	var buf bytes.Buffer
	if _, err := gobble.BuildPlan(g, gobble.WriteTo(&buf)); err != nil {
		t.Fatalf("BuildPlan(strelka) error = %v, want nil", err)
	}
	assertSpineStrelkaPlan(t, buf.Bytes())

	g2, err := gobble.Compose(wgsE2ESpineBCFToolsPipeline())
	if err != nil {
		t.Fatalf("Compose(bcftools) error = %v, want nil", err)
	}
	if err := gobble.Validate(g2); err != nil {
		t.Fatalf("Validate(bcftools) error = %v, want nil", err)
	}
	buf.Reset()
	if _, err := gobble.BuildPlan(g2, gobble.WriteTo(&buf)); err != nil {
		t.Fatalf("BuildPlan(bcftools) error = %v, want nil", err)
	}
	assertSpineBCFToolsPlan(t, buf.Bytes())
}

func TestWGSSpineRun(t *testing.T) {
	requireDocker(t)

	rec := spineRecord{
		BWATag:       wgsE2EThinBWA,
		SamtoolsTag:  wgsE2EThinSamtools,
		StrelkaTag:   wgsE2ESpineStrelka,
		NewWorkspace: true,
	}

	dir := t.TempDir()
	if _, err := os.Stat(filepath.Join(dir, ".gobble", "run.json")); !os.IsNotExist(err) {
		t.Fatalf("new workspace run.json: %v, want not exist", err)
	}
	stageWGSSpineWorkspace(t, dir)
	rec.Inputs = thinSliceInputFacts(t, dir)

	g := mustCompose(wgsE2ESpineStrelkaPipeline)(t)
	err := gobble.Run(g, dir, 1)
	rec.StrelkaAttempted = true
	rec.StrelkaError = formatRunError(err)
	rec.TaskLogs = spineLogs(dir, "index", "mem", "sort", "bai", "faidx", "strelka")
	rec.TaskState = thinSliceTaskState(dir)

	if err == nil && regularNonEmpty(dir, wgsE2ESpineVCF) {
		assertSpineFiles(t, dir)
		assertSpineFAI(t, dir)
		assertGzipVCF(t, dir, wgsE2ESpineVCF)
		rec.Caller = "strelka"
		rec.VCFPaths = []string{wgsE2ESpineVCF}
		rec.FilesOK = true
		rec.BWADigest = dockerImageDigest(t, wgsE2EThinBWA)
		rec.SamtoolsDigest = dockerImageDigest(t, wgsE2EThinSamtools)
		rec.StrelkaDigest = dockerImageDigest(t, wgsE2ESpineStrelka)
		writeSpineRecord(t, rec)
		t.Logf("strelka published %s; digest %s", wgsE2ESpineVCF, rec.StrelkaDigest)
		return
	}

	rec.StrelkaFailure = classifyStrelkaFailure(err, dir)
	t.Logf("strelka isolate failed: %s", rec.StrelkaFailure)

	fallback := t.TempDir()
	if _, err := os.Stat(filepath.Join(fallback, ".gobble", "run.json")); !os.IsNotExist(err) {
		t.Fatalf("fallback workspace run.json: %v, want not exist", err)
	}
	stageWGSSpineWorkspace(t, fallback)
	rec.FallbackNewWorkspace = true
	rec.BCFToolsTag = wgsE2ESpineBCFTools

	g2 := mustCompose(wgsE2ESpineBCFToolsPipeline)(t)
	err2 := gobble.Run(g2, fallback, 1)
	rec.BCFToolsError = formatRunError(err2)
	rec.TaskLogs += spineLogs(fallback, "index", "mem", "sort", "bai", "faidx", "mpileup", "call")
	rec.TaskState += "\n" + thinSliceTaskState(fallback)
	rec.BWADigest = dockerImageDigest(t, wgsE2EThinBWA)
	rec.SamtoolsDigest = dockerImageDigest(t, wgsE2EThinSamtools)
	rec.StrelkaDigest = dockerImageDigest(t, wgsE2ESpineStrelka)

	if err2 != nil || !regularNonEmpty(fallback, wgsE2ESpinePileup) || !regularNonEmpty(fallback, wgsE2ESpineCalls) {
		rec.Stop = "D5 joint hole: both callers failed isolate. Do not invent directory-output or HOME APIs."
		writeSpineRecord(t, rec)
		t.Fatalf("D5 joint hole: strelka %s; bcftools error = %v\n%s\n%s",
			rec.StrelkaFailure, err2, rec.TaskState, rec.TaskLogs)
	}

	assertSpineFiles(t, fallback)
	assertSpineFAI(t, fallback)
	assertTextVCF(t, fallback, wgsE2ESpinePileup)
	assertTextVCF(t, fallback, wgsE2ESpineCalls)
	rec.Caller = "bcftools"
	rec.Substitution = true
	rec.VCFPaths = []string{wgsE2ESpinePileup, wgsE2ESpineCalls}
	rec.FilesOK = true
	rec.BCFToolsDigest = dockerImageDigest(t, wgsE2ESpineBCFTools)
	writeSpineRecord(t, rec)
	t.Logf("bcftools substitution after strelka isolate failure: %s; digest %s",
		rec.StrelkaFailure, rec.BCFToolsDigest)
}

func assertSpineStrelkaPlan(t *testing.T, raw []byte) {
	t.Helper()
	got := decodeSpinePlan(t, raw)
	wantCmd := map[string][]string{
		"index":   {"bwa", "index", wgsE2EStagedFASTA},
		"mem":     {"bwa", "mem", "-o", wgsE2EThinSAM, wgsE2EStagedFASTA, wgsE2EStagedR1, wgsE2EStagedR2},
		"sort":    {"samtools", "sort", "-o", wgsE2EThinBAM, wgsE2EThinSAM},
		"bai":     {"samtools", "index", wgsE2EThinBAM, wgsE2EThinBAI},
		"faidx":   {"samtools", "faidx", wgsE2EStagedFASTA},
		"strelka": {"sh", "-c", wgsE2EStrelkaScript},
	}
	wantImage := map[string]string{
		"index":   wgsE2EThinBWA,
		"mem":     wgsE2EThinBWA,
		"sort":    wgsE2EThinSamtools,
		"bai":     wgsE2EThinSamtools,
		"faidx":   wgsE2EThinSamtools,
		"strelka": wgsE2ESpineStrelka,
	}
	assertSpinePlanTasks(t, got, wantCmd, wantImage, "strelka")
	if path := thinSliceOutputPath(t, got, "faidx", "fai"); path != wgsE2ESpineFAI {
		t.Fatalf("faidx.fai path got %q, want %q", path, wgsE2ESpineFAI)
	}
	if path := thinSliceOutputPath(t, got, "bai", "bai"); path != wgsE2EThinBAI {
		t.Fatalf("bai.bai path got %q, want %q", path, wgsE2EThinBAI)
	}
	if path := thinSliceOutputPath(t, got, "strelka", "vcf"); path != wgsE2ESpineVCF {
		t.Fatalf("strelka.vcf path got %q, want %q", path, wgsE2ESpineVCF)
	}
	for _, task := range got.Tasks {
		if task.ID == "mpileup" || task.ID == "call" {
			t.Fatalf("strelka-first plan has %s; do not start on bcftools", task.ID)
		}
	}
}

func assertSpineBCFToolsPlan(t *testing.T, raw []byte) {
	t.Helper()
	got := decodeSpinePlan(t, raw)
	wantCmd := map[string][]string{
		"index":   {"bwa", "index", wgsE2EStagedFASTA},
		"mem":     {"bwa", "mem", "-o", wgsE2EThinSAM, wgsE2EStagedFASTA, wgsE2EStagedR1, wgsE2EStagedR2},
		"sort":    {"samtools", "sort", "-o", wgsE2EThinBAM, wgsE2EThinSAM},
		"bai":     {"samtools", "index", wgsE2EThinBAM, wgsE2EThinBAI},
		"faidx":   {"samtools", "faidx", wgsE2EStagedFASTA},
		"mpileup": {"bcftools", "mpileup", "-f", wgsE2EStagedFASTA, "-o", wgsE2ESpinePileup, wgsE2EThinBAM},
		"call":    {"bcftools", "call", "-mv", "-o", wgsE2ESpineCalls, wgsE2ESpinePileup},
	}
	wantImage := map[string]string{
		"index":   wgsE2EThinBWA,
		"mem":     wgsE2EThinBWA,
		"sort":    wgsE2EThinSamtools,
		"bai":     wgsE2EThinSamtools,
		"faidx":   wgsE2EThinSamtools,
		"mpileup": wgsE2ESpineBCFTools,
		"call":    wgsE2ESpineBCFTools,
	}
	assertSpinePlanTasks(t, got, wantCmd, wantImage, "")
	if path := thinSliceOutputPath(t, got, "mpileup", "pileup"); path != wgsE2ESpinePileup {
		t.Fatalf("mpileup.pileup path got %q, want %q", path, wgsE2ESpinePileup)
	}
	if path := thinSliceOutputPath(t, got, "call", "calls"); path != wgsE2ESpineCalls {
		t.Fatalf("call.calls path got %q, want %q", path, wgsE2ESpineCalls)
	}
	for _, task := range got.Tasks {
		if task.ID == "strelka" {
			t.Fatalf("fallback plan has strelka; fallback is bcftools only")
		}
		if thinSliceHasShell(task.Command) {
			t.Fatalf("%s command has sh -c: %#v", task.ID, task.Command)
		}
	}
}

func decodeSpinePlan(t *testing.T, raw []byte) thinSlicePlan {
	t.Helper()
	var got thinSlicePlan
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return got
}

func assertSpinePlanTasks(t *testing.T, got thinSlicePlan, wantCmd map[string][]string, wantImage map[string]string, shellTask string) {
	t.Helper()
	if len(got.Tasks) != len(wantCmd) {
		t.Fatalf("tasks got %d, want %d", len(got.Tasks), len(wantCmd))
	}
	seen := map[string]bool{}
	for _, task := range got.Tasks {
		want, ok := wantCmd[task.ID]
		if !ok {
			t.Fatalf("unexpected task id %q", task.ID)
		}
		seen[task.ID] = true
		if !reflect.DeepEqual(task.Command, want) {
			t.Fatalf("%s command got %#v, want %#v", task.ID, task.Command, want)
		}
		if task.Image != wantImage[task.ID] {
			t.Fatalf("%s image got %q, want %q", task.ID, task.Image, wantImage[task.ID])
		}
		hasShell := thinSliceHasShell(task.Command)
		if task.ID == shellTask {
			if !hasShell {
				t.Fatalf("%s command missing labeled sh -c: %#v", task.ID, task.Command)
			}
		} else if hasShell {
			t.Fatalf("%s command has sh -c: %#v", task.ID, task.Command)
		}
		assertSpineArgv(t, task, shellTask)
	}
	for id := range wantCmd {
		if !seen[id] {
			t.Fatalf("missing task %q", id)
		}
	}
}

func assertSpineArgv(t *testing.T, task thinSlicePlanTask, shellTask string) {
	t.Helper()
	binds := thinSliceBindPaths(task)
	for _, tok := range task.Command {
		if tok == "/work" || strings.HasPrefix(tok, "/work/") || strings.Contains(tok, " /work/") {
			t.Fatalf("%s argv token contains /work: %q", task.ID, tok)
		}
		if strings.Contains(tok, "..") {
			t.Fatalf("%s argv token %q contains ..", task.ID, tok)
		}
		if filepath.IsAbs(tok) || strings.HasPrefix(tok, "/") {
			t.Fatalf("%s argv token %q is an absolute path", task.ID, tok)
		}
		if task.ID == shellTask {
			continue
		}
		if !thinSlicePathShaped(tok) {
			continue
		}
		if !binds[tok] {
			t.Fatalf("%s path-shaped argv token %q is not a declared bind; binds %v", task.ID, tok, bindPathList(binds))
		}
	}
	if task.ID != shellTask {
		return
	}
	script := ""
	for i, tok := range task.Command {
		if tok == "-c" && i+1 < len(task.Command) {
			script = task.Command[i+1]
			break
		}
	}
	for _, p := range []string{wgsE2EThinBAM, wgsE2EStagedFASTA, wgsE2ESpineVCF} {
		if !strings.Contains(script, p) {
			t.Fatalf("strelka script missing %q", p)
		}
	}
	if strings.Contains(script, "--runDir work/") || strings.Contains(script, "--runDir "+wgsE2ESpineVCF) {
		t.Fatalf("strelka script declares the run directory as an output path")
	}
}

func stageWGSSpineWorkspace(t *testing.T, dir string) {
	t.Helper()
	stageWGSFixture(t, dir)
	if _, err := os.Stat(filepath.Join(dir, wgsE2ESpineFAI)); !os.IsNotExist(err) {
		t.Fatalf("staged FAI: %v, want not exist", err)
	}
}

func assertSpineFiles(t *testing.T, workspace string) {
	t.Helper()
	assertThinSliceFiles(t, workspace)
	assertThinSliceRealOutputs(t, workspace)
	for _, rel := range []string{wgsE2ESpineFAI} {
		if !regularNonEmpty(workspace, rel) {
			t.Fatalf("published %s missing or empty", rel)
		}
	}
}

func assertSpineFAI(t *testing.T, workspace string) {
	t.Helper()
	path := filepath.Join(workspace, filepath.FromSlash(wgsE2ESpineFAI))
	want := wgsE2EFileByName(t)["genome.fasta.fai"]
	sum, size := wgsE2EHashFile(t, path)
	if size != want.Bytes || sum != want.SHA256 {
		t.Fatalf("published FAI size %d sha256 %s, want %d %s", size, sum, want.Bytes, want.SHA256)
	}
}

func assertGzipVCF(t *testing.T, workspace, rel string) {
	t.Helper()
	if !regularNonEmpty(workspace, rel) {
		t.Fatalf("published %s missing or empty", rel)
	}
	got := readPrefix(t, filepath.Join(workspace, filepath.FromSlash(rel)), 2)
	if len(got) < 2 || got[0] != 0x1f || got[1] != 0x8b {
		t.Fatalf("%s magic got %x, want gzip 1f8b", rel, got)
	}
}

func assertTextVCF(t *testing.T, workspace, rel string) {
	t.Helper()
	if !regularNonEmpty(workspace, rel) {
		t.Fatalf("published %s missing or empty", rel)
	}
	got := readPrefix(t, filepath.Join(workspace, filepath.FromSlash(rel)), 2)
	if string(got) != "##" && (len(got) == 0 || got[0] != '#') {
		t.Fatalf("%s prefix got %q, want VCF header", rel, got)
	}
}

func regularNonEmpty(workspace, rel string) bool {
	info, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(rel)))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func classifyStrelkaFailure(err error, workspace string) string {
	path := filepath.Join(workspace, filepath.FromSlash(wgsE2ESpineVCF))
	info, statErr := os.Stat(path)
	switch {
	case statErr == nil && info.IsDir():
		return "directory-only at " + wgsE2ESpineVCF
	case statErr == nil && !info.Mode().IsRegular():
		return "declared VCF is not a regular file"
	}
	isolateDir := filepath.Join(workspace, ".gobble", "tasks", "strelka", "_", "0", "1", "work", "strelka")
	if di, derr := os.Stat(isolateDir); derr == nil && di.IsDir() && (statErr != nil || info.Size() == 0) {
		if err != nil {
			return "directory-only: isolate has strelka/ run dir; " + formatRunError(err)
		}
		return "directory-only: isolate has strelka/ run dir, missing " + wgsE2ESpineVCF
	}
	if err != nil {
		return "nonzero: " + formatRunError(err)
	}
	return "missing declared VCF " + wgsE2ESpineVCF
}

func formatRunError(err error) string {
	if err == nil {
		return ""
	}
	return formatRunDefects(err)
}

func spineLogs(workspace string, ids ...string) string {
	var b strings.Builder
	for _, id := range ids {
		for _, name := range []string{"stderr", "stdout"} {
			path := filepath.Join(workspace, ".gobble", "tasks", id, "_", "0", "1", name)
			data, err := os.ReadFile(path)
			if err != nil || len(data) == 0 {
				continue
			}
			b.WriteString(id)
			b.WriteByte('/')
			b.WriteString(name)
			b.WriteString(":\n")
			b.Write(data)
			if data[len(data)-1] != '\n' {
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

type spineRecord struct {
	BWATag               string
	BWADigest            string
	SamtoolsTag          string
	SamtoolsDigest       string
	StrelkaTag           string
	StrelkaDigest        string
	BCFToolsTag          string
	BCFToolsDigest       string
	Inputs               []thinSliceInputFact
	Caller               string
	VCFPaths             []string
	NewWorkspace         bool
	FallbackNewWorkspace bool
	StrelkaAttempted     bool
	Substitution         bool
	StrelkaFailure       string
	StrelkaError         string
	BCFToolsError        string
	FilesOK              bool
	TaskLogs             string
	TaskState            string
	Stop                 string
}

func writeSpineRecord(t *testing.T, rec spineRecord) {
	t.Helper()
	path := wgsSpineRecordPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(formatSpineRecord(rec)), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	t.Logf("wgs-spine record: %s", path)
}

func wgsSpineRecordPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(wgsE2ERecordDir(t), "wgs-spine-record.md")
}

func formatSpineRecord(rec spineRecord) string {
	var b strings.Builder
	b.WriteString("# WGS spine proof record\n\n")
	b.WriteString("## Workspace\n\n")
	b.WriteString("- new workspace: " + strconv.FormatBool(rec.NewWorkspace) + "\n")
	b.WriteString("- reused thin-slice workspace: no\n")
	b.WriteString("- fallback new workspace: " + strconv.FormatBool(rec.FallbackNewWorkspace) + "\n\n")
	b.WriteString("## Caller\n\n")
	b.WriteString("- first caller: strelka\n")
	b.WriteString("- strelka attempted: " + strconv.FormatBool(rec.StrelkaAttempted) + "\n")
	b.WriteString("- published by: " + rec.Caller + "\n")
	b.WriteString("- substitution: " + strconv.FormatBool(rec.Substitution) + "\n")
	if rec.StrelkaFailure != "" {
		b.WriteString("- strelka isolate: " + rec.StrelkaFailure + "\n")
	} else if rec.Caller == "strelka" {
		b.WriteString("- strelka isolate: published " + wgsE2ESpineVCF + "\n")
	}
	if rec.StrelkaError != "" {
		b.WriteString("- strelka error: " + rec.StrelkaError + "\n")
	}
	if rec.BCFToolsError != "" {
		b.WriteString("- bcftools error: " + rec.BCFToolsError + "\n")
	}
	b.WriteString("\n## VCF paths\n\n")
	if len(rec.VCFPaths) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, p := range rec.VCFPaths {
			b.WriteString("- " + p + "\n")
		}
	}
	b.WriteString("\n## Images\n\n")
	b.WriteString("- bwa tag: " + rec.BWATag + "\n")
	b.WriteString("- bwa digest: " + rec.BWADigest + "\n")
	b.WriteString("- samtools tag: " + rec.SamtoolsTag + "\n")
	b.WriteString("- samtools digest: " + rec.SamtoolsDigest + "\n")
	b.WriteString("- strelka tag: " + rec.StrelkaTag + "\n")
	b.WriteString("- strelka digest: " + rec.StrelkaDigest + "\n")
	if rec.BCFToolsTag != "" {
		b.WriteString("- bcftools tag: " + rec.BCFToolsTag + "\n")
		b.WriteString("- bcftools digest: " + rec.BCFToolsDigest + "\n")
	}
	b.WriteString("\n## Inputs\n\n")
	for _, in := range rec.Inputs {
		b.WriteString("- " + in.Path + " sha256 " + in.SHA256 + " bytes " + strconv.FormatInt(in.Bytes, 10) + "\n")
	}
	b.WriteString("\n## Run\n\n")
	b.WriteString("- spine files and FAI: " + strconv.FormatBool(rec.FilesOK) + "\n")
	if rec.Stop != "" {
		b.WriteString("\n## Stop\n\n")
		b.WriteString(rec.Stop + "\n")
	}
	if rec.TaskState != "" {
		b.WriteString("\n## Task state\n\n```json\n")
		b.WriteString(rec.TaskState)
		if !strings.HasSuffix(rec.TaskState, "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("```\n")
	}
	if rec.TaskLogs != "" {
		b.WriteString("\n## Task logs\n\n```\n")
		b.WriteString(rec.TaskLogs)
		b.WriteString("```\n")
	}
	return b.String()
}
