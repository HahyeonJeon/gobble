package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
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
	StatusUnknown    = "unknown"
)

const (
	executorProcess = "process"
	executorDocker  = "docker"
)

const pollInterval = 20 * time.Millisecond

const defaultCompletionBound = 30 * time.Second

var errEngineBound = errors.New("engine bound")

// completionBoundNanos is the Engine ceiling on each Submit/Poll/Cancel/Reconcile
// call that has not returned a known disposition. Tests may shorten it.
// It is not a public timeout API.
var completionBoundNanos atomic.Int64

func init() {
	setCompletionBound(defaultCompletionBound)
}

func setCompletionBound(d time.Duration) {
	completionBoundNanos.Store(int64(d))
}

func currentBound() time.Duration {
	n := completionBoundNanos.Load()
	if n <= 0 {
		return defaultCompletionBound
	}
	return time.Duration(n)
}

// runExecutor is an optional test seam. Nil means each occupy creates
// its own Local executor.
var runExecutor exec.Executor

func schedulerExecutor() exec.Executor {
	if runExecutor != nil {
		return runExecutor
	}
	return exec.Local()
}

// Run occupies workspace after Check, schedules doc, and returns when
// no more independent work can start. A nil result means every task
// succeeded. When ctx is done, in-flight work is canceled, those
// identities are persisted incomplete, occupancy stays active, and the
// result is DefectCanceled.
func Run(ctx context.Context, req Request) []Defect {
	if ctx == nil {
		ctx = context.Background()
	}
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
	return s.loop(ctx, n)
}

type sched struct {
	workspace string
	doc       Document
	run       jsonRun
	snapshot  string
	tasks     map[string]*jsonTaskState
	history   []jsonTaskState
	resume    map[string]reuseDecision
	launched  map[string]bool
	persist   error
	budget    resourceBudget
	exec      exec.Executor
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
	Snapshot      string         `json:"snapshot,omitempty"`
	ID            string         `json:"id"`
	Status        string         `json:"status"`
	Started       string         `json:"started"`
	Ended         string         `json:"ended,omitempty"`
	Occupancy     *jsonOccupancy `json:"occupancy"`
}

type jsonTaskState struct {
	ID               string         `json:"id"`
	Instance         string         `json:"instance"`
	ShardIndex       int            `json:"shard_index"`
	ShardCount       int            `json:"shard_count"`
	Attempt          int            `json:"attempt"`
	Status           string         `json:"status"`
	Executor         string         `json:"executor"`
	Image            string         `json:"image"`
	Command          []string       `json:"command"`
	Script           string         `json:"script,omitempty"`
	Resources        jsonResources  `json:"resources"`
	Params           []jsonParam    `json:"params"`
	EnvDigest        string         `json:"env_digest,omitempty"`
	RuntimeID        string         `json:"runtime_id,omitempty"`
	ImageDigest      string         `json:"image_digest,omitempty"`
	ExecutablePath   string         `json:"executable_path,omitempty"`
	ExecutableSHA256 string         `json:"executable_sha256,omitempty"`
	Reason           string         `json:"reason"`
	Error            *jsonTaskErr   `json:"error,omitempty"`
	Stdout           string         `json:"stdout,omitempty"`
	Stderr           string         `json:"stderr,omitempty"`
	Started          string         `json:"started,omitempty"`
	Ended            string         `json:"ended,omitempty"`
	Fingerprints     []jsonFileHash `json:"fingerprints,omitempty"`
	Checksums        []jsonFileHash `json:"checksums,omitempty"`
	Lineage          []jsonLineage  `json:"lineage,omitempty"`
	Decision         string         `json:"decision,omitempty"`
	ReuseReason      string         `json:"reuse_reason,omitempty"`
	Differing        []string       `json:"differing,omitempty"`
	Change           string         `json:"change,omitempty"`
}

type jsonTaskErr struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

type jsonTasksFile struct {
	SchemaVersion int             `json:"schema_version"`
	Snapshot      string          `json:"snapshot,omitempty"`
	Tasks         []jsonTaskState `json:"tasks"`
}

func occupy(req Request) (*sched, []Defect) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	host, err := currentHost()
	if err != nil {
		return nil, pathDefects(err)
	}
	root := filepath.Join(req.Workspace, ControlDir)
	lock, defects := claimOccupy(root)
	if len(defects) > 0 {
		return nil, defects
	}
	if existing, exists, err := readRunIdentity(req.Workspace); err != nil {
		lock.Close()
		return nil, pathDefects(err)
	} else if exists && occupancyIsActive(existing) {
		lock.Close()
		return nil, occupiedDefect()
	}
	doc := cloneDocument(req.Document)
	for i := range doc.Tasks {
		applyReservedDefaults(&doc.Tasks[i])
	}
	lease := newOccupancyID()
	snapshot := newOccupancyID()
	ex := schedulerExecutor()
	s := &sched{
		workspace: req.Workspace,
		doc:       doc,
		snapshot:  snapshot,
		run: jsonRun{
			SchemaVersion: SchemaVersion,
			Snapshot:      snapshot,
			ID:            runID(doc),
			Status:        StatusRunning,
			Started:       now,
			Occupancy: &jsonOccupancy{
				Active:  true,
				Host:    host,
				PID:     os.Getpid(),
				Lease:   lease,
				Started: now,
			},
		},
		tasks:  make(map[string]*jsonTaskState, len(doc.Tasks)),
		budget: newBudget(readHostCapacity()),
		exec:   ex,
	}
	for _, t := range doc.Tasks {
		st := initialTask(t)
		s.tasks[reservedIdentity(t)] = &st
	}
	if err := s.writeControl(); err != nil {
		lock.Close()
		return nil, pathDefects(err)
	}
	retainLease(req.Workspace, lock, ex)
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
		Script:     t.Script,
		Resources: jsonResources{
			CPU:    t.Resources.CPU,
			Memory: t.Resources.Memory,
		},
		Params:    encodeParams(t.Params),
		EnvDigest: envDigest(t.Env),
	}
}

type report struct {
	ID               string
	Exit             int
	Message          string
	Stdout           string
	Stderr           string
	Published        bool
	RuntimeID        string
	ImageDigest      string
	ExecutablePath   string
	ExecutableSHA256 string
	Unknown          bool
}

type startEvent struct {
	ident  string
	handle exec.Handle
	sub    exec.Report
}

func (s *sched) executor() exec.Executor {
	if s.exec != nil {
		return s.exec
	}
	return schedulerExecutor()
}

func (s *sched) loop(ctx context.Context, n int) []Defect {
	if s.resume != nil {
		s.reapLive()
	}
	reports := make(chan report, n)
	starts := make(chan startEvent, n)
	running := 0
	handles := make(map[string]exec.Handle)
	inflight := make(map[string]bool)
	canceled := false
	finishIdent := func(ident string) {
		if !inflight[ident] {
			return
		}
		delete(inflight, ident)
		delete(handles, ident)
		running--
	}
	cancelInFlight := func() {
		canceled = true
		for ident, h := range handles {
			if err := boundedCancel(s.executor(), h); err != nil {
				s.markUnknown(ident, err.Error())
				finishIdent(ident)
			}
		}
	}
	for {
		if ctx.Err() != nil && !canceled {
			cancelInFlight()
		}
		for !canceled && s.persist == nil && running < n {
			id := s.nextReady()
			if id == "" {
				break
			}
			s.launch(id, starts, reports)
			inflight[id] = true
			running++
		}
		if running == 0 {
			break
		}
		if canceled {
			select {
			case st := <-starts:
				if !inflight[st.ident] {
					continue
				}
				handles[st.ident] = st.handle
				s.persistStart(st)
				if err := boundedCancel(s.executor(), st.handle); err != nil {
					s.markUnknown(st.ident, err.Error())
					finishIdent(st.ident)
				}
			case r := <-reports:
				if !inflight[r.ID] {
					continue
				}
				finishIdent(r.ID)
				if r.Unknown {
					s.markUnknown(r.ID, r.Message)
				} else {
					s.markIncomplete(r.ID)
				}
			}
			continue
		}
		select {
		case st := <-starts:
			if !inflight[st.ident] {
				continue
			}
			handles[st.ident] = st.handle
			s.persistStart(st)
		case r := <-reports:
			if !inflight[r.ID] {
				continue
			}
			finishIdent(r.ID)
			s.apply(r)
		case <-ctx.Done():
			cancelInFlight()
		}
	}
	if canceled {
		s.syncUnknown()
		s.notePersist(s.persistControl())
		out := s.cancelDefects()
		if unknown := s.unknownUnits(); len(unknown) > 0 {
			out = append(out, unknownBackendDefects(unknown)...)
		}
		return out
	}
	s.assignBlockedUpstream()
	s.finish()
	return s.failures()
}

func (s *sched) persistStart(st startEvent) {
	task := s.tasks[st.ident]
	if task == nil {
		return
	}
	if st.handle.RuntimeID == "" || st.sub.RuntimeID == "" {
		s.markUnknown(st.ident, "empty runtime id")
		return
	}
	task.Status = StatusRunning
	task.Reason = "ready"
	task.RuntimeID = st.sub.RuntimeID
	if st.sub.ImageDigest != "" {
		task.ImageDigest = st.sub.ImageDigest
	}
	s.notePersist(s.persistControl())
}

func (s *sched) markIncomplete(ident string) {
	st := s.tasks[ident]
	if st == nil {
		return
	}
	if task, ok := s.taskByIdent(ident); ok {
		s.budget.release(task)
	}
	st.Status = StatusIncomplete
	st.Reason = "canceled"
	if st.Ended == "" {
		st.Ended = time.Now().UTC().Format(time.RFC3339Nano)
	}
	s.notePersist(s.persistControl())
}

func (s *sched) markUnknown(ident, message string) {
	st := s.tasks[ident]
	if st == nil {
		return
	}
	if st.Status != StatusUnknown {
		if task, ok := s.taskByIdent(ident); ok && st.Status == StatusRunning {
			s.budget.release(task)
		}
	}
	st.Status = StatusUnknown
	st.Reason = "unknown-backend"
	if message != "" {
		st.Error = &jsonTaskErr{Unit: ident, Message: message}
	}
	if st.Ended == "" {
		st.Ended = time.Now().UTC().Format(time.RFC3339Nano)
	}
	s.syncUnknown()
	s.notePersist(s.persistControl())
}

func (s *sched) reapLive() {
	ex := s.executor()
	for ident, st := range s.tasks {
		if st == nil {
			continue
		}
		if st.Status != StatusRunning && st.Status != StatusIncomplete && st.Status != StatusUnknown {
			continue
		}
		h, ok := backendHandle(s.workspace, ident, st)
		if !ok {
			if st.Status == StatusUnknown {
				continue
			}
			continue
		}
		r, err := boundedReconcile(ex, h)
		if err != nil {
			s.markUnknown(ident, err.Error())
			continue
		}
		if r.Running {
			if err := boundedCancel(ex, h); err != nil {
				s.markUnknown(ident, err.Error())
				continue
			}
			stopped := false
			for {
				pr, perr := boundedPoll(ex, h)
				if perr != nil {
					s.markUnknown(ident, perr.Error())
					stopped = true
					break
				}
				if !pr.Running {
					stopped = true
					break
				}
				time.Sleep(pollInterval)
			}
			if !stopped || (s.tasks[ident] != nil && s.tasks[ident].Status == StatusUnknown) {
				continue
			}
		}
		if st.Status == StatusRunning {
			st.Status = StatusIncomplete
			st.Reason = reasonPreviousIncomplete
			if st.Ended == "" {
				st.Ended = time.Now().UTC().Format(time.RFC3339Nano)
			}
		}
	}
	s.syncUnknown()
	s.notePersist(s.persistControl())
}

func backendHandle(workspace, ident string, st *jsonTaskState) (exec.Handle, bool) {
	backend := st.Executor
	if backend == "" {
		if st.Image != "" {
			backend = executorDocker
		} else {
			backend = executorProcess
		}
	}
	if st.RuntimeID != "" {
		return exec.Handle{Identity: ident, Backend: backend, RuntimeID: st.RuntimeID}, true
	}
	h, sub, ok := awaitLateSubmit(workspace, ident)
	if !ok {
		return exec.Handle{}, false
	}
	if sub.RuntimeID != "" {
		st.RuntimeID = sub.RuntimeID
	}
	if h.Backend == "" {
		h.Backend = backend
	}
	if h.Identity == "" {
		h.Identity = ident
	}
	return h, true
}

func boundedReconcile(ex exec.Executor, h exec.Handle) (exec.Report, error) {
	ctx, cancel := context.WithTimeout(context.Background(), currentBound())
	defer cancel()
	r, err := ex.Reconcile(ctx, h)
	return r, boundCallErrAfter(ctx, err)
}

func (s *sched) nextReady() string {
	for _, t := range s.doc.Tasks {
		ident := reservedIdentity(t)
		st := s.tasks[ident]
		if st == nil {
			continue
		}
		if s.resume != nil {
			if s.resume[ident].Decision != reuseRerun || s.launched[ident] {
				continue
			}
		} else if st.Status != StatusNotStarted {
			continue
		}
		if !s.upstreamReady(t.ID) {
			continue
		}
		if !s.budget.fits(t) {
			continue
		}
		return ident
	}
	return ""
}

func (s *sched) stateByTaskID(id string) *jsonTaskState {
	if t, ok := s.taskByID(id); ok {
		return s.tasks[reservedIdentity(t)]
	}
	return s.tasks[id]
}

func (s *sched) taskByIdent(ident string) (TaskPlan, bool) {
	for _, t := range s.doc.Tasks {
		if reservedIdentity(t) == ident {
			return t, true
		}
	}
	return TaskPlan{}, false
}

func (s *sched) upstreamReady(id string) bool {
	for _, e := range s.doc.Edges {
		if e.ToTask != id {
			continue
		}
		if e.FromTask != "" {
			up := s.stateByTaskID(e.FromTask)
			if up == nil || up.Status != StatusSucceeded {
				return false
			}
			if s.resume != nil {
				upTask, ok := s.taskByID(e.FromTask)
				if !ok {
					return false
				}
				ident := reservedIdentity(upTask)
				if s.resume[ident].Decision == reuseRerun && !s.succeededThisResume(ident) {
					return false
				}
			}
		} else if s.resume != nil {
			for _, path := range e.Wait {
				if s.waitPathPendingRerun(path) {
					return false
				}
			}
		}
		for _, path := range e.Wait {
			if path == "" || !waitPathReady(s.workspace, path) {
				return false
			}
		}
	}
	return true
}

func waitPathReady(workspace, path string) bool {
	abs, present, err := containedRel(workspace, path, true)
	if err != nil || !present {
		return false
	}
	if regularFile(abs) {
		return true
	}
	if !isDir(abs) {
		return false
	}
	man := filepath.Join(abs, treeManifestName)
	return regularFile(man)
}

func (s *sched) succeededThisResume(ident string) bool {
	if s.launched == nil || !s.launched[ident] {
		return false
	}
	for _, t := range s.doc.Tasks {
		if reservedIdentity(t) != ident {
			continue
		}
		st := s.tasks[ident]
		return st != nil && st.Status == StatusSucceeded
	}
	return false
}

func (s *sched) waitPathPendingRerun(path string) bool {
	if path == "" {
		return false
	}
	for _, t := range s.doc.Tasks {
		ident := reservedIdentity(t)
		if s.resume[ident].Decision != reuseRerun {
			continue
		}
		if !declaresOutputPath(t, path) {
			continue
		}
		if !s.succeededThisResume(ident) {
			return true
		}
	}
	return false
}

func declaresOutputPath(t TaskPlan, path string) bool {
	for _, f := range declaredIOFiles(t.Outputs) {
		if f.path == path {
			return true
		}
	}
	return false
}

func (s *sched) taskByID(id string) (TaskPlan, bool) {
	for _, t := range s.doc.Tasks {
		if t.ID == id {
			return t, true
		}
	}
	return TaskPlan{}, false
}

func (s *sched) launch(ident string, starts chan startEvent, reports chan report) {
	if s.resume != nil {
		s.beginResumeAttempt(ident)
	}
	st := s.tasks[ident]
	st.Status = StatusUnknown
	st.Reason = "unknown-backend"
	st.Started = time.Now().UTC().Format(time.RFC3339Nano)
	st.RuntimeID = ""
	task, _ := s.taskByIdent(ident)
	if st != nil {
		task.Attempt = st.Attempt
		task.Instance = st.Instance
		task.ShardIndex = st.ShardIndex
		task.ShardCount = st.ShardCount
	}
	if s.resume != nil {
		task.Replace = true
	}
	s.budget.occupy(task)
	s.notePersist(s.persistControl())
	ws := s.workspace
	ex := s.executor()
	go func() {
		s.runJob(ws, task, ex, starts, reports)
	}()
}

func (s *sched) runJob(workspace string, task TaskPlan, ex exec.Executor, starts chan startEvent, reports chan report) {
	applyReservedDefaults(&task)
	ident := reservedIdentity(task)
	rel := isolateRel(task)
	workRel := rel + "/work"
	isolate, _, err := containedRel(workspace, workRel, false)
	r := report{ID: ident, Stdout: rel + "/stdout", Stderr: rel + "/stderr"}
	if err != nil {
		r.Message = err.Error()
		reports <- r
		return
	}
	if err := os.MkdirAll(isolate, 0o755); err != nil {
		r.Message = err.Error()
		reports <- r
		return
	}
	if err := prepareIsolate(workspace, isolate, task); err != nil {
		r.Message = err.Error()
		reports <- r
		return
	}
	memBytes, _ := parseMemory(task.Resources.Memory)
	argv := append([]string(nil), executeArgv(task)...)
	job := exec.Job{
		Identity:    ident,
		Isolate:     isolate,
		Argv:        argv,
		Env:         copyStringMap(task.Env),
		Image:       task.Image,
		CPU:         task.Resources.CPU,
		Memory:      task.Resources.Memory,
		MemoryBytes: memBytes,
	}
	if task.Image == "" && len(argv) > 0 {
		resolved, err := exec.ResolveArgv0(argv[0], task.Env)
		if err != nil {
			r.Message = err.Error()
			reports <- r
			return
		}
		job.Argv[0] = resolved
		sum, err := sha256File(resolved)
		if err != nil {
			r.Message = err.Error()
			reports <- r
			return
		}
		r.ExecutablePath = resolved
		r.ExecutableSHA256 = sum
	}
	h, sub, err := boundedSubmit(workspace, ex, job)
	if err != nil {
		r.Message = err.Error()
		if errors.Is(err, errEngineBound) {
			r.Unknown = true
		}
		reports <- r
		return
	}
	if h.RuntimeID == "" || sub.RuntimeID == "" {
		r.Unknown = true
		r.Message = "empty runtime id"
		reports <- r
		return
	}
	r.RuntimeID = sub.RuntimeID
	r.ImageDigest = sub.ImageDigest
	starts <- startEvent{ident: ident, handle: h, sub: sub}
	for {
		pr, perr := boundedPoll(ex, h)
		if perr != nil {
			r.Unknown = true
			r.Message = perr.Error()
			reports <- r
			return
		}
		if pr.Running {
			time.Sleep(pollInterval)
			continue
		}
		r.Exit = pr.Exit
		r.Message = pr.Message
		r.Published = pr.Published
		if pr.RuntimeID != "" {
			r.RuntimeID = pr.RuntimeID
		}
		if pr.ImageDigest != "" {
			r.ImageDigest = pr.ImageDigest
		}
		if pr.Exit == 0 && pr.Message == "" && !pr.Published {
			if err := inspectOutputs(isolate, task); err != nil {
				r.Message = err.Error()
			} else {
				var pubErr error
				if task.Replace {
					pubErr = publishReplace(workspace, isolate, task)
				} else {
					pubErr = publishAll(workspace, isolate, task)
				}
				if pubErr != nil {
					r.Message = pubErr.Error()
				} else {
					r.Published = true
				}
			}
		}
		reports <- r
		return
	}
}

func (s *sched) beginResumeAttempt(ident string) {
	old := s.tasks[ident]
	task, _ := s.taskByIdent(ident)
	dec, hasDec := s.resume[ident]
	if old != nil && dec.Change == changeAdded && old.Status == StatusNotStarted {
		applyResumeDecision(old, dec, hasDec)
		if s.launched != nil {
			s.launched[ident] = true
		}
		return
	}
	st := initialTask(task)
	if old != nil {
		applyTaskStateDefaults(old)
		s.history = append(s.history, *old)
		st.Attempt = old.Attempt + 1
		applyResumeDecision(&st, dec, hasDec)
		if s.launched != nil {
			s.launched[ident] = true
		}
	} else if s.launched != nil {
		s.launched[ident] = true
		applyResumeDecision(&st, dec, hasDec)
	}
	s.tasks[ident] = &st
}

func applyResumeDecision(st *jsonTaskState, dec reuseDecision, ok bool) {
	if st == nil || !ok {
		return
	}
	st.Decision = dec.Decision
	st.ReuseReason = dec.Reason
	st.Differing = append([]string(nil), dec.Differing...)
	st.Change = dec.Change
}

func (s *sched) assignBlockedUpstream() {
	if s.resume == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, t := range s.doc.Tasks {
		ident := reservedIdentity(t)
		if s.resume[ident].Decision != reuseRerun || s.launched[ident] {
			continue
		}
		st := s.tasks[ident]
		if st == nil {
			continue
		}
		if s.waitProducerFailedThisResume(t.ID) {
			st.Decision = decisionBlockedUpstream
			st.ReuseReason = s.blockedReason(t.ID)
			st.Differing = nil
			st.Status = StatusBlocked
			if st.Ended == "" {
				st.Ended = now
			}
			continue
		}
		st.Status = StatusNotStarted
	}
}

func (s *sched) waitProducerFailedThisResume(id string) bool {
	for _, e := range s.doc.Edges {
		if e.ToTask != id {
			continue
		}
		if e.FromTask != "" {
			up := s.stateByTaskID(e.FromTask)
			if up == nil {
				continue
			}
			upIdent := reservedIdentity(taskPlanFromState(*up))
			if s.launched[upIdent] && up.Status != StatusSucceeded {
				return true
			}
			continue
		}
		for _, path := range e.Wait {
			if s.waitPathProducerFailedThisResume(path) {
				return true
			}
		}
	}
	return false
}

func (s *sched) waitPathProducerFailedThisResume(path string) bool {
	if path == "" {
		return false
	}
	for _, t := range s.doc.Tasks {
		if !declaresOutputPath(t, path) {
			continue
		}
		ident := reservedIdentity(t)
		st := s.tasks[ident]
		if s.launched[ident] && st != nil && st.Status != StatusSucceeded {
			return true
		}
	}
	return false
}

func (s *sched) blockedReason(id string) string {
	for _, e := range s.doc.Edges {
		if e.ToTask != id {
			continue
		}
		if e.FromTask != "" {
			up := s.stateByTaskID(e.FromTask)
			if up == nil {
				continue
			}
			upIdent := reservedIdentity(taskPlanFromState(*up))
			if s.launched[upIdent] && up.Status != StatusSucceeded {
				return reasonDownstreamOfRerun
			}
			if up.Decision == decisionBlockedUpstream || up.Status != StatusSucceeded {
				return reasonPreviousUnsuccessful
			}
			continue
		}
		for _, path := range e.Wait {
			if s.waitPathProducerFailedThisResume(path) {
				return reasonDownstreamOfRerun
			}
		}
	}
	return reasonDownstreamOfRerun
}

func (s *sched) apply(r report) {
	st := s.tasks[r.ID]
	if st == nil {
		return
	}
	if task, ok := s.taskByIdent(r.ID); ok {
		s.budget.release(task)
		st.Script = task.Script
		st.EnvDigest = envDigest(task.Env)
	}
	st.Stdout = r.Stdout
	st.Stderr = r.Stderr
	if r.RuntimeID != "" {
		st.RuntimeID = r.RuntimeID
	}
	if r.ImageDigest != "" {
		st.ImageDigest = r.ImageDigest
	}
	if r.ExecutablePath != "" {
		st.ExecutablePath = r.ExecutablePath
	}
	if r.ExecutableSHA256 != "" {
		st.ExecutableSHA256 = r.ExecutableSHA256
	}
	st.Ended = time.Now().UTC().Format(time.RFC3339Nano)
	if r.Unknown {
		st.Status = StatusUnknown
		st.Reason = "unknown-backend"
		st.Error = &jsonTaskErr{Unit: r.ID, Message: r.Message}
		s.syncUnknown()
		s.notePersist(s.persistControl())
		return
	}
	if r.Published && r.Exit == 0 && r.Message == "" {
		st.Status = StatusSucceeded
		st.Reason = "ready"
		st.Error = nil
		if task, ok := s.taskByIdent(r.ID); ok {
			s.notePersist(s.recordSuccess(st, task))
		}
	} else {
		st.Status = StatusFailed
		st.Reason = "ready"
		msg := r.Message
		if msg == "" {
			msg = "exit " + strconv.Itoa(r.Exit)
		}
		unit := r.ID
		if task, ok := s.taskByIdent(r.ID); ok {
			unit = task.ID
		}
		st.Error = &jsonTaskErr{Unit: unit, Message: msg}
		s.blockFrom(r.ID)
	}
	s.notePersist(s.persistControl())
}

func (s *sched) recordSuccess(st *jsonTaskState, task TaskPlan) error {
	inputs, err := inputRecords(s.workspace, task.Inputs)
	if err != nil {
		return err
	}
	outputs, err := destRecords(s.workspace, task.Outputs)
	if err != nil {
		return err
	}
	st.Fingerprints = inputs
	st.Checksums = outputs
	st.Lineage = successLineage(s, task, inputs, outputs)
	return nil
}

func (s *sched) blockFrom(ident string) {
	src := s.tasks[ident]
	why := "upstream failed"
	if src != nil && src.Status == StatusBlocked {
		why = "upstream blocked"
	}
	fromID := ident
	if task, ok := s.taskByIdent(ident); ok {
		fromID = task.ID
	}
	for _, e := range s.doc.Edges {
		if e.FromTask != fromID {
			continue
		}
		to, ok := s.taskByID(e.ToTask)
		if !ok {
			continue
		}
		depIdent := reservedIdentity(to)
		dep := s.tasks[depIdent]
		if dep == nil || dep.Status != StatusNotStarted {
			continue
		}
		dep.Status = StatusBlocked
		dep.Reason = why
		dep.Ended = time.Now().UTC().Format(time.RFC3339Nano)
		s.blockFrom(depIdent)
	}
}

func (s *sched) finish() {
	s.syncUnknown()
	if len(s.unknownUnits()) > 0 {
		s.run.Status = StatusUnknown
		s.notePersist(s.persistControl())
		return
	}
	s.run.Ended = time.Now().UTC().Format(time.RFC3339Nano)
	s.run.Status = StatusSucceeded
	for _, t := range s.doc.Tasks {
		st := s.tasks[reservedIdentity(t)]
		if st == nil || st.Status == StatusSucceeded {
			continue
		}
		s.run.Status = StatusFailed
		break
	}
	s.notePersist(s.persistControl())
}

func (s *sched) notePersist(err error) {
	if err != nil && s.persist == nil {
		s.persist = err
	}
}

func (s *sched) failures() []Defect {
	var out []Defect
	for _, t := range s.doc.Tasks {
		st := s.tasks[reservedIdentity(t)]
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
		case StatusUnknown:
			out = append(out, Defect{
				Code:    DefectUnknownBackend,
				Unit:    reservedIdentity(t),
				Message: "unknown backend",
			})
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

func (s *sched) persistControl() error {
	s.snapshot = newOccupancyID()
	s.run.Snapshot = s.snapshot
	s.run.SchemaVersion = SchemaVersion
	if err := s.writePlan(); err != nil {
		return err
	}
	if err := s.writeTasks(); err != nil {
		return err
	}
	return s.writeRun()
}

func (s *sched) writeControl() error {
	s.run.Snapshot = s.snapshot
	s.run.SchemaVersion = SchemaVersion
	if err := s.writePlan(); err != nil {
		return err
	}
	if err := s.writeTasks(); err != nil {
		return err
	}
	return s.writeRun()
}

func (s *sched) writePlan() error {
	plan, err := marshalControlPlan(s.doc, s.snapshot)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.workspace, ControlDir, PlanFile), plan)
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
		Snapshot:      s.snapshot,
		Tasks:         make([]jsonTaskState, 0, len(s.history)+len(s.doc.Tasks)),
	}
	doc.Tasks = append(doc.Tasks, s.history...)
	for _, t := range s.doc.Tasks {
		if st := s.tasks[reservedIdentity(t)]; st != nil {
			doc.Tasks = append(doc.Tasks, *st)
		}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.workspace, ControlDir, TasksFile), append(data, '\n'))
}

func (s *sched) syncUnknown() {
	units := s.unknownUnits()
	if s.run.Occupancy == nil {
		return
	}
	s.run.Occupancy.Unknown = units
}

func (s *sched) unknownUnits() []string {
	var units []string
	seen := map[string]bool{}
	for _, t := range s.doc.Tasks {
		ident := reservedIdentity(t)
		st := s.tasks[ident]
		if st != nil && st.Status == StatusUnknown && !seen[ident] {
			units = append(units, ident)
			seen[ident] = true
		}
	}
	sort.Strings(units)
	return units
}

func (s *sched) cancelDefects() []Defect {
	return []Defect{{
		Code:    DefectCanceled,
		Message: "canceled",
	}}
}

func boundCallErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errEngineBound
	}
	return err
}

func boundCallErrAfter(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return errEngineBound
	}
	return boundCallErr(err)
}

func boundedSubmit(workspace string, ex exec.Executor, job exec.Job) (exec.Handle, exec.Report, error) {
	ls := registerLateSubmit(workspace, job.Identity)
	ctx, cancel := context.WithTimeout(context.Background(), currentBound())
	defer cancel()
	h, r, err := ex.Submit(ctx, job)
	finishLateSubmit(ls, h, r, err)
	return h, r, boundCallErrAfter(ctx, err)
}

func boundedPoll(ex exec.Executor, h exec.Handle) (exec.Report, error) {
	ctx, cancel := context.WithTimeout(context.Background(), currentBound())
	defer cancel()
	r, err := ex.Poll(ctx, h)
	return r, boundCallErrAfter(ctx, err)
}

func boundedCancel(ex exec.Executor, h exec.Handle) error {
	ctx, cancel := context.WithTimeout(context.Background(), currentBound())
	defer cancel()
	return boundCallErrAfter(ctx, ex.Cancel(ctx, h))
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
