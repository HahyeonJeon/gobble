package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/monitor"
	"github.com/charmbracelet/x/ansi"
)

func TestReviewSearchPasteAndCancel(t *testing.T) {
	m := testModel(t)
	m.screen, m.task, m.detailReturn = detailScreen, "S01.align", tasksScreen
	press(m, '/', "/")
	m.Update(tea.PasteMsg{Content: "S01\x1b]52;c;secret\a\n"})
	if m.query != "S01" {
		t.Fatalf("pasted sample ID: %q", m.query)
	}
	press(m, tea.KeyEscape, "")
	if m.screen != detailScreen || m.task != "S01.align" {
		t.Fatal("cancelling search lost task context")
	}
}

func TestReviewExactCaseAndSearchFocusSurviveRefresh(t *testing.T) {
	m := testModel(t)
	s := testSnapshot()
	task := s.Tasks[1]
	task.Identity, task.TaskID = "lower.align", "lower"
	task.Display = gobble.TaskDisplay{Samples: []string{"s01"}}
	s.Tasks = append(s.Tasks, task)
	m.Update(loadedMsg{snapshot: s})
	press(m, '/', "/")
	for _, r := range "s01" {
		press(m, r, string(r))
	}
	press(m, tea.KeyEnter, "")
	if m.sample != "s01" {
		t.Fatalf("selected similarly named sample %q", m.sample)
	}
	press(m, '/', "/")
	m.searchIndex = 1 // S010
	added := s.Tasks[1]
	added.Identity, added.TaskID = "new.align", "new"
	added.Display.Samples = []string{"S00"}
	s.Tasks = append(s.Tasks, added)
	m.Update(loadedMsg{snapshot: s})
	press(m, tea.KeyEnter, "")
	if m.sample != "S010" {
		t.Fatalf("refresh moved highlighted result to %q", m.sample)
	}
}

func TestReviewDetailRefreshPreservesReturnPosition(t *testing.T) {
	m := testModel(t)
	m.screen, m.listIndex = tasksScreen, 2
	press(m, tea.KeyEnter, "")
	s := testSnapshot()
	added := s.Tasks[0]
	added.Identity, added.TaskID = "added", "added"
	s.Tasks = append([]monitor.Task{added}, s.Tasks...)
	m.Update(loadedMsg{snapshot: s})
	press(m, tea.KeyEscape, "")
	if m.listIndex != 3 {
		t.Fatalf("return position changed to %d", m.listIndex)
	}
}

func TestReviewHelpCanReachEveryInstruction(t *testing.T) {
	m := testModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	press(m, '?', "?")
	for range 10 {
		press(m, tea.KeyPgDown, "")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "NO_COLOR") {
		t.Fatal("short terminal cannot read the end of help")
	}
}

func TestReviewFullMetadataIsReachable(t *testing.T) {
	m := testModel(t)
	s := testSnapshot()
	task := &s.Tasks[2]
	task.Command = []string{"tool", strings.Repeat("long argument ", 90) + "ARGUMENT_END"}
	task.Script = strings.Repeat("echo step\n", 50) + "SCRIPT_END"
	task.Reason = strings.Repeat("Long failure reason. ", 40) + "REASON_END"
	task.Error = &monitor.TaskError{Unit: "failure.location", Message: task.Reason}
	task.Decision, task.ReuseReason = "rerun", "input-fingerprint-changed"
	task.Started, task.Ended = "2026-09-05T11:00:00Z", "2026-09-05T11:05:00Z"
	m.Update(loadedMsg{snapshot: s})
	for _, size := range [][2]int{{80, 24}, {132, 44}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.screen, m.listIndex = tasksScreen, 2
		press(m, '3', "3")
		var seen strings.Builder
		for range 100 {
			seen.WriteString(ansi.Strip(m.View().Content))
			press(m, tea.KeyDown, "")
		}
		press(m, tea.KeyEnd, "")
		seen.WriteString(ansi.Strip(m.View().Content))
		for _, want := range []string{"ARGUMENT_END", "SCRIPT_END", "REASON_END", "failure.location", "input-fingerprint-changed", "2026-09-05T11:00:00Z"} {
			if !strings.Contains(seen.String(), want) {
				t.Fatalf("full facts hide %q at %v", want, size)
			}
		}
		press(m, '1', "1")
		if m.showMetadata || m.logStream != "stdout" {
			t.Fatal("cannot return to logs")
		}
	}
}

func TestReviewOverlayRefreshKeepsLogsAndAttentionContext(t *testing.T) {
	m := testModel(t)
	m.screen, m.task, m.detailReturn = detailScreen, "S010.align", attentionScreen
	var requested string
	m.read = func(id string) (monitor.Snapshot, error) { requested = id; return testSnapshot(), nil }
	press(m, '?', "?")
	cmd := m.refresh()
	m.Update(cmd())
	if requested != "S010.align" || len(m.listTasks()) != 1 {
		t.Fatal("help discarded the inspector context")
	}
	s := testSnapshot()
	s.Tasks = s.Tasks[:2]
	s.Edges = s.Edges[:1]
	m.Update(loadedMsg{snapshot: s})
	press(m, tea.KeyEscape, "")
	if m.screen != attentionScreen || m.task != "" {
		t.Fatal("removed task reopened a broken inspector")
	}
}

func TestReviewNarrowRunStateAndUnicodeLabels(t *testing.T) {
	m := testModel(t)
	s := testSnapshot()
	s.Run.Status = "failed"
	s.Tasks[0].Display.Stage = "Cafe\u0301 🧑‍🔬"
	m.Update(loadedMsg{snapshot: s})
	m.Update(tea.WindowSizeMsg{Width: 44, Height: 20})
	content := ansi.Strip(m.View().Content)
	if !strings.Contains(content, "× Failed") {
		t.Fatal("narrow header hid run failure")
	}
	if !strings.Contains(content, "Cafe\u0301 🧑‍🔬") {
		t.Fatalf("graph split Unicode graphemes: %s", content)
	}
	for _, line := range wrapPlain("Cafe\u0301 🧑‍🔬 한국어", 7) {
		if strings.Contains(line, "\u200d") && !strings.Contains(line, "🧑‍🔬") {
			t.Fatal("line wrap split joined emoji")
		}
	}
}

func TestReviewRefreshRevealsFocusedStageAfterInsertion(t *testing.T) {
	m := referencePreview(t)
	m.height = 30
	m.stage = m.data.Stages[len(m.data.Stages)-1].ID
	m.revealStage()
	s := m.data.Snapshot
	for i := range 10 {
		id := fmt.Sprintf("new-%d", i)
		s.Tasks = append([]monitor.Task{{Identity: id, TaskID: id, Name: id, Status: "not-started"}}, s.Tasks...)
	}
	m.Update(loadedMsg{snapshot: s})
	r := layout(m.data.Stages, m.graphWidth()).nodes[m.stage]
	if r.y < m.graphOffset || r.y+r.h > m.graphOffset+m.graphViewportHeight() {
		t.Fatal("refresh scrolled selected stage out of view")
	}
}
