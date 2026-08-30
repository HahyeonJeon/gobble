package bismarkreport_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	bismarkreport "github.com/HahyeonJeon/gobble/assets/modules/bismark-report"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestBismarkReportBindsEveryNamedInput(t *testing.T) {
	task := cc.Task(t, bismarkreport.Pipeline(
		gobble.Literal("work/sample_PE_report.txt"),
		gobble.Literal("work/sample_pe.deduplication_report.txt"),
		gobble.Literal("work/sample_pe.deduplicated_splitting_report.txt"),
		gobble.Literal("work/sample_pe.deduplicated.M-bias.txt"),
		bismarkreport.Options{OutDir: gobble.Dir("results/reports/sample"), Prefix: "sample"},
	), "bismark_report")
	if !pc.ContainsAll(task.Command, "bismark2report", "--alignment_report", "work/sample_PE_report.txt", "--dedup_report", "work/sample_pe.deduplication_report.txt", "--splitting_report", "work/sample_pe.deduplicated_splitting_report.txt", "--mbias_report", "work/sample_pe.deduplicated.M-bias.txt", "--output", "sample.bismark_report.html") {
		t.Fatalf("command = %#v", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "html", "results/reports/sample/sample.bismark_report.html")
}
