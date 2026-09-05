package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble/monitor"
)

const nodeHeight = 4

type rect struct{ x, y, w, h int }
type graphLayout struct {
	nodes        map[string]rect
	height, rail int
}

// Layout keeps a stable plan order within each topological rank. Excess nodes
// wrap within their rank; edges still connect their exact group identities.
func layout(stages []monitor.Stage, width int) graphLayout {
	columns := max(1, min(3, (width-8)/27))
	nodeWidth := max(12, (width-8-(columns-1)*3)/columns)
	out := graphLayout{nodes: map[string]rect{}, rail: width - 4}
	y, column, previous := 0, 0, -1
	for _, stage := range stages {
		if previous >= 0 && stage.Rank != previous {
			y += nodeHeight + 3
			column = 0
		} else if column == columns {
			y += nodeHeight + 3
			column = 0
		}
		out.nodes[stage.ID] = rect{x: column * (nodeWidth + 3), y: y, w: nodeWidth, h: nodeHeight}
		out.height = y + nodeHeight
		column++
		previous = stage.Rank
	}
	return out
}

type cell struct {
	text string
	role int
}
type canvas struct {
	cells                 [][]cell
	width, height, offset int
}

func newCanvas(width, height, offset int) *canvas {
	c := &canvas{width: width, height: height, offset: offset, cells: make([][]cell, height)}
	for y := range c.cells {
		c.cells[y] = make([]cell, width)
		for x := range c.cells[y] {
			c.cells[y][x].text = " "
		}
	}
	return c
}

func (c *canvas) put(x, y int, text string, role int) {
	y -= c.offset
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		c.cells[y][x] = cell{text: text, role: role}
	}
}

func (c *canvas) text(x, y int, text string, role, limit int) {
	used := 0
	for _, r := range clean(text) {
		w := lipgloss.Width(string(r))
		if used+w > limit {
			break
		}
		if w == 0 {
			continue
		}
		c.put(x+used, y, string(r), role)
		for j := 1; j < w; j++ {
			c.put(x+used+j, y, "", role)
		}
		used += w
	}
}

func (c *canvas) hline(x1, x2, y, role int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := max(0, x1); x <= min(c.width-1, x2); x++ {
		c.stroke(x, y, "─", role)
	}
}

func (c *canvas) vline(x, y1, y2, role int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := max(c.offset, y1); y <= min(c.offset+c.height-1, y2); y++ {
		c.stroke(x, y, "│", role)
	}
}

func (c *canvas) stroke(x, y int, glyph string, role int) {
	row := y - c.offset
	if x < 0 || x >= c.width || row < 0 || row >= c.height {
		return
	}
	old := c.cells[row][x]
	if old.text != " " && old.text != glyph {
		glyph = "┼"
	}
	// A highlighted edge remains highlighted when another edge crosses it.
	if old.role == 2 {
		role = 2
	}
	c.put(x, y, glyph, role)
}

func (c *canvas) render(t theme) []string {
	styles := []lipgloss.Style{t.plain, t.dim, t.active, t.good, t.bad, t.warn, t.selected}
	lines := make([]string, c.height)
	for y, row := range c.cells {
		var out, span strings.Builder
		role := 0
		flush := func() { out.WriteString(styles[role].Render(span.String())); span.Reset() }
		for _, cell := range row {
			if cell.role != role {
				flush()
				role = cell.role
			}
			span.WriteString(cell.text)
		}
		flush()
		lines[y] = out.String()
	}
	return lines
}

func nodeStatus(c monitor.Counts) string {
	if c.Total == 0 && c.Templates == 0 {
		return "no owned work"
	}
	parts := []string{}
	for _, item := range []struct {
		n     int
		label string
	}{{c.Running, "run"}, {c.Failed, "fail"}, {c.Blocked, "blocked"}, {c.Pending, "wait"}, {c.Skipped, "skip"}, {c.Incomplete, "incomplete"}, {c.Unknown, "unknown"}, {c.Unfinalized, "unfinalized"}} {
		if item.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", item.n, item.label))
		}
	}
	if c.Unexpanded > 0 {
		parts = append(parts, "expanding")
	}
	if len(parts) == 0 {
		return "complete"
	}
	return strings.Join(parts, " · ")
}

func (m *model) graphWidth() int {
	w := m.width
	if w >= 100 {
		w -= 32
	}
	return max(12, w)
}

func (m *model) revealStage() {
	g := layout(m.data.Stages, m.graphWidth())
	r, ok := g.nodes[m.stage]
	if !ok {
		return
	}
	h := max(1, m.bodyHeight()-2)
	if r.y < m.graphOffset {
		m.graphOffset = r.y
	}
	if r.y+r.h > m.graphOffset+h {
		m.graphOffset = max(0, r.y+r.h-h)
	}
	m.graphOffset = min(m.graphOffset, max(0, g.height-h))
}

func (m *model) graph(width, height int) []string {
	g := layout(m.data.Stages, width)
	offset := min(m.graphOffset, max(0, g.height-height))
	c := newCanvas(width, height, offset)
	for i, edge := range m.data.Edges {
		from, to := g.nodes[edge.From], g.nodes[edge.To]
		x1, x2 := from.x+from.w/2, to.x+to.w/2
		y1, y2 := from.y+from.h, to.y-1
		role := 1
		if edge.From == m.stage || edge.To == m.stage {
			role = 2
		}
		if x1 == x2 {
			c.vline(x1, y1, y2, role)
		} else if y2-y1 == 2 {
			// Adjacent ranks share an empty routing row. Connect there without
			// extending an unnecessary line to the outer rail.
			c.vline(x1, y1, y1+1, role)
			c.hline(x1, x2, y1+1, role)
			c.vline(x2, y1+1, y2, role)
		} else {
			rail := g.rail + i%3
			c.vline(x1, y1, y1+1, role)
			c.hline(x1, rail, y1+1, role)
			c.vline(rail, y1+1, y2-1, role)
			c.hline(rail, x2, y2-1, role)
			c.vline(x2, y2-1, y2, role)
		}
		c.put(x2, y2, "▼", role)
	}
	for _, stage := range m.data.Stages {
		r := g.nodes[stage.ID]
		counts := m.data.Count(m.data.StageTasks(stage, m.sample))
		role := 1
		if counts.Failed > 0 {
			role = 4
		} else if counts.Attention() > 0 {
			role = 5
		} else if counts.Running > 0 {
			role = 2
		} else if counts.Successful() {
			role = 3
		}
		if stage.ID == m.stage {
			role = 2
		}
		for y := r.y; y < r.y+r.h; y++ {
			for x := r.x; x < r.x+r.w; x++ {
				c.put(x, y, " ", 0)
			}
		}
		c.hline(r.x+1, r.x+r.w-2, r.y, role)
		c.hline(r.x+1, r.x+r.w-2, r.y+r.h-1, role)
		c.vline(r.x, r.y+1, r.y+r.h-2, role)
		c.vline(r.x+r.w-1, r.y+1, r.y+r.h-2, role)
		c.put(r.x, r.y, "┌", role)
		c.put(r.x+r.w-1, r.y, "┐", role)
		c.put(r.x, r.y+r.h-1, "└", role)
		c.put(r.x+r.w-1, r.y+r.h-1, "┘", role)
		name := " " + oneLine(stage.Name) + " "
		if stage.ID == m.stage {
			name = " ▸ " + oneLine(stage.Name) + " "
		}
		c.text(r.x+1, r.y, name, role, r.w-3)
		first := fmt.Sprintf("%d / %d succeeded", counts.Succeeded, counts.Total)
		if m.sample != "" && stage.Scope != "sample" {
			first = "Context: " + first
		}
		if counts.Total == 0 && counts.Templates == 0 {
			first = "Outside sample scope"
		}
		c.text(r.x+2, r.y+1, fit(first, r.w-4), 0, r.w-4)
		c.text(r.x+2, r.y+2, fit(nodeStatus(counts), r.w-4), role, r.w-4)
	}
	return c.render(m.style)
}
