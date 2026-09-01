package resume_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario"
)

func TestMethylExtractorChangePreservesUpstreamIdentity(t *testing.T) {
	plain := methylseqscenario.Plan(t, methylseq.DefaultConfig())
	changedConfig := methylseq.DefaultConfig()
	changedConfig.Extractor.CoverageCutoff = 3
	changed := methylseqscenario.Plan(t, changedConfig)
	for _, id := range []string{"SRR389222_sub1.bismark_align", "SRR389222_sub1.bismark_deduplicate"} {
		if !slices.Equal(pc.TaskByID(t, plain, id).Command, pc.TaskByID(t, changed, id).Command) {
			t.Fatalf("extractor-only change altered upstream task %s", id)
		}
	}
	lifecycle := methylseq.Lifecycle()
	if slices.Equal(pc.TaskByID(t, plain, "SRR389222_sub1.bismark_methylation_extractor").Command, pc.TaskByID(t, changed, "SRR389222_sub1.bismark_methylation_extractor").Command) || !lifecycle.Resume || lifecycle.PreLiftResumable {
		t.Fatal("Methyl selective resume or graph-generation boundary is incorrect")
	}
}

func TestMethylInspectReleaseResumeReusesCompletedGraph(t *testing.T) {
	runtime := methylseqscenario.NewRuntime(t, methylseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(Methyl graph): %v", err)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release(Methyl graph): %v", err)
	}
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume(Methyl graph): %v", err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	if len(reuse) == 0 {
		t.Fatal("Methyl Resume reuse is empty")
	}
	for _, record := range reuse {
		if record["decision"] != "reused" || record["reason"] != "reused-identity-matched" {
			t.Fatalf("reuse = %#v, want reused-identity-matched", record)
		}
	}
}

func TestMethylChangedGraphRerunsAffectedWorkAndReusesUnaffectedWork(t *testing.T) {
	tests := []struct {
		name   string
		mutate func([]methylseq.Sample, *methylseq.Config)
		want   func(string) string
	}{
		{
			name: "changed sample run",
			mutate: func(samples []methylseq.Sample, _ *methylseq.Config) {
				samples[0].Runs[0].Fastq1 = "in/reads/SRR389222_sub1.changed.fastq.gz"
			},
			want: func(identity string) string {
				if strings.HasPrefix(identity, "SRR389222_sub1.") || identity == "bismark_summary" || identity == "multiqc" {
					return "rerun"
				}
				return "reused"
			},
		},
		{
			name: "changed reference",
			mutate: func(_ []methylseq.Sample, config *methylseq.Config) {
				config.Reference.FASTA = gobble.Literal("in/reference/genome-v2.fa")
			},
			want: func(identity string) string {
				if identity == "reference.bismark_genome_preparation" || strings.Contains(identity, ".bismark_") || identity == "bismark_summary" || identity == "multiqc" {
					return "rerun"
				}
				return "reused"
			},
		},
		{
			name: "changed extractor command",
			mutate: func(_ []methylseq.Sample, config *methylseq.Config) {
				config.Extractor.CoverageCutoff = 3
			},
			want: func(identity string) string {
				if strings.HasSuffix(identity, ".bismark_methylation_extractor") || strings.HasSuffix(identity, ".bismark_report") || identity == "bismark_summary" || identity == "multiqc" {
					return "rerun"
				}
				return "reused"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runtime := methylseqscenario.NewRuntime(t, methylseq.DefaultConfig())
			if err := runtime.Run(t.Context()); err != nil {
				t.Fatalf("Run(Methyl graph): %v", err)
			}
			if err := runtime.Release(); err != nil {
				t.Fatalf("Release(Methyl graph): %v", err)
			}
			samples, _ := methylseqscenario.Samples(t)
			config := methylseq.DefaultConfig()
			test.mutate(samples, &config)
			if err := runtime.ResumeWith(t.Context(), samples, config); err != nil {
				t.Fatalf("ResumeWith(changed Methyl graph): %v", err)
			}
			reuse := runtime.InspectRecords(gobble.ViewReuse)
			if len(reuse) == 0 {
				t.Fatal("changed Methyl Resume reuse is empty")
			}
			for _, record := range reuse {
				identity, ok := record["identity"].(string)
				if !ok || identity == "" {
					t.Fatalf("reuse record has no identity: %#v", record)
				}
				want := test.want(identity)
				if record["decision"] != want {
					t.Errorf("reuse decision for %s = %q (%q), want %q", identity, record["decision"], record["reason"], want)
				}
				reason, _ := record["reason"].(string)
				if want == "reused" && reason != "reused-identity-matched" {
					t.Errorf("reuse reason for %s = %q, want reused-identity-matched", identity, reason)
				}
				if want == "rerun" && (reason == "" || reason == "reused-identity-matched") {
					t.Errorf("rerun reason for %s = %q, want changed-graph reason", identity, reason)
				}
			}
		})
	}
}
