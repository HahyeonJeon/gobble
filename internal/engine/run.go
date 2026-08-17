package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Control-plane document names under ControlDir.
const (
	PlanFile  = "plan.json"
	TasksFile = "tasks.json"
)

// Task status values persisted in task state.
const (
	StatusNotStarted = "not-started"
	StatusRunning    = "running"
	StatusSucceeded  = "succeeded"
	StatusFailed     = "failed"
	StatusBlocked    = "blocked"
)

const (
	executorProcess = "process"
	executorDocker  = "docker"
)

// execTask is the scheduler-to-executor seam. The scheduler never
// starts a host process; this function does.
var execTask = executeTask

// Run occupies workspace after Check, schedules doc, and returns when
// no more independent work can start. A nil result means every task
// succeeded.
func Run(req Request) []Defect {
	if d := Check(req); len(d) > 0 {
		return d
	}
	n := req.Cap
	if n == 0 {
		n = DefaultCap
	}
	s, defects := occupy(req)
	if len(defects) > 0 {
		return defects
	}
	return s.loop(n)
}

type sched struct {
	workspace string
	doc       Document
	run       jsonRun
	tasks     map[string]*jsonTaskState
	persist   error
}

type jsonRun struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Started string `json:"started"`
	Ended   string `json:"ended,omitempty"`
}

type jsonTaskState struct {
	ID        string        `json:"id"`
	Status    string        `json:"status"`
	Executor  string        `json:"executor"`
	Image     string        `json:"image"`
	Command   []string      `json:"command"`
	Resources jsonResources `json:"resources"`
	Params    []jsonParam   `json:"params"`
	Reason    string        `json:"reason"`
	Error     *jsonTaskErr  `json:"error,omitempty"`
	Stdout    string        `json:"stdout,omitempty"`
	Stderr    string        `json:"stderr,omitempty"`
}

type jsonTaskErr struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

type jsonTasksFile struct {
	Tasks []jsonTaskState `json:"tasks"`
}

func occupy(req Request) (*sched, []Defect) {
	s := &sched{
		workspace: req.Workspace,
		doc:       req.Document,
		run: jsonRun{
			ID:      runID(req.Document),
			Status:  StatusRunning,
			Started: time.Now().UTC().Format(time.RFC3339Nano),
		},
		tasks: make(map[string]*jsonTaskState, len(req.Document.Tasks)),
	}
	for _, t := range req.Document.Tasks {
		st := initialTask(t)
		s.tasks[t.ID] = &st
	}
	root := filepath.Join(req.Workspace, ControlDir)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, pathDefects(err)
	}
	ident := filepath.Join(root, RunIdentityFile)
	f, err := os.OpenFile(ident, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, []Defect{{
				Code:    DefectOccupiedWorkspace,
				Message: "occupied workspace",
				Paths:   []string{ControlDir + "/" + RunIdentityFile},
			}}
		}
		return nil, pathDefects(err)
	}
	if err := f.Close(); err != nil {
		return nil, pathDefects(err)
	}
	plan, err := marshalPlan(req.Document)
	if err != nil {
		return nil, pathDefects(err)
	}
	if err := writeAtomic(filepath.Join(root, PlanFile), plan); err != nil {
		return nil, pathDefects(err)
	}
	if err := s.writeTasks(); err != nil {
		return nil, pathDefects(err)
	}
	if err := s.writeRun(); err != nil {
		return nil, pathDefects(err)
	}
	return s, nil
}

func pathDefects(err error) []Defect {
	return []Defect{{
		Code:    DefectInvalidPath,
		Message: err.Error(),
	}}
}

func runID(doc Document) string {
	if doc.Name != "" {
		return doc.Name
	}
	return "run"
}

func initialTask(t TaskPlan) jsonTaskState {
	kind := executorProcess
	if t.Image != "" {
		kind = executorDocker
	}
	return jsonTaskState{
		ID:       t.ID,
		Status:   StatusNotStarted,
		Executor: kind,
		Image:    t.Image,
		Command:  jsonStrings(t.Command),
		Resources: jsonResources{
			CPU:    t.Resources.CPU,
			Memory: t.Resources.Memory,
		},
		Params: encodeParams(t.Params),
	}
}

func (s *sched) loop(n int) []Defect {
	reports := make(chan report, n)
	running := 0
	for {
		for s.persist == nil && running < n {
			id := s.nextReady()
			if id == "" {
				break
			}
			s.launch(id, reports)
			running++
		}
		if running == 0 {
			break
		}
		r := <-reports
		running--
		s.apply(r)
	}
	s.finish()
	return s.failures()
}

func (s *sched) nextReady() string {
	for _, t := range s.doc.Tasks {
		st := s.tasks[t.ID]
		if st == nil || st.Status != StatusNotStarted {
			continue
		}
		if s.upstreamReady(t.ID) {
			return t.ID
		}
	}
	return ""
}

func (s *sched) upstreamReady(id string) bool {
	for _, e := range s.doc.Edges {
		if e.ToTask != id || e.FromTask == "" {
			continue
		}
		up := s.tasks[e.FromTask]
		if up == nil || up.Status != StatusSucceeded {
			return false
		}
		path := s.inputPath(id, e.ToPort)
		if path == "" || !regularFile(workspaceFile(s.workspace, path)) {
			return false
		}
	}
	return true
}

func (s *sched) inputPath(taskID, port string) string {
	for _, t := range s.doc.Tasks {
		if t.ID != taskID {
			continue
		}
		for _, in := range t.Inputs {
			if in.Name == port {
				return in.Path
			}
		}
	}
	return ""
}

func (s *sched) launch(id string, reports chan report) {
	st := s.tasks[id]
	st.Status = StatusRunning
	st.Reason = "ready"
	s.notePersist(s.writeTasks())
	var task TaskPlan
	for _, t := range s.doc.Tasks {
		if t.ID == id {
			task = t
			break
		}
	}
	ws := s.workspace
	go func() {
		reports <- execTask(ws, task)
	}()
}

func (s *sched) apply(r report) {
	st := s.tasks[r.ID]
	if st == nil {
		return
	}
	st.Stdout = r.Stdout
	st.Stderr = r.Stderr
	if r.Published && r.Exit == 0 && r.Message == "" {
		st.Status = StatusSucceeded
		st.Reason = "ready"
		st.Error = nil
	} else {
		st.Status = StatusFailed
		st.Reason = "ready"
		msg := r.Message
		if msg == "" {
			msg = "exit " + strconv.Itoa(r.Exit)
		}
		st.Error = &jsonTaskErr{Unit: r.ID, Message: msg}
		s.blockFrom(r.ID)
	}
	s.notePersist(s.writeTasks())
}

func (s *sched) blockFrom(id string) {
	src := s.tasks[id]
	why := "upstream failed"
	if src != nil && src.Status == StatusBlocked {
		why = "upstream blocked"
	}
	for _, e := range s.doc.Edges {
		if e.FromTask != id {
			continue
		}
		dep := s.tasks[e.ToTask]
		if dep == nil || dep.Status != StatusNotStarted {
			continue
		}
		dep.Status = StatusBlocked
		dep.Reason = why
		s.blockFrom(e.ToTask)
	}
}

func (s *sched) finish() {
	s.run.Ended = time.Now().UTC().Format(time.RFC3339Nano)
	s.run.Status = StatusSucceeded
	for _, t := range s.doc.Tasks {
		st := s.tasks[t.ID]
		if st == nil || st.Status == StatusSucceeded {
			continue
		}
		s.run.Status = StatusFailed
		break
	}
	s.notePersist(s.writeRun())
}

func (s *sched) notePersist(err error) {
	if err != nil && s.persist == nil {
		s.persist = err
	}
}

func (s *sched) failures() []Defect {
	var out []Defect
	for _, t := range s.doc.Tasks {
		st := s.tasks[t.ID]
		if st == nil {
			out = append(out, Defect{
				Code:    DefectFailed,
				Unit:    t.ID,
				Message: "task failed",
			})
			continue
		}
		switch st.Status {
		case StatusSucceeded, StatusBlocked:
			continue
		case StatusNotStarted:
			out = append(out, Defect{
				Code:    DefectFailed,
				Unit:    t.ID,
				Message: "not started",
			})
		default:
			msg := "task failed"
			if st.Error != nil && st.Error.Message != "" {
				msg = st.Error.Message
			}
			out = append(out, Defect{
				Code:    DefectFailed,
				Unit:    t.ID,
				Message: msg,
			})
		}
	}
	if s.persist != nil {
		out = append(out, Defect{
			Code:    DefectInvalidPath,
			Message: "persist: " + s.persist.Error(),
		})
	}
	return out
}

func (s *sched) writeRun() error {
	data, err := json.MarshalIndent(s.run, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.workspace, ControlDir, RunIdentityFile), append(data, '\n'))
}

func (s *sched) writeTasks() error {
	doc := jsonTasksFile{Tasks: make([]jsonTaskState, 0, len(s.doc.Tasks))}
	for _, t := range s.doc.Tasks {
		if st := s.tasks[t.ID]; st != nil {
			doc.Tasks = append(doc.Tasks, *st)
		}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.workspace, ControlDir, TasksFile), append(data, '\n'))
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil {
		os.Remove(tmp)
		return werr
	}
	if cerr != nil {
		os.Remove(tmp)
		return cerr
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
