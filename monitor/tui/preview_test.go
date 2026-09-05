package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/monitor"
)

// The accepted preview used these eight samples and six stages. Reusing those
// exact states makes visual comparisons about layout, not different workloads.
func referencePreview(t *testing.T) *model {
	t.Helper()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	s := monitor.Snapshot{SchemaVersion: 2, Revision: "design-preview", Pipeline: "RNA-seq / cohort-01", ReadAt: now}
	s.Run.Status = "running"
	s.Run.Started = "2026-09-05T11:17:42Z"
	s.Run.Occupancy.Live = true
	shared := func(id, name, status, scope string) monitor.Task {
		return monitor.Task{Identity: id, TaskID: id, Name: name, Status: status, Attempt: 1, Display: gobble.TaskDisplay{Stage: name, Scope: scope}}
	}
	s.Tasks = []monitor.Task{shared("reference/index", "Reference", "succeeded", "shared"), shared("cohort/report", "Cohort report", "blocked", "cohort")}
	s.Tasks[0].Decision = "reused"
	s.Tasks[1].Reason = "Required quantification outputs are incomplete."
	states := [][4]string{
		{"succeeded", "succeeded", "succeeded", "succeeded"},
		{"succeeded", "succeeded", "succeeded", "succeeded"},
		{"succeeded", "succeeded", "succeeded", "succeeded"},
		{"succeeded", "succeeded", "running", "not-started"},
		{"succeeded", "succeeded", "succeeded", "running"},
		{"succeeded", "succeeded", "failed", "blocked"},
		{"succeeded", "running", "not-started", "not-started"},
		{"succeeded", "succeeded", "running", "not-started"},
	}
	names := []string{"Read QC", "Trim reads", "Alignment", "Quantification"}
	ids := []string{"qc", "trim", "align", "quant"}
	durations := []time.Duration{72 * time.Second, 272 * time.Second, 728 * time.Second, 189 * time.Second}
	for i, states := range states {
		sample := fmt.Sprintf("S%02d", i+1)
		for j, status := range states {
			id := sample + "/" + ids[j]
			task := monitor.Task{Identity: id, TaskID: id, Name: ids[j], Status: status, Attempt: 1, Executor: "docker", Display: gobble.TaskDisplay{Stage: names[j], Samples: []string{sample}, Scope: "sample"}}
			if status != "not-started" && status != "blocked" {
				task.Started = now.Add(-durations[j]).Format(time.RFC3339)
				if status != "running" {
					task.Ended = now.Format(time.RFC3339)
				}
			}
			if sample == "S03" && j < 2 {
				task.Decision = "reused"
			}
			if j == 2 {
				task.Resources.CPU = 8
				task.Resources.Memory = "32 GiB"
				task.Image = "demo/star:alignment"
				task.Command = []string{"STAR", "--runThreadN", "8", "--genomeDir", "reference/star", "--readFilesIn", sample + "_R1.fastq", sample + "_R2.fastq"}
			}
			if status == "failed" {
				task.Reason = "Input FASTQ could not be read. Inspect the staged file and command."
			}
			if status == "blocked" {
				task.Reason = "Blocked by failed upstream task S06 / Alignment."
			}
			s.Tasks = append(s.Tasks, task)
			if j > 0 {
				s.Edges = append(s.Edges, monitor.Edge{From: sample + "/" + ids[j-1], To: id})
			}
			if j == 2 {
				s.Edges = append(s.Edges, monitor.Edge{From: "reference/index", To: id})
			}
			if j == 3 {
				s.Edges = append(s.Edges, monitor.Edge{From: id, To: "cohort/report"})
			}
		}
	}
	stderr := strings.Join([]string{"[start] S06 / Alignment", "[info] Loading reference/star", "[info] Reading paired-end input", "", "[error] Cannot read S06_R2.fastq", "[error] Truncated FASTQ record", "[error] Alignment exited with code 1", "", "[state] Quantification blocked", "", "Illustrative preview log."}, "\n")
	s.Logs = []monitor.Log{{Identity: "S06/align", StderrTail: stderr, StderrSize: int64(len(stderr)), StdoutTail: "[start] S06/align\n[info] Preparing input data", StdoutSize: 43}}
	m, err := newModel("/workspaces/rnaseq-cohort-01", func(string) (monitor.Snapshot, error) { return s, nil }, s, false)
	if err != nil {
		t.Fatal(err)
	}
	m.now = now
	m.width = 132
	m.height = 44
	// No node is preselected in the reference preview. Keyboard focus starts at
	// Reference; its subtle selected surface provides the keyboard equivalent.
	return m
}

func TestRenderPreview(t *testing.T) {
	path := os.Getenv("GOBBLE_UI_PREVIEW")
	if path == "" {
		t.Skip("preview export not requested")
	}
	m := referencePreview(t)
	if err := os.WriteFile(path, []byte(m.View().Content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCaptureScreens(t *testing.T) {
	dir := os.Getenv("GOBBLE_UI_CAPTURE_DIR")
	if dir == "" {
		t.Skip("screen export not requested")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	m := referencePreview(t)
	save := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name+".ansi"), []byte(m.View().Content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	save("01-dashboard")
	press(m, '/', "/")
	press(m, 'S', "S")
	press(m, '0', "0")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyRight, "")
	save("02-sample-search")
	press(m, tea.KeyEnter, "")
	save("03-sample-dashboard")
	press(m, 't', "t")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyDown, "")
	save("04-task-inspector")
	cmd := press(m, tea.KeyEnter, "")
	if cmd != nil {
		m.Update(cmd())
	}
	save("05-task-logs")
	press(m, '3', "3")
	save("06-task-facts")
}
