package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	intpath "github.com/HahyeonJeon/gobble/internal/path"
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
	StatusSkipped    = "skipped"
)

const (
	conditionMissingFile = "missing-file"
	conditionFalseParam  = "false-param"
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
	whenDown  map[string]bool
	launched  map[string]bool
	persist   error
	escape    *Defect
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
	Scatter          string         `json:"scatter,omitempty"`
	Gather           string         `json:"gather,omitempty"`
	When             string         `json:"when,omitempty"`
	Condition        string         `json:"condition,omitempty"`
	Expansion        *jsonExpansion `json:"expansion,omitempty"`
}

type jsonExpansion struct {
	Producer string   `json:"producer,omitempty"`
	Members  []string `json:"members"`
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
		Scatter:   t.Scatter,
		Gather:    t.Gather,
		When:      t.When,
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
	InvalidPath      bool
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
		s.freshenWhenBranchAfterReap()
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
		for !canceled && s.persist == nil && s.escape == nil && running < n {
			id, d := s.nextReady()
			if d != nil {
				s.escape = d
				break
			}
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
	if s.escape != nil {
		return []Defect{*s.escape}
	}
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

func (s *sched) nextReady() (string, *Defect) {
	if d := s.maybeSkip(); d != nil {
		return "", d
	}
	if d := s.maybeExpand(); d != nil {
		return "", d
	}
	for _, ident := range s.readyCandidates() {
		st := s.tasks[ident]
		if st == nil {
			continue
		}
		if isScatterTemplateState(st) {
			continue
		}
		if st.Status == StatusSkipped {
			continue
		}
		if s.resume != nil {
			if !s.resumeAdmit(ident, st) {
				continue
			}
		} else if st.Status != StatusNotStarted {
			continue
		}
		task, ok := s.taskByIdent(ident)
		if !ok {
			continue
		}
		ready, d := s.upstreamReady(task, ident)
		if d != nil {
			return "", d
		}
		if !ready {
			continue
		}
		if !s.budget.fits(task) {
			continue
		}
		return ident, nil
	}
	return "", nil
}

func (s *sched) resumeAdmit(ident string, st *jsonTaskState) bool {
	if s.launched[ident] {
		return false
	}
	if st != nil && (st.Status == StatusRunning || st.Status == StatusUnknown || st.Status == StatusSkipped) {
		return false
	}
	if dec, ok := s.resume[ident]; ok {
		return dec.Decision == reuseRerun
	}
	if st == nil || st.Instance == "" {
		return false
	}
	parent, ok := s.taskByID(st.ID)
	if !ok {
		return false
	}
	parentIdent := reservedIdentity(parent)
	return s.resume[parentIdent].Decision == reuseRerun
}

func (s *sched) readyCandidates() []string {
	seen := make(map[string]bool, len(s.tasks))
	var out []string
	for _, t := range s.doc.Tasks {
		ident := reservedIdentity(t)
		if s.tasks[ident] != nil {
			out = append(out, ident)
			seen[ident] = true
		}
		var members []string
		for ident, st := range s.tasks {
			if st == nil || st.ID != t.ID || st.Instance == "" || seen[ident] {
				continue
			}
			members = append(members, ident)
		}
		sort.Strings(members)
		out = append(out, members...)
		for _, ident := range members {
			seen[ident] = true
		}
	}
	var extra []string
	for ident := range s.tasks {
		if !seen[ident] {
			extra = append(extra, ident)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
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
			out := cloneTaskPlan(t)
			s.specializeGatherIO(&out)
			return out, true
		}
	}
	st := s.tasks[ident]
	if st == nil || st.Instance == "" {
		return TaskPlan{}, false
	}
	t, ok := s.taskByID(st.ID)
	if !ok {
		return TaskPlan{}, false
	}
	member := cloneTaskPlan(t)
	member.Instance = st.Instance
	member.ShardIndex = st.ShardIndex
	member.ShardCount = st.ShardCount
	member.Attempt = st.Attempt
	s.specializeMemberIO(&member, st.Instance)
	return member, true
}

func (s *sched) upstreamReady(task TaskPlan, ident string) (bool, *Defect) {
	id := task.ID
	inst := ""
	if st := s.tasks[ident]; st != nil {
		inst = st.Instance
	}
	for _, e := range s.doc.Edges {
		if e.ToTask != id {
			continue
		}
		if e.FromTask != "" {
			if s.isScatterTaskID(e.FromTask) {
				if inst != "" && s.sameScatterID(id, e.FromTask) {
					if !s.scatterMemberReady(e.FromTask, inst) {
						return false, nil
					}
					continue
				}
				ready, d := s.scatterMembersReady(e.FromTask)
				if d != nil || !ready {
					return ready, d
				}
				continue
			}
			up := s.stateByTaskID(e.FromTask)
			if up == nil || up.Status != StatusSucceeded {
				return false, nil
			}
			if s.resume != nil {
				upTask, ok := s.taskByID(e.FromTask)
				if !ok {
					return false, nil
				}
				ident := reservedIdentity(upTask)
				if s.resume[ident].Decision == reuseRerun && !s.succeededThisResume(ident) {
					return false, nil
				}
			}
		} else if s.resume != nil {
			for _, path := range e.Wait {
				if s.waitPathPendingRerun(path) {
					return false, nil
				}
			}
		}
		for _, path := range e.Wait {
			if path == "" {
				continue
			}
			ready, d := waitPathReady(s.workspace, path)
			if d != nil {
				if d.Unit == "" {
					d.Unit = id
				}
				return false, d
			}
			if !ready {
				return false, nil
			}
		}
	}
	return true, nil
}

func waitPathReady(workspace, path string) (bool, *Defect) {
	abs, present, err := containedRel(workspace, path, false)
	if err != nil {
		d := escapeDefect("", path)
		return false, &d
	}
	if !present {
		return false, nil
	}
	if regularFile(abs) {
		return true, nil
	}
	if !isDir(abs) {
		return false, nil
	}
	man := filepath.Join(abs, treeManifestName)
	return regularFile(man), nil
}

func (s *sched) succeededThisResume(ident string) bool {
	if s.launched == nil || !s.launched[ident] {
		return false
	}
	st := s.tasks[ident]
	return st != nil && st.Status == StatusSucceeded
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
		r.Message = "path escapes directory"
		r.InvalidPath = true
		reports <- r
		return
	}
	if err := os.MkdirAll(isolate, 0o755); err != nil {
		r.Message = err.Error()
		reports <- r
		return
	}
	for _, logRel := range []string{rel + "/stdout", rel + "/stderr"} {
		if _, _, err := containedRel(workspace, logRel, false); err != nil {
			r.Message = "path escapes directory"
			r.InvalidPath = true
			reports <- r
			return
		}
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
		if errors.Is(err, exec.ErrEscapedPath) {
			r.Message = "path escapes directory"
			r.InvalidPath = true
		}
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
			r.Message = perr.Error()
			if errors.Is(perr, exec.ErrEscapedPath) {
				r.Message = "path escapes directory"
				r.InvalidPath = true
			} else {
				r.Unknown = true
			}
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
	if old != nil && old.Status == StatusNotStarted && (dec.Change == changeAdded || dec.Decision == reuseRerun) {
		if prev := s.latestHistoryAttempt(ident); prev >= old.Attempt {
			old.Attempt = prev + 1
		}
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
	for _, ident := range s.readyCandidates() {
		st := s.tasks[ident]
		if st == nil || s.launched[ident] {
			continue
		}
		task, ok := s.taskByIdent(ident)
		if !ok || isScatterTemplate(task) {
			continue
		}
		if !s.resumeSkipAdmit(ident, st) {
			continue
		}
		if st.Status == StatusSkipped || st.Status == StatusSucceeded {
			continue
		}
		if s.cascadeSkip(task) || s.scatterParentSkipped(st) {
			st.Status = StatusSkipped
			st.Reason = "skipped"
			if st.Ended == "" {
				st.Ended = now
			}
			continue
		}
		if s.waitProducerFailedThisResume(task.ID) {
			st.Decision = decisionBlockedUpstream
			st.ReuseReason = s.blockedReason(task.ID)
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
	if r.InvalidPath {
		st.Status = StatusFailed
		st.Reason = "path escapes directory"
		unit := r.ID
		if task, ok := s.taskByIdent(r.ID); ok {
			unit = task.ID
		}
		st.Error = &jsonTaskErr{Unit: unit, Message: "path escapes directory"}
		s.blockFrom(r.ID)
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
	for ident, st := range s.tasks {
		if st == nil || isNonFailureState(st) {
			continue
		}
		_ = ident
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
	seen := make(map[string]bool)
	for _, t := range s.doc.Tasks {
		ident := reservedIdentity(t)
		st := s.tasks[ident]
		if isScatterTemplate(t) {
			seen[ident] = true
			continue
		}
		if st == nil {
			out = append(out, Defect{
				Code:    DefectFailed,
				Unit:    t.ID,
				Message: "task failed",
			})
			continue
		}
		seen[ident] = true
		out = append(out, s.failureOf(ident, t.ID, st)...)
	}
	var extra []string
	for ident, st := range s.tasks {
		if seen[ident] || st == nil || isScatterTemplateState(st) {
			continue
		}
		extra = append(extra, ident)
	}
	sort.Strings(extra)
	for _, ident := range extra {
		st := s.tasks[ident]
		out = append(out, s.failureOf(ident, st.ID, st)...)
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
		Tasks:         make([]jsonTaskState, 0, len(s.history)+len(s.tasks)),
	}
	doc.Tasks = append(doc.Tasks, s.history...)
	seen := make(map[string]bool, len(s.tasks))
	for _, t := range s.doc.Tasks {
		ident := reservedIdentity(t)
		if st := s.tasks[ident]; st != nil {
			doc.Tasks = append(doc.Tasks, *st)
			seen[ident] = true
		}
	}
	var extra []string
	for ident := range s.tasks {
		if !seen[ident] {
			extra = append(extra, ident)
		}
	}
	sort.Strings(extra)
	for _, ident := range extra {
		if st := s.tasks[ident]; st != nil {
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
	for ident, st := range s.tasks {
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

func isScatterTemplate(t TaskPlan) bool {
	return t.Scatter != "" && t.Instance == ""
}

func isScatterTemplateState(st *jsonTaskState) bool {
	return st != nil && st.Scatter != "" && st.Instance == ""
}

func isNonFailureState(st *jsonTaskState) bool {
	if st == nil {
		return false
	}
	switch st.Status {
	case StatusSucceeded, StatusSkipped, StatusBlocked:
		return true
	case StatusNotStarted:
		return isScatterTemplateState(st)
	default:
		return isScatterTemplateState(st)
	}
}

func isKnownTerminal(status string) bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusSkipped, StatusBlocked, StatusIncomplete:
		return true
	default:
		return false
	}
}

func (s *sched) failureOf(ident, unit string, st *jsonTaskState) []Defect {
	if st == nil {
		return []Defect{{Code: DefectFailed, Unit: unit, Message: "task failed"}}
	}
	switch st.Status {
	case StatusSucceeded, StatusBlocked, StatusSkipped:
		return nil
	case StatusUnknown:
		return []Defect{{
			Code:    DefectUnknownBackend,
			Unit:    ident,
			Message: "unknown backend",
		}}
	case StatusNotStarted:
		if isScatterTemplateState(st) {
			return nil
		}
		return []Defect{{
			Code:    DefectFailed,
			Unit:    unit,
			Message: "not started",
		}}
	default:
		msg := "task failed"
		if st.Error != nil && st.Error.Message != "" {
			msg = st.Error.Message
		}
		code := DefectFailed
		if st.Reason == "path escapes directory" || msg == "path escapes directory" {
			code = DefectInvalidPath
		}
		if st.Reason == "never-ready" || msg == "never-ready" {
			code = DefectNeverReady
		}
		return []Defect{{
			Code:    code,
			Unit:    unit,
			Message: msg,
		}}
	}
}

func (s *sched) isScatterTaskID(id string) bool {
	t, ok := s.taskByID(id)
	return ok && isScatterTemplate(t)
}

func (s *sched) scatterTemplateState(id string) *jsonTaskState {
	t, ok := s.taskByID(id)
	if !ok {
		return nil
	}
	return s.tasks[reservedIdentity(t)]
}

func (s *sched) sameScatterID(a, b string) bool {
	ta, oka := s.taskByID(a)
	tb, okb := s.taskByID(b)
	return oka && okb && sameScatter(ta, tb)
}

func sameScatter(a, b TaskPlan) bool {
	return a.Scatter != "" && a.Scatter == b.Scatter &&
		a.ScatterFromTask == b.ScatterFromTask &&
		a.ScatterFromPort == b.ScatterFromPort
}

func (s *sched) scatterMemberReady(id, key string) bool {
	t, ok := s.taskByID(id)
	if !ok {
		return false
	}
	t.Instance = key
	t.ShardIndex = DefaultShardIndex
	applyReservedDefaults(&t)
	ms := s.tasks[reservedIdentity(t)]
	return ms != nil && ms.Status == StatusSucceeded
}

func (s *sched) scatterMembersReady(id string) (bool, *Defect) {
	st := s.scatterTemplateState(id)
	if st == nil || st.Expansion == nil {
		return false, nil
	}
	if len(st.Expansion.Members) == 0 {
		d := Defect{Code: DefectNeverReady, Unit: id, Message: "never-ready"}
		return false, &d
	}
	allSucceeded := true
	for _, key := range st.Expansion.Members {
		member, ok := s.taskByID(id)
		if !ok {
			return false, nil
		}
		member.Instance = key
		member.ShardIndex = DefaultShardIndex
		applyReservedDefaults(&member)
		ident := reservedIdentity(member)
		ms := s.tasks[ident]
		if ms == nil {
			return false, nil
		}
		if ms.Status == StatusRunning || ms.Status == StatusUnknown || ms.Status == StatusNotStarted {
			return false, nil
		}
		if !isKnownTerminal(ms.Status) {
			return false, nil
		}
		if ms.Status != StatusSucceeded {
			allSucceeded = false
		}
	}
	if !allSucceeded {
		return false, nil
	}
	return true, nil
}

func (s *sched) maybeExpand() *Defect {
	for _, t := range s.doc.Tasks {
		if !isScatterTemplate(t) {
			continue
		}
		st := s.tasks[reservedIdentity(t)]
		if st == nil || st.Status == StatusSkipped {
			continue
		}
		if st.Expansion != nil {
			if d := s.seedMembers(t, st.Expansion.Members); d != nil {
				return d
			}
			continue
		}
		keys, producer, ready, d := s.expansionKeys(t)
		if d != nil {
			return d
		}
		if !ready {
			continue
		}
		st.Expansion = &jsonExpansion{Producer: producer, Members: keys}
		if st.Expansion.Members == nil {
			st.Expansion.Members = []string{}
		}
		s.notePersist(s.persistControl())
		if d := s.seedMembers(t, st.Expansion.Members); d != nil {
			return d
		}
	}
	return nil
}

func (s *sched) seedMembers(t TaskPlan, keys []string) *Defect {
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if seen[key] {
			return &Defect{Code: DefectInvalidValue, Unit: t.ID, Message: "invalid-value"}
		}
		seen[key] = true
		member := cloneTaskPlan(t)
		member.Instance = key
		member.ShardIndex = DefaultShardIndex
		applyReservedDefaults(&member)
		ident := reservedIdentity(member)
		if _, _, err := containedRel(s.workspace, isolateRel(member), false); err != nil {
			d := escapeDefect(t.ID, key)
			return &d
		}
		if s.tasks[ident] != nil {
			continue
		}
		st := initialTask(member)
		if s.resume != nil {
			parentIdent := reservedIdentity(t)
			if pdec, ok := s.resume[parentIdent]; ok && pdec.Decision == reuseRerun {
				dec := reuseDecision{
					Identity:  ident,
					Decision:  reuseRerun,
					Change:    pdec.Change,
					Reason:    pdec.Reason,
					Differing: append([]string(nil), pdec.Differing...),
				}
				s.resume[ident] = dec
				applyResumeDecision(&st, dec, true)
			}
		}
		s.tasks[ident] = &st
	}
	return nil
}

func (s *sched) expansionKeys(t TaskPlan) ([]string, string, bool, *Defect) {
	producer := t.ScatterFromTask
	if producer == "" {
		producer = t.ScatterFromPort
	}
	if t.ScatterFromTask != "" {
		up := s.stateByTaskID(t.ScatterFromTask)
		if up == nil || up.Status != StatusSucceeded {
			return nil, producer, false, nil
		}
		if s.resume != nil {
			upTask, ok := s.taskByID(t.ScatterFromTask)
			if !ok {
				return nil, producer, false, nil
			}
			ident := reservedIdentity(upTask)
			if s.resume[ident].Decision == reuseRerun && !s.succeededThisResume(ident) {
				return nil, producer, false, nil
			}
		}
		keys, d := s.producerMemberKeys(t)
		return keys, producer, d == nil, d
	}
	if t.ScatterFromKind == ArtifactTree {
		keys, d := s.staticTreeKeys(t)
		return keys, producer, d == nil, d
	}
	if t.ScatterMembers != nil {
		return append([]string(nil), t.ScatterMembers...), producer, true, nil
	}
	return []string{}, producer, true, nil
}

func (s *sched) producerMemberKeys(t TaskPlan) ([]string, *Defect) {
	src, ok := s.taskByID(t.ScatterFromTask)
	if !ok {
		return nil, &Defect{Code: DefectMissingInput, Unit: t.ID, Message: "missing input"}
	}
	io, ok := findProducerIO(src, t.ScatterFromPort)
	if !ok {
		return nil, &Defect{Code: DefectMissingInput, Unit: t.ID, Message: "missing input"}
	}
	switch t.ScatterFromKind {
	case ArtifactGroup:
		if io.Members == nil {
			return []string{}, nil
		}
		keys := make([]string, 0, len(io.Members))
		for _, m := range io.Members {
			keys = append(keys, m.Name)
		}
		return keys, nil
	case ArtifactTree:
		return treeMemberKeys(s.workspace, io, t.ID)
	default:
		path := io.Path
		if path == "" {
			return []string{}, nil
		}
		return []string{path}, nil
	}
}

func (s *sched) staticTreeKeys(t TaskPlan) ([]string, *Defect) {
	if t.ScatterFromPath != "" {
		io := IO{Kind: ArtifactTree, Path: t.ScatterFromPath, Source: t.ScatterFromPath}
		return treeMemberKeys(s.workspace, io, t.ID)
	}
	var io IO
	found := false
	for _, in := range t.Inputs {
		if s.ioFromScatterProducer(t, in.Name) || in.Name == t.ScatterFromPort {
			io = in
			found = true
			break
		}
	}
	if !found {
		return []string{}, nil
	}
	return treeMemberKeys(s.workspace, io, t.ID)
}

func treeMemberKeys(workspace string, io IO, unit string) ([]string, *Defect) {
	files := treeSourceMemberPaths(workspace, io)
	keys := make([]string, 0, len(files))
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		if seen[f.name] {
			return nil, &Defect{Code: DefectInvalidValue, Unit: unit, Message: "invalid-value"}
		}
		seen[f.name] = true
		keys = append(keys, f.name)
	}
	return keys, nil
}

func findProducerIO(t TaskPlan, port string) (IO, bool) {
	for _, out := range t.Outputs {
		if out.Name == port {
			return out, true
		}
	}
	for _, in := range t.Inputs {
		if in.Name == port {
			return in, true
		}
	}
	return IO{}, false
}

func (s *sched) specializeGatherIO(t *TaskPlan) {
	if t.Gather == "" {
		return
	}
	for i := range t.Inputs {
		fromTask, fromPort := s.edgeFrom(t.ID, t.Inputs[i].Name)
		if fromTask == "" || !s.isScatterTaskID(fromTask) {
			continue
		}
		st := s.scatterTemplateState(fromTask)
		if st == nil || st.Expansion == nil {
			continue
		}
		src, ok := s.taskByID(fromTask)
		if !ok {
			continue
		}
		members := make([]IOMember, 0, len(st.Expansion.Members))
		for _, key := range st.Expansion.Members {
			member := cloneTaskPlan(src)
			member.Instance = key
			member.ShardIndex = DefaultShardIndex
			applyReservedDefaults(&member)
			s.specializeMemberIO(&member, key)
			io, ok := findProducerIO(member, fromPort)
			if !ok || io.Path == "" {
				continue
			}
			members = append(members, IOMember{Name: key, Path: io.Path, Source: io.Path, Spec: io.Spec})
		}
		if len(members) == 0 {
			continue
		}
		t.Inputs[i].Kind = ArtifactGroup
		t.Inputs[i].Path = ""
		t.Inputs[i].Source = ""
		t.Inputs[i].Members = members
		t.Inputs[i].Manifest = ""
	}
}

func (s *sched) edgeFrom(toTask, toPort string) (string, string) {
	for _, e := range s.doc.Edges {
		if e.ToTask == toTask && e.ToPort == toPort {
			return e.FromTask, e.FromPort
		}
	}
	return "", ""
}

func (s *sched) specializeMemberIO(t *TaskPlan, key string) {
	s.specializeMemberChain(t, key, make(map[string]bool))
}

func (s *sched) specializeMemberChain(t *TaskPlan, key string, seen map[string]bool) {
	if seen[t.ID] {
		return
	}
	seen[t.ID] = true
	defer delete(seen, t.ID)
	memberPath, memberSpec := s.memberSource(t, key)
	if memberPath != "" || !isZeroPath(memberSpec) {
		for i := range t.Inputs {
			if s.ioFromScatterProducer(*t, t.Inputs[i].Name) {
				s.applyMemberIO(&t.Inputs[i], memberPath, memberSpec, true)
			}
		}
		for i := range t.Outputs {
			if s.ioFromScatterProducer(*t, t.Outputs[i].Name) {
				s.applyMemberIO(&t.Outputs[i], memberPath, memberSpec, false)
			}
		}
	}
	s.specializeSiblingChain(t, key, true, seen)
	s.specializeSiblingChain(t, key, false, seen)
}

func (s *sched) specializeSiblingChain(t *TaskPlan, key string, inputs bool, seen map[string]bool) {
	ports := t.Outputs
	if inputs {
		ports = t.Inputs
	}
	for i := range ports {
		fromTask, fromPort := s.edgeFrom(t.ID, ports[i].Name)
		if fromTask == "" || fromTask == t.ID {
			continue
		}
		src, ok := s.taskByID(fromTask)
		if !ok || !sameScatter(*t, src) {
			continue
		}
		sibling := cloneTaskPlan(src)
		sibling.Instance = key
		sibling.ShardIndex = DefaultShardIndex
		applyReservedDefaults(&sibling)
		s.specializeMemberChain(&sibling, key, seen)
		io, ok := findProducerIO(sibling, fromPort)
		if !ok || io.Path == "" {
			continue
		}
		path := io.Path
		spec := io.Spec
		if inputs {
			s.applyMemberIO(&t.Inputs[i], path, spec, true)
			continue
		}
		s.applyMemberIO(&t.Outputs[i], path, spec, false)
	}
}

func (s *sched) ioFromScatterProducer(t TaskPlan, port string) bool {
	if t.Scatter == "" {
		return false
	}
	for _, e := range s.doc.Edges {
		if e.ToTask != t.ID || e.ToPort != port {
			continue
		}
		if e.FromTask == t.ScatterFromTask && e.FromPort == t.ScatterFromPort {
			return true
		}
	}
	return false
}

func (s *sched) memberSource(t *TaskPlan, key string) (string, Path) {
	if t.ScatterFromTask != "" {
		src, ok := s.taskByID(t.ScatterFromTask)
		if !ok {
			return key, literalPath(key)
		}
		io, ok := findProducerIO(src, t.ScatterFromPort)
		if !ok {
			return key, literalPath(key)
		}
		switch t.ScatterFromKind {
		case ArtifactGroup:
			for _, m := range io.Members {
				if m.Name == key {
					path := m.Path
					if m.Source != "" {
						path = m.Source
					}
					return path, m.Spec
				}
			}
		case ArtifactTree:
			dir := treeSourceDir(io)
			path := key
			if dir != "" {
				path = strings.TrimSuffix(strings.ReplaceAll(dir, `\`, "/"), "/") + "/" + key
			}
			return path, literalPath(path)
		default:
			path := io.Path
			if path == "" {
				path = key
			}
			return path, io.Spec
		}
		return key, literalPath(key)
	}
	for i, name := range t.ScatterMembers {
		if name != key {
			continue
		}
		path := key
		if i < len(t.ScatterMemberPaths) && t.ScatterMemberPaths[i] != "" {
			path = t.ScatterMemberPaths[i]
		}
		return path, literalPath(path)
	}
	if t.ScatterFromKind == ArtifactFile && len(t.ScatterMembers) == 1 {
		return t.ScatterMembers[0], literalPath(t.ScatterMembers[0])
	}
	for _, in := range t.Inputs {
		if !s.ioFromScatterProducer(*t, in.Name) {
			continue
		}
		if in.Members != nil {
			for _, m := range in.Members {
				if m.Name == key {
					path := m.Path
					if m.Source != "" {
						path = m.Source
					}
					return path, m.Spec
				}
			}
		}
		if t.ScatterFromKind == ArtifactTree {
			dir := treeSourceDir(in)
			if dir == "" {
				dir = t.ScatterFromPath
			}
			path := key
			if dir != "" {
				path = strings.TrimSuffix(strings.ReplaceAll(dir, `\`, "/"), "/") + "/" + key
			}
			return path, literalPath(path)
		}
	}
	if t.ScatterFromKind == ArtifactTree && t.ScatterFromPath != "" {
		dir := strings.TrimSuffix(strings.ReplaceAll(t.ScatterFromPath, `\`, "/"), "/")
		path := key
		if dir != "" {
			path = dir + "/" + key
		}
		return path, literalPath(path)
	}
	return key, literalPath(key)
}

func (s *sched) applyMemberIO(io *IO, memberPath string, memberSpec Path, asInput bool) {
	from := memberSpec
	if isZeroPath(from) {
		from = literalPath(memberPath)
	}
	classified := pathFromSpec(intpath.Classify(io.Spec.spec(), from.spec(), intpath.DeriveAppend))
	path, d := classified.Render()
	if d != nil || path == "" {
		path = memberPath
		classified = from
	}
	io.Kind = ArtifactFile
	io.Members = nil
	io.Manifest = ""
	io.Spec = classified
	if asInput {
		io.Source = memberPath
		io.Path = path
		if io.Path == io.Source {
			io.Source = ""
		}
		return
	}
	io.Path = path
	io.Source = ""
}

func literalPath(path string) Path {
	return Path{Literal: true, Opaque: path}
}

func (s *sched) latestHistoryAttempt(ident string) int {
	best := 0
	for _, st := range s.history {
		h := reservedIdentity(taskPlanFromState(st))
		if h == ident && st.Attempt > best {
			best = st.Attempt
		}
	}
	return best
}

func (s *sched) maybeSkip() *Defect {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, ident := range s.readyCandidates() {
		st := s.tasks[ident]
		if st == nil || st.Status != StatusNotStarted {
			continue
		}
		if s.resume != nil {
			if s.launched[ident] {
				continue
			}
			if !s.resumeSkipAdmit(ident, st) {
				continue
			}
		}
		task, ok := s.taskByIdent(ident)
		if !ok {
			continue
		}
		if s.cascadeSkip(task) || s.scatterParentSkipped(st) {
			st.Status = StatusSkipped
			st.Reason = "skipped"
			if st.Ended == "" {
				st.Ended = now
			}
			s.notePersist(s.persistControl())
			continue
		}
		if task.When == "" {
			continue
		}
		skip, reason, ready, d := s.evalWhen(task)
		if d != nil {
			return d
		}
		if !ready || !skip {
			continue
		}
		st.Status = StatusSkipped
		st.Reason = "skipped"
		st.Condition = reason
		if st.Ended == "" {
			st.Ended = now
		}
		s.notePersist(s.persistControl())
	}
	return nil
}

func (s *sched) resumeSkipAdmit(ident string, st *jsonTaskState) bool {
	if dec, ok := s.resume[ident]; ok {
		return dec.Decision == reuseRerun
	}
	if st == nil || st.Instance == "" {
		return false
	}
	parent, ok := s.taskByID(st.ID)
	if !ok {
		return false
	}
	return s.resume[reservedIdentity(parent)].Decision == reuseRerun
}

func (s *sched) scatterParentSkipped(st *jsonTaskState) bool {
	if st == nil || st.Instance == "" {
		return false
	}
	t, ok := s.taskByID(st.ID)
	if !ok {
		return false
	}
	up := s.tasks[reservedIdentity(t)]
	return up != nil && up.Status == StatusSkipped
}

func (s *sched) onWhenSkipBranch(t TaskPlan) bool {
	if t.When != "" {
		return true
	}
	return s.whenDown[t.ID]
}

func (s *sched) freshenWhenBranchAfterReap() {
	if s.resume == nil {
		return
	}
	var idents []string
	for ident, st := range s.tasks {
		if st == nil {
			continue
		}
		switch st.Status {
		case StatusSucceeded, StatusSkipped, StatusNotStarted:
			continue
		case StatusRunning:
			if _, ok := backendHandle(s.workspace, ident, st); ok {
				continue
			}
		}
		task, ok := s.taskByIdent(ident)
		if !ok || isScatterTemplate(task) {
			continue
		}
		if !s.onWhenSkipBranch(task) {
			continue
		}
		idents = append(idents, ident)
	}
	sort.Strings(idents)
	for _, ident := range idents {
		st := s.tasks[ident]
		task, ok := s.taskByIdent(ident)
		if !ok || st == nil {
			continue
		}
		dec, hasDec := s.resume[ident]
		if !hasDec {
			dec = reuseDecision{Identity: ident, Decision: reuseRerun, Reason: reasonPreviousUnsuccessful}
			if p, pok := s.taskByID(st.ID); pok {
				pdec := s.resume[reservedIdentity(p)]
				if pdec.Decision == reuseRerun {
					dec.Change = pdec.Change
					dec.Reason = pdec.Reason
					dec.Differing = append([]string(nil), pdec.Differing...)
				}
			}
			hasDec = true
		}
		s.history = append(s.history, *st)
		fresh := initialTask(task)
		fresh.Attempt = st.Attempt + 1
		applyResumeDecision(&fresh, dec, hasDec)
		s.tasks[ident] = &fresh
		s.resume[ident] = dec
	}
}

func (s *sched) cascadeSkip(t TaskPlan) bool {
	for _, e := range s.doc.Edges {
		if e.ToTask != t.ID || e.FromTask == "" {
			continue
		}
		if s.isScatterTaskID(e.FromTask) {
			up := s.scatterTemplateState(e.FromTask)
			if up != nil && up.Status == StatusSkipped {
				return true
			}
			continue
		}
		up := s.stateByTaskID(e.FromTask)
		if up != nil && up.Status == StatusSkipped {
			return true
		}
	}
	return false
}

func (s *sched) evalWhen(t TaskPlan) (skip bool, reason string, ready bool, d *Defect) {
	if t.SkipIfFalse != "" {
		val, ok := paramValue(t, t.SkipIfFalse)
		if !ok || (val != "true" && val != "false") {
			return false, "", true, &Defect{Code: DefectInvalidValue, Unit: t.ID, Message: "invalid-value"}
		}
		if val == "false" {
			skip = true
			reason = conditionFalseParam
		}
	}
	if t.SkipIfMissingPort == "" && t.SkipIfMissingPath == "" {
		return skip, reason, true, nil
	}
	if t.SkipIfMissingTask != "" {
		up := s.stateByTaskID(t.SkipIfMissingTask)
		if up == nil {
			return false, "", false, nil
		}
		if up.Status == StatusUnknown || up.Status == StatusRunning || up.Status == StatusNotStarted {
			return false, "", false, nil
		}
		if up.Status != StatusSucceeded {
			return skip, reason, true, nil
		}
	}
	path := t.SkipIfMissingPath
	if path == "" {
		path = s.skipMissingDest(t)
	}
	if path == "" {
		return skip, reason, true, nil
	}
	abs, present, err := containedRel(s.workspace, path, false)
	if err != nil {
		esc := escapeDefect(t.ID, path)
		return false, "", true, &esc
	}
	if !present || !regularFile(abs) {
		return true, conditionMissingFile, true, nil
	}
	info, err := os.Lstat(abs)
	if err != nil || info.Size() == 0 {
		return true, conditionMissingFile, true, nil
	}
	return skip, reason, true, nil
}

func paramValue(t TaskPlan, name string) (string, bool) {
	for _, p := range t.Params {
		if p.Name == name {
			return p.Value, true
		}
	}
	return "", false
}

func (s *sched) skipMissingDest(t TaskPlan) string {
	if t.SkipIfMissingTask == "" {
		return t.SkipIfMissingPath
	}
	src, ok := s.taskByID(t.SkipIfMissingTask)
	if !ok {
		return t.SkipIfMissingPath
	}
	io, ok := findProducerIO(src, t.SkipIfMissingPort)
	if !ok {
		return t.SkipIfMissingPath
	}
	return io.Path
}
