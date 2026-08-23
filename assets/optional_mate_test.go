package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestOptionalMateMixedRowsRun(t *testing.T) {
	dir := t.TempDir()
	writeProofFile(t, filepath.Join(dir, "in", "se.txt"), "single")
	writeProofFile(t, filepath.Join(dir, "in", "pe_r1.txt"), "mate1")
	writeProofFile(t, filepath.Join(dir, "in", "pe_r2.txt"), "mate2")
	csv := "sample,read1,read2\n" +
		"se,in/se.txt,\n" +
		"pe,in/pe_r1.txt,in/pe_r2.txt\n"
	withSampleSheet(t, writeTempSheet(t, csv))

	raw := mustPlanJSON(t, OptionalMate())
	tasks := planAllTasks(t, raw)
	mustHaveTaskID(t, tasks, "se.copy_r1")
	mustHaveTaskID(t, tasks, "pe.copy_r1")
	mustHaveTaskID(t, tasks, "pe.copy_r2")
	assertNoTaskID(t, tasks, "se.copy_r2")
	if planTask(t, raw, "se.copy_r1").Image != "" || planTask(t, raw, "pe.copy_r2").Image != "" {
		t.Fatalf("optional-mate tasks used an image, want process")
	}

	g := mustComposeProof(t, OptionalMate())
	mustRunProof(t, g, dir, 2)
	assertFile(t, filepath.Join(dir, "work", "se", "r1.txt"), "single")
	assertFile(t, filepath.Join(dir, "work", "pe", "r1.txt"), "mate1")
	assertFile(t, filepath.Join(dir, "work", "pe", "r2.txt"), "mate2")
	if _, err := os.Stat(filepath.Join(dir, "work", "se", "r2.txt")); !os.IsNotExist(err) {
		t.Fatalf("single-end row published read2 dest, want absent")
	}
	got := inspectByIdentity(t, dir, gobble.ViewInstances)
	for _, id := range []string{"se.copy_r1", "pe.copy_r1", "pe.copy_r2"} {
		if got[id].Status != "succeeded" {
			t.Fatalf("%s status = %q, want succeeded", id, got[id].Status)
		}
	}
}

func TestOptionalMateOmittedRead2Header(t *testing.T) {
	dir := t.TempDir()
	writeProofFile(t, filepath.Join(dir, "in", "se.txt"), "single")
	withSampleSheet(t, writeTempSheet(t, "sample,read1\nse,in/se.txt\n"))
	raw := mustPlanJSON(t, OptionalMate())
	assertNoTaskID(t, planAllTasks(t, raw), "se.copy_r2")
	mustRunProof(t, mustComposeProof(t, OptionalMate()), dir, 0)
	assertFile(t, filepath.Join(dir, "work", "se", "r1.txt"), "single")
}

func TestOptionalMateBadSheetIsSampleSheetError(t *testing.T) {
	withSampleSheet(t, filepath.Join(t.TempDir(), "missing.csv"))
	mustComposeSheetError(t, OptionalMate())
}
