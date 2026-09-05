package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble/monitor"
)

func (m *model) searchResultHeight() int {
	return min(9, max(4, m.bodyHeight()/2))
}

func (m *model) graphViewportHeight() int {
	height := m.bodyHeight() - 2
	if m.screen == searchScreen {
		height -= len(m.searchResults(m.graphWidth(), m.searchResultHeight())) + 1
	}
	return max(1, height)
}

func (m *model) dashboardView() []string {
	w, h := m.graphWidth(), m.bodyHeight()
	var left []string
	if m.screen == searchScreen {
		left = m.searchResults(w, m.searchResultHeight())
		left = append(left, "")
	}
	title := ends(m.style.dim.Render("PIPELINE GRAPH"), m.style.dim.Render(fmt.Sprintf("%d stages · ↑↓ select", len(m.data.Stages))), w)
	left = append(left, title, "")
	graphHeight := max(0, h-len(left))
	g := layout(m.data.Stages, w)
	if graphHeight > 0 {
		if g.height > graphHeight {
			left[len(left)-2] = ends(m.style.dim.Render("PIPELINE GRAPH"), m.style.dim.Render("PgUp / PgDn pan"), w)
		}
		left = append(left, m.graph(w, graphHeight)...)
	}
	if m.hasSidebar() {
		sw := m.sidebarWidth()
		right := frame(m.sidebar(sw, h), sw, h)
		for i := range right {
			right[i] = m.style.line.Render("│") + "  " + right[i]
		}
		return beside(left, right, w, sw+3, 1, h)
	}
	return frame(left, w, h)
}

func (m *model) searchResults(width, height int) []string {
	matches := m.data.SearchSamples(m.query)
	if len(matches) == 0 {
		message := "No matching samples. Try another sample ID."
		if len(m.data.Samples) == 0 {
			message = "This run has no sample labels. Task navigation is available with t."
		}
		return panelRows(append([]string{"SAMPLE SEARCH", ""}, wrapPlain(message, width-4)...), width, min(height, 4), m.style.panel)
	}
	columns := searchColumns(width)
	cw := (width - (columns-1)*2) / columns
	rows := min(max(1, height-2), (len(matches)+columns-1)/columns)
	firstRow := max(0, m.searchIndex/columns-rows+1)
	lines := []string{m.style.dim.Render(fmt.Sprintf("SAMPLE SEARCH · %d matches", len(matches)))}
	for row := firstRow; row < firstRow+rows; row++ {
		line := ""
		for col := 0; col < columns; col++ {
			if col > 0 {
				line += "  "
			}
			index := row*columns + col
			if index >= len(matches) {
				line += strings.Repeat(" ", cw)
				continue
			}
			s := matches[index]
			mark := "  "
			surface := m.style.panel
			if index == m.searchIndex {
				mark = "▸ "
				surface = m.style.selected
			}
			state := countsStatus(s.Counts)
			if !m.style.monochrome {
				surface = surface.Foreground(stateColor(state))
			}
			label := fmt.Sprintf("%s%s  %d/%d · %s", mark, oneLine(s.ID), s.Counts.Succeeded, s.Counts.Total, stateLabel(state))
			line += surface.Render(fit(label, cw))
		}
		lines = append(lines, line)
	}
	return append(lines, m.style.dim.Render("↑↓ / ←→ choose · Enter selects · Esc closes"))
}

func stateColor(status string) color.Color {
	switch status {
	case "succeeded":
		return lipgloss.Color(goodColor)
	case "running":
		return lipgloss.Color(activeColor)
	case "failed":
		return lipgloss.Color(badColor)
	case "not-started", "skipped":
		return lipgloss.Color(mutedColor)
	default:
		return lipgloss.Color(warnColor)
	}
}

func (m *model) sidebar(width, height int) []string {
	if height < 24 {
		return m.compactSidebar(width, height)
	}
	lines := []string{ends(m.style.dim.Render("ATTENTION"), m.style.bad.Render(fmt.Sprintf("%d failed", m.data.Total.Failed)), width), ""}
	// Show a direct failure and a cohort consequence when both exist, preserving
	// visibility across sample scopes. The full attention list remains one key away.
	shown := []int{}
	if len(m.data.Attention) > 0 {
		shown = append(shown, m.data.Attention[0])
	}
	for _, index := range m.data.Attention {
		if len(shown) > 0 && index != shown[0] && m.data.Snapshot.Tasks[index].Display.Scope == "cohort" {
			shown = append(shown, index)
			break
		}
	}
	if len(shown) < 2 && len(m.data.Attention) > 1 {
		shown = append(shown, m.data.Attention[1])
	}
	for _, index := range shown {
		task := m.data.Snapshot.Tasks[index]
		label := taskLabel(task)
		style := m.style.state(task.Status)
		mark := strings.Fields(stateLabel(task.Status))[0]
		lines = append(lines, style.Bold(true).Render(mark+" "+label))
		reason := task.Reason
		if reason == "" {
			reason = stateLabel(task.Status) + " · Press ! to inspect"
		}
		wrapped := wrapWords(reason, width-2)
		for _, line := range wrapped[:min(2, len(wrapped))] {
			lines = append(lines, m.style.dim.Render("  "+line))
		}
		lines = append(lines, "")
	}
	if len(shown) == 0 {
		lines = append(lines, m.style.good.Render("✓ No task failures recorded"), "")
	}
	lines = append(lines, m.style.dim.Render(fmt.Sprintf("!  Inspect all %d attention tasks", len(m.data.Attention))), "")
	lines = append(lines, m.style.line.Render(strings.Repeat("─", width)), "", m.style.dim.Render("SAMPLE PROGRESS"))
	complete, active, failed := 0, 0, 0
	for _, s := range m.data.Samples {
		if s.Counts.Successful() {
			complete++
		} else if s.Counts.Failed > 0 {
			failed++
		} else if s.Counts.Running > 0 || s.Counts.Pending > 0 {
			active++
		}
	}
	lines = append(lines, m.style.title.Render(fmt.Sprintf("%d / %d sample task sets complete", complete, len(m.data.Samples))), "")
	lines = append(lines, m.sampleGrid(width, 12)...)
	lines = append(lines, "", m.style.dim.Render(fmt.Sprintf("%d complete · %d active · %d failed", complete, active, failed)), m.style.dim.Render(fmt.Sprintf("Shared / cohort: %d / %d succeeded", m.data.Shared.Succeeded, m.data.Shared.Total)))
	if i := m.stageIndex(); i >= 0 && height-len(lines) >= 4 {
		up, down := m.data.Neighbors(m.stage)
		lines = append(lines, "", m.style.dim.Render("SELECTED · "+oneLine(m.data.Stages[i].Name)))
		if len(up) > 0 {
			lines = append(lines, m.style.dim.Render("From "+strings.Join(up, ", ")))
		}
		if len(down) > 0 {
			lines = append(lines, m.style.dim.Render("To   "+strings.Join(down, ", ")))
		}
	}
	return frame(lines, width, height)
}

func taskLabel(t monitor.Task) string {
	name := t.Display.Stage
	if name == "" {
		name = t.Name
	}
	if name == "" {
		return oneLine(t.Identity)
	}
	owner := "Shared"
	if len(t.Display.Samples) > 0 {
		owner = strings.Join(t.Display.Samples, ", ")
	} else if t.Display.Scope == "cohort" {
		owner = "Cohort"
	}
	return oneLine(owner + " / " + name)
}

func (m *model) sampleGrid(width, limit int) []string {
	var lines []string
	maxSamples := min(limit, len(m.data.Samples))
	cols := min(4, max(1, width/8))
	cw := (width - (cols - 1)) / cols
	for row := 0; row*cols < maxSamples; row++ {
		line := ""
		for col := 0; col < cols; col++ {
			i := row*cols + col
			if i >= maxSamples {
				break
			}
			if col > 0 {
				line += " "
			}
			sample := m.data.Samples[i]
			style := m.style.panel
			if sample.ID == m.sample {
				style = m.style.selected.Bold(true)
			}
			if !m.style.monochrome {
				style = style.Foreground(stateColor(countsStatus(sample.Counts)))
			}
			line += style.Render(fit(" "+oneLine(sample.ID), cw))
		}
		lines = append(lines, line)
	}
	if len(m.data.Samples) > maxSamples {
		lines = append(lines, m.style.dim.Render(fmt.Sprintf("+ %d more · / find sample", len(m.data.Samples)-maxSamples)))
	}
	if len(m.data.Samples) == 0 {
		lines = append(lines, m.style.dim.Render("No sample labels in this plan"))
	}

	return lines
}

func (m *model) compactSidebar(width, height int) []string {
	lines := []string{m.style.dim.Render("ATTENTION")}
	if len(m.data.Attention) > 0 {
		task := m.data.Snapshot.Tasks[m.data.Attention[0]]
		lines = append(lines, m.style.state(task.Status).Render(taskLabel(task)))
	} else {
		lines = append(lines, m.style.good.Render("✓ No task failures recorded"))
	}
	lines = append(lines, m.style.dim.Render(fmt.Sprintf("! %d tasks need attention", len(m.data.Attention))), "", m.style.dim.Render("SAMPLE PROGRESS"))
	complete := 0
	for _, sample := range m.data.Samples {
		if sample.Counts.Successful() {
			complete++
		}
	}
	lines = append(lines, m.style.title.Render(fmt.Sprintf("%d / %d sample task sets complete", complete, len(m.data.Samples))), "")
	lines = append(lines, m.sampleGrid(width, 8)...)
	lines = append(lines, "", m.style.dim.Render(fmt.Sprintf("Shared / cohort: %d / %d", m.data.Shared.Succeeded, m.data.Shared.Total)))
	return frame(lines, width, height)
}
