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
	workspace      string
	read           readFunc
	data           *monitor.Dashboard
	style          theme
	width          int
	height         int
	now            time.Time
	err            error
	reading        bool
	screen         screen
	previous       screen
	searchReturn   screen
	detailReturn   screen
	sample         string
	stage          string
	task           string
	stageFilter    string
	query          string
	searchIndex    int
	listIndex      int
	graphOffset    int
	logOffset      int
	metadataOffset int
	helpOffset     int
	showMetadata   bool
	logStream      string
	follow         bool
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
	if m.contextScreen() == detailScreen {
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
		oldStage := layout(m.data.Stages, m.graphWidth()).nodes[m.stage]
		stageWasVisible := oldStage.y >= m.graphOffset && oldStage.y+oldStage.h <= m.graphOffset+m.graphViewportHeight()
		focused := ""
		if m.contextScreen() == detailScreen {
			focused = m.task
		} else if m.contextScreen() == tasksScreen || m.contextScreen() == attentionScreen {
			items := m.listTasks()
			if m.listIndex < len(items) {
				focused = m.data.Snapshot.Tasks[items[m.listIndex]].Identity
			}
		}
		searchFocus := ""
		if matches := m.data.SearchSamples(m.query); m.screen == searchScreen && m.searchIndex < len(matches) {
			searchFocus = matches[m.searchIndex].ID
		}
		m.reading = false
		m.err = msg.err
		if msg.err == nil {
			var d *monitor.Dashboard
			d, m.err = monitor.Build(msg.snapshot)
			if m.err == nil {
				m.data = d
				m.reconcileSelection(focused)
				if stageWasVisible {
					m.revealStage()
				}
				for i, sample := range m.data.SearchSamples(m.query) {
					if sample.ID == searchFocus {
						m.searchIndex = i
						break
					}
				}
			}
		}
	case tea.KeyPressMsg:
		return m, m.key(msg)
	case tea.PasteMsg:
		if m.screen == searchScreen {
			m.appendQuery(msg.Content)
		}
	}
	return m, nil
}

func (m *model) key(key tea.KeyPressMsg) tea.Cmd {
	k := key.String()
	if k == "ctrl+c" {
		return tea.Quit
	}
	if k == "q" && (m.width < 44 || m.height < 20) {
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
			m.helpOffset = 0
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
		m.searchReturn = m.contextScreen()
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
		case "down":
			m.moveStageSpatial(0, 1)
		case "up":
			m.moveStageSpatial(0, -1)
		case "right", "l":
			m.moveStageSpatial(1, 0)
		case "left", "h":
			m.moveStageSpatial(-1, 0)
		case "j":
			m.moveStage(1)
		case "k":
			m.moveStage(-1)
		case "pgdown":
			m.panGraph(1)
		case "pgup":
			m.panGraph(-1)
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
			m.listIndex = min(max(0, len(items)-1), m.listIndex+taskPageSize(m.bodyHeight()-2))
		case "pgup":
			m.listIndex = max(0, m.listIndex-taskPageSize(m.bodyHeight()-2))
		case "enter", "l", "3":
			if len(items) > 0 {
				task := m.data.Snapshot.Tasks[items[m.listIndex]]
				m.detailReturn = m.screen
				m.task, m.screen, m.follow, m.logOffset = task.Identity, detailScreen, true, 0
				m.showMetadata, m.metadataOffset = k == "3", 0
				return m.refresh()
			}
		}
	case detailScreen:
		if k == "3" {
			m.showMetadata = true
			m.metadataOffset = 0
			return nil
		}
		if m.showMetadata && k != "1" && k != "2" {
			m.metadataOffset = scrollOffset(k, m.metadataOffset, m.metadataLineCount(), m.metadataHeight())
			return nil
		}
		switch k {
		case "1":
			m.showMetadata = false
			m.logStream, m.logOffset = "stdout", 0
		case "2":
			m.showMetadata = false
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
	case helpScreen:
		m.helpOffset = scrollOffset(k, m.helpOffset, len(m.helpRows()), m.bodyHeight())
	}
	return nil
}

func (m *model) searchKey(key tea.KeyPressMsg) tea.Cmd {
	matches := m.data.SearchSamples(m.query)
	switch key.String() {
	case "esc":
		m.screen = m.searchReturn
	case "up":
		m.searchIndex = max(0, m.searchIndex-searchColumns(m.graphWidth()))
	case "down":
		m.searchIndex = min(max(0, len(matches)-1), m.searchIndex+searchColumns(m.graphWidth()))
	case "left":
		m.searchIndex = max(0, m.searchIndex-1)
	case "right":
		m.searchIndex = min(max(0, len(matches)-1), m.searchIndex+1)
	case "pgdown":
		m.panGraph(1)
	case "pgup":
		m.panGraph(-1)
	case "enter":
		if len(matches) > 0 {
			// Prefer an exact entered ID to a similarly named first result.
			for i, match := range matches {
				if match.ID == strings.TrimSpace(m.query) {
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
		m.appendQuery(key.Key().Text)
	}
	return nil
}

func (m *model) appendQuery(text string) {
	text = strings.NewReplacer("\n", "", "\t", "").Replace(clean(text))
	runes := []rune(text)
	remaining := max(0, 128-utf8.RuneCountInString(m.query))
	if len(runes) > 0 && remaining > 0 {
		m.query += string(runes[:min(len(runes), remaining)])
		m.searchIndex = 0
	}
}

// Overlay screens retain the underlying inspector context while refreshing.
func (m *model) contextScreen() screen {
	if m.screen == helpScreen {
		return m.previous
	}
	if m.screen == searchScreen {
		return m.searchReturn
	}
	return m.screen
}

func (m *model) reconcileSelection(focused string) {
	if _, ok := m.data.Task(m.task); m.task != "" && !ok {
		m.task = ""
		if m.contextScreen() == detailScreen {
			switch m.screen {
			case helpScreen:
				m.previous = m.detailReturn
			case searchScreen:
				m.searchReturn = m.detailReturn
			default:
				m.screen = m.detailReturn
			}
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
	if m.contextScreen() == tasksScreen || m.contextScreen() == attentionScreen || m.contextScreen() == detailScreen {
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
	if m.contextScreen() == attentionScreen || (m.contextScreen() == detailScreen && m.detailReturn == attentionScreen) {
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
