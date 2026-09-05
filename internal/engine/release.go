package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

// Release closes occupancy on workspace after bounded settlement. The
// occupying process may Release after Run returns. A later process while the
// occupying process is live is live-occupancy. Later-process unproved process
// identities become incomplete or published-unfinalized; Docker unknowns keep
// occupancy active.
func Release(workspace string, supplied *InstallIdentity) []Defect {
	if d := checkWorkspace(workspace); len(d) > 0 {
		return d
	}
	run, exists, err := readRunIdentity(workspace)
	if err != nil {
		return pathDefects(err)
	}
	if !exists {
		return []Defect{{
			Code:    DefectNotFound,
			Message: "run not found",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	if d := unsupportedControlSchema(workspace, run); len(d) > 0 {
		return d
	}
	if supplied != nil {
		if d := ValidateInstallIdentity(supplied); len(d) > 0 {
			return d
		}
	}
	if d := workspaceIdentityDefects(run.Identity, installIdentityForWorkspace(run.Identity, supplied), identityRelease); len(d) > 0 {
		return d
	}
	if !occupancyIsActive(run) {
		return []Defect{{
			Code:    DefectAlreadyReleased,
			Message: "already released",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	host, err := currentHost()
	if err != nil {
		return pathDefects(err)
	}
	if run.Occupancy != nil && run.Occupancy.Host != "" && run.Occupancy.Host != host {
		return []Defect{{
			Code:    DefectForeignHost,
			Message: "foreign host",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	held := heldLeaseFor(workspace)
	owner := held != nil
	var lock *os.File
	if owner {
		held.mutator.Lock()
		if heldLeaseFor(workspace) != held {
			held.mutator.Unlock()
			return liveOccupancyDefect()
		}
		defer held.mutator.Unlock()
	} else {
		claimed, d := claimOccupy(filepath.Join(workspace, ControlDir))
		if len(d) > 0 {
			if hasDefectCode(d, DefectOccupiedWorkspace) {
				return liveOccupancyDefect()
			}
			return d
		}
		lock = claimed
		defer lock.Close()
	}
	run, plan, hasPlan, taskFile, _, d := readCoherentControl(workspace)
	if len(d) > 0 {
		return d
	}
	if d := workspaceIdentityDefects(run.Identity, installIdentityForWorkspace(run.Identity, supplied), identityRelease); len(d) > 0 {
		return d
	}
	tasks := taskFile.Tasks
	if len(latestAttempts(tasks)) == 0 {
		return emptyTaskStateDefects()
	}
	if !occupancyIsActive(run) {
		return []Defect{{
			Code:    DefectAlreadyReleased,
			Message: "already released",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	occ := run.Occupancy
	if occ != nil && occ.Host != "" && occ.Host != host {
		return []Defect{{
			Code:    DefectForeignHost,
			Message: "foreign host",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	ex := heldExecutor(workspace)
	if ex == nil {
		ex = schedulerExecutor()
	}
	recorded := Document{}
	if hasPlan {
		recorded = documentFromPlan(plan)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	settle := newSettlement()
	defer settle.close()
	tasks, marked, unknown := reconcileRelease(workspace, ex, settle, recorded, tasks, owner, now)
	if occ == nil {
		occ = &jsonOccupancy{}
	}
	occ.Unknown = unknown
	run.Occupancy = occ
	run.SchemaVersion = SchemaVersion
	if len(unknown) > 0 {
		snapshot := newOccupancyID()
		run.Snapshot = snapshot
		s := releaseSched(workspace, run, tasks)
		s.snapshot = snapshot
		if err := s.writeReleasedCheckpoint(plan); err != nil {
			return pathDefects(err)
		}
		return unknownBackendDefects(unknown)
	}
	occ.Active = false
	occ.Closed = now
	occ.Incomplete = marked
	occ.Unknown = nil
	run.Occupancy = occ
	latest := latestAttempts(tasks)
	run.Status = runStatusFromTasks(latest)
	if run.Ended == "" {
		run.Ended = now
	}
	snapshot := newOccupancyID()
	run.Snapshot = snapshot
	s := releaseSched(workspace, run, tasks)
	s.snapshot = snapshot
	if err := s.writeReleasedCheckpoint(plan); err != nil {
		return pathDefects(err)
	}
	DropHeldLease(workspace)
	return nil
}

func (s *sched) writeReleasedCheckpoint(plan jsonPlan) error {
	plan.Snapshot = s.snapshot
	plan.SchemaVersion = SchemaVersion
	out, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return s.writeCheckpoint(append(out, '\n'))
}

func releaseSched(workspace string, run jsonRun, tasks []jsonTaskState) *sched {
	latest := latestAttempts(tasks)
	s := &sched{
		workspace: workspace,
		run:       run,
		snapshot:  run.Snapshot,
		tasks:     make(map[string]*jsonTaskState, len(latest)),
		history:   priorAttempts(tasks),
	}
	for _, st := range latest {
		cp := st
		tp := taskPlanFromState(st)
		s.tasks[reservedIdentity(tp)] = &cp
		s.doc.Tasks = append(s.doc.Tasks, tp)
	}
	return s
}

func unsupportedControlSchema(workspace string, run jsonRun) []Defect {
	lock, root, committed, err := openCheckpoint(workspace)
	if err != nil {
		return pathDefects(err)
	}
	defer closeCheckpointLock(lock)
	if committed {
		_, _, _, err := readCommittedControl(workspace, root)
		if err != nil {
			return checkpointDefects(err)
		}
		return nil
	}
	return unsupportedLegacyControlSchema(workspace, run)
}

func unsupportedLegacyControlSchema(workspace string, run jsonRun) []Defect {
	if schemaUnsupported(run.SchemaVersion) || pidOnlyOccupancy(run) {
		return schemaDefect(ControlDir + "/" + RunIdentityFile)
	}
	root := filepath.Join(workspace, ControlDir)
	for _, name := range []string{PlanFile, TasksFile} {
		ver, exists, err := readSchemaFile(filepath.Join(root, name))
		if err != nil {
			return pathDefects(err)
		}
		if exists && schemaUnsupported(ver) {
			return schemaDefect(ControlDir + "/" + name)
		}
	}
	return nil
}

func schemaDefect(path string) []Defect {
	return []Defect{{
		Code:    DefectUnsupportedSchema,
		Message: "unsupported schema",
		Paths:   []string{path},
	}}
}

func reconcileRelease(workspace string, ex exec.Executor, settle *settlement, recorded Document, tasks []jsonTaskState, owner bool, now string) ([]jsonTaskState, []string, []string) {
	latest := latestAttempts(tasks)
	latestAttempt := make(map[string]int, len(latest))
	stateByIdentity := make(map[string]*jsonTaskState, len(latest))
	for i := range latest {
		st := &latest[i]
		applyTaskStateDefaults(st)
		ident := reservedIdentity(taskPlanFromState(*st))
		latestAttempt[ident] = st.Attempt
		stateByIdentity[ident] = st
	}
	releaseState := &sched{workspace: workspace, doc: recorded, tasks: stateByIdentity}
	var marked []string
	var unknown []string
	for i := range tasks {
		st := &tasks[i]
		applyTaskStateDefaults(st)
		ident := reservedIdentity(taskPlanFromState(*st))
		if latestAttempt[ident] != st.Attempt {
			continue
		}
		switch st.Status {
		case StatusUnknown:
			if !reconcileIdentity(workspace, ex, settle, releaseState, st, owner, now) {
				unknown = append(unknown, ident)
				continue
			}
			if st.Status == StatusIncomplete {
				marked = append(marked, ident)
				continue
			}
			if st.Status == StatusRunning || st.Status == StatusUnknown {
				st.Status = StatusIncomplete
				st.Ended = now
				markReleasedReason(st)
				marked = append(marked, ident)
			}
		case StatusRunning:
			if !reconcileIdentity(workspace, ex, settle, releaseState, st, owner, now) {
				st.Status = StatusUnknown
				st.Reason = "unknown-backend"
				unknown = append(unknown, ident)
				continue
			}
			if st.Status == StatusIncomplete {
				marked = append(marked, ident)
				continue
			}
			if st.Status == StatusRunning {
				st.Status = StatusIncomplete
				st.Ended = now
				markReleasedReason(st)
				marked = append(marked, ident)
			}
		default:
			if dockerLeftover(st) && !retryDockerLeftover(workspace, ex, settle, st) {
				unknown = append(unknown, ident)
			}
		}
	}
	sort.Strings(unknown)
	return tasks, marked, unknown
}

func markReleasedReason(st *jsonTaskState) {
	if st.Reason == "" || st.Reason == "ready" || st.Reason == "unknown-backend" {
		st.Reason = "released"
	}
}

func dockerLeftover(st *jsonTaskState) bool {
	return st.RuntimeID != "" && (st.Executor == executorDocker || st.Image != "")
}

func retryDockerLeftover(workspace string, ex exec.Executor, settle *settlement, st *jsonTaskState) bool {
	ident := reservedIdentity(taskPlanFromState(*st))
	h, ok := backendHandle(workspace, ident, st)
	if !ok {
		return true
	}
	r, err := settle.reconcile(ex, h)
	if err != nil {
		// The terminal task state is prior stopped proof. A cleanup retry
		// failure does not make that disposition unknown again.
		return true
	}
	if r.Running {
		return false
	}
	st.RuntimeID = r.RuntimeID
	return true
}

func reconcileIdentity(workspace string, ex exec.Executor, settle *settlement, releaseState *sched, st *jsonTaskState, owner bool, now string) bool {
	ident := reservedIdentity(taskPlanFromState(*st))
	backend := st.Executor
	if backend == "" {
		if st.Image != "" {
			backend = executorDocker
		} else {
			backend = executorProcess
		}
	}
	if !owner && backend == executorProcess {
		if h, ok := backendHandle(workspace, ident, st); ok {
			_, _ = settle.reconcile(ex, h)
		}
		task, ok := releaseState.taskByIdent(ident)
		if ok && destComplete(workspace, task.Outputs) {
			st.Status = StatusPublishedUnfinalized
			st.Reason = StatusPublishedUnfinalized
			st.Error = nil
		} else {
			st.Status = StatusIncomplete
			st.Reason = "released"
		}
		if st.Ended == "" {
			st.Ended = now
		}
		return true
	}
	h, ok := backendHandle(workspace, ident, st)
	if !ok {
		if st.Status == StatusUnknown || st.Status == StatusRunning {
			st.Status = StatusUnknown
			st.Reason = "unknown-backend"
			return false
		}
		return true
	}
	r, err := settle.reconcile(ex, h)
	if err != nil {
		st.Status = StatusUnknown
		st.Reason = "unknown-backend"
		st.Error = &jsonTaskErr{Unit: ident, Message: err.Error()}
		return false
	}
	if r.Running {
		if err := settle.cancelHandle(ex, h); err != nil {
			st.Status = StatusUnknown
			st.Reason = "unknown-backend"
			st.Error = &jsonTaskErr{Unit: ident, Message: err.Error()}
			return false
		}
		for {
			pr, perr := settle.poll(ex, h)
			if perr != nil {
				st.Status = StatusUnknown
				st.Reason = "unknown-backend"
				st.Error = &jsonTaskErr{Unit: ident, Message: perr.Error()}
				return false
			}
			if !pr.Running {
				applyDockerStoppedReport(st, backend, pr)
				return true
			}
			if err := settle.pause(); err != nil {
				st.Status = StatusUnknown
				st.Reason = "unknown-backend"
				st.Error = &jsonTaskErr{Unit: ident, Message: err.Error()}
				return false
			}
		}
	}
	applyDockerStoppedReport(st, backend, r)
	return true
}

func applyDockerStoppedReport(st *jsonTaskState, backend string, r exec.Report) {
	if backend != executorDocker || r.Running {
		return
	}
	st.RuntimeID = r.RuntimeID
	if r.Reason != "" {
		st.Reason = r.Reason
	}
}

func destComplete(workspace string, outputs []IO) bool {
	if len(outputs) == 0 {
		return false
	}
	for _, out := range outputs {
		if isTreeIO(out) {
			if !completeDirectory(workspace, treeDir(out)) ||
				!completeRegularFile(workspace, treeManifestPath(out)) {
				return false
			}
			continue
		}
		files := namedIOFiles(out)
		if len(files) == 0 {
			return false
		}
		for _, file := range files {
			if !completeRegularFile(workspace, file.path) {
				return false
			}
		}
	}
	return true
}

func completeRegularFile(workspace, path string) bool {
	abs, present, err := containedRel(workspace, path, false)
	return err == nil && present && regularFile(abs)
}

func completeDirectory(workspace, path string) bool {
	abs, present, err := containedRel(workspace, path, false)
	return err == nil && present && isDir(abs)
}

func runStatusFromTasks(tasks []jsonTaskState) string {
	if len(tasks) == 0 {
		return StatusFailed
	}
	for _, st := range tasks {
		if st.Status == StatusUnknown {
			return StatusUnknown
		}
	}
	for _, st := range tasks {
		if st.Status == StatusSucceeded || st.Status == StatusSkipped {
			continue
		}
		if st.Scatter != "" && st.Instance == "" {
			continue
		}
		return StatusFailed
	}
	return StatusSucceeded
}

func hasDefectCode(defects []Defect, code string) bool {
	for _, d := range defects {
		if d.Code == code {
			return true
		}
	}
	return false
}
