package sampleretentiontrimmed_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	sampleretentiontrimmed "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-trimmed"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContractAndBoundary(t *testing.T) {
	read := gobble.Literal("in/read.fastq.gz")
	task := cc.Task(t, sampleretentiontrimmed.Pipeline(read, 2, sampleretentiontrimmed.Options{}), "sample_retention_trimmed")
	if task.Script != "" || len(task.Command) != 6 || task.Command[0] != "python" || task.Command[1] != "-c" || !strings.Contains(task.Command[2], "line_count % 4") || task.Command[4] != "2" {
		t.Fatalf("command = %#v, script = %q, want one Python command with complete-FASTQ boundary at two records", task.Command, task.Script)
	}
	if task.Image != string(sampleretentiontrimmed.DefaultImage) {
		t.Fatalf("image = %q, want %q", task.Image, sampleretentiontrimmed.DefaultImage)
	}
	pc.AssertIOPath(t, task.Outputs, "accepted", "work/sample-retention-trimmed/trimmed_reads.accepted.txt")

	cc.Invalid(t, sampleretentiontrimmed.Pipeline(read, -1, sampleretentiontrimmed.Options{}))
	cc.Invalid(t, sampleretentiontrimmed.Pipeline(read, 2, sampleretentiontrimmed.Options{Options: modules.Options{ExtraArgs: []string{"--bypass"}}}))
}
