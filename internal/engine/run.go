package engine

import (
	"context"
	"errors"
	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"
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
	StatusUnknown    = "unknown"
	StatusSkipped    = "skipped"
	// StatusPublishedUnfinalized means a later-process Release found every
	// declared destination complete without proof that the process finished.
	StatusPublishedUnfinalized = "published-unfinalized"
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

// Docker observations invoke a client process; avoid a 50 Hz loop per task.
const dockerPollInterval = 500 * time.Millisecond

const settlementBound = 30 * time.Second

var errEngineBound = errors.New("engine bound")

// settlementBoundNanos bounds proof after stop or cancellation begins and
// Release reconciliation starts. Ordinary execution uses only caller ctx.
// Tests may shorten it. It is not a public timeout API.
var settlementBoundNanos atomic.Int64

func init() {
	setSettlementBound(settlementBound)
}

func setSettlementBound(d time.Duration) {
	settlementBoundNanos.Store(int64(d))
}

func currentSettlementBound() time.Duration {
	n := settlementBoundNanos.Load()
	if n <= 0 {
		return settlementBound
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
// no more independent work can start. A nil result means scheduling
// completed without a task failure. When ctx is done, in-flight work
// is canceled, those identities are persisted incomplete, occupancy
// stays active, and the result is DefectCanceled.
func Run(ctx context.Context, req Request) []Defect {
	if ctx == nil {
		ctx = context.Background()
	}
	if d := ValidateInstallIdentity(req.Identity); len(d) > 0 {
		return d
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
	lease     *heldLease
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
			Identity:      cloneInstallIdentity(req.Identity),
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
	s.lease = retainLease(req.Workspace, lock, ex)
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
	Reason           string
	Stdout           string
	Stderr           string
	Published        bool
	RuntimeID        string
	ImageDigest      string
	ExecutablePath   string
	ExecutableSHA256 string
	Fingerprints     []jsonFileHash
	Unknown          bool
	Canceled         bool
	InvalidPath      bool
}

type startEvent struct {
	ident  string
	handle exec.Handle
	sub    exec.Report
	ack    chan error
}

type settlementResult struct {
	ident   string
	unknown bool
	message string
}

func (s *sched) executor() exec.Executor {
	if s.exec != nil {
		return s.exec
	}
	return schedulerExecutor()
}

func (s *sched) loop(ctx context.Context, n int) []Defect {
	if s.lease != nil {
		defer s.lease.mutator.Unlock()
	}
	if s.resume != nil {
		s.reapLive()
		s.freshenWhenBranchAfterReap()
	}
	reports := make(chan report, n)
	starts := make(chan startEvent, n)
	settled := make(chan settlementResult, n)
	running := 0
	handles := make(map[string]exec.Handle)
	inflight := make(map[string]bool)
	settling := make(map[string]bool)
	canceled := false
	var settle *settlement
	var settlementDone <-chan struct{}
	finishIdent := func(ident string) {
		if !inflight[ident] {
			return
		}
		delete(inflight, ident)
		delete(handles, ident)
		delete(settling, ident)
		running--
	}
	settleHandle := func(ident string, h exec.Handle) {
		if settling[ident] {
			return
		}
		settling[ident] = true
		go settleBackend(s.executor(), settle, ident, h, settled)
	}
	startSettlement := func() {
		if canceled {
			return
		}
		canceled = true
		settle = newSettlement()
		settlementDone = settle.ctx.Done()
		for ident, h := range handles {
			settleHandle(ident, h)
		}
	}
	handleCanceledReport := func(r report) {
		if !inflight[r.ID] {
			return
		}
		if _, ok := handles[r.ID]; !ok && r.RuntimeID != "" {
			backend := executorProcess
			if st := s.tasks[r.ID]; st != nil && (st.Executor == executorDocker || st.Image != "") {
				backend = executorDocker
			}
			h := exec.Handle{Identity: r.ID, Backend: backend, RuntimeID: r.RuntimeID}
			if st := s.tasks[r.ID]; st != nil {
				st.RuntimeID = r.RuntimeID
				if durable, ok := backendHandle(s.workspace, r.ID, st); ok {
					h = durable
				}
				st.Status = StatusRunning
				s.notePersist(s.persistControl())
			}
			handles[r.ID] = h
		}
		if r.Canceled {
			if h, ok := handles[r.ID]; ok {
				settleHandle(r.ID, h)
				return
			}
			s.markIncomplete(r.ID)
			finishIdent(r.ID)
			return
		}
		if r.Unknown {
			s.markUnknown(r.ID, r.Message)
		} else {
			s.markIncomplete(r.ID)
		}
		finishIdent(r.ID)
	}
	for {
		if ctx.Err() != nil && !canceled {
			startSettlement()
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
			if !s.launch(ctx, id, starts, reports) {
				break
			}
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
					acknowledgeStart(st, context.Canceled)
					continue
				}
				if st.handle.RuntimeID != "" {
					handles[st.ident] = st.handle
				}
				s.persistStart(st)
				acknowledgeStart(st, context.Canceled)
				if st.handle.RuntimeID != "" {
					settleHandle(st.ident, st.handle)
				}
			case r := <-reports:
				handleCanceledReport(r)
			case result := <-settled:
				if !inflight[result.ident] {
					continue
				}
				if result.unknown {
					s.markUnknown(result.ident, result.message)
				} else {
					s.markIncomplete(result.ident)
				}
				finishIdent(result.ident)
			case <-settlementDone:
				for ident := range inflight {
					s.markUnknown(ident, "settlement bound exceeded")
					finishIdent(ident)
				}
			}
			continue
		}
		select {
		case st := <-starts:
			if !inflight[st.ident] {
				acknowledgeStart(st, context.Canceled)
				continue
			}
			if st.handle.RuntimeID != "" {
				handles[st.ident] = st.handle
			}
			s.persistStart(st)
			acknowledgeStart(st, s.persist)
		case r := <-reports:
			if !inflight[r.ID] {
				continue
			}
			if ctx.Err() != nil || r.Canceled {
				startSettlement()
				handleCanceledReport(r)
				continue
			}
			finishIdent(r.ID)
			s.apply(r)
		case <-ctx.Done():
			startSettlement()
		}
	}
	if canceled {
		settle.close()
		s.syncUnknown()
		s.notePersist(s.persistControl())
		out := s.cancelDefects()
		if unknown := s.unknownUnits(); len(unknown) > 0 {
			out = append(out, unknownBackendDefects(unknown)...)
		}
		out = append(out, s.persistenceDefects()...)
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
	if st.handle.Submission != nil {
		cp := *st.handle.Submission
		task.Submission = &cp
		if !cp.Created {
			s.notePersist(s.persistControl())
			return
		}
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

func acknowledgeStart(st startEvent, err error) {
	if st.ack != nil {
		st.ack <- err
	}
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
	settle := newSettlement()
	defer settle.close()
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
		r, err := settle.reconcile(ex, h)
		if err != nil {
			s.markUnknown(ident, err.Error())
			continue
		}
		if r.Running || r.NeedsRemoval {
			if err := settle.cancelHandle(ex, h); err != nil {
				s.markUnknown(ident, err.Error())
				continue
			}
			stopped := false
			for {
				pr, perr := settle.poll(ex, h)
				if perr != nil {
					s.markUnknown(ident, perr.Error())
					stopped = true
					break
				}
				if !pr.Running && !pr.NeedsRemoval {
					stopped = true
					break
				}
				if err := settle.pause(); err != nil {
					s.markUnknown(ident, err.Error())
					stopped = true
					break
				}
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
	if st.Submission != nil && backend == executorDocker {
		isolate, _, err := containedRel(workspace, isolateRel(taskPlanFromState(*st))+"/work", false)
		if err != nil {
			// Keep a handle that fails closed rather than treating it as absent.
			isolate = ""
		}
		return exec.Handle{Identity: ident, Backend: backend, RuntimeID: st.RuntimeID,
			Isolate: isolate, Submission: st.Submission}, true
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

type settlement struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func newSettlement() *settlement {
	ctx, cancel := context.WithTimeout(context.Background(), currentSettlementBound())
	return &settlement{ctx: ctx, cancel: cancel}
}

func (s *settlement) close() {
	s.cancel()
}

func (s *settlement) reconcile(ex exec.Executor, h exec.Handle) (exec.Report, error) {
	type result struct {
		report exec.Report
		err    error
	}
	done := make(chan result, 1)
	go func() {
		r, err := ex.Reconcile(s.ctx, h)
		done <- result{report: r, err: err}
	}()
	select {
	case got := <-done:
		return got.report, boundCallErrAfter(s.ctx, got.err)
	case <-s.ctx.Done():
		return exec.Report{}, errEngineBound
	}
}

func (s *settlement) cancelHandle(ex exec.Executor, h exec.Handle) error {
	done := make(chan error, 1)
	go func() { done <- ex.Cancel(s.ctx, h) }()
	select {
	case err := <-done:
		return boundCallErrAfter(s.ctx, err)
	case <-s.ctx.Done():
		return errEngineBound
	}
}

func (s *settlement) poll(ex exec.Executor, h exec.Handle) (exec.Report, error) {
	type result struct {
		report exec.Report
		err    error
	}
	done := make(chan result, 1)
	go func() {
		r, err := ex.Poll(s.ctx, h)
		done <- result{report: r, err: err}
	}()
	select {
	case got := <-done:
		return got.report, boundCallErrAfter(s.ctx, got.err)
	case <-s.ctx.Done():
		return exec.Report{}, errEngineBound
	}
}

func (s *settlement) pause() error {
	timer := time.NewTimer(pollInterval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-s.ctx.Done():
		return errEngineBound
	}
}

func settleBackend(ex exec.Executor, settle *settlement, ident string, h exec.Handle, out chan<- settlementResult) {
	if err := settle.cancelHandle(ex, h); err != nil {
		out <- settlementResult{ident: ident, unknown: true, message: err.Error()}
		return
	}
	for {
		report, err := settle.poll(ex, h)
		if err != nil {
			out <- settlementResult{ident: ident, unknown: true, message: err.Error()}
			return
		}
		if !report.Running && !report.NeedsRemoval {
			out <- settlementResult{ident: ident}
			return
		}
		if err := settle.pause(); err != nil {
			out <- settlementResult{ident: ident, unknown: true, message: err.Error()}
			return
		}
	}
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
			if up == nil || (up.Status != StatusSucceeded && up.Status != StatusPublishedUnfinalized) {
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

func (s *sched) launch(ctx context.Context, ident string, starts chan startEvent, reports chan report) bool {
	if s.resume != nil {
		s.beginResumeAttempt(ident)
	}
	st := s.tasks[ident]
	st.Status = StatusUnknown
	st.Reason = "unknown-backend"
	st.Started = time.Now().UTC().Format(time.RFC3339Nano)
	st.RuntimeID = ""
	st.Submission = nil
	ex := s.executor()
	task, _ := s.taskByIdent(ident)
	if durable, ok := ex.(interface{ DurableDocker() bool }); ok && durable.DurableDocker() && task.Image != "" {
		st.Submission = &exec.Submission{Token: newOccupancyID()}
	}
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
	if s.persist != nil {
		s.budget.release(task)
		return false
	}
	ws := s.workspace
	var submission *exec.Submission
	if st.Submission != nil {
		cp := *st.Submission
		submission = &cp
	}
	go func() {
		s.runJob(ctx, ws, task, ex, submission, starts, reports)
	}()
	return true
}

func (s *sched) runJob(ctx context.Context, workspace string, task TaskPlan, ex exec.Executor, submission *exec.Submission, starts chan startEvent, reports chan report) {
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
	fingerprints, err := stagedInputRecords(workspace, isolate, task.Inputs)
	if err != nil {
		r.Message = err.Error()
		reports <- r
		return
	}
	r.Fingerprints = fingerprints
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
		Submission:  submission,
	}
	if submission != nil {
		job.Record = func(ctx context.Context, h exec.Handle, sub exec.Report) error {
			ack := make(chan error, 1)
			select {
			case starts <- startEvent{ident: ident, handle: h, sub: sub, ack: ack}:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case err := <-ack:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
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
	h, sub, err := submitJob(ctx, workspace, ex, job)
	if err != nil {
		r.Message = err.Error()
		r.Unknown = h.Submission != nil
		if h.RuntimeID != "" {
			r.RuntimeID = h.RuntimeID
			r.Unknown = true
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			r.Canceled = true
		}
		if errors.Is(err, exec.ErrEscapedPath) {
			r.Message = "path escapes directory"
			r.InvalidPath = true
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
	if submission == nil {
		starts <- startEvent{ident: ident, handle: h, sub: sub}
	}
	for {
		pr, perr := ex.Poll(ctx, h)
		if perr != nil {
			r.Message = perr.Error()
			if errors.Is(perr, exec.ErrEscapedPath) {
				r.Message = "path escapes directory"
				r.InvalidPath = true
			} else if errors.Is(perr, context.Canceled) || errors.Is(perr, context.DeadlineExceeded) {
				r.Canceled = true
			} else {
				r.Unknown = true
			}
			reports <- r
			return
		}
		if pr.Running {
			interval := pollInterval
			if task.Image != "" {
				interval = dockerPollInterval
			}
			timer := time.NewTimer(interval)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				r.Canceled = true
				r.Message = ctx.Err().Error()
				reports <- r
				return
			}
			continue
		}
		r.Exit = pr.Exit
		r.Message = pr.Message
		r.Reason = pr.Reason
		r.Published = pr.Published
		r.RuntimeID = pr.RuntimeID
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
		if st.Status == StatusSkipped || st.Status == StatusSucceeded || st.Status == StatusUnknown {
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
	st.RuntimeID = r.RuntimeID
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
		st.Reason = terminalReason(r)
		st.Error = nil
		if task, ok := s.taskByIdent(r.ID); ok {
			s.notePersist(s.recordSuccess(st, task, r.Fingerprints))
		}
	} else {
		st.Status = StatusFailed
		st.Reason = terminalReason(r)
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

func terminalReason(r report) string {
	if r.Reason != "" {
		return r.Reason
	}
	return "ready"
}

func (s *sched) recordSuccess(st *jsonTaskState, task TaskPlan, inputs []jsonFileHash) error {
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
	return append(out, s.persistenceDefects()...)
}

func (s *sched) persistenceDefects() []Defect {
	if s.persist == nil {
		return nil
	}
	return []Defect{{
		Code:    DefectInvalidPath,
		Message: "persist: " + s.persist.Error(),
	}}
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

func submitJob(ctx context.Context, workspace string, ex exec.Executor, job exec.Job) (exec.Handle, exec.Report, error) {
	ls := registerLateSubmit(workspace, job.Identity)
	h, r, err := ex.Submit(ctx, job)
	finishLateSubmit(ls, h, r, err)
	return h, r, err
}
