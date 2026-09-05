package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble/monitor"
)

func (m *model) inspectorWidths() (int, int) {
	w := m.contentWidth()
	if w < 90 {
		return 0, w
	}
	left := max(32, w*4/10)
	return left, w - left - 3
}

func (m *model) inspectedTask() (monitor.Task, bool) {
	if m.screen == detailScreen {
		return m.data.Task(m.task)
	}
	items := m.listTasks()
	if len(items) == 0 {
		return monitor.Task{}, false
	}
	return m.data.Snapshot.Tasks[items[min(m.listIndex, len(items)-1)]], true
}

func (m *model) inspectorView() []string {
	width, height := m.contentWidth(), m.bodyHeight()
	leftWidth, rightWidth := m.inspectorWidths()
	title := "TASK INSPECTOR"
	if m.screen == attentionScreen || m.screen == detailScreen && m.detailReturn == attentionScreen {
		title = "ATTENTION · ALL SAMPLES"
	}
	hint := "↑↓ select task · Enter opens logs"
	if m.screen == detailScreen {
		hint = "1 / 2 stream · f follow · Esc back"
	}
	lines := []string{ends(m.style.dim.Render(title), m.style.dim.Render(hint), width), ""}
	paneHeight := max(1, height-len(lines))
	task, ok := m.inspectedTask()
	if !ok {
		return append(lines, m.style.dim.Render("No tasks in this selection."))
	}
	if leftWidth == 0 {
		if m.screen == detailScreen {
			return append(lines, m.taskDetails(task, rightWidth, paneHeight, true)...)
		}
		return append(lines, m.taskList(width, paneHeight, task.Identity)...)
	}
	left := m.taskList(leftWidth, paneHeight, task.Identity)
	right := m.taskDetails(task, rightWidth, paneHeight, m.screen == detailScreen)
	return append(lines, beside(left, right, leftWidth, rightWidth, 3, paneHeight)...)
}

func (m *model) taskList(width, height int, selected string) []string {
	items := m.listTasks()
	c := m.data.Count(items)
	lines := []string{m.style.title.Render(fmt.Sprintf("%d / %d tasks succeeded", c.Succeeded, c.Total)), distribution(c, width, m.style), ""}
	cursor := 0
	for i, index := range items {
		if m.data.Snapshot.Tasks[index].Identity == selected {
			cursor = i
			break
		}
	}
	pageSize := taskPageSize(height)
	start := max(0, cursor-pageSize+1)
	for i := start; i < len(items) && i < start+pageSize && len(lines)+2 <= height-1; i++ {
		task := m.data.Snapshot.Tasks[items[i]]
		active := task.Identity == selected
		surface := m.style.panel
		if active {
			surface = m.style.selected
		}
		name := taskLabel(task)
		prefix := "  "
		if active {
			prefix = "▸ "
		}
		lines = append(lines, surface.Bold(active).Render(fit(prefix+name, width)))
		state := stateLabel(task.Status)
		if task.Template {
			state = "◇ Scatter template"
		} else if task.Decision == "reused" {
			state = "↺ Reused"
		}
		style := surface
		if !m.style.monochrome {
			style = style.Foreground(stateColor(task.Status))
		}
		row := ends("  "+state, elapsed(task.Started, task.Ended, m.observationTime())+" ", width)
		lines = append(lines, style.Render(row), "")
	}
	if len(items) > pageSize {
		lines = append(lines, m.style.dim.Render(fmt.Sprintf("%d–%d of %d · PgUp / PgDn", start+1, min(len(items), start+pageSize), len(items))))
	}
	return frame(lines, width, height)
}

func taskPageSize(paneHeight int) int {
	return max(1, (paneHeight-4)/3)
}

func (m *model) surfaceLine(text string, width int, style lipgloss.Style) string {
	if !m.style.monochrome {
		style = style.Background(lipgloss.Color(panelColor))
	}
	return style.Render("  " + fit(text, max(0, width-4)) + "  ")
}

func (m *model) detailHeader(task monitor.Task, width, height int) []string {
	inner := max(1, width-4)
	line := func(text string, style lipgloss.Style) string { return m.surfaceLine(text, width, style) }
	rows := []string{line(taskLabel(task), m.style.title), line(oneLine(task.Identity), m.style.dim), line(stateLabel(task.Status)+fmt.Sprintf(" · Attempt %d", task.Attempt), m.style.state(task.Status)), line("", m.style.plain)}
	cpu := "—"
	if task.Resources.CPU > 0 {
		cpu = fmt.Sprintf("%g", task.Resources.CPU)
	}
	memory := oneLine(task.Resources.Memory)
	if memory == "" {
		memory = "—"
	}
	rows = append(rows, line("CPU request "+cpu+"   RAM request "+memory, m.style.dim))
	executor := oneLine(task.Executor)
	if executor == "" {
		executor = "Not started"
	}
	image := oneLine(task.Image)
	if image != "" {
		executor += " · " + image
	}
	rows = append(rows, line(executor, m.style.dim))
	command := strings.Join(task.Command, " ")
	if task.Script != "" {
		command = task.Script
	}
	if command != "" {
		for _, text := range limitedWrap("$ "+oneLine(command), inner, 2) {
			rows = append(rows, line(text, m.style.plain))
		}
	}
	if task.Reason != "" {
		rows = append(rows, line("", m.style.plain))
		for _, text := range limitedWrap(task.Reason, inner, 3) {
			rows = append(rows, line(text, m.style.state(task.Status)))
		}
	}
	// Reserve visible log space in short terminals; task identity and state stay
	// first, while full facts remain available through inspect monitor.
	return rows[:min(len(rows), max(1, height-6))]
}

func limitedWrap(text string, width, limit int) []string {
	rows := wrapWords(text, width)
	if len(rows) > limit {
		rows = rows[:limit]
		rows[limit-1] = fit(rows[limit-1], max(1, width-1)) + "…"
	}
	return rows
}

func (m *model) taskDetails(task monitor.Task, width, height int, withLogs bool) []string {
	rows := m.detailHeader(task, width, height)
	blank := m.surfaceLine("", width, m.style.plain)
	rows = append(rows, blank)
	if !withLogs {
		rows = append(rows, m.surfaceLine("Enter  Open task logs", width, m.style.active))
		rows = append(rows, m.surfaceLine("1 stdout  /  2 stderr", width, m.style.dim))
		for len(rows) < height {
			rows = append(rows, blank)
		}
		return rows[:min(height, len(rows))]
	}
	tabs := "[1 STDOUT]    2 stderr"
	if m.logStream == "stderr" {
		tabs = " 1 stdout   [2 STDERR]"
	}
	follow := "PAUSED"
	if m.follow {
		follow = "FOLLOWING"
	}
	rows = append(rows, m.surfaceLine(ends(tabs, follow, width-4), width, m.style.active))
	_, size := m.currentLog()
	rows = append(rows, m.surfaceLine(fmt.Sprintf("Last 4 KiB · %d bytes in stream", size), width, m.style.dim), m.surfaceLine(strings.Repeat("─", max(1, width-4)), width, m.style.line))
	logs := m.logLines()
	offset := min(m.logOffset, m.logTailOffset())
	if m.follow {
		offset = m.logTailOffset()
	}
	available := max(0, height-len(rows))
	for _, text := range frame(logs[offset:], width-4, available) {
		rows = append(rows, m.surfaceLine(text, width, m.style.plain))
	}
	return rows[:min(height, len(rows))]
}

func (m *model) currentLog() (string, int64) {
	for _, log := range m.data.Snapshot.Logs {
		if log.Identity == m.task {
			if m.logStream == "stdout" {
				return log.StdoutTail, log.StdoutSize
			}
			return log.StderrTail, log.StderrSize
		}
	}
	return "", 0
}

func (m *model) logLines() []string {
	_, width := m.inspectorWidths()
	text, _ := m.currentLog()
	if text == "" {
		text = "No output available in this tail."
	}
	return wrapPlain(strings.ReplaceAll(clean(text), "\t", "    "), max(1, width-4))
}

func (m *model) logTailOffset() int {
	_, width := m.inspectorWidths()
	height := max(1, m.bodyHeight()-2)
	task, ok := m.data.Task(m.task)
	if !ok {
		return 0
	}
	header := len(m.detailHeader(task, width, height)) + 4
	return max(0, len(m.logLines())-max(1, height-header))
}
