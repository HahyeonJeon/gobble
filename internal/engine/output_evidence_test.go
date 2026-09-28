package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

func completedOutput(t *testing.T) (string, []byte, preparation.LaunchIntent, OutputRequest) {
	t.Helper()
	ws, raw, in := launchFixture(t)
	in.SchemaVersion = 2
	continuationTools(t, in)
	useExec(t, continuationExecutor(t, in, "", nil))
	if d := RunPrepared(t.Context(), ws, raw, in, testInstallIdentity()); len(d) != 0 {
		t.Fatal(d)
	}
	DropHeldLease(ws)
	run, _, _ := readRunIdentity(ws)
	return ws, raw, in, OutputRequest{run.ID, "qc", 1, "html"}
}
func TestOutputEvidenceRecordedBytesAndNoMutation(t *testing.T) {
	ws, raw, in, q := completedOutput(t)
	// Retained output evidence does not require the source input or installed tools.
	if err := os.Remove(filepath.Join(ws, in.Binding.InputPath)); err != nil {
		t.Fatal(err)
	}
	lookupImageID = func(string) string { t.Fatal("output read queried installed tools"); return "" }
	before := workspaceContents(t, ws)
	got, err := ReadOutputEvidence(ws, raw, in, q, testInstallIdentity())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(ws, got.Path))
	if err != nil {
		t.Fatal(err)
	}
	if got.Recipe != "fastqc-v1" || got.SHA256 != preparedDigest(data) || got.Size != int64(len(data)) || got.Attempt != 1 || got.RunID != q.RunID || got.OriginDigest != LaunchDigest(in) || got.Snapshot == "" {
		t.Fatalf("attribution: %+v", got)
	}
	if !reflect.DeepEqual(before, workspaceContents(t, ws)) {
		t.Fatal("read mutated workspace")
	}
	// Attribution reports recorded bytes even if the source is changed; acquisition
	// verifies live bytes in the native host rather than changing engine history.
	writeCheckFile(t, filepath.Join(ws, got.Path), "changed")
	again, err := ReadOutputEvidence(ws, raw, in, q, testInstallIdentity())
	if err != nil || again != got {
		t.Fatal(again, err)
	}
}
func TestOutputEvidenceRefusals(t *testing.T) {
	cases := []struct {
		name, want string
		mutate     func(*jsonRun, *jsonPlan, *jsonTasksFile)
	}{
		{"changed-plan", "output saved plan changed", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) { p.Tasks[0].Command = []string{"other"} }},
		{"duplicate-attempt", "output attempts are ambiguous", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) { f.Tasks = append(f.Tasks, f.Tasks[len(f.Tasks)-1]) }},
		{"newer-failure", "output attempt is no longer current", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) {
			st := f.Tasks[len(f.Tasks)-1]
			st.Attempt++
			st.Status = StatusFailed
			f.Tasks = append(f.Tasks, st)
		}},
		{"failed", "output attempt has not succeeded", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) { f.Tasks[len(f.Tasks)-1].Status = StatusFailed }},
		{"wrong-command", "output task identity changed", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) { f.Tasks[len(f.Tasks)-1].Command = []string{"false"} }},
		{"wrong-image", "output task identity changed", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) {
			f.Tasks[len(f.Tasks)-1].ImageDigest = "sha256:" + strings.Repeat("c", 64)
		}},
		{"missing-checksum", "output checksum is unavailable", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) { f.Tasks[len(f.Tasks)-1].Checksums = nil }},
		{"duplicate-checksum", "output checksums are ambiguous", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) {
			st := &f.Tasks[len(f.Tasks)-1]
			st.Checksums = append(st.Checksums, st.Checksums[0])
		}},
		{"invalid-checksum", "output checksums are ambiguous", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) { f.Tasks[len(f.Tasks)-1].Checksums[0].SHA256 = "bad" }},
		{"negative-size", "output checksums are ambiguous", func(r *jsonRun, p *jsonPlan, f *jsonTasksFile) { f.Tasks[len(f.Tasks)-1].Checksums[0].Size = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws, raw, in, q := completedOutput(t)
			mutateContinuationControl(t, ws, tc.mutate)
			_, err := ReadOutputEvidence(ws, raw, in, q, testInstallIdentity())
			if err == nil || err.Error() != tc.want {
				t.Fatalf("want %s: %v", tc.want, err)
			}
		})
	}
}
func TestOutputEvidenceRequestAndOrigin(t *testing.T) {
	ws, raw, in, q := completedOutput(t)
	for _, bad := range []OutputRequest{{q.RunID, "qc", 0, "html"}, {"other", "qc", 1, "html"}, {q.RunID, "other", 1, "html"}, {q.RunID, "qc", 2, "html"}, {q.RunID, "qc", 1, "missing"}} {
		if _, err := ReadOutputEvidence(ws, raw, in, bad, testInstallIdentity()); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	other := in
	other.RequestID = "req_other"
	if _, err := ReadOutputEvidence(ws, raw, other, q, testInstallIdentity()); err == nil {
		t.Fatal("accepted another launch")
	}
	if _, err := ReadOutputEvidence(ws, append(raw, ' '), in, q, testInstallIdentity()); err == nil {
		t.Fatal("accepted altered declaration")
	}
	if _, err := ReadOutputEvidence(ws, raw, in, q, nil); err == nil {
		t.Fatal("accepted missing identity")
	}
}
func TestOutputEvidenceAfterContinuation(t *testing.T) {
	ws, raw, in := stoppedContinuation(t, "qc")
	review, err := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity())
	if err != nil {
		t.Fatal(err)
	}
	useExec(t, continuationExecutor(t, in, "", nil))
	if _, d := ContinuePrepared(t.Context(), ws, raw, in, continuationIntent(review), testInstallIdentity()); len(d) > 0 {
		t.Fatal(d)
	}
	DropHeldLease(ws)
	run, _, _ := readRunIdentity(ws)
	q := OutputRequest{run.ID, "qc", 2, "html"}
	got, err := ReadOutputEvidence(ws, raw, in, q, testInstallIdentity())
	if err != nil || got.Attempt != 2 {
		t.Fatal(got, err)
	}
	q.Attempt = 1
	if _, err := ReadOutputEvidence(ws, raw, in, q, testInstallIdentity()); err == nil {
		t.Fatal("accepted older attempt")
	}
	q = OutputRequest{run.ID, "trim", 1, "trimmed_read1"}
	if got, err = ReadOutputEvidence(ws, raw, in, q, testInstallIdentity()); err != nil || got.Attempt != 1 || got.Recipe != "trim-galore-v1" {
		t.Fatal(got, err)
	}
}
