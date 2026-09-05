package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble/monitor"
	"github.com/charmbracelet/x/ansi"
)

const nodeHeight = 7
const nodeGap = 2

type rect struct{ x, y, w, h int }
type graphLayout struct {
	nodes        map[string]rect
	height, rail int
}

// Fold the topological sequence into alternating rows. A two-column layout
// preserves the preview's compact flow, with explicit left/right arrows in a
// row. Edges always connect the original stage IDs; ranks still govern grouping.
func layout(stages []monitor.Stage, width int) graphLayout {
	columns := 1
	if width >= 62 {
		columns = 2
	}
	gap := 7
	nw := max(12, (width-4-(columns-1)*gap)/columns)
	out := graphLayout{nodes: map[string]rect{}, rail: width - 2}
	for i, s := range stages {
		row, col := i/columns, i%columns
		if columns == 2 && row%2 == 1 {
			col = 1 - col
		}
		y := row * (nodeHeight + nodeGap)
		out.nodes[s.ID] = rect{x: col * (nw + gap), y: y, w: nw, h: nodeHeight}
		out.height = y + nodeHeight
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
		c.cells[y][x] = cell{text, role}
	}
}

func (c *canvas) text(x, y int, text string, role, limit int) {
	used := 0
	lastX := -1
	text = clean(text)
	var state byte
	for text != "" {
		cluster, w, n, next := ansi.DecodeSequence(text, state, nil)
		text, state = text[n:], next
		if used+w > limit {
			break
		}
		if w == 0 {
			// The ANSI decoder can return a combining mark separately after an
			// ASCII fast path. Keep it attached to the previous visible cell.
			row := y - c.offset
			if lastX >= 0 && lastX < c.width && row >= 0 && row < c.height {
				c.cells[row][lastX].text += cluster
			}
			continue
		}
		lastX = x + used
		c.put(x+used, y, cluster, role)
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
	if old.role == 2 {
		role = 2
	}
	c.put(x, y, glyph, role)
}

func (c *canvas) render(t theme) []string {
	base := []lipgloss.Style{t.plain, t.line, t.active, t.good, t.bad, t.warn, t.dim, t.title}
	styles := append([]lipgloss.Style(nil), base...)
	for _, color := range []string{panelColor, selectedColor} {
		for _, style := range base {
			if !t.monochrome {
				style = style.Background(lipgloss.Color(color))
			}
			styles = append(styles, style)
		}
	}
	lines := make([]string, c.height)
	for y, row := range c.cells {
		var out, span strings.Builder
		role := 0
		flush := func() {
			if span.Len() > 0 {
				out.WriteString(styles[role].Render(span.String()))
				span.Reset()
			}
		}
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

func (m *model) graphWidth() int {
	w := m.contentWidth()
	if m.hasSidebar() {
		w -= m.sidebarWidth() + 4
	}
	return max(12, w)
}

func (m *model) panGraph(direction int) {
	height := m.graphViewportHeight()
	limit := max(0, layout(m.data.Stages, m.graphWidth()).height-height)
	m.graphOffset = min(limit, max(0, m.graphOffset+direction*height))
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
	for i, e := range m.data.Edges {
		a, b := g.nodes[e.From], g.nodes[e.To]
		role := 1
		if e.From == m.stage || e.To == m.stage {
			role = 2
		}
		if a.y == b.y {
			y := a.y + a.h/2
			if a.x < b.x {
				c.hline(a.x+a.w, b.x-1, y, role)
				c.put(b.x-1, y, "▶", role)
			} else {
				c.hline(b.x+b.w, a.x-1, y, role)
				c.put(b.x+b.w, y, "◀", role)
			}
			continue
		}
		x1, x2 := a.x+a.w/2, b.x+b.w/2
		y1, y2 := a.y+a.h, b.y-1
		switch {
		case b.y-a.y == nodeHeight+nodeGap && x1 == x2:
			c.vline(x1, y1, y2, role)
		case b.y-a.y == nodeHeight+nodeGap:
			c.vline(x1, y1, y1, role)
			c.hline(x1, x2, y1, role)
			c.vline(x2, y1, y2, role)
		default:
			rail := g.rail + i%2
			c.hline(x1, rail, y1, role)
			c.vline(rail, y1, y2, role)
			c.hline(rail, x2, y2, role)
		}
		c.put(x2, y2, "▼", role)
	}
	for _, stage := range m.data.Stages {
		m.drawNode(c, g.nodes[stage.ID], stage)
	}
	return c.render(m.style)
}

func (m *model) drawNode(c *canvas, r rect, s monitor.Stage) {
	counts := m.data.Count(m.data.StageTasks(s, m.sample))
	status := countsStatus(counts)
	base := 8
	if s.ID == m.stage {
		base = 16
	}
	for y := r.y; y < r.y+r.h; y++ {
		for x := r.x; x < r.x+r.w; x++ {
			c.put(x, y, " ", base)
		}
	}
	border := base + 1
	if s.ID == m.stage {
		border = base + 2
	}
	for _, y := range []int{r.y, r.y + r.h - 1} {
		c.hline(r.x+1, r.x+r.w-2, y, border)
	}
	c.vline(r.x, r.y+1, r.y+r.h-2, border)
	c.vline(r.x+r.w-1, r.y+1, r.y+r.h-2, border)
	c.put(r.x, r.y, "╭", border)
	c.put(r.x+r.w-1, r.y, "╮", border)
	c.put(r.x, r.y+r.h-1, "╰", border)
	c.put(r.x+r.w-1, r.y+r.h-1, "╯", border)
	stateRole := 6
	switch status {
	case "succeeded":
		stateRole = 3
	case "running":
		stateRole = 2
	case "failed":
		stateRole = 4
	case "blocked", "unknown", "incomplete", "published-unfinalized":
		stateRole = 5
	}
	if counts.Failed > 0 {
		c.vline(r.x, r.y+1, r.y+r.h-2, base+4)
	}
	scope := "UNASSIGNED"
	switch s.Scope {
	case "sample":
		scope = "PER SAMPLE"
	case "shared":
		scope = "SHARED"
	case "cohort":
		scope = "COHORT"
	}
	marker := strings.Fields(stateLabel(status))[0]
	if s.ID == m.stage {
		scope = "▸ " + scope
	}
	c.text(r.x+2, r.y+1, scope, base+6, r.w-6)
	c.text(r.x+r.w-3, r.y+1, marker, base+stateRole, 1)
	c.text(r.x+2, r.y+2, fit(oneLine(s.Name), r.w-4), base+7, r.w-4)
	c.text(r.x+2, r.y+3, fmt.Sprintf("%d / %d succeeded", counts.Succeeded, counts.Total), base, r.w-4)
	note := nodeStatus(counts)
	if m.sample != "" && s.Scope != "sample" {
		note = "Context · " + note
	}
	c.text(r.x+2, r.y+4, fit(note, r.w-4), base+stateRole, r.w-4)
	parts := stateParts(counts)
	width := r.w - 4
	used := 0
	for i, p := range parts {
		end := used
		if counts.Total > 0 {
			before := 0
			for j := 0; j <= i; j++ {
				before += parts[j].n
			}
			end = before * width / counts.Total
		}
		role := 6
		switch p.label {
		case "succeeded":
			role = 3
		case "running":
			role = 2
		case "failed":
			role = 4
		case "pending":
			role = 1
		default:
			role = 5
		}
		for x := used; x < end; x++ {
			c.put(r.x+2+x, r.y+5, "━", base+role)
		}
		used = end
	}
	if counts.Total == 0 {
		for x := 0; x < width; x++ {
			c.put(r.x+2+x, r.y+5, "┄", base+1)
		}
	}
}

// Arrow keys follow screen geometry; j/k retain dependency-order traversal.
func (m *model) moveStageSpatial(dx, dy int) {
	g := layout(m.data.Stages, m.graphWidth())
	from, ok := g.nodes[m.stage]
	if !ok {
		return
	}
	abs := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	best, score := "", int(^uint(0)>>1)
	for _, stage := range m.data.Stages {
		to := g.nodes[stage.ID]
		x, y := to.x-from.x, to.y-from.y
		if (dx != 0 && x*dx <= 0) || (dy != 0 && y*dy <= 0) {
			continue
		}
		cost := abs(x) + 4*abs(y)
		if dy != 0 {
			cost = abs(y) + 4*abs(x)
		}
		if cost < score {
			best, score = stage.ID, cost
		}
	}
	if best != "" {
		m.stage = best
		m.revealStage()
	}
}
