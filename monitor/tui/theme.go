package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Colors match the accepted dashboard preview. Surfaces and spacing carry the
// hierarchy; state colors are accents, not full panel outlines.
const (
	backgroundColor = "#10171B"
	panelColor      = "#162127"
	selectedColor   = "#203C46"
	textColor       = "#DBE8EC"
	mutedColor      = "#9AB0BC"
	borderColor     = "#354B57"
	activeColor     = "#8CCDDD"
	goodColor       = "#8AC9AD"
	badColor        = "#F19BAC"
	warnColor       = "#D5B78D"
)

type theme struct {
	plain, dim, active, good, bad, warn, selected lipgloss.Style
	line, panel, title                            lipgloss.Style
	monochrome                                    bool
}

func newTheme(monochrome bool) theme {
	base := lipgloss.NewStyle()
	if monochrome {
		return theme{plain: base, dim: base, active: base.Bold(true), good: base, bad: base.Bold(true), warn: base,
			selected: base.Reverse(true), line: base, panel: base, title: base.Bold(true), monochrome: true}
	}
	fg := func(c string) lipgloss.Style { return base.Foreground(lipgloss.Color(c)) }
	return theme{plain: fg(textColor), dim: fg(mutedColor), active: fg(activeColor), good: fg(goodColor), bad: fg(badColor), warn: fg(warnColor),
		selected: fg(textColor).Background(lipgloss.Color(selectedColor)), line: fg(borderColor),
		panel: fg(textColor).Background(lipgloss.Color(panelColor)), title: fg(textColor).Bold(true)}
}

// Never interpret escape sequences embedded in task names, errors, or logs.
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

func oneLine(s string) string { return strings.Join(strings.Fields(clean(s)), " ") }
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

func stateLabel(status string) string {
	switch status {
	case "succeeded":
		return "✓ Succeeded"
	case "running":
		return "● Running"
	case "failed":
		return "× Failed"
	case "not-started":
		return "· Pending"
	case "blocked":
		return "! Blocked"
	case "skipped":
		return "− Skipped"
	case "incomplete":
		return "! Incomplete"
	case "published-unfinalized":
		return "! Unfinalized"
	default:
		return "? " + oneLine(status)
	}
}
