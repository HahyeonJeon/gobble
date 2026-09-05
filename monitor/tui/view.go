package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m *model) margin() int {
	if m.width >= 60 {
		return 2
	}
	return 1
}

func (m *model) contentWidth() int { return max(1, m.width-2*m.margin()) }
func (m *model) comfortable() bool { return m.width >= 100 && m.height >= 32 }
func (m *model) bodyHeight() int   { return max(1, m.height-len(m.header())-2) }
func (m *model) hasSidebar() bool  { return m.contentWidth() >= 96 }
func (m *model) sidebarWidth() int { return min(32, max(27, m.contentWidth()/4)) }

func (m *model) View() tea.View {
	w := m.contentWidth()
	var lines []string
	if m.width < 44 || m.height < 20 {
		lines = []string{m.style.active.Bold(true).Render("GOBBLE / pipeline monitor"), oneLine(m.data.Snapshot.Pipeline), "Run: " + stateLabel(m.data.Snapshot.Run.Status),
			fmt.Sprintf("Succeeded %d/%d · Failed %d", m.data.Total.Succeeded, m.data.Total.Total, m.data.Total.Failed),
			fmt.Sprintf("Running %d · Attention %d", m.data.Total.Running, m.data.Total.Attention()), "Resize to at least 44 × 20", "q exits monitor; execution continues"}
		if m.err != nil {
			lines = append(lines, "STALE · "+oneLine(m.err.Error()))
		}
	} else {
		lines = append([]string{""}, m.header()...)
		var body []string
		switch m.screen {
		case dashboardScreen, searchScreen:
			body = m.dashboardView()
		case tasksScreen, attentionScreen, detailScreen:
			body = m.inspectorView()
		case helpScreen:
			body = m.helpView()
		}
		lines = append(lines, frame(body, w, m.bodyHeight())...)
		lines = append(lines, m.footer())
	}
	lines = frame(lines, w, m.height)
	for i := range lines {
		lines[i] = strings.Repeat(" ", m.margin()) + lines[i] + strings.Repeat(" ", m.margin())
	}
	v := tea.NewView(strings.Join(frame(lines, m.width, m.height), "\n"))
	v.AltScreen = true
	if !m.style.monochrome {
		v.BackgroundColor = lipgloss.Color(backgroundColor)
		v.ForegroundColor = lipgloss.Color(textColor)
	}
	return v
}

func elapsed(start, end string, now time.Time) string {
	t, err := time.Parse(time.RFC3339Nano, start)
	if err != nil {
		return "—"
	}
	if end != "" {
		if e, err := time.Parse(time.RFC3339Nano, end); err == nil {
			now = e
		}
	}
	return max(time.Duration(0), now.Sub(t)).Truncate(time.Second).String()
}

func (m *model) observationTime() time.Time {
	if m.err != nil || !m.data.Snapshot.Run.Occupancy.Live {
		return m.data.Snapshot.ReadAt
	}
	return m.now
}

func (m *model) header() []string {
	w := m.contentWidth()
	c := m.data.Total
	s := m.data.Snapshot
	name := s.Pipeline
	if name == "" {
		name = s.Run.ID
	}
	title := ends(m.style.title.Render(oneLine(name)), m.style.state(s.Run.Status).Render(stateLabel(s.Run.Status))+m.style.dim.Render("  "+elapsed(s.Run.Started, s.Run.Ended, m.observationTime())+" elapsed"), w)
	lines := []string{ends(m.style.active.Bold(true).Render("G O B B L E")+m.style.dim.Render(" / pipeline monitor"), m.style.dim.Render("READ ONLY"), w), title, m.style.dim.Render(oneLine(m.workspace))}
	if w < 90 {
		lines[1] = ends(m.style.title.Render(oneLine(name)), m.style.state(s.Run.Status).Render(stateLabel(s.Run.Status)), w)
		lines[2] = m.style.dim.Render(elapsed(s.Run.Started, s.Run.Ended, m.observationTime()) + " elapsed · " + oneLine(m.workspace))
	}
	if m.comfortable() {
		lines = append(lines, "")
		lines = append(lines, m.metrics(w)...)
		lines = append(lines, "", distribution(c, w, m.style))
	} else {
		lines = append(lines, fmt.Sprintf("✓ %d/%d succeeded   ● %d running   × %d failed", c.Succeeded, c.Total, c.Running, c.Failed), distribution(c, w, m.style))
	}
	legend := []string{}
	for i, p := range stateParts(c) {
		if i < 5 || p.n > 0 {
			style := m.style.dim
			if !m.style.monochrome {
				color := p.color
				if p.label == "pending" {
					color = mutedColor
				}
				style = style.Foreground(lipgloss.Color(color))
			}
			legend = append(legend, style.Render(fmt.Sprintf("%d %s", p.n, p.label)))
		}
	}
	if c.Reused > 0 {
		legend = append(legend, m.style.dim.Render(fmt.Sprintf("↺ %d reused", c.Reused)))
	}
	if c.Unexpanded > 0 {
		legend = append(legend, m.style.warn.Render(fmt.Sprintf("%d expanding", c.Unexpanded)))
	}
	if c.SkippedTemplates > 0 {
		legend = append(legend, m.style.dim.Render(fmt.Sprintf("%d templates skipped", c.SkippedTemplates)))
	}
	lines = append(lines, wrapStyledParts(legend, w)...)
	if m.comfortable() {
		lines = append(lines, "")
	}
	query := "Sample ID, e.g. S06"
	style := m.style.panel
	if m.sample != "" {
		query = m.sample + "    · Esc clears selection"
	}
	if m.screen == searchScreen {
		query = m.query + "▏"
		style = m.style.selected
	}
	lines = append(lines, style.Render(fit("  /  FIND SAMPLE    "+oneLine(query), w)))
	scope := fmt.Sprintf("All %d samples · stage totals", len(m.data.Samples))
	hint := "Enter opens stage tasks"
	if m.sample != "" {
		sc := m.data.Count(m.data.SampleTasks(m.sample))
		scope = fmt.Sprintf("Sample %s · %d / %d owned tasks succeeded", oneLine(m.sample), sc.Succeeded, sc.Total)
		hint = "Shared / cohort nodes are context"
	}
	lines = append(lines, ends(m.style.active.Render(scope), m.style.dim.Render(hint), w))
	if m.err != nil {
		lines = append(lines, m.style.bad.Render("STALE · "+oneLine(m.err.Error())))
	} else if s.Run.Unknown {
		lines = append(lines, m.style.warn.Render("BACKEND UNCONFIRMED · inspect affected work before recovery"))
	} else if s.Run.Status == "running" && !s.Run.Occupancy.Live {
		lines = append(lines, m.style.warn.Render("OWNER NOT LIVE · showing recorded progress"))
	}
	if m.comfortable() {
		lines = append(lines, "")
	}
	return lines
}

func wrapStyledParts(parts []string, width int) []string {
	var rows []string
	line := ""
	for _, p := range parts {
		if line != "" && lipgloss.Width(line)+3+lipgloss.Width(p) > width {
			rows = append(rows, line)
			line = ""
		}
		if line != "" {
			line += "   "
		}
		line += p
	}
	if line != "" {
		rows = append(rows, line)
	}
	return rows
}

func (m *model) metrics(width int) []string {
	c := m.data.Total
	activeStages := 0
	for _, s := range m.data.Stages {
		if s.Counts.Running > 0 {
			activeStages++
		}
	}
	widths := []int{(width - 4) * 4 / 10, (width - 4) * 3 / 10, 0}
	widths[2] = width - 4 - widths[0] - widths[1]
	labels := []string{"SUCCESSFUL TASKS", "RUNNING NOW", "TASKS FAILED"}
	values := []string{fmt.Sprintf("%d / %d     %.0f%%", c.Succeeded, c.Total, c.Percent()), fmt.Sprint(c.Running), fmt.Sprint(c.Failed)}
	notes := []string{"Known work · includes reused", fmt.Sprintf("%d active stages", activeStages), fmt.Sprintf("%d tasks blocked", c.Blocked)}
	colors := []string{textColor, activeColor, badColor}
	rows := make([]string, 3)
	for i, width := range widths {
		label, value, note := m.style.panel, m.style.panel.Bold(true), m.style.panel
		if !m.style.monochrome {
			label = label.Foreground(lipgloss.Color(mutedColor))
			note = note.Foreground(lipgloss.Color(mutedColor))
			value = value.Foreground(lipgloss.Color(colors[i]))
		}
		if i > 0 {
			for r := range rows {
				rows[r] += "  "
			}
		}
		rows[0] += label.Render(fit("  "+labels[i], width))
		rows[1] += value.Render(fit("  "+values[i], width))
		rows[2] += note.Render(fit("  "+notes[i], width))
	}
	return rows
}

func (m *model) footer() string {
	left := "/ find sample   Enter inspect   ! attention   ? help   q quit"
	if m.screen == detailScreen {
		left = "1 stdout  2 stderr  3 facts   ↑↓ scroll   f follow   Esc back   q quit"
		if m.showMetadata {
			left = "1 stdout  2 stderr  3 facts   ↑↓ scroll   Home/End   Esc back   q quit"
		}
	} else if m.screen == tasksScreen || m.screen == attentionScreen {
		left = "↑↓ select   Enter logs   3 facts   / find sample   Esc back   q quit"
	}
	if m.screen == searchScreen {
		left = "↑↓ choose   Enter select   Esc close search   Ctrl+C quit"
		if m.width < 100 {
			return m.style.dim.Render("Enter select  Esc back  Ctrl+C quit")
		}
	}
	if m.width < 60 {
		return m.style.dim.Render("/ search  Enter open  Esc back  q quit")
	}
	if m.width < 100 {
		return m.style.dim.Render("/ search  Enter open  Esc back  ? help  q quit")
	}
	fresh := "Read " + m.data.Snapshot.ReadAt.Format("15:04:05")
	return ends(m.style.dim.Render(left), m.style.dim.Render(fresh), m.contentWidth())
}

func wrapPlain(text string, width int) []string {
	return strings.Split(ansi.Hardwrap(clean(text), max(1, width), true), "\n")
}

func (m *model) helpRows() []string {
	lines := []string{"KEYBOARD · PgUp / PgDn scroll", "", "Arrows        Select a graph card, sample, or task", "j / k         Traverse stages or task lists", "Enter         Open stage tasks or task logs", "/ or s        Find a sample; paste is supported", "t             Tasks in the current sample scope", "!             Global attention list", "PgUp / PgDn   Pan graph or scroll lists and logs", "1 / 2         Switch stdout / stderr in task details", "3             Full task facts, commands, and errors", "f / End       Toggle follow / follow newest log tail", "r             Refresh now", "Esc           Back; on dashboard, clear sample scope", "q / Ctrl+C    Close monitor; pipeline keeps running", "", "Counts describe known tasks, not remaining compute time.", "Shared/cohort work is excluded from sample completion.", "Log history is bounded to the last 4 KiB per stream.", "NO_COLOR keeps symbols and selection without colors."}
	var rows []string
	for _, line := range lines {
		rows = append(rows, wrapWords(line, m.contentWidth())...)
	}
	return rows
}

func (m *model) helpView() []string {
	rows := m.helpRows()
	offset := min(m.helpOffset, max(0, len(rows)-m.bodyHeight()))
	return frame(rows[offset:], m.contentWidth(), m.bodyHeight())
}
