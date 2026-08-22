package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

// Release closes occupancy on workspace after Reconcile. The occupying
// process may Release after Run returns. A later process while the
// occupying process is live is live-occupancy. Occupancy does not close
// while any identity remains unknown.
func Release(workspace string) []Defect {
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
	owner := holdsLease(workspace)
	var lock *os.File
	if !owner {
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
	run, exists, err = readRunIdentity(workspace)
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tasks, marked, unknown, found, err := reconcileRelease(workspace, ex, now)
	if err != nil {
		return pathDefects(err)
	}
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
		if err := rewritePlanSnapshot(workspace, snapshot); err != nil {
			return pathDefects(err)
		}
		if found {
			if err := s.writeTasks(); err != nil {
				return pathDefects(err)
			}
		}
		if err := s.writeRun(); err != nil {
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
	if err := rewritePlanSnapshot(workspace, snapshot); err != nil {
		return pathDefects(err)
	}
	if found {
		if err := s.writeTasks(); err != nil {
			return pathDefects(err)
		}
	}
	if err := s.writeRun(); err != nil {
		return pathDefects(err)
	}
	DropHeldLease(workspace)
	return nil
}

func rewritePlanSnapshot(workspace, snapshot string) error {
	path := filepath.Join(workspace, ControlDir, PlanFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var plan jsonPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return err
	}
	plan.Snapshot = snapshot
	out, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(out, '\n'))
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

func reconcileRelease(workspace string, ex exec.Executor, now string) ([]jsonTaskState, []string, []string, bool, error) {
	path := filepath.Join(workspace, ControlDir, TasksFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil, false, nil
		}
		return nil, nil, nil, false, err
	}
	var doc jsonTasksFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, nil, false, err
	}
	var marked []string
	var unknown []string
	for i := range doc.Tasks {
		st := &doc.Tasks[i]
		applyTaskStateDefaults(st)
		ident := reservedIdentity(taskPlanFromState(*st))
		switch st.Status {
		case StatusUnknown:
			if !reconcileIdentity(workspace, ex, st) {
				unknown = append(unknown, ident)
				continue
			}
			if st.Status == StatusRunning || st.Status == StatusUnknown {
				st.Status = StatusIncomplete
				st.Ended = now
				st.Reason = "released"
				marked = append(marked, ident)
			}
		case StatusRunning:
			if !reconcileIdentity(workspace, ex, st) {
				st.Status = StatusUnknown
				st.Reason = "unknown-backend"
				unknown = append(unknown, ident)
				continue
			}
			if st.Status == StatusRunning {
				st.Status = StatusIncomplete
				st.Ended = now
				st.Reason = "released"
				marked = append(marked, ident)
			}
		}
	}
	sort.Strings(unknown)
	return doc.Tasks, marked, unknown, true, nil
}

func reconcileIdentity(workspace string, ex exec.Executor, st *jsonTaskState) bool {
	ident := reservedIdentity(taskPlanFromState(*st))
	h, ok := backendHandle(workspace, ident, st)
	if !ok {
		if st.Status == StatusUnknown || st.Status == StatusRunning {
			st.Status = StatusUnknown
			st.Reason = "unknown-backend"
			return false
		}
		return true
	}
	r, err := boundedReconcile(ex, h)
	if err != nil {
		st.Status = StatusUnknown
		st.Reason = "unknown-backend"
		st.Error = &jsonTaskErr{Unit: ident, Message: err.Error()}
		return false
	}
	if r.Running {
		if err := boundedCancel(ex, h); err != nil {
			st.Status = StatusUnknown
			st.Reason = "unknown-backend"
			st.Error = &jsonTaskErr{Unit: ident, Message: err.Error()}
			return false
		}
		for {
			pr, perr := boundedPoll(ex, h)
			if perr != nil {
				st.Status = StatusUnknown
				st.Reason = "unknown-backend"
				st.Error = &jsonTaskErr{Unit: ident, Message: perr.Error()}
				return false
			}
			if !pr.Running {
				return true
			}
			time.Sleep(pollInterval)
		}
	}
	return true
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
