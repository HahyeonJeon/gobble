package sampleretentionmapped_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	sampleretentionmapped "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-mapped"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContractAndBoundary(t *testing.T) {
	starLog := gobble.Literal("in/Log.final.out")
	task := cc.Task(t, sampleretentionmapped.Pipeline(starLog, 5, sampleretentionmapped.Options{}), "sample_retention_mapped")
	if task.Script != "" || len(task.Command) != 6 || task.Command[0] != "python" || task.Command[1] != "-c" || !strings.Contains(task.Command[2], "Uniquely mapped reads %") || task.Command[4] != "5" {
		t.Fatalf("command = %#v, script = %q, want one Python command with STAR uniquely-mapped boundary at five percent", task.Command, task.Script)
	}
	if task.Image != string(sampleretentionmapped.DefaultImage) {
		t.Fatalf("image = %q, want %q", task.Image, sampleretentionmapped.DefaultImage)
	}
	pc.AssertIOPath(t, task.Outputs, "accepted", "work/sample-retention-mapped/mapped_reads.accepted.txt")

	cc.Invalid(t, sampleretentionmapped.Pipeline(starLog, 101, sampleretentionmapped.Options{}))
	cc.Invalid(t, sampleretentionmapped.Pipeline(starLog, 5, sampleretentionmapped.Options{Options: modules.Options{ExtraArgs: []string{"--bypass"}}}))
}
