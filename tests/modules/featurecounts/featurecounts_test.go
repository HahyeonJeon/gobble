package featurecounts_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestFeatureCountsStandaloneComposeBuildPlan(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".bam"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	opts := FeatureCountsOptions{
		ExtraArgs:    []string{"-Q", "10"},
		Resources:    gobble.Resources{CPU: 2},
		Strandedness: gobble.StrandednessReverse,
	}
	p := FeatureCountsPipeline(bam, gtf, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "featurecounts")
	if task.Name != "featurecounts" {
		t.Fatalf("task name = %q, want featurecounts", task.Name)
	}
	if task.Image != "quay.io/biocontainers/subread:2.1.1--h577a1d6_0" {
		t.Fatalf("image = %q, want locked subread pin", task.Image)
	}
	if containsToken(task.Command, "-t") || containsToken(task.Command, "-g") {
		t.Fatalf("command = %#v, must not pass -t or -g", task.Command)
	}
	if !pc.ContainsAll(task.Command,
		"featureCounts",
		"-a", "in/genes.gtf",
		"-o", "work/featurecounts/counts.txt",
		"-p",
		"-s", "2",
		"-T", "2",
		"-Q", "10",
		"in/aligned.bam",
	) {
		t.Fatalf("command = %#v, want named flags, extra-args, then BAM", task.Command)
	}
	n := len(task.Command)
	if n < 3 || task.Command[n-1] != "in/aligned.bam" || task.Command[n-3] != "-Q" || task.Command[n-2] != "10" {
		t.Fatalf("command tail = %#v, want extra-args then BAM", task.Command[max(0, n-4):])
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Inputs, "bam", "in/aligned.bam")
	pc.AssertIOPath(t, task.Inputs, "gtf", "in/genes.gtf")
	pc.AssertIOPath(t, task.Outputs, "counts", "work/featurecounts/counts.txt")
}

func TestFeatureCountsNestedModule(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".bam"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	p := gobble.NewPipeline("assay")
	hb := p.AddInput("bam", bam)
	hg := p.AddInput("gtf", gtf)
	mod := p.AddModule("quant")
	ports := AddFeatureCounts(mod, hb, hg, FeatureCountsOptions{
		ExtraArgs:    []string{"-Q", "10"},
		Strandedness: gobble.StrandednessForward,
	})
	if ports.Counts.IsZero() {
		t.Fatalf("ports.Counts IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "quant.featurecounts")
	if task.Name != "featurecounts" || task.Module != "quant" {
		t.Fatalf("nested task = %+v, want name featurecounts module quant", task)
	}
	if !pc.ContainsAll(task.Command, "-s", "1", "-Q", "10") {
		t.Fatalf("command = %#v, want strandedness 1 and extra-args", task.Command)
	}
}

func TestFeatureCountsStrandFlag(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".bam"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	tests := []struct {
		name         string
		strandedness string
		want         string
	}{
		{name: "unstranded", strandedness: gobble.StrandednessUnstranded, want: "0"},
		{name: "forward", strandedness: gobble.StrandednessForward, want: "1"},
		{name: "reverse", strandedness: gobble.StrandednessReverse, want: "2"},
		{name: "empty", strandedness: "", want: "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := FeatureCountsPipeline(bam, gtf, FeatureCountsOptions{Strandedness: tt.strandedness})
			raw := pc.MustPlanJSON(t, p)
			task := pc.TaskByID(t, raw, "featurecounts")
			got, ok := flagValue(task.Command, "-s")
			if !ok || got != tt.want {
				t.Fatalf("command = %#v, -s = %q found %v, want %q", task.Command, got, ok, tt.want)
			}
		})
	}
}

func containsToken(got []string, want string) bool {
	for _, s := range got {
		if s == want {
			return true
		}
	}
	return false
}

func flagValue(cmd []string, flag string) (string, bool) {
	for i, tok := range cmd {
		if tok == flag && i+1 < len(cmd) {
			return cmd[i+1], true
		}
	}
	return "", false
}
