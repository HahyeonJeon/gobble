package engine

import (
	"context"
	"os"
	"time"
)

// Resume occupies a released existing run after checks, classifies
// every reserved identity as Change, executes reruns as new
// attempts, and persists decisions. A nil result means every
// supplied identity succeeded. ctx cancel matches Run: stop admit,
// cancel in-flight, persist incomplete, occupancy stays active,
// DefectCanceled.
func Resume(ctx context.Context, req Request) []Defect {
	if ctx == nil {
		ctx = context.Background()
	}
	if d := checkResume(req); len(d) > 0 {
		return d
	}
	n := req.Cap
	if n == 0 {
		n = DefaultCap
	}
	s, defects := occupyResume(req)
	if len(defects) > 0 {
		return defects
	}
	return s.loop(ctx, n)
}

func checkResume(req Request) []Defect {
	if d := CheckResumeStart(req.Workspace, req.Cap); len(d) > 0 {
		return d
	}
	if d := checkOccupied(req.Workspace); len(d) > 0 {
		return d
	}
	if d := checkPlanPaths(req.Document); len(d) > 0 {
		return d
	}
	if d := checkBackends(req.Document); len(d) > 0 {
		return d
	}
	if d := checkImages(req.Document); len(d) > 0 {
		return d
	}
	if d := checkInputs(req.Workspace, req.Document); len(d) > 0 {
		return d
	}
	if d := checkCapacity(req.Document, readHostCapacity()); len(d) > 0 {
		return d
	}
	run, exists, err := readRunIdentity(req.Workspace)
	if err != nil {
		return pathDefects(err)
	}
	if !exists {
		return []Defect{{
			Code:    DefectNothingToResume,
			Message: "nothing to resume",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	if d := unsupportedControlSchema(req.Workspace, run); len(d) > 0 {
		return d
	}
	recorded, hasPlan, d := readInspectPlan(req.Workspace)
	if len(d) > 0 {
		return d
	}
	var recordedDoc Document
	if hasPlan {
		recordedDoc = documentFromPlan(recorded)
	}
	tasks, d := readInspectTasks(req.Workspace)
	if len(d) > 0 {
		return d
	}
	if unknown := unknownTaskUnits(tasks); len(unknown) > 0 {
		return unknownBackendDefects(unknown)
	}
	class := classifyResume(req.Workspace, recordedDoc, req.Document, tasks)
	return checkResumeOutputs(req.Workspace, req.Document, tasks, class)
}

func checkResumeOutputs(workspace string, doc Document, tasks []jsonTaskState, class remainingClass) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		applyReservedDefaults(&t)
		ident := reservedIdentity(t)
		if class.Decision[ident].Decision != reuseRerun {
			continue
		}
		for _, out := range t.Outputs {
			unit := bindUnit(t.ID, out.Name)
			if isTreeIO(out) {
				if !pathPresent(workspaceFile(workspace, out.Path)) {
					continue
				}
				if treePublishedBy(tasks, ident, workspace, out) {
					continue
				}
				defects = append(defects, Defect{
					Code:    DefectOutputExists,
					Unit:    unit,
					Message: "output exists",
					Paths:   []string{out.Path},
				})
				continue
			}
			for _, f := range namedIOFiles(out) {
				if !pathPresent(workspaceFile(workspace, f.path)) {
					continue
				}
				if destPublished(tasks, ident, f.path) {
					continue
				}
				defUnit := unit
				if out.Members != nil {
					defUnit = bindUnit(unit, f.name)
				}
				defects = append(defects, Defect{
					Code:    DefectOutputExists,
					Unit:    defUnit,
					Message: "output exists",
					Paths:   []string{f.path},
				})
			}
		}
	}
	return defects
}

func treePublishedBy(tasks []jsonTaskState, ident, workspace string, out IO) bool {
	if destPublished(tasks, ident, treeManifestPath(out)) {
		return true
	}
	for _, f := range treeDestMemberPaths(workspace, out) {
		if destPublished(tasks, ident, f.path) {
			return true
		}
	}
	return false
}

func destPublished(tasks []jsonTaskState, ident, path string) bool {
	for _, st := range tasks {
		applyTaskStateDefaults(&st)
		if reservedIdentity(taskPlanFromState(st)) != ident {
			continue
		}
		switch st.Status {
		case StatusSucceeded, StatusFailed, StatusIncomplete, StatusRunning, StatusUnknown:
		default:
			continue
		}
		for _, h := range st.Checksums {
			if h.Path == path {
				return true
			}
		}
		for _, lin := range st.Lineage {
			if lin.Path == path && lin.Producer == ident {
				return true
			}
		}
	}
	return false
}

func occupyResume(req Request) (*sched, []Defect) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	host, err := currentHost()
	if err != nil {
		return nil, pathDefects(err)
	}
	existing, exists, err := readRunIdentity(req.Workspace)
	if err != nil {
		return nil, pathDefects(err)
	}
	if !exists {
		return nil, []Defect{{
			Code:    DefectNothingToResume,
			Message: "nothing to resume",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	recorded, hasPlan, d := readInspectPlan(req.Workspace)
	if len(d) > 0 {
		return nil, d
	}
	var recordedDoc Document
	if hasPlan {
		recordedDoc = documentFromPlan(recorded)
	}
	tasks, d := readInspectTasks(req.Workspace)
	if len(d) > 0 {
		return nil, d
	}
	if unknown := unknownTaskUnits(tasks); len(unknown) > 0 {
		return nil, unknownBackendDefects(unknown)
	}
	root := workspaceFile(req.Workspace, ControlDir)
	lock, defects := claimOccupy(root)
	if len(defects) > 0 {
		return nil, defects
	}
	if current, found, err := readRunIdentity(req.Workspace); err != nil {
		lock.Close()
		return nil, pathDefects(err)
	} else if found && occupancyIsActive(current) {
		lock.Close()
		return nil, occupiedDefect()
	}
	doc := cloneDocument(req.Document)
	for i := range doc.Tasks {
		applyReservedDefaults(&doc.Tasks[i])
	}
	class := classifyResume(req.Workspace, recordedDoc, doc, tasks)
	latest := latestAttempts(tasks)
	byIdent := make(map[string]jsonTaskState, len(latest))
	for _, st := range latest {
		byIdent[reservedIdentity(taskPlanFromState(st))] = st
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
			ID:            existing.ID,
			Status:        StatusRunning,
			Started:       existing.Started,
			Occupancy: &jsonOccupancy{
				Active:  true,
				Host:    host,
				PID:     os.Getpid(),
				Lease:   lease,
				Started: now,
			},
		},
		tasks:    make(map[string]*jsonTaskState, len(doc.Tasks)),
		history:  priorAttempts(tasks),
		resume:   class.Decision,
		launched: make(map[string]bool),
		budget:   newBudget(readHostCapacity()),
		exec:     ex,
	}
	if s.run.ID == "" {
		s.run.ID = runID(doc)
	}
	if s.run.Started == "" {
		s.run.Started = now
	}
	for _, t := range doc.Tasks {
		ident := reservedIdentity(t)
		dec := class.Decision[ident]
		if st, ok := byIdent[ident]; ok {
			cp := st
			cp.Change = dec.Change
			if dec.Decision == reuseReused {
				cp.Decision = dec.Decision
				cp.ReuseReason = dec.Reason
				cp.Differing = append([]string(nil), dec.Differing...)
			}
			s.tasks[ident] = &cp
			continue
		}
		st := initialTask(t)
		st.Decision = dec.Decision
		st.ReuseReason = dec.Reason
		st.Differing = append([]string(nil), dec.Differing...)
		st.Change = dec.Change
		s.tasks[ident] = &st
	}
	for ident, st := range byIdent {
		if s.tasks[ident] != nil {
			continue
		}
		cp := st
		if dec := class.Decision[ident]; dec.Change == changeRemoved {
			cp.Change = changeRemoved
		}
		s.history = append(s.history, cp)
	}
	if err := s.writeControl(); err != nil {
		lock.Close()
		return nil, pathDefects(err)
	}
	retainLease(req.Workspace, lock, ex)
	return s, nil
}

func unknownTaskUnits(tasks []jsonTaskState) []string {
	var units []string
	seen := map[string]bool{}
	for _, st := range latestAttempts(tasks) {
		if st.Status != StatusUnknown {
			continue
		}
		ident := reservedIdentity(taskPlanFromState(st))
		if seen[ident] {
			continue
		}
		seen[ident] = true
		units = append(units, ident)
	}
	return units
}
