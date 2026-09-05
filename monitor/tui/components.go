package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble/monitor"
)

// These helpers size terminal cells, preserving ANSI spans and full surfaces.
func frame(lines []string, width, height int) []string {
	out := make([]string, max(0, height))
	for i := range out {
		if i < len(lines) {
			out[i] = fit(lines[i], width)
		} else {
			out[i] = strings.Repeat(" ", max(0, width))
		}
	}
	return out
}

func beside(left, right []string, lw, rw, gap, height int) []string {
	a, b := frame(left, lw, height), frame(right, rw, height)
	for i := range a {
		a[i] += strings.Repeat(" ", gap) + b[i]
	}
	return a
}

func ends(left, right string, width int) string {
	rw := lipgloss.Width(right)
	if rw > width/2 {
		return fit(left, width)
	}
	return fit(left, max(0, width-rw)) + right
}

func panelRows(lines []string, width, height int, style lipgloss.Style) []string {
	rows := frame(lines, max(0, width-4), height)
	for i := range rows {
		rows[i] = style.Render("  " + rows[i] + "  ")
	}
	return rows
}

type statePart struct {
	n     int
	color string
	label string
}

func stateParts(c monitor.Counts) []statePart {
	return []statePart{{c.Succeeded, goodColor, "succeeded"}, {c.Running, activeColor, "running"}, {c.Failed, badColor, "failed"},
		{c.Pending, borderColor, "pending"}, {c.Blocked, warnColor, "blocked"}, {c.Skipped, mutedColor, "skipped"},
		{c.Incomplete, warnColor, "incomplete"}, {c.Unknown, warnColor, "unknown"}, {c.Unfinalized, warnColor, "unfinalized"}}
}

// Largest-remainder allocation keeps segments at exactly width cells. The
// legend always carries exact counts, including states too small for a cell.
func distribution(c monitor.Counts, width int, t theme) string {
	if width <= 0 {
		return ""
	}
	if c.Total == 0 {
		return t.line.Render(strings.Repeat("━", width))
	}
	parts := stateParts(c)
	cells := make([]int, len(parts))
	remainders := make([]int, len(parts))
	used := 0
	for i, p := range parts {
		cells[i] = p.n * width / c.Total
		remainders[i] = p.n * width % c.Total
		used += cells[i]
	}
	for used < width {
		best := 0
		for i := range parts {
			if remainders[i] > remainders[best] {
				best = i
			}
		}
		cells[best]++
		remainders[best] = -1
		used++
	}
	var out strings.Builder
	for i, p := range parts {
		style := t.plain
		if !t.monochrome {
			style = style.Foreground(lipgloss.Color(p.color))
		}
		glyph := "━"
		if t.monochrome && p.label != "succeeded" {
			glyph = "┄"
		}
		out.WriteString(style.Render(strings.Repeat(glyph, cells[i])))
	}
	return out.String()
}

func countsStatus(c monitor.Counts) string {
	switch {
	case c.Failed > 0:
		return "failed"
	case c.Unknown > 0:
		return "unknown"
	case c.Incomplete > 0:
		return "incomplete"
	case c.Unfinalized > 0:
		return "published-unfinalized"
	case c.Blocked > 0:
		return "blocked"
	case c.Running > 0:
		return "running"
	case c.Successful():
		return "succeeded"
	case (c.Skipped > 0 || c.SkippedTemplates > 0) && c.Pending == 0 && c.Unexpanded == 0:
		return "skipped"
	default:
		return "not-started"
	}
}

func nodeStatus(c monitor.Counts) string {
	if c.Total == 0 && c.Templates == 0 {
		return "No owned work"
	}
	if c.Total == 0 && c.Unexpanded == 0 && c.SkippedTemplates == 0 {
		return "No executable instances"
	}
	parts := []string{}
	for _, p := range stateParts(c) {
		if p.n > 0 && p.label != "succeeded" {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.label))
		}
	}
	if c.Unexpanded > 0 {
		parts = append(parts, fmt.Sprintf("%d expanding", c.Unexpanded))
	}
	if c.SkippedTemplates > 0 {
		parts = append(parts, fmt.Sprintf("%d templates skipped", c.SkippedTemplates))
	}
	if len(parts) == 0 {
		return "All tasks succeeded"
	}
	return strings.Join(parts, " · ")
}

func searchColumns(width int) int {
	if width >= 62 {
		return 2
	}
	return 1
}

// Prose wraps at word boundaries. Logs retain their original whitespace and
// use wrapPlain instead, including indentation and long unbroken paths.
func wrapWords(text string, width int) []string {
	width = max(1, width)
	var rows []string
	for _, paragraph := range strings.Split(clean(text), "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			if line != "" && lipgloss.Width(line)+1+lipgloss.Width(word) > width {
				rows = append(rows, line)
				line = ""
			}
			if lipgloss.Width(word) > width {
				pieces := wrapPlain(word, width)
				rows = append(rows, pieces[:len(pieces)-1]...)
				word = pieces[len(pieces)-1]
			}
			if line != "" {
				line += " "
			}
			line += word
		}
		rows = append(rows, line)
	}
	return rows
}
