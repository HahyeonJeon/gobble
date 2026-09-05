package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble/monitor"
)

// Full task facts have a separate scrollable pane. The concise log header can
// stay small without making long identities, commands, or errors inaccessible.
func (m *model) metadataRows(task monitor.Task, width int) []string {
	var rows []string
	field := func(label, value string) {
		if value == "" {
			value = "Not recorded"
		}
		rows = append(rows, wrapWords(label+": "+value, width)...)
	}
	field("Identity", task.Identity)
	field("Task", task.TaskID)
	field("Stage", task.Display.Stage)
	field("Module", task.Module)
	field("Scope", task.Display.Scope)
	if len(task.Display.Samples) > 0 {
		field("Samples", strings.Join(task.Display.Samples, ", "))
	}
	field("Status", stateLabel(task.Status))
	field("Attempt", strconv.Itoa(task.Attempt))
	if task.Template {
		field("Scatter template", fmt.Sprintf("expanded=%t", task.Expanded))
	}
	field("Started", task.Started)
	field("Ended", task.Ended)
	field("Elapsed", elapsed(task.Started, task.Ended, m.observationTime()))
	field("Executor", task.Executor)
	field("Image", task.Image)
	field("Requested CPU", fmt.Sprintf("%g", task.Resources.CPU))
	field("Requested RAM", task.Resources.Memory)
	if task.Decision != "" {
		field("Decision", task.Decision)
	}
	if task.ReuseReason != "" {
		field("Reuse reason", task.ReuseReason)
	}
	for _, difference := range task.Differing {
		field("Changed", difference)
	}
	if task.Reason != "" {
		rows = append(rows, "")
		field("Reason", task.Reason)
	}
	if task.Error != nil {
		field("Error unit", task.Error.Unit)
		if task.Error.Message != task.Reason {
			field("Error", task.Error.Message)
		}
	}
	for _, log := range m.data.Snapshot.Logs {
		if log.Identity == task.Identity {
			if log.Error != "" {
				field("Log unavailable", log.Error)
				continue
			}
			field("Stdout path", log.Stdout)
			field("Stderr path", log.Stderr)
		}
	}
	rows = append(rows, "", "COMMAND ARGUMENTS")
	for i, arg := range task.Command {
		field(strconv.Itoa(i), strconv.Quote(arg))
	}
	if len(task.Command) == 0 {
		rows = append(rows, "Not recorded")
	}
	if task.Script != "" {
		rows = append(rows, "", "SCRIPT")
		rows = append(rows, wrapPlain(strings.ReplaceAll(clean(task.Script), "\t", "    "), width)...)
	}
	return rows
}

func (m *model) metadataHeight() int { return max(1, m.bodyHeight()-4) }
func (m *model) metadataLineCount() int {
	task, ok := m.data.Task(m.task)
	if !ok {
		return 0
	}
	_, width := m.inspectorWidths()
	return len(m.metadataRows(task, max(1, width-4)))
}
func (m *model) metadataView(task monitor.Task, width, height int) []string {
	rows := m.metadataRows(task, max(1, width-4))
	available := max(1, height-2)
	offset := min(m.metadataOffset, max(0, len(rows)-available))
	header := fmt.Sprintf("[3 FACTS]  %d–%d / %d", offset+1, min(len(rows), offset+available), len(rows))
	lines := []string{m.surfaceLine(header, width, m.style.active), m.surfaceLine("↑↓ scroll · PgUp/PgDn · Home/End", width, m.style.dim)}
	for _, row := range frame(rows[offset:], width-4, available) {
		lines = append(lines, m.surfaceLine(row, width, m.style.plain))
	}
	return frame(lines, width, height)
}

func scrollOffset(key string, offset, total, height int) int {
	switch key {
	case "up", "k":
		offset--
	case "down", "j":
		offset++
	case "pgup":
		offset -= height
	case "pgdown":
		offset += height
	case "home":
		offset = 0
	case "end":
		offset = total
	}
	return min(max(0, total-height), max(0, offset))
}
