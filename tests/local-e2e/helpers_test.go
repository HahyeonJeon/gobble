package local_e2e_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/tests/internal/fixture"
	methylseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/methylseq"
	rnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/rnaseq"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

var testIdentityOnce sync.Once
var testIdentity gobble.Identity
var testIdentityErr error

func testOccupyOption(t *testing.T) gobble.OccupyOption {
	t.Helper()
	testIdentityOnce.Do(func() {
		testIdentity, testIdentityErr = gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/tests/local-e2e_test")
	})
	if testIdentityErr != nil {
		t.Fatalf("IdentityFromBuildInfo() error = %v", testIdentityErr)
	}
	return gobble.WithIdentity(testIdentity)
}

const (
	rnaSheetRel    = "tests/pipelines/rnaseq/testdata/rnaseq-live-samplesheet.csv"
	methylSheetRel = "tests/pipelines/methylseq/testdata/methylseq-samplesheet.csv"
	runLocalInput  = "testdata/run-local/in/sample.txt"
	runLocalImage  = "alpine:3.21"
	runLocalPkg    = "./tests/cli-valid/runlocal"
	wgsPkg         = "./tests/local-e2e/wgs"
	rnaSeqPkg      = "./tests/local-e2e/rnaseq"
	methylSeqPkg   = "./tests/local-e2e/methylseq"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker info: %v", err)
	}
}

func withSampleSheet(t *testing.T, path string) {
	t.Helper()
	prev := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(path)
	t.Cleanup(func() { gobble.SetSampleSheetPath(prev) })
}

func packSheet(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join(moduleRoot(t), filepath.FromSlash(rel))
}

func readModuleFile(t *testing.T, rel string) []byte {
	t.Helper()
	path := filepath.Join(moduleRoot(t), filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return data
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

func fetchPin(t *testing.T, cacheDir string, pin fixture.Pin) string {
	t.Helper()
	if !filepath.IsAbs(cacheDir) {
		cacheDir = filepath.Join(moduleRoot(t), filepath.FromSlash(cacheDir))
	}
	src, err := fixture.Fetch(cacheDir, pin)
	if err != nil {
		t.Fatalf("download %s: %v", pin.URL, err)
	}
	return src
}

func stagePins(t *testing.T, dir, cacheDir string, pins []struct {
	pin fixture.Pin
	rel string
}) {
	t.Helper()
	for _, p := range pins {
		stageFile(t, dir, p.rel, fetchPin(t, cacheDir, p.pin))
	}
}

func stageWGSPins(t *testing.T, dir string) {
	t.Helper()
	stagePins(t, dir, wgsevidence.CacheDir, []struct {
		pin fixture.Pin
		rel string
	}{
		{wgsevidence.MustPin("test_1.fastq.gz"), "in/reads/test_1.fastq.gz"},
		{wgsevidence.MustPin("test_2.fastq.gz"), "in/reads/test_2.fastq.gz"},
		{wgsevidence.MustPin("test2_1.fastq.gz"), "in/reads/test2_1.fastq.gz"},
		{wgsevidence.MustPin("test2_2.fastq.gz"), "in/reads/test2_2.fastq.gz"},
		{wgsevidence.MustPin("genome.fasta"), "in/reference/genome.fasta"},
		{wgsevidence.MustPin("genome.fasta.fai"), "in/reference/genome.fasta.fai"},
		{wgsevidence.MustPin("genome.dict"), "in/reference/genome.dict"},
		{wgsevidence.MustPin("dbsnp_146.hg38.vcf.gz"), "in/reference/known-sites/dbsnp_146.hg38.vcf.gz"},
		{wgsevidence.MustPin("dbsnp_146.hg38.vcf.gz.tbi"), "in/reference/known-sites/dbsnp_146.hg38.vcf.gz.tbi"},
		{wgsevidence.MustPin("mills_and_1000G.indels.vcf.gz"), "in/reference/known-sites/mills_and_1000G.indels.vcf.gz"},
		{wgsevidence.MustPin("mills_and_1000G.indels.vcf.gz.tbi"), "in/reference/known-sites/mills_and_1000G.indels.vcf.gz.tbi"},
		{wgsevidence.MustPin("genome.multi_intervals.bed"), "in/reference/genome.multi_intervals.bed"},
	})
	data, err := os.ReadFile(filepath.Join(dir, "in", "reference", "genome.multi_intervals.bed"))
	if err != nil {
		t.Fatalf("read WGS interval source: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("WGS interval source has %d members, want 2", len(lines))
	}
	for i, line := range lines {
		writeFile(t, filepath.Join(dir, "in", "reference", "intervals", "interval_00"+strconv.Itoa(i+1)+".bed"), string(line)+"\n")
	}
	withSampleSheet(t, packSheet(t, wgsevidence.FixtureSheet))
}

func stageRNASeqPins(t *testing.T, dir string) {
	t.Helper()
	stagePins(t, dir, rnaseqevidence.CacheDir, []struct {
		pin fixture.Pin
		rel string
	}{
		{rnaseqevidence.GenomeFASTA, "in/reference/genome.fasta"},
		{rnaseqevidence.GTF, "in/reference/genes_with_empty_tid.gtf.gz"},
		{rnaseqevidence.Ctrl1FASTQ1, "in/reads/SRR6357070_1.fastq.gz"},
		{rnaseqevidence.Ctrl1FASTQ2, "in/reads/SRR6357070_2.fastq.gz"},
		{rnaseqevidence.Ctrl2FASTQ1, "in/reads/SRR6357071_1.fastq.gz"},
		{rnaseqevidence.Ctrl2FASTQ2, "in/reads/SRR6357071_2.fastq.gz"},
		{rnaseqevidence.Test1FASTQ, "in/reads/SRR6357072_1.fastq.gz"},
		{rnaseqevidence.Test2FASTQ, "in/reads/SRR6357072_2.fastq.gz"},
		{rnaseqevidence.Treat2FASTQ1, "in/reads/SRR6357073_1.fastq.gz"},
		{rnaseqevidence.Single2Run1, "in/reads/SRR6357074_1.fastq.gz"},
		{rnaseqevidence.Single2Run2, "in/reads/SRR6357075_1.fastq.gz"},
		{rnaseqevidence.FinalFASTQ1, "in/reads/SRR6357076_1.fastq.gz"},
		{rnaseqevidence.FinalFASTQ2, "in/reads/SRR6357076_2.fastq.gz"},
	})
}

func stageMethylPins(t *testing.T, dir string) {
	t.Helper()
	stagePins(t, dir, methylseqevidence.CacheDir, []struct {
		pin fixture.Pin
		rel string
	}{
		{methylseqevidence.GenomeFASTA, "in/reference/genome.fa"},
		{methylseqevidence.Single1FASTQ, "in/reads/SRR389222_sub1.fastq.gz"},
		{methylseqevidence.Single2FASTQ, "in/reads/SRR389222_sub2.fastq.gz"},
		{methylseqevidence.Single3FASTQ, "in/reads/SRR389222_sub3.fastq.gz"},
		{methylseqevidence.Test1FASTQ, "in/reads/Ecoli_10K_methylated_R1.fastq.gz"},
		{methylseqevidence.Test2FASTQ, "in/reads/Ecoli_10K_methylated_R2.fastq.gz"},
	})
}

func stageMethylReadyIndex(t *testing.T, dir string) {
	t.Helper()
	archive := fetchPin(t, methylseqevidence.CacheDir, methylseqevidence.ReadyIndexArchive)
	tree := filepath.Join(dir, filepath.FromSlash("in/reference/BismarkIndex"))
	if err := methylseqevidence.PrepareReadyIndex(archive, tree); err != nil {
		t.Fatalf("prepare official ready Bismark index: %v", err)
	}
}

func copyRunLocalInput(t *testing.T, workspace string) {
	t.Helper()
	data := readModuleFile(t, runLocalInput)
	dst := filepath.Join(workspace, "in", "sample.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", dst, err)
	}
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

func requireRegularFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published %s: %v", path, err)
	}
}

func requireFixtureText(t *testing.T, path string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if string(got) != "fixture\n" && string(got) != "fixture" {
		t.Fatalf("%s got %q, want fixture", path, got)
	}
}

func mustJSONFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("%s is not complete JSON: %v\n%s", path, err, data)
	}
	return data
}

func mustCompose(pipe func() *gobble.Pipeline) func(t *testing.T) *gobble.Graph {
	return func(t *testing.T) *gobble.Graph {
		t.Helper()
		g, err := gobble.Compose(pipe())
		if err != nil {
			t.Fatalf("Compose() error = %v, want nil", err)
		}
		if g == nil {
			t.Fatalf("Compose() graph = nil, want compose-valid graph")
		}
		return g
	}
}

func TestFormatAPIError(t *testing.T) {
	err := &gobble.Error{
		Op: "run",
		Defects: []gobble.Defect{
			{Code: gobble.DefectFailed, Unit: "a", Message: "boom", Paths: []string{"out/a"}},
			{Code: gobble.DefectFailed, Unit: "b", Message: "bang", Paths: []string{"out/b", "work/b"}},
		},
	}
	if err.Error() != "run: 2 defects" {
		t.Fatalf("Error() = %q, want collapsed multi-defect text", err.Error())
	}
	got := formatAPIError("Run(wgs.Pipeline())", err)
	for _, want := range []string{
		"Run(wgs.Pipeline()) op=run",
		"code=failed unit=a path=out/a message=boom",
		"code=failed unit=b path=out/b,work/b message=bang",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatAPIError() = %q, want substring %q", got, want)
		}
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
	return decodeJSONL(t, data)
}

func decodeJSONL(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	var out []map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("JSONL: %v\n%s", err, data)
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

func assertOccupied(t *testing.T, dir string) {
	t.Helper()
	run := inspectObject(t, dir, gobble.ViewRun)
	occ, _ := run["occupancy"].(map[string]any)
	if occ["active"] != true {
		t.Fatalf("occupancy got %#v, want active", occ)
	}
}

func recoverAfterSuccessAPI(t *testing.T, g *gobble.Graph, dir string, cap int) {
	t.Helper()
	if remaining := inspectJSONL(t, dir, gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("success remaining got %#v, want empty", remaining)
	}
	err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
	requireRunError(t, "second Run", err, gobble.DefectOccupiedWorkspace, "")

	if err := gobble.Release(dir); err != nil {
		fatalAPIError(t, "Release()", err)
	}
	run := inspectObject(t, dir, gobble.ViewRun)
	occ, _ := run["occupancy"].(map[string]any)
	if occ["active"] != false {
		t.Fatalf("released occupancy got %#v", occ)
	}

	if err := gobble.Resume(t.Context(), g, dir, cap, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Resume()", err)
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

func dumpTaskLogs(t *testing.T, workspace string, ids ...string) {
	t.Helper()
	for _, id := range ids {
		for _, name := range []string{"stderr", "stdout"} {
			path := filepath.Join(workspace, engine.ControlDir, "tasks", id, "_", "0", "1", name)
			data, err := os.ReadFile(path)
			if err != nil || len(data) == 0 {
				continue
			}
			t.Logf("%s/%s:\n%s", id, name, data)
		}
	}
}

func planTasks(t *testing.T, g *gobble.Graph) []planTask {
	t.Helper()
	var buf bytes.Buffer
	if _, err := gobble.BuildPlan(g, gobble.WriteTo(&buf)); err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	var got struct {
		Tasks []planTask `json:"tasks"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("plan JSON: %v\n%s", err, buf.Bytes())
	}
	return got.Tasks
}

type planTask struct {
	ID      string       `json:"id"`
	Inputs  []planTaskIO `json:"inputs"`
	Outputs []planTaskIO `json:"outputs"`
}

type planTaskIO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func assertMultiQCOmitsBAM(t *testing.T, g *gobble.Graph) {
	t.Helper()
	for _, task := range planTasks(t, g) {
		if task.ID != "multiqc" && !strings.HasSuffix(task.ID, ".multiqc") {
			continue
		}
		for _, in := range task.Inputs {
			if strings.HasSuffix(in.Path, ".bam") {
				t.Fatalf("multiqc input %q path = %q, MultiQC must not consume BAM", in.Name, in.Path)
			}
		}
	}
}

func assertRNAProductOutputs(t *testing.T, dir string) {
	t.Helper()
	for _, rel := range []string{
		"results/rnaseq/matrices/gene_counts.tsv",
		"results/rnaseq/matrices/transcript_tpm.tsv",
		"results/rnaseq/deseq2-qc/pca.pdf",
		"results/rnaseq/deseq2-qc/sample_distance.pdf",
		"results/rnaseq/multiqc/multiqc_report.html",
	} {
		requireRegularFile(t, filepath.Join(dir, filepath.FromSlash(rel)))
	}
}

func assertSTARMappedAndSplices(t *testing.T, dir string, samples []string) {
	t.Helper()
	ok := false
	for _, sample := range samples {
		path := filepath.Join(dir, filepath.FromSlash("work/"+sample+"/star/Log.final.out"))
		mapped := uniquelyMappedReads(t, path)
		splices := starLogInt(t, path, starSplicesTotalField)
		t.Logf("%s uniquely mapped = %d splices = %d", sample, mapped, splices)
		if mapped >= 10 && splices >= 1 {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("no sample STAR log has uniquely mapped reads >= 10 and splices >= 1")
	}
}

func uniquelyMappedReads(t *testing.T, path string) int {
	t.Helper()
	return starLogInt(t, path, "Uniquely mapped reads number")
}

const starSplicesTotalField = "Number of splices: Total"

func starLogInt(t *testing.T, path, field string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, field) {
			continue
		}
		i := strings.LastIndex(line, "|")
		if i < 0 {
			t.Fatalf("%s line %q: missing |", field, line)
		}
		n, err := strconv.Atoi(strings.TrimSpace(line[i+1:]))
		if err != nil {
			t.Fatalf("%s line %q: %v", field, line, err)
		}
		return n
	}
	t.Fatalf("%s: missing %s", path, field)
	return 0
}

const uniquePEAlignmentField = "Number of paired-end alignments with a unique best hit:"

func uniquePEAlignments(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, uniquePEAlignmentField) {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("unique-alignment line %q missing value", line)
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			t.Fatalf("unique-alignment count in %q: %v", line, err)
		}
		return n
	}
	t.Fatalf("%s missing unique-alignment field %q", path, uniquePEAlignmentField)
	return 0
}

func assertUniqueAlignmentFloor(t *testing.T, unique int) {
	t.Helper()
	if unique < 1 {
		t.Fatalf("unique paired-end alignments = %d, want floor > 0", unique)
	}
}

func assertMethylationCallRows(t *testing.T, unique int, paths ...string) {
	t.Helper()
	rows := 0
	for _, path := range paths {
		n := methylationCallRows(t, path)
		t.Logf("methylation call rows in %s = %d", filepath.Base(path), n)
		rows += n
	}
	if unique > 0 && rows == 0 {
		t.Fatalf("no methylation call row in %v", paths)
	}
}

func methylationCallRows(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s): %v", path, err)
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			t.Fatalf("gzip %s: %v", path, err)
		}
		defer gz.Close()
		r = gz
	}
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "Bismark") || strings.HasPrefix(line, "#") {
			continue
		}
		if len(strings.Fields(line)) < 4 {
			continue
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return n
}

func assertMethylExtractorOutputs(t *testing.T, dir string, samples []string) {
	t.Helper()
	for _, sample := range samples {
		prefix := sample
		if sample == "Ecoli_10K_methylated" {
			prefix += "_pe"
		}
		calls := "results/methylseq/methylation-calls/" + sample + "/"
		for _, rel := range []string{
			"results/methylseq/bismark/" + sample + "/" + prefix + ".deduplicated.bam",
			"results/methylseq/bismark/" + sample + "/" + prefix + ".deduplication_report.txt",
			calls + prefix + ".deduplicated.bedGraph.gz",
			calls + prefix + ".deduplicated.bismark.cov.gz",
			calls + prefix + ".deduplicated_splitting_report.txt",
			calls + prefix + ".deduplicated.M-bias.txt",
			calls + "CpG_context_" + prefix + ".deduplicated.txt.gz",
			calls + "CHG_context_" + prefix + ".deduplicated.txt.gz",
			calls + "CHH_context_" + prefix + ".deduplicated.txt.gz",
			"results/methylseq/reports/" + sample + "/" + sample + ".bismark_report.html",
		} {
			requireRegularFile(t, filepath.Join(dir, filepath.FromSlash(rel)))
		}
	}
}
