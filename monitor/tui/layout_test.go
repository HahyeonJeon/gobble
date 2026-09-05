package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble/monitor"
	"github.com/charmbracelet/x/ansi"
)

func TestReferenceLayoutAndSpatialNavigation(t *testing.T) {
	m := referencePreview(t)
	g := layout(m.data.Stages, m.graphWidth())
	byName := map[string]rect{}
	for _, s := range m.data.Stages {
		byName[s.Name] = g.nodes[s.ID]
	}
	ref, qc, trim, align, quant, report := byName["Reference"], byName["Read QC"], byName["Trim reads"], byName["Alignment"], byName["Quantification"], byName["Cohort report"]
	if ref.y != qc.y || align.y != trim.y || quant.y != report.y || ref.x != align.x || qc.x != trim.x || align.y <= ref.y || quant.y <= align.y {
		t.Fatalf("preview flow not preserved: %+v", byName)
	}
	press(m, tea.KeyRight, "")
	if m.data.Stages[m.stageIndex()].Name != "Read QC" {
		t.Fatal("right did not move to adjacent card")
	}
	press(m, tea.KeyDown, "")
	if m.data.Stages[m.stageIndex()].Name != "Trim reads" {
		t.Fatal("down did not stay in column")
	}
	press(m, tea.KeyLeft, "")
	if m.data.Stages[m.stageIndex()].Name != "Alignment" {
		t.Fatal("left did not follow folded row")
	}
	press(m, tea.KeyDown, "")
	if m.data.Stages[m.stageIndex()].Name != "Quantification" {
		t.Fatal("down lost stage selection")
	}
}

func TestSearchPreservesGlobalContextAndSelectsGridCell(t *testing.T) {
	m := referencePreview(t)
	press(m, '/', "/")
	press(m, 'S', "S")
	press(m, '0', "0")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyRight, "")
	content := ansi.Strip(m.View().Content)
	for _, want := range []string{"SUCCESSFUL TASKS", "23 / 34", "SAMPLE SEARCH", "PIPELINE GRAPH", "ATTENTION", "SAMPLE PROGRESS"} {
		if !strings.Contains(content, want) {
			t.Fatalf("search hid %q", want)
		}
	}
	press(m, tea.KeyEnter, "")
	if m.sample != "S06" {
		t.Fatalf("grid selected %q", m.sample)
	}
	content = ansi.Strip(m.View().Content)
	if !strings.Contains(content, "23 / 34") || !strings.Contains(content, "2 / 4 owned tasks") {
		t.Fatal("sample selection changed global metrics")
	}
}

func TestInspectorKeepsListAndLogsInOneFrame(t *testing.T) {
	m := referencePreview(t)
	m.sample = "S06"
	press(m, 't', "t")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyDown, "")
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil {
		t.Fatal("detail did not request logs")
	}
	m.Update(cmd())
	content := ansi.Strip(m.View().Content)
	for _, want := range []string{"S06 / Read QC", "S06 / Quantification", "S06 / Alignment", "[2 STDERR]", "Cannot read S06_R2.fastq", "CPU request 8"} {
		if !strings.Contains(content, want) {
			t.Fatalf("inspector hid %q", want)
		}
	}
	press(m, tea.KeyEscape, "")
	if m.screen != tasksScreen || m.listIndex != 2 || m.sample != "S06" {
		t.Fatal("back lost selection")
	}
}

func TestReferenceFramesAtTerminalSizes(t *testing.T) {
	m := referencePreview(t)
	for _, size := range [][2]int{{44, 20}, {80, 24}, {100, 32}, {132, 44}, {160, 50}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, page := range []screen{dashboardScreen, searchScreen, tasksScreen, detailScreen, attentionScreen, helpScreen} {
			m.screen = page
			m.task = "S06/align"
			m.sample = "S06"
			for _, mono := range []bool{false, true} {
				m.style = newTheme(mono)
				v := m.View()
				lines := strings.Split(v.Content, "\n")
				if len(lines) != size[1] {
					t.Fatalf("height at %v/%v: %d", size, page, len(lines))
				}
				for _, line := range lines {
					if lipgloss.Width(line) != size[0] {
						t.Fatalf("width at %v/%v: %q", size, page, line)
					}
				}
				if !strings.Contains(ansi.Strip(lines[len(lines)-1]), "quit") {
					t.Fatalf("footer lost at %v/%v", size, page)
				}
			}
		}
	}
}

func TestFoldedLayoutKeepsCardsDisjointAndVisibleWhenFocused(t *testing.T) {
	m := referencePreview(t)
	for _, width := range []int{44, 80, 132} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		g := layout(m.data.Stages, m.graphWidth())
		for i, a := range m.data.Stages {
			ra := g.nodes[a.ID]
			if ra.x < 0 || ra.x+ra.w > m.graphWidth() {
				t.Fatal("node outside graph width")
			}
			for _, b := range m.data.Stages[i+1:] {
				rb := g.nodes[b.ID]
				if ra.x < rb.x+rb.w && ra.x+ra.w > rb.x && ra.y < rb.y+rb.h && ra.y+ra.h > rb.y {
					t.Fatal("overlapping cards")
				}
			}
		}
		m.stage = m.data.Stages[len(m.data.Stages)-1].ID
		m.revealStage()
		r := g.nodes[m.stage]
		if m.graphOffset > r.y || r.y+r.h > m.graphOffset+max(1, m.bodyHeight()-2) {
			t.Fatal("focused node outside viewport")
		}
	}
}

func TestDistributionPreservesCellBudget(t *testing.T) {
	counts := monitor.Counts{Total: 101, Succeeded: 93, Running: 3, Failed: 1, Blocked: 2, Unknown: 2}
	for width := 1; width <= 128; width++ {
		if got := lipgloss.Width(distribution(counts, width, newTheme(false))); got != width {
			t.Fatalf("width %d rendered %d", width, got)
		}
	}
}

func TestPagingMatchesVisibleContent(t *testing.T) {
	m := referencePreview(t)
	press(m, '/', "/")
	press(m, tea.KeyPgDown, "")
	if m.graphOffset == 0 || !strings.Contains(ansi.Strip(m.View().Content), "Cohort report") {
		t.Fatal("search graph did not pan to downstream cards")
	}
	for range 5 {
		press(m, tea.KeyPgDown, "")
	}
	press(m, tea.KeyPgUp, "")
	if m.graphOffset != 0 {
		t.Fatal("paging past the graph end accumulated hidden scroll distance")
	}
	press(m, tea.KeyEscape, "")
	press(m, 't', "t")
	width, _ := m.inspectorWidths()
	before := m.taskList(width, m.bodyHeight()-2, m.data.Snapshot.Tasks[m.listTasks()[0]].Identity)
	pageSize := 0
	for _, line := range before {
		if strings.Contains(ansi.Strip(line), " / ") && !strings.Contains(ansi.Strip(line), "tasks succeeded") && !strings.Contains(ansi.Strip(line), "PgDn") {
			pageSize++
		}
	}
	if pageSize < 2 || !strings.Contains(ansi.Strip(strings.Join(before, "\n")), "of 34") {
		t.Fatal("task page hid its rows or page indicator")
	}
	press(m, tea.KeyPgDown, "")
	if m.listIndex != pageSize {
		t.Fatalf("page key skipped tasks: %d visible, moved %d", pageSize, m.listIndex)
	}
	press(m, tea.KeyPgUp, "")
	if m.listIndex != 0 {
		t.Fatal("page up did not return to first task")
	}
}
