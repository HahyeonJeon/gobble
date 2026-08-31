//go:build live

package install_e2e_test

const workflowSource = `package workflowcase

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("workflow-case")
	readsR1 := p.AddInput("reads_r1", gobble.PathSpec{
		Dir: gobble.Dir("in"), Prefix: "sample_S1_L001_R1_", Base: "001", Ext: ".fastq.gz",
	})
	readsR2 := p.AddInput("reads_r2", gobble.PathSpec{
		Dir: gobble.Dir("in"), Prefix: "sample_S1_L001_R2_", Base: "001", Ext: ".fastq.gz",
	})
	prep := p.AddModule("prep")
	fastp := prep.AddTask(gobble.TaskSpec{
		Name: "fastp", Command: []string{"fastp"}, Image: "example/fastp:0",
		Inputs: []gobble.Bind{{Name: "r1", From: readsR1}, {Name: "r2", From: readsR2}},
		Outputs: []gobble.Bind{
			{Name: "clean_r1", Spec: gobble.PathSpec{Dir: gobble.Dir("work/prep"), Prefix: "sample_S1_L001_R1_", Base: "001", Suffixes: []string{"clean"}, Ext: ".fastq.gz"}},
			{Name: "clean_r2", Spec: gobble.PathSpec{Dir: gobble.Dir("work/prep"), Prefix: "sample_S1_L001_R2_", Base: "001", Suffixes: []string{"clean"}, Ext: ".fastq.gz"}},
		},
	})
	call := p.AddModule("call")
	align := call.Branch("align")
	qc := call.Branch("qc")
	join := call.Merge("join")
	bam := gobble.PathSpec{Dir: gobble.Dir("work/align"), Base: "sample", Suffixes: []string{"sorted"}, Ext: ".bam"}
	bwa := align.AddTask(gobble.TaskSpec{
		Name: "bwa", Command: []string{"bwa"}, Image: "example/bwa:0",
		Inputs: []gobble.Bind{{Name: "r1", From: fastp.Out("clean_r1")}, {Name: "r2", From: fastp.Out("clean_r2")}},
		Outputs: []gobble.Bind{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bam.AppendExt(".bai")}},
	})
	fastqc := qc.AddTask(gobble.TaskSpec{
		Name: "fastqc", Command: []string{"fastqc"}, Image: "example/fastqc:0",
		Inputs: []gobble.Bind{{Name: "r1", From: fastp.Out("clean_r1")}, {Name: "r2", From: fastp.Out("clean_r2")}},
		Outputs: []gobble.Bind{{Name: "html", Spec: gobble.Literal("sample_clean_fastqc.html").WithDir(gobble.Dir("work/qc"))}},
	})
	join.AddTask(gobble.TaskSpec{
		Name: "report", Command: []string{"report"},
		Inputs: []gobble.Bind{{Name: "bam", From: bwa.Out("bam")}, {Name: "bai", From: bwa.Out("bai")}, {Name: "html", From: fastqc.Out("html")}},
		Outputs: []gobble.Bind{{Name: "summary", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "report", Ext: ".json"}}},
	})
	return p
}
`

const wgsSource = `package wgs

import (
	"github.com/HahyeonJeon/gobble"
	product "github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

func Pipeline() *gobble.Pipeline {
	return product.Build([]product.Sample{
		{Patient: "patient1", Name: "testN", Sex: "XX", Lanes: []product.Lane{
			{ID: "L001", Fastq1: "in/reads/test_1.fastq.gz", Fastq2: "in/reads/test_2.fastq.gz"},
			{ID: "L002", Fastq1: "in/reads/test_1.fastq.gz", Fastq2: "in/reads/test_2.fastq.gz"},
		}},
		{Patient: "patient2", Name: "testT", Sex: "XY", Lanes: []product.Lane{
			{ID: "L001", Fastq1: "in/reads/test2_1.fastq.gz", Fastq2: "in/reads/test2_2.fastq.gz"},
		}},
	}, product.DefaultConfig())
}
`

const printpipeSource = `package printpipe

import (
	"fmt"
	"github.com/HahyeonJeon/gobble"
)

func init() { fmt.Print("user init output\n") }

func Pipeline() *gobble.Pipeline {
	fmt.Print("user Pipeline output\n")
	p := gobble.NewPipeline("printed")
	p.AddTask(gobble.TaskSpec{
		Name: "copy", Command: []string{"true"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "out", Ext: ".txt"}}},
	})
	return p
}
`

const hiddenSource = `package hidden

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline { return gobble.NewPipeline("hidden") }
`

const markerSource = `package markerpipe

import (
	"os"
	"github.com/HahyeonJeon/gobble"
)

func Pipeline() *gobble.Pipeline {
	if marker := os.Getenv("ASSAY_PIPELINE_MARKER"); marker != "" {
		_ = os.WriteFile(marker, []byte("called\n"), 0o600)
	}
	p := gobble.NewPipeline("marker")
	p.AddTask(gobble.TaskSpec{
		Name: "write", Command: []string{"sh", "-c", "echo called > out/called.txt"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "called", Ext: ".txt"}}},
	})
	return p
}
`

const stageWGSSource = `package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: stage-wgs WORKSPACE")
		os.Exit(2)
	}
	workspace := os.Args[1]
	destinations := map[string]string{
		"test_1.fastq.gz": "in/reads/test_1.fastq.gz",
		"test_2.fastq.gz": "in/reads/test_2.fastq.gz",
		"test2_1.fastq.gz": "in/reads/test2_1.fastq.gz",
		"test2_2.fastq.gz": "in/reads/test2_2.fastq.gz",
		"genome.fasta": "in/reference/genome.fasta",
		"genome.fasta.fai": "in/reference/genome.fasta.fai",
		"genome.dict": "in/reference/genome.dict",
		"genome.multi_intervals.bed": "in/reference/genome.multi_intervals.bed",
		"dbsnp_146.hg38.vcf.gz": "in/reference/known-sites/dbsnp_146.hg38.vcf.gz",
		"dbsnp_146.hg38.vcf.gz.tbi": "in/reference/known-sites/dbsnp_146.hg38.vcf.gz.tbi",
		"mills_and_1000G.indels.vcf.gz": "in/reference/known-sites/mills_and_1000G.indels.vcf.gz",
		"mills_and_1000G.indels.vcf.gz.tbi": "in/reference/known-sites/mills_and_1000G.indels.vcf.gz.tbi",
	}
	for _, pin := range wgsevidence.MustPins() {
		rel, ok := destinations[pin.Name]
		if !ok {
			continue
		}
		source, err := wgsevidence.Fetch("testdata/cache", pin)
		if err != nil {
			fail(err)
		}
		destination := filepath.Join(workspace, filepath.FromSlash(rel))
		if err := copyFile(source, destination); err != nil {
			fail(err)
		}
	}
	intervals, err := os.ReadFile(filepath.Join(workspace, "in", "reference", "genome.multi_intervals.bed"))
	if err != nil {
		fail(err)
	}
	lines := strings.Split(strings.TrimSpace(string(intervals)), "\n")
	if len(lines) != 2 {
		fail(fmt.Errorf("interval member count %d, want 2", len(lines)))
	}
	for i, line := range lines {
		destination := filepath.Join(workspace, "in", "reference", "intervals", fmt.Sprintf("interval_%03d.bed", i+1))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(destination, []byte(line+"\n"), 0o644); err != nil {
			fail(err)
		}
	}
}

func copyFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
`

const apiAssaySource = `package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"example.com/gobble-install-assay/wgs"
	"example.com/gobble-install-assay/workflowcase"
	"github.com/HahyeonJeon/gobble"
)

type result struct {
	Op              string          ` + "`json:\"op\"`" + `
	RunningIdentity string          ` + "`json:\"running_identity,omitempty\"`" + `
	Remaining       []string        ` + "`json:\"remaining,omitempty\"`" + `
	Identity        gobble.Identity ` + "`json:\"identity\"`" + `
}

type instanceRecord struct {
	Identity string ` + "`json:\"identity\"`" + `
	Status   string ` + "`json:\"status\"`" + `
	Executor string ` + "`json:\"executor\"`" + `
}

type remainingRecord struct {
	Identity  string ` + "`json:\"identity\"`" + `
	Remaining bool   ` + "`json:\"remaining\"`" + `
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fail(errors.New("usage: api-assay workflow|prepare|resume [WORKSPACE]"))
	}
	var err error
	switch os.Args[1] {
	case "workflow":
		err = checkWorkflow()
	case "prepare", "resume":
		if len(os.Args) != 3 {
			err = errors.New("workspace required")
		} else if os.Args[1] == "prepare" {
			err = prepare(os.Args[2])
		} else {
			err = resume(os.Args[2])
		}
	default:
		err = errors.New("unknown mode")
	}
	if err != nil {
		fail(err)
	}
}

func checkWorkflow() error {
	g, err := gobble.Compose(workflowcase.Pipeline())
	if err != nil {
		return err
	}
	if err := gobble.Validate(g); err != nil {
		return err
	}
	p, err := gobble.BuildPlan(g)
	if err != nil {
		return err
	}
	if len(p.TaskIDs()) == 0 {
		return errors.New("workflow plan has no tasks")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"op": "workflow", "tasks": len(p.TaskIDs())})
}

func prepare(workspace string) error {
	id, err := processIdentity()
	if err != nil {
		return err
	}
	g, err := gobble.Compose(wgs.Pipeline())
	if err != nil {
		return err
	}
	if err := gobble.Validate(g); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- gobble.Run(ctx, g, workspace, 1, gobble.WithIdentity(id))
	}()
	running, err := waitForRunningDocker(workspace, id, done)
	if err != nil {
		return err
	}
	cancel()
	runErr := <-done
	if !hasDefect(runErr, gobble.DefectCanceled) {
		return fmt.Errorf("Run cancellation = %v, want canceled defect", runErr)
	}
	remaining, err := remainingIdentities(workspace, id)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("Run cancellation left empty remaining work")
	}
	return json.NewEncoder(os.Stdout).Encode(result{Op: "prepare", RunningIdentity: running, Remaining: remaining, Identity: id})
}

func resume(workspace string) error {
	id, err := processIdentity()
	if err != nil {
		return err
	}
	before, err := remainingIdentities(workspace, id)
	if err != nil {
		return err
	}
	if len(before) == 0 {
		return errors.New("Resume started with empty remaining work")
	}
	if err := gobble.Release(workspace, gobble.WithIdentity(id)); err != nil {
		return err
	}
	g, err := gobble.Compose(wgs.Pipeline())
	if err != nil {
		return err
	}
	if err := gobble.Resume(context.Background(), g, workspace, 1, gobble.WithIdentity(id)); err != nil {
		return err
	}
	after, err := remainingIdentities(workspace, id)
	if err != nil {
		return err
	}
	if len(after) != 0 {
		return fmt.Errorf("Resume left remaining work: %v", after)
	}
	if err := requireSuccessfulDocker(workspace, id); err != nil {
		return err
	}
	for _, rel := range []string{
		"results/wgs/multiqc/multiqc_report.html",
		"results/wgs/samples/testN/alignment/testN.recalibrated.bam",
		"results/wgs/samples/testT/alignment/testT.recalibrated.bam",
		"results/wgs/joint/joint_germline.vcf.gz",
	} {
		info, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("published output %s: %v", rel, err)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(result{Op: "resume", Remaining: before, Identity: id})
}

func processIdentity() (gobble.Identity, error) {
	id, err := gobble.IdentityFromBuildInfo("example.com/gobble-install-assay/wgs")
	if err != nil {
		return id, err
	}
	digest, err := executableDigest()
	if err != nil {
		return id, err
	}
	if id.GobbleExecutableSHA256 != digest {
		return id, fmt.Errorf("identity executable digest %s, want %s", id.GobbleExecutableSHA256, digest)
	}
	return id, nil
}

func executableDigest() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func waitForRunningDocker(workspace string, id gobble.Identity, done <-chan error) (string, error) {
	deadline := time.Now().Add(4 * time.Minute)
	var last []instanceRecord
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			return "", fmt.Errorf("Run ended before a Docker task was observed running: %v; last instances: %#v", err, last)
		default:
		}
		raw, err := gobble.Inspect(workspace, gobble.ViewInstances, "", gobble.WithIdentity(id))
		if err == nil {
			last, err = decodeInstances(raw)
			if err != nil {
				return "", err
			}
			for _, rec := range last {
				if rec.Executor == "docker" && rec.Status == "running" {
					return rec.Identity, nil
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out waiting for running Docker task; last instances: %#v", last)
}

func remainingIdentities(workspace string, id gobble.Identity) ([]string, error) {
	raw, err := gobble.Inspect(workspace, gobble.ViewRemaining, "", gobble.WithIdentity(id))
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var out []string
	for {
		var rec remainingRecord
		if err := dec.Decode(&rec); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		if rec.Remaining {
			out = append(out, rec.Identity)
		}
	}
	return out, nil
}

func requireSuccessfulDocker(workspace string, id gobble.Identity) error {
	raw, err := gobble.Inspect(workspace, gobble.ViewInstances, "", gobble.WithIdentity(id))
	if err != nil {
		return err
	}
	records, err := decodeInstances(raw)
	if err != nil {
		return err
	}
	found := false
	for _, rec := range records {
		if rec.Executor == "docker" && rec.Status == "succeeded" {
			found = true
		}
	}
	if !found {
		return errors.New("Resume has no successful Docker instance")
	}
	return nil
}

func decodeInstances(raw []byte) ([]instanceRecord, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var out []instanceRecord
	for {
		var rec instanceRecord
		if err := dec.Decode(&rec); errors.Is(err, io.EOF) {
			return out, nil
		} else if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
}

func hasDefect(err error, code gobble.DefectCode) bool {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return false
	}
	for _, defect := range ge.Defects {
		if defect.Code == code {
			return true
		}
	}
	return false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
`
