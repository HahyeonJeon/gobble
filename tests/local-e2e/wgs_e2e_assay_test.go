//go:build live

package local_e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

func TestWGSSuccessInspectReleaseResume(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageWGSPins(t, dir)
	g, err := gobble.Compose(wgs.Pipeline())
	if err != nil {
		t.Fatalf("Compose(wgs.Pipeline()) error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run(wgs.Pipeline())", err)
	}
	assertOccupied(t, dir)
	for _, rel := range []string{
		"results/wgs/multiqc/multiqc_report.html",
		"results/wgs/samples/testN/alignment/testN.recalibrated.bam",
		"results/wgs/samples/testT/alignment/testT.recalibrated.bam",
		"results/wgs/joint/joint_germline.vcf.gz",
	} {
		requireRegularFile(t, filepath.Join(dir, filepath.FromSlash(rel)))
	}
	recoverAfterSuccessAPI(t, g, dir, 2)
}

func TestThinFailFixtureInspectReleaseResume(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	g, err := gobble.Compose(thinFailFixture())
	if err != nil {
		t.Fatalf("Compose(thin fail fixture) error = %v", err)
	}
	err = gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t))
	requireRunError(t, "contained failure", err, gobble.DefectFailed, "fail")
	assertOccupied(t, dir)
	if _, statErr := os.Stat(filepath.Join(dir, "out", "fail.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("failed dest was published")
	}
	got, readErr := os.ReadFile(filepath.Join(dir, "out", "ok.txt"))
	if readErr != nil {
		t.Fatalf("independent dest: %v", readErr)
	}
	if string(got) != "ok\n" && string(got) != "ok" {
		t.Fatalf("independent dest got %q, want ok", got)
	}

	remaining := instanceByID(inspectJSONL(t, dir, gobble.ViewRemaining))
	if remaining["fail"] == nil || remaining["fail"]["remaining"] != true {
		t.Fatalf("fail remaining got %#v", remaining["fail"])
	}

	if err := gobble.Release(dir); err != nil {
		fatalAPIError(t, "Release()", err)
	}
	err = gobble.Resume(t.Context(), g, dir, 2, testOccupyOption(t))
	requireRunError(t, "resume remaining", err, gobble.DefectFailed, "fail")
	assertOccupied(t, dir)
	reuse := instanceByID(inspectJSONL(t, dir, gobble.ViewReuse))
	if reuse["fail"] == nil || reuse["fail"]["decision"] != "rerun" {
		t.Fatalf("fail resume reuse got %#v", reuse["fail"])
	}
	if remaining := instanceByID(inspectJSONL(t, dir, gobble.ViewRemaining)); remaining["fail"] == nil {
		t.Fatalf("resume remaining missing fail: %#v", remaining)
	}
}

func thinFailFixture() *gobble.Pipeline {
	p := gobble.NewPipeline("thin-fail")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "fail",
		Command: []string{"sh", "-c", "exit 1"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "fail", Ext: ".txt"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "ok",
		Command: []string{"sh", "-c", "echo ok > out/ok.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "ok", Ext: ".txt"},
		}},
	})
	return p
}
