package salmonquant_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	salmonquant "github.com/HahyeonJeon/gobble/assets/modules/salmon-quant"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestAlignmentCommandContract(t *testing.T) {
	bam, transcriptome, gtf := gobble.Literal("in/transcript.bam"), gobble.Literal("in/transcriptome.fa"), gobble.Literal("in/genes.gtf")
	task := cc.Task(t, salmonquant.AlignmentPipeline(bam, transcriptome, gtf, salmonquant.Options{LibType: "ISR"}), "salmon_quant")
	if !pc.ContainsAll(task.Command, "salmon", "quant", "--geneMap", "in/genes.gtf", "--libType", "ISR", "-t", "in/transcriptome.fa", "-a", "in/transcript.bam", "-o", "work/salmon-quant/quant") {
		t.Fatalf("command = %#v, want alignment-mode Salmon argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "quant", "work/salmon-quant/quant/quant.sf")
	cc.Invalid(t, salmonquant.AlignmentPipeline(bam, transcriptome, gtf, salmonquant.Options{Options: modules.Options{Image: "alpine:latest"}}))
}

func TestInferenceProducesTypedStrandedness(t *testing.T) {
	p := gobble.NewPipeline("salmon-inference")
	index := p.AddInputTree("index", gobble.DeclareTree(gobble.Dir("in/index")))
	r1 := p.AddInput("read1", gobble.Literal("in/read1.fastq.gz"))
	r2 := p.AddInput("read2", gobble.Literal("in/read2.fastq.gz"))
	ports, err := salmonquant.AddInference(p, index, r1, r2, gobble.Handle{}, salmonquant.InferenceThresholds{StrandedFraction: 0.8, UnstrandedDifference: 0.1}, salmonquant.Options{})
	if err != nil || ports.Strandedness.IsZero() {
		t.Fatalf("AddInference() = (%+v, %v), want typed strand port", ports, err)
	}
	task := cc.Task(t, p, "salmon_strandedness")
	if !strings.Contains(task.Script, "expected_format") || !strings.Contains(task.Script, "strand_mapping_bias") || !strings.Contains(task.Script, "limit=0.8") || !strings.Contains(task.Script, "limit=0.1") {
		t.Fatalf("script = %q, want thresholded Salmon inference", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "strandedness", "work/salmon-quant/quant/strandedness.txt")
}

func TestInferenceScriptAppliesThresholds(t *testing.T) {
	tests := []struct {
		name   string
		format string
		bias   float64
		want   string
	}{
		{name: "paired forward", format: "ISF", bias: 0.9, want: "forward\n"},
		{name: "paired reverse", format: "ISR", bias: 0.1, want: "reverse\n"},
		{name: "unstranded", format: "IU", bias: 0.52, want: "unstranded\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := gobble.NewPipeline("salmon-inference")
			index := p.AddInputTree("index", gobble.DeclareTree(gobble.Dir("in/index")))
			r1 := p.AddInput("read1", gobble.Literal("in/read1.fastq.gz"))
			_, err := salmonquant.AddInference(p, index, r1, gobble.Handle{}, gobble.Handle{}, salmonquant.InferenceThresholds{StrandedFraction: 0.8, UnstrandedDifference: 0.1}, salmonquant.Options{OutDir: gobble.Dir("out"), Prefix: "sample"})
			if err != nil {
				t.Fatalf("AddInference: %v", err)
			}
			task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "salmon_strandedness")
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatalf("MkdirAll bin: %v", err)
			}
			fake := fmt.Sprintf(`#!/bin/sh
set -eu
out=
while test "$#" -gt 0; do
  if test "$1" = "-o"; then out=$2; shift 2; continue; fi
  shift
done
mkdir -p "$out/aux_info" "$out/logs"
printf '{"expected_format":"%s","strand_mapping_bias":%g}\n' > "$out/lib_format_counts.json"
: > "$out/quant.sf"
: > "$out/aux_info/meta_info.json"
: > "$out/logs/salmon_quant.log"
`, test.format, test.bias)
			if err := os.WriteFile(filepath.Join(bin, "salmon"), []byte(fake), 0o755); err != nil {
				t.Fatalf("WriteFile fake salmon: %v", err)
			}
			cmd := exec.Command("sh", "-c", "set -eu\n"+task.Script)
			cmd.Dir = dir
			cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin"}
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("inference script: %v\n%s\n%s", err, output, task.Script)
			}
			got, err := os.ReadFile(filepath.Join(dir, "out", "sample", "strandedness.txt"))
			if err != nil || string(got) != test.want {
				t.Fatalf("strandedness = %q, %v, want %q", got, err, test.want)
			}
		})
	}
}
