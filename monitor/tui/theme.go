package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type theme struct {
	plain, dim, active, good, bad, warn, selected lipgloss.Style
	monochrome bool
}

func newTheme(monochrome bool) theme {
	base := lipgloss.NewStyle()
	if monochrome {
		return theme{plain: base, dim: base, active: base.Bold(true), good: base, bad: base.Bold(true), warn: base, selected: base.Reverse(true), monochrome: true}
	}
	return theme{
		plain: base.Foreground(lipgloss.Color("#DBE8EC")),
		dim: base.Foreground(lipgloss.Color("#8FA7B3")),
		active: base.Foreground(lipgloss.Color("#86CEDB")),
		good: base.Foreground(lipgloss.Color("#8AC9AD")),
		bad: base.Foreground(lipgloss.Color("#F19BAC")),
		warn: base.Foreground(lipgloss.Color("#D5B78D")),
		selected: base.Foreground(lipgloss.Color("#DBE8EC")).Background(lipgloss.Color("#25414B")),
	}
}

// Treat terminal escape sequences in task names, errors, and logs as data.
// OSC clipboard/window/title sequences and other controls must never execute.
func clean(s string) string {
	s = ansi.Strip(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(clean(s)), " ")
}

func fit(s string, width int) string {
	if width < 1 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s)))
}

func (t theme) state(status string) lipgloss.Style {
	switch status {
	case "succeeded":
		return t.good
	case "running":
		return t.active
	case "failed":
		return t.bad
	case "blocked", "incomplete", "unknown", "unknown-backend", "published-unfinalized":
		return t.warn
	default:
		return t.dim
	}
}
