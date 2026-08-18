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
	StatusIncomplete = "incomplete"
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
	budget    resourceBudget
}

type resourceBudget struct {
	host hostCapacity
	cpu  float64
	mem  int64
}

func newBudget(host hostCapacity) resourceBudget {
	return resourceBudget{host: host, cpu: host.CPU, mem: host.Memory}
}

func (b resourceBudget) fits(t TaskPlan) bool {
	if b.host.CPUKnown && t.Resources.CPU > 0 && t.Resources.CPU > b.cpu {
		return false
	}
	n, ok := parseMemory(t.Resources.Memory)
	if ok && b.host.MemKnown && n > 0 && n > b.mem {
		return false
	}
	return true
}

func (b *resourceBudget) occupy(t TaskPlan) {
	if t.Resources.CPU > 0 {
		b.cpu -= t.Resources.CPU
	}
	if n, ok := parseMemory(t.Resources.Memory); ok && n > 0 {
		b.mem -= n
	}
}

func (b *resourceBudget) release(t TaskPlan) {
	if t.Resources.CPU > 0 {
		b.cpu += t.Resources.CPU
	}
	if n, ok := parseMemory(t.Resources.Memory); ok && n > 0 {
		b.mem += n
	}
}

type jsonRun struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	Status        string         `json:"status"`
	Started       string         `json:"started"`
	Ended         string         `json:"ended,omitempty"`
	Occupancy     *jsonOccupancy `json:"occupancy"`
}

type jsonTaskState struct {
	ID           string         `json:"id"`
	Instance     string         `json:"instance"`
	ShardIndex   int            `json:"shard_index"`
	ShardCount   int            `json:"shard_count"`
	Attempt      int            `json:"attempt"`
	Status       string         `json:"status"`
	Executor     string         `json:"executor"`
	Image        string         `json:"image"`
	Command      []string       `json:"command"`
	Resources    jsonResources  `json:"resources"`
	Params       []jsonParam    `json:"params"`
	Reason       string         `json:"reason"`
	Error        *jsonTaskErr   `json:"error,omitempty"`
	Stdout       string         `json:"stdout,omitempty"`
	Stderr       string         `json:"stderr,omitempty"`
	Started      string         `json:"started,omitempty"`
	Ended        string         `json:"ended,omitempty"`
	Fingerprints []jsonFileHash `json:"fingerprints,omitempty"`
	Checksums    []jsonFileHash `json:"checksums,omitempty"`
	Lineage      []jsonLineage  `json:"lineage,omitempty"`
	Decision     string         `json:"decision,omitempty"`
	ReuseReason  string         `json:"reuse_reason,omitempty"`
	Differing    []string       `json:"differing,omitempty"`
}

type jsonTaskErr struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

type jsonTasksFile struct {
	SchemaVersion int             `json:"schema_version"`
	Tasks         []jsonTaskState `json:"tasks"`
}

func occupy(req Request) (*sched, []Defect) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	host, err := currentHost()
	if err != nil {
		return nil, pathDefects(err)
	}
	doc := req.Document
	for i := range doc.Tasks {
		applyReservedDefaults(&doc.Tasks[i])
	}
	s := &sched{
		workspace: req.Workspace,
		doc:       doc,
		run: jsonRun{
			SchemaVersion: SchemaVersion,
			ID:            runID(doc),
			Status:        StatusRunning,
			Started:       now,
			Occupancy: &jsonOccupancy{
				Active:  true,
				Host:    host,
				PID:     os.Getpid(),
				Started: now,
			},
		},
		tasks:  make(map[string]*jsonTaskState, len(doc.Tasks)),
		budget: newBudget(readHostCapacity()),
	}
	for _, t := range doc.Tasks {
		st := initialTask(t)
		s.tasks[t.ID] = &st
	}
	root := filepath.Join(req.Workspace, ControlDir)
	lock, defects := claimOccupy(root)
	if len(defects) > 0 {
		return nil, defects
	}
	defer lock.Close()
	if existing, exists, err := readRunIdentity(req.Workspace); err != nil {
		return nil, pathDefects(err)
	} else if exists && occupancyIsActive(existing) {
		return nil, occupiedDefect()
	}
	plan, err := marshalControlPlan(doc)
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
	applyReservedDefaults(&t)
	return jsonTaskState{
		ID:         t.ID,
		Instance:   t.Instance,
		ShardIndex: t.ShardIndex,
		ShardCount: t.ShardCount,
		Attempt:    t.Attempt,
		Status:     StatusNotStarted,
		Executor:   kind,
		Image:      t.Image,
		Command:    jsonStrings(t.Command),
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
		if !s.upstreamReady(t.ID) {
			continue
		}
		if !s.budget.fits(t) {
			continue
		}
		return t.ID
	}
	return ""
}

func (s *sched) upstreamReady(id string) bool {
	for _, e := range s.doc.Edges {
		if e.ToTask != id {
			continue
		}
		if e.FromTask != "" {
			up := s.tasks[e.FromTask]
			if up == nil || up.Status != StatusSucceeded {
				return false
			}
		}
		for _, path := range e.Wait {
			if path == "" || !regularFile(workspaceFile(s.workspace, path)) {
				return false
			}
		}
	}
	return true
}

func (s *sched) taskByID(id string) (TaskPlan, bool) {
	for _, t := range s.doc.Tasks {
		if t.ID == id {
			return t, true
		}
	}
	return TaskPlan{}, false
}

func (s *sched) launch(id string, reports chan report) {
	st := s.tasks[id]
	st.Status = StatusRunning
	st.Reason = "ready"
	st.Started = time.Now().UTC().Format(time.RFC3339Nano)
	task, _ := s.taskByID(id)
	s.budget.occupy(task)
	s.notePersist(s.writeTasks())
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
	if task, ok := s.taskByID(r.ID); ok {
		s.budget.release(task)
	}
	st.Stdout = r.Stdout
	st.Stderr = r.Stderr
	st.Ended = time.Now().UTC().Format(time.RFC3339Nano)
	if r.Published && r.Exit == 0 && r.Message == "" {
		st.Status = StatusSucceeded
		st.Reason = "ready"
		st.Error = nil
		if task, ok := s.taskByID(r.ID); ok {
			s.notePersist(s.recordSuccess(st, task))
		}
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

func (s *sched) recordSuccess(st *jsonTaskState, task TaskPlan) error {
	inputs, err := fileHashes(s.workspace, task.Inputs)
	if err != nil {
		return err
	}
	outputs, err := fileHashes(s.workspace, task.Outputs)
	if err != nil {
		return err
	}
	st.Fingerprints = inputs
	st.Checksums = outputs
	st.Lineage = successLineage(s, task, inputs, outputs)
	return nil
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
		dep.Ended = time.Now().UTC().Format(time.RFC3339Nano)
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
	doc := jsonTasksFile{
		SchemaVersion: SchemaVersion,
		Tasks:         make([]jsonTaskState, 0, len(s.doc.Tasks)),
	}
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
