package tui

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/monitor"
	"github.com/charmbracelet/x/ansi"
)

func testSnapshot() monitor.Snapshot {
	s := monitor.Snapshot{SchemaVersion: 2, Revision: "one", Pipeline: "RNA-seq", ReadAt: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)}
	s.Run.Status, s.Run.Started, s.Run.Occupancy.Live = "running", "2026-09-05T11:58:00Z", true
	s.Tasks = []monitor.Task{
		{Identity: "reference", TaskID: "reference", Name: "Reference", Status: "succeeded"},
		{Identity: "S01.align", TaskID: "a", Name: "Alignment", Status: "running", Display: gobble.TaskDisplay{Samples: []string{"S01"}}},
		{Identity: "S010.align", TaskID: "b", Name: "Alignment", Status: "failed", Reason: "Input read failed", Display: gobble.TaskDisplay{Samples: []string{"S010"}}},
	}
	s.Edges = []monitor.Edge{{From: "reference", To: "a"}, {From: "reference", To: "b"}}
	return s
}

func testModel(t *testing.T) *model {
	t.Helper()
	m, err := newModel("/runs/rnaseq", func(string) (monitor.Snapshot, error) { return testSnapshot(), nil }, testSnapshot(), true)
	if err != nil {
		t.Fatal(err)
	}
	m.now = m.data.Snapshot.ReadAt
	return m
}

func press(m *model, code rune, text string) tea.Cmd {
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: code, Text: text}))
	return cmd
}

func TestSearchExactSelectionAndAttentionBack(t *testing.T) {
	m := testModel(t)
	press(m, '/', "/")
	for _, r := range "S01" {
		press(m, r, string(r))
	}
	press(m, tea.KeyEnter, "")
	if m.sample != "S01" || m.screen != dashboardScreen {
		t.Fatalf("selection: %q %v", m.sample, m.screen)
	}
	press(m, '!', "!")
	press(m, tea.KeyEnter, "")
	if m.task != "S010.align" || m.screen != detailScreen {
		t.Fatal("global attention did not open failed task")
	}
	press(m, tea.KeyEscape, "")
	if m.screen != attentionScreen || m.sample != "S01" {
		t.Fatal("detail lost previous navigation context")
	}
	press(m, tea.KeyEscape, "")
	press(m, tea.KeyEscape, "")
	if m.sample != "" {
		t.Fatal("escape did not clear sample scope")
	}
}

func TestRefreshSingleFlightStaleDataAndIdentityFocus(t *testing.T) {
	m := testModel(t)
	cmd := m.refresh()
	if cmd == nil || m.refresh() != nil {
		t.Fatal("overlapping reads permitted")
	}
	m.Update(loadedMsg{err: errors.New("snapshot changed")})
	if m.reading || m.data.Snapshot.Revision != "one" || !strings.Contains(ansi.Strip(m.View().Content), "STALE") {
		t.Fatal("failed read did not retain stale snapshot")
	}
	m.screen, m.listIndex = tasksScreen, 2
	s := testSnapshot()
	s.Revision = "two"
	s.Tasks = append([]monitor.Task{{Identity: "new", TaskID: "new", Name: "New", Status: "running"}}, s.Tasks...)
	m.Update(loadedMsg{snapshot: s})
	if m.err != nil || m.listIndex != 3 {
		t.Fatal("refresh lost selected identity")
	}
	if m.refresh() == nil {
		t.Fatal("refresh remained blocked")
	}
}

func TestTerminalFramesAndUntrustedText(t *testing.T) {
	m := testModel(t)
	m.task = "S010.align"
	m.data.Snapshot.Logs = []monitor.Log{{Identity: m.task, StderrTail: "hello\x1b]52;c;secret\a\x1b[31m world\x1b[0m\n한국어", StderrSize: 40}}
	for _, size := range [][2]int{{20, 8}, {44, 16}, {80, 24}, {120, 36}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, page := range []screen{dashboardScreen, searchScreen, tasksScreen, detailScreen, attentionScreen, helpScreen} {
			m.screen = page
			v := m.View()
			lines := strings.Split(v.Content, "\n")
			if !v.AltScreen || len(lines) != size[1] {
				t.Fatalf("frame %v %v height %d", size, page, len(lines))
			}
			for _, line := range lines {
				if lipgloss.Width(line) != size[0] {
					t.Fatalf("frame width %d: %q", size[0], line)
				}
			}
			if strings.Contains(v.Content, "secret") || strings.Contains(v.Content, "\x1b]52") {
				t.Fatal("terminal control escaped sanitization")
			}
		}
	}
	if got := clean("hello\x1b]52;c;secret\a world"); got != "hello world" {
		t.Fatalf("sanitized: %q", got)
	}
}

func TestLogPauseKeepsVisibleTail(t *testing.T) {
	m := testModel(t)
	m.screen, m.task = detailScreen, "S01.align"
	m.data.Snapshot.Logs = []monitor.Log{{Identity: m.task, StderrTail: strings.Repeat("line\n", 100)}}
	want := m.logTailOffset()
	press(m, 'f', "f")
	if m.follow || m.logOffset != want {
		t.Fatalf("pause jumped from tail: %d want %d", m.logOffset, want)
	}
	press(m, tea.KeyUp, "")
	if m.logOffset >= want {
		t.Fatal("paused tail did not scroll")
	}
	press(m, tea.KeyEnd, "")
	if !m.follow {
		t.Fatal("End did not resume following")
	}
}

func TestWatchRejectsNonTerminalBeforeReading(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "redirected")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := Watch(t.Context(), "missing", f, f); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("redirected watch: %v", err)
	}
}

// CI can export a deterministic rendering of the actual terminal View for
// visual review. No mock HTML or separate rendering implementation is used.
func TestRenderPreview(t *testing.T) {
	path := os.Getenv("GOBBLE_UI_PREVIEW")
	if path == "" {
		t.Skip("preview export not requested")
	}
	m := testModel(t)
	s := testSnapshot()
	s.Pipeline = "RNA-seq / engineering preview"
	s.Tasks = []monitor.Task{{Identity: "reference.star_index", TaskID: "ref", Name: "STAR index", Status: "succeeded", Display: gobble.TaskDisplay{Scope: gobble.DisplayShared}}}
	s.Edges = nil
	for _, sample := range []string{"S01", "S02", "S03", "S04"} {
		for i, name := range []string{"FastQC", "Fastp", "STAR alignment", "Salmon quant"} {
			id := sample + "." + name
			status := "succeeded"
			if i == 2 && sample == "S03" {
				status = "running"
			}
			if i == 2 && sample == "S04" {
				status = "failed"
			}
			if i == 3 && sample == "S03" {
				status = "not-started"
			}
			if i == 3 && sample == "S04" {
				status = "blocked"
			}
			task := monitor.Task{Identity: id, TaskID: id, Name: name, Status: status, Display: gobble.TaskDisplay{Samples: []string{sample}}}
			if status == "failed" {
				task.Reason = "Input read failed · inspect stderr"
			}
			s.Tasks = append(s.Tasks, task)
			if i == 2 {
				s.Edges = append(s.Edges, monitor.Edge{From: "ref", To: id}, monitor.Edge{From: sample + ".Fastp", To: id})
			}
			if i == 3 {
				s.Edges = append(s.Edges, monitor.Edge{From: sample + ".STAR alignment", To: id}, monitor.Edge{From: id, To: "report"})
			}
		}
	}
	s.Tasks = append(s.Tasks, monitor.Task{Identity: "cohort.multiqc", TaskID: "report", Name: "MultiQC report", Status: "blocked", Display: gobble.TaskDisplay{Scope: gobble.DisplayCohort}})
	d, err := monitor.Build(s)
	if err != nil {
		t.Fatal(err)
	}
	m.data = d
	for _, stage := range d.Stages {
		if stage.Name == "STAR alignment" {
			m.stage = stage.ID
		}
	}
	m.style = newTheme(false)
	m.width, m.height = 128, 40
	if err := os.WriteFile(path, []byte(m.View().Content), 0600); err != nil {
		t.Fatal(err)
	}
}
