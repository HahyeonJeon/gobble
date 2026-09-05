package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/HahyeonJeon/gobble/monitor"
)

type screen int

const (
	dashboardScreen screen = iota
	searchScreen
	tasksScreen
	detailScreen
	attentionScreen
	helpScreen
)

type readFunc func(string) (monitor.Snapshot, error)
type tickMsg time.Time
type loadedMsg struct {
	snapshot monitor.Snapshot
	err      error
}

type model struct {
	workspace    string
	read         readFunc
	data         *monitor.Dashboard
	style        theme
	width        int
	height       int
	now          time.Time
	err          error
	reading      bool
	screen       screen
	previous     screen
	detailReturn screen
	sample       string
	stage        string
	task         string
	stageFilter  string
	query        string
	searchIndex  int
	listIndex    int
	graphOffset  int
	logOffset    int
	logStream    string
	follow       bool
}

func newModel(workspace string, read readFunc, initial monitor.Snapshot, monochrome bool) (*model, error) {
	d, err := monitor.Build(initial)
	if err != nil {
		return nil, err
	}
	m := &model{workspace: workspace, read: read, data: d, style: newTheme(monochrome), width: 100, height: 32, now: time.Now(), follow: true, logStream: "stderr"}
	if len(d.Stages) > 0 {
		m.stage = d.Stages[0].ID
	}
	return m, nil
}

func clockTick() tea.Cmd {
	return tea.Tick(time.Second, func(now time.Time) tea.Msg { return tickMsg(now) })
}

func (m *model) Init() tea.Cmd { return clockTick() }

// Exactly one file read can be outstanding. Commands capture values, never
// mutable model fields, so the UI thread owns all selection and view state.
func (m *model) refresh() tea.Cmd {
	if m.reading {
		return nil
	}
	m.reading = true
	read, instance := m.read, ""
	if m.screen == detailScreen {
		instance = m.task
	}
	return func() tea.Msg {
		snapshot, err := read(instance)
		return loadedMsg{snapshot: snapshot, err: err}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
		m.revealStage()
	case tickMsg:
		m.now = time.Time(msg)
		return m, tea.Batch(clockTick(), m.refresh())
	case loadedMsg:
		focused := ""
		if m.screen == tasksScreen || m.screen == attentionScreen {
			items := m.listTasks()
			if m.listIndex < len(items) {
				focused = m.data.Snapshot.Tasks[items[m.listIndex]].Identity
			}
		}
		m.reading = false
		m.err = msg.err
		if msg.err == nil {
			var d *monitor.Dashboard
			d, m.err = monitor.Build(msg.snapshot)
			if m.err == nil {
				m.data = d
				m.reconcileSelection(focused)
			}
		}
	case tea.KeyPressMsg:
		return m, m.key(msg)
	}
	return m, nil
}

func (m *model) key(key tea.KeyPressMsg) tea.Cmd {
	k := key.String()
	if k == "ctrl+c" {
		return tea.Quit
	}
	if m.screen == searchScreen {
		return m.searchKey(key)
	}
	switch k {
	case "q":
		return tea.Quit
	case "?":
		if m.screen == helpScreen {
			m.screen = m.previous
		} else {
			m.previous, m.screen = m.screen, helpScreen
		}
		return nil
	case "esc":
		switch m.screen {
		case detailScreen:
			m.screen = m.detailReturn
		case helpScreen:
			m.screen = m.previous
		case dashboardScreen:
			m.sample = ""
		default:
			m.screen = dashboardScreen
		}
		return nil
	case "/", "s":
		m.query, m.searchIndex, m.screen = "", 0, searchScreen
		return nil
	case "!":
		m.screen, m.listIndex = attentionScreen, 0
		return nil
	case "r":
		return m.refresh()
	}
	switch m.screen {
	case dashboardScreen:
		switch k {
		case "down", "j", "right", "l":
			m.moveStage(1)
		case "up", "k", "left", "h":
			m.moveStage(-1)
		case "pgdown":
			m.graphOffset += max(1, m.bodyHeight()-2)
		case "pgup":
			m.graphOffset = max(0, m.graphOffset-m.bodyHeight()+2)
		case "home":
			m.graphOffset = 0
		case "enter":
			m.stageFilter, m.screen, m.listIndex = m.stage, tasksScreen, 0
		case "t":
			m.stageFilter, m.screen, m.listIndex = "", tasksScreen, 0
		}
	case tasksScreen, attentionScreen:
		items := m.listTasks()
		switch k {
		case "down", "j":
			m.listIndex = min(max(0, len(items)-1), m.listIndex+1)
		case "up", "k":
			m.listIndex = max(0, m.listIndex-1)
		case "pgdown":
			m.listIndex = min(max(0, len(items)-1), m.listIndex+max(1, m.bodyHeight()-3))
		case "pgup":
			m.listIndex = max(0, m.listIndex-max(1, m.bodyHeight()-3))
		case "enter", "l":
			if len(items) > 0 {
				task := m.data.Snapshot.Tasks[items[m.listIndex]]
				m.detailReturn = m.screen
				m.task, m.screen, m.follow, m.logOffset = task.Identity, detailScreen, true, 0
				return m.refresh()
			}
		}
	case detailScreen:
		switch k {
		case "1":
			m.logStream, m.logOffset = "stdout", 0
		case "2":
			m.logStream, m.logOffset = "stderr", 0
		case "f":
			if m.follow {
				m.logOffset = m.logTailOffset()
			}
			m.follow = !m.follow
		case "up", "k", "pgup":
			if m.follow {
				m.logOffset = m.logTailOffset()
			}
			m.follow = false
			m.logOffset = max(0, m.logOffset-3)
		case "down", "j", "pgdown":
			if m.follow {
				m.logOffset = m.logTailOffset()
			}
			m.follow = false
			m.logOffset = min(m.logTailOffset(), m.logOffset+3)
		case "end":
			m.follow = true
		}
	}
	return nil
}

func (m *model) searchKey(key tea.KeyPressMsg) tea.Cmd {
	matches := m.data.SearchSamples(m.query)
	switch key.String() {
	case "esc":
		m.screen = dashboardScreen
	case "up":
		m.searchIndex = max(0, m.searchIndex-1)
	case "down":
		m.searchIndex = min(max(0, len(matches)-1), m.searchIndex+1)
	case "enter":
		if len(matches) > 0 {
			// Prefer an exact entered ID to a similarly named first result.
			for i, match := range matches {
				if strings.EqualFold(match.ID, strings.TrimSpace(m.query)) {
					m.searchIndex = i
					break
				}
			}
			m.sample, m.screen = matches[m.searchIndex].ID, dashboardScreen
		}
	case "backspace":
		if m.query != "" {
			_, size := utf8.DecodeLastRuneInString(m.query)
			m.query = m.query[:len(m.query)-size]
		}
		m.searchIndex = 0
	case "ctrl+u":
		m.query, m.searchIndex = "", 0
	default:
		text := clean(key.Key().Text)
		if text != "" && utf8.RuneCountInString(m.query+text) <= 128 {
			m.query += strings.ReplaceAll(text, "\n", "")
			m.searchIndex = 0
		}
	}
	return nil
}

func (m *model) reconcileSelection(focused string) {
	if _, ok := m.data.Task(m.task); m.task != "" && !ok {
		m.task = ""
		if m.screen == detailScreen {
			m.screen = tasksScreen
		}
	}
	if m.sample != "" && len(m.data.SampleTasks(m.sample)) == 0 {
		m.sample = ""
	}
	if m.stageIndex() < 0 && len(m.data.Stages) > 0 {
		m.stage = m.data.Stages[0].ID
	}
	items := m.listTasks()
	// Identity preserves list focus when another task is inserted or removed.
	if m.screen == tasksScreen || m.screen == attentionScreen {
		for i, index := range items {
			if m.data.Snapshot.Tasks[index].Identity == focused {
				m.listIndex = i
				break
			}
		}
	}
	m.listIndex = min(m.listIndex, max(0, len(items)-1))
	m.searchIndex = min(m.searchIndex, max(0, len(m.data.SearchSamples(m.query))-1))
}

func (m *model) listTasks() []int {
	if m.screen == attentionScreen {
		return m.data.Attention
	}
	if m.stageFilter != "" {
		for _, stage := range m.data.Stages {
			if stage.ID == m.stageFilter {
				return m.data.StageTasks(stage, m.sample)
			}
		}
	}
	if m.sample != "" {
		return m.data.SampleTasks(m.sample)
	}
	items := make([]int, len(m.data.Snapshot.Tasks))
	for i := range items {
		items[i] = i
	}
	return items
}

func (m *model) stageIndex() int {
	for i, stage := range m.data.Stages {
		if stage.ID == m.stage {
			return i
		}
	}
	return -1
}

func (m *model) moveStage(delta int) {
	if len(m.data.Stages) == 0 {
		return
	}
	i := min(len(m.data.Stages)-1, max(0, m.stageIndex()+delta))
	m.stage = m.data.Stages[i].ID
	m.revealStage()
}
