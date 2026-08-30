package local_e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

const (
	wgsE2EThinBWA         = "quay.io/biocontainers/bwa:0.7.18--h577a1d6_2"
	wgsE2EThinBWAFallback = "quay.io/biocontainers/bwa:0.7.17--hed695b0_7"
	wgsE2EThinSamtools    = "quay.io/biocontainers/samtools:1.24--h9dcdb79_1"
	wgsE2EThinSAM         = "work/aligned.sam"
	wgsE2EThinBAM         = "work/aligned.bam"
	wgsE2EThinBAI         = "work/aligned.bam.bai"
)

var wgsE2EThinIndexSiblings = []string{
	"in/genome.fasta.amb",
	"in/genome.fasta.ann",
	"in/genome.fasta.bwt",
	"in/genome.fasta.pac",
	"in/genome.fasta.sa",
}

var wgsE2EThinPublished = append(append([]string{}, wgsE2EThinIndexSiblings...),
	wgsE2EThinSAM, wgsE2EThinBAM, wgsE2EThinBAI)

// wgsE2EThinPipeline is the locked four-task public-API graph.
func wgsE2EThinPipeline() *gobble.Pipeline {
	return wgsE2EThinPipelineWithBWA(wgsE2EThinBWA)
}

func wgsE2EThinPipelineWithBWA(bwaImage string) *gobble.Pipeline {
	p := gobble.NewPipeline("wgs-e2e-thin")
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	inFASTA := p.AddInput("fasta", fasta)
	inR1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"})
	inR2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"})

	index := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Image:   bwaImage,
		Command: []string{"bwa", "index", wgsE2EStagedFASTA},
		Inputs:  []gobble.Bind{{Name: "fasta", From: inFASTA}},
		Outputs: []gobble.Bind{
			{Name: "amb", Spec: fasta.AppendExt(".amb")},
			{Name: "ann", Spec: fasta.AppendExt(".ann")},
			{Name: "bwt", Spec: fasta.AppendExt(".bwt")},
			{Name: "pac", Spec: fasta.AppendExt(".pac")},
			{Name: "sa", Spec: fasta.AppendExt(".sa")},
		},
	})

	sam := gobble.PathSpec{Dir: gobble.Dir("work"), Base: "aligned", Ext: ".sam"}
	mem := p.AddTask(gobble.TaskSpec{
		Name:  "mem",
		Image: bwaImage,
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

	bam := gobble.PathSpec{Dir: gobble.Dir("work"), Base: "aligned", Ext: ".bam"}
	sort := p.AddTask(gobble.TaskSpec{
		Name:    "sort",
		Image:   wgsE2EThinSamtools,
		Command: []string{"samtools", "sort", "-o", wgsE2EThinBAM, wgsE2EThinSAM},
		Inputs:  []gobble.Bind{{Name: "sam", From: mem.Out("sam")}},
		Outputs: []gobble.Bind{{Name: "bam", Spec: bam}},
	})

	// BAI is a second-task related-file From. Same-task From is a cycle.
	// Do not re-declare the BAM on this task.
	p.AddTask(gobble.TaskSpec{
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
	return p
}

// TestWGSThinSlicePlan is a demoted four-task graph. It is not the WGS
// product proof; that is tests/local-e2e executing assets/pipelines/wgs.
func TestWGSThinSlicePlan(t *testing.T) {
	p := wgsE2EThinPipeline()
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("Compose() graph = nil, want non-nil")
	}
	if err := gobble.Validate(g); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	var buf bytes.Buffer
	plan, err := gobble.BuildPlan(g, gobble.WriteTo(&buf))
	if err != nil {
		t.Fatalf("BuildPlan() error = %v, want nil", err)
	}
	if plan == nil {
		t.Fatalf("BuildPlan() plan = nil, want non-nil")
	}
	assertThinSlicePlan(t, buf.Bytes())
}

type thinSlicePlan struct {
	Tasks []thinSlicePlanTask `json:"tasks"`
}

type thinSlicePlanTask struct {
	ID      string            `json:"id"`
	Command []string          `json:"command"`
	Image   string            `json:"image"`
	Inputs  []thinSlicePlanIO `json:"inputs"`
	Outputs []thinSlicePlanIO `json:"outputs"`
}

type thinSlicePlanIO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func assertThinSlicePlan(t *testing.T, raw []byte) {
	t.Helper()
	var got thinSlicePlan
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	wantCmd := map[string][]string{
		"index": {"bwa", "index", wgsE2EStagedFASTA},
		"mem":   {"bwa", "mem", "-o", wgsE2EThinSAM, wgsE2EStagedFASTA, wgsE2EStagedR1, wgsE2EStagedR2},
		"sort":  {"samtools", "sort", "-o", wgsE2EThinBAM, wgsE2EThinSAM},
		"bai":   {"samtools", "index", wgsE2EThinBAM, wgsE2EThinBAI},
	}
	wantImage := map[string]string{
		"index": wgsE2EThinBWA,
		"mem":   wgsE2EThinBWA,
		"sort":  wgsE2EThinSamtools,
		"bai":   wgsE2EThinSamtools,
	}
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
		if thinSliceHasShell(task.Command) {
			t.Fatalf("%s command has sh -c: %#v", task.ID, task.Command)
		}
		binds := thinSliceBindPaths(task)
		for _, tok := range task.Command {
			if tok == "/work" || strings.HasPrefix(tok, "/work/") {
				t.Fatalf("%s argv token %q contains /work", task.ID, tok)
			}
			if strings.Contains(tok, "..") {
				t.Fatalf("%s argv token %q contains ..", task.ID, tok)
			}
			if filepath.IsAbs(tok) || strings.HasPrefix(tok, "/") {
				t.Fatalf("%s argv token %q is an absolute path", task.ID, tok)
			}
			if !thinSlicePathShaped(tok) {
				continue
			}
			if !binds[tok] {
				t.Fatalf("%s path-shaped argv token %q is not a declared bind; binds %v", task.ID, tok, bindPathList(binds))
			}
		}
	}
	for id := range wantCmd {
		if !seen[id] {
			t.Fatalf("missing task %q", id)
		}
	}
	if path := thinSliceOutputPath(t, got, "bai", "bai"); path != wgsE2EThinBAI {
		t.Fatalf("bai.bai path got %q, want %q", path, wgsE2EThinBAI)
	}
}

func thinSlicePathShaped(tok string) bool {
	if tok == "" || strings.HasPrefix(tok, "-") {
		return false
	}
	return strings.ContainsAny(tok, `/\`)
}

func thinSliceHasShell(cmd []string) bool {
	hasSH, hasC := false, false
	for _, tok := range cmd {
		if tok == "sh" || strings.HasSuffix(tok, "/sh") {
			hasSH = true
		}
		if tok == "-c" {
			hasC = true
		}
	}
	return hasSH && hasC
}

func thinSliceBindPaths(task thinSlicePlanTask) map[string]bool {
	out := make(map[string]bool, len(task.Inputs)+len(task.Outputs))
	for _, b := range task.Inputs {
		out[b.Path] = true
	}
	for _, b := range task.Outputs {
		out[b.Path] = true
	}
	return out
}

func bindPathList(binds map[string]bool) []string {
	out := make([]string, 0, len(binds))
	for p := range binds {
		out = append(out, p)
	}
	return out
}

func thinSliceOutputPath(t *testing.T, plan thinSlicePlan, taskID, port string) string {
	t.Helper()
	for _, task := range plan.Tasks {
		if task.ID != taskID {
			continue
		}
		for _, b := range task.Outputs {
			if b.Name == port {
				return b.Path
			}
		}
	}
	t.Fatalf("plan missing %s.%s", taskID, port)
	return ""
}

func assertThinSliceFiles(t *testing.T, workspace string) {
	t.Helper()
	for _, rel := range wgsE2EThinPublished {
		path := filepath.Join(workspace, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%s) error = %v", path, err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("published %s is not a regular file", path)
		}
		if info.Size() == 0 {
			t.Fatalf("published %s is empty", path)
		}
	}
}

func assertThinSliceRealOutputs(t *testing.T, workspace string) {
	t.Helper()
	sam := readPrefix(t, filepath.Join(workspace, filepath.FromSlash(wgsE2EThinSAM)), 4)
	if sam[0] != '@' {
		t.Fatalf("SAM magic got %q, want @ header (not a stand-in)", sam)
	}
	bam := readPrefix(t, filepath.Join(workspace, filepath.FromSlash(wgsE2EThinBAM)), 2)
	if bam[0] != 0x1f || bam[1] != 0x8b {
		t.Fatalf("BAM magic got %x, want gzip/BGZF 1f8b (not a stand-in)", bam)
	}
	bai := readPrefix(t, filepath.Join(workspace, filepath.FromSlash(wgsE2EThinBAI)), 3)
	if string(bai) != "BAI" {
		t.Fatalf("BAI magic got %q, want BAI (not a stand-in)", bai)
	}
}

func readPrefix(t *testing.T, path string, n int) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", path, err)
	}
	defer f.Close()
	buf := make([]byte, n)
	got, err := f.Read(buf)
	if err != nil && got == 0 {
		t.Fatalf("Read(%s) error = %v", path, err)
	}
	return buf[:got]
}

func countMappedReads(t *testing.T, workspace string) (int, string) {
	t.Helper()
	abs, err := filepath.Abs(workspace)
	if err != nil {
		t.Fatalf("Abs(%s) error = %v", workspace, err)
	}
	user := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	// Biocontainers ENTRYPOINT is env-execute; override so argv is view ...
	args := []string{
		"run", "--rm",
		"--user", user,
		"--network=none",
		"--entrypoint", "samtools",
		"-v", abs + ":/work",
		"-w", "/work",
		wgsE2EThinSamtools,
		"view", "-c", "-F", "4", wgsE2EThinBAM,
	}
	out, err := exec.Command("docker", args...).CombinedOutput()
	method := "docker run --rm --user --network=none --entrypoint samtools -v workspace:/work -w /work " + wgsE2EThinSamtools + " view -c -F 4 " + wgsE2EThinBAM
	if err == nil {
		n, parseErr := strconv.Atoi(strings.TrimSpace(string(out)))
		if parseErr == nil {
			return n, method
		}
		t.Logf("docker mapped-read parse %v; output %q", parseErr, out)
	} else {
		t.Logf("docker mapped-read helper: %v\n%s", err, out)
	}

	host := exec.Command("samtools", "view", "-c", "-F", "4", wgsE2EThinBAM)
	host.Dir = workspace
	out, err = host.CombinedOutput()
	method = "host samtools view -c -F 4 " + wgsE2EThinBAM
	if err != nil {
		t.Fatalf("mapped-read count failed (docker and host): %v\n%s", err, out)
	}
	n, parseErr := strconv.Atoi(strings.TrimSpace(string(out)))
	if parseErr != nil {
		t.Fatalf("host mapped-read parse %v; output %q", parseErr, out)
	}
	return n, method
}

func dockerImageDigest(t *testing.T, image string) string {
	t.Helper()
	out, err := exec.Command("docker", "image", "inspect", "--format",
		`{{range $i, $d := .RepoDigests}}{{if $i}},{{end}}{{$d}}{{end}}`, image).Output()
	if err == nil {
		if d := strings.TrimSpace(string(out)); d != "" {
			return d
		}
	}
	out, err = exec.Command("docker", "image", "inspect", "--format", "{{.Id}}", image).Output()
	if err != nil {
		t.Fatalf("docker image inspect %s: %v", image, err)
	}
	return strings.TrimSpace(string(out))
}

func bwaWriteFallback(err error, workspace string) bool {
	if err == nil {
		return false
	}
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return false
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(err.Error()))
	for _, d := range ge.Defects {
		b.WriteByte(' ')
		b.WriteString(strings.ToLower(string(d.Code)))
		b.WriteByte(' ')
		b.WriteString(strings.ToLower(d.Unit))
		b.WriteByte(' ')
		b.WriteString(strings.ToLower(d.Message))
	}
	b.WriteString(strings.ToLower(thinSliceLogs(workspace)))
	blob := b.String()
	if strings.Contains(blob, "docker pull") || strings.Contains(blob, "no such image") {
		return false
	}
	needles := []string{
		"unrecognized option",
		"invalid option",
		"unknown option",
		"illegal option",
		"permission denied",
		"read-only file system",
		"operation not permitted",
		"cannot open",
		"can't open",
		"fail to open",
		"failed to open",
		"error opening",
		"missing output",
	}
	for _, n := range needles {
		if strings.Contains(blob, n) {
			return true
		}
	}
	return false
}

func thinSliceLogs(workspace string) string {
	var b strings.Builder
	for _, id := range []string{"index", "mem", "sort", "bai"} {
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

type thinSliceRecord struct {
	BWATag         string
	BWADigest      string
	SamtoolsTag    string
	SamtoolsDigest string
	Inputs         []thinSliceInputFact
	PairCount      int
	MappedCount    int
	MappedMethod   string
	OccupyError    string
	OccupyOK       bool
	FilesOK        bool
	RunError       string
	TaskLogs       string
	TaskState      string
	Stop           string
}

func finishThinSliceAfterRunError(t *testing.T, g *gobble.Graph, dir string, rec *thinSliceRecord, runErr error) {
	t.Helper()
	rec.BWADigest = dockerImageDigest(t, rec.BWATag)
	rec.SamtoolsDigest = dockerImageDigest(t, wgsE2EThinSamtools)
	published := thinSliceExisting(dir)
	rec.FilesOK = len(published) == len(wgsE2EThinPublished)
	if fileExists(dir, wgsE2EThinBAM) {
		count, method := countMappedReads(t, dir)
		rec.MappedCount = count
		rec.MappedMethod = method
		t.Logf("mapped reads after failed Run: %d via %s", count, method)
	}
	occErr := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
	var ge *gobble.Error
	if errors.As(occErr, &ge) {
		rec.OccupyError = occErr.Error()
		hasOccupied, hasOutput := false, false
		for _, d := range ge.Defects {
			if d.Code == gobble.DefectOccupiedWorkspace {
				hasOccupied = true
			}
			if d.Code == gobble.DefectOutputExists {
				hasOutput = true
			}
		}
		rec.OccupyOK = ge.Op == "run" && hasOccupied && !hasOutput
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".gobble", "run.json")); statErr != nil {
		rec.OccupyOK = false
	}
	if thinSliceBAINotStarted(runErr, rec.TaskState) {
		rec.Stop = "D5 joint hole: locked BAI related-file From creates a scheduler edge to output port bai; Run upstreamReady requires that ToPort be an existing input file, so task bai stays not-started. Engine Run cannot host the locked D1 output From. Do not drop From. Do not add thin-slice sh -c. Do not edit engine in this task."
	}
	writeThinSliceRecord(t, *rec)
}

func thinSliceBAINotStarted(err error, state string) bool {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return false
	}
	for _, d := range ge.Defects {
		if d.Unit == "bai" && strings.Contains(strings.ToLower(d.Message), "not started") {
			return true
		}
	}
	return strings.Contains(state, `"id": "bai"`) && strings.Contains(state, `"status": "not-started"`)
}

func thinSliceExisting(workspace string) []string {
	var out []string
	for _, rel := range wgsE2EThinPublished {
		if fileExists(workspace, rel) {
			out = append(out, rel)
		}
	}
	return out
}

func fileExists(workspace, rel string) bool {
	info, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(rel)))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func formatRunDefects(err error) string {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return err.Error()
	}
	var b strings.Builder
	b.WriteString(err.Error())
	for _, d := range ge.Defects {
		b.WriteString("\n  ")
		b.WriteString(string(d.Code))
		b.WriteString(" unit=")
		b.WriteString(d.Unit)
		b.WriteString(" msg=")
		b.WriteString(d.Message)
	}
	return b.String()
}

func thinSliceTaskState(workspace string) string {
	data, err := os.ReadFile(filepath.Join(workspace, ".gobble", "tasks.json"))
	if err != nil {
		return "tasks.json: " + err.Error()
	}
	return string(data)
}

type thinSliceInputFact struct {
	Path   string
	SHA256 string
	Bytes  int64
}

func thinSliceInputFacts(t *testing.T, workspace string) []thinSliceInputFact {
	t.Helper()
	out := make([]thinSliceInputFact, 0, len(wgsE2EStaged))
	for _, rel := range wgsE2EStaged {
		sum, size := wgsE2EHashFile(t, filepath.Join(workspace, rel))
		out = append(out, thinSliceInputFact{Path: rel, SHA256: sum, Bytes: size})
	}
	return out
}

func writeThinSliceRecord(t *testing.T, rec thinSliceRecord) {
	t.Helper()
	path := thinSliceRecordPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(formatThinSliceRecord(rec)), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	t.Logf("thin-slice record: %s", path)
}

func thinSliceRecordPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(wgsE2ERecordDir(t), "thin-slice-record.md")
}

func wgsE2ERecordDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("GOBBLE_WGS_E2E_RECORD_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
		return dir
	}
	return t.TempDir()
}

func formatThinSliceRecord(rec thinSliceRecord) string {
	var b strings.Builder
	b.WriteString("# Thin-slice proof record\n\n")
	b.WriteString("## Images\n\n")
	b.WriteString("- bwa tag: " + rec.BWATag + "\n")
	b.WriteString("- bwa digest: " + rec.BWADigest + "\n")
	b.WriteString("- samtools tag: " + rec.SamtoolsTag + "\n")
	b.WriteString("- samtools digest: " + rec.SamtoolsDigest + "\n\n")
	b.WriteString("## Inputs\n\n")
	for _, in := range rec.Inputs {
		b.WriteString("- " + in.Path + " sha256 " + in.SHA256 + " bytes " + strconv.FormatInt(in.Bytes, 10) + "\n")
	}
	b.WriteString("- pair count: " + strconv.Itoa(rec.PairCount) + "\n\n")
	b.WriteString("## Mapped reads\n\n")
	b.WriteString("- count: " + strconv.Itoa(rec.MappedCount) + "\n")
	b.WriteString("- method: " + rec.MappedMethod + "\n")
	if rec.Stop != "" && strings.Contains(rec.Stop, "mapped-read count is 0") {
		b.WriteString("- stopped for count=0: yes\n")
	} else {
		b.WriteString("- stopped for count=0: no\n")
	}
	b.WriteString("\n## Occupy\n\n")
	if rec.OccupyOK {
		b.WriteString("- second Run: occupied-workspace\n")
	} else {
		b.WriteString("- second Run: not checked\n")
	}
	b.WriteString("- error: " + rec.OccupyError + "\n")
	b.WriteString("- output-exists: not reported\n\n")
	b.WriteString("## Run\n\n")
	b.WriteString("- eight regular files: " + strconv.FormatBool(rec.FilesOK) + "\n")
	if rec.RunError != "" {
		b.WriteString("- run error: " + rec.RunError + "\n")
	}
	if rec.Stop != "" {
		b.WriteString("\n## Stop\n\n")
		b.WriteString(rec.Stop + "\n")
		b.WriteString("\nDo not PASS. Do not rewrite testdata/manifest.json, wgs_e2e_fixture_test.go, testdata/README.md, .gitignore, or the cache helper.\n")
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
