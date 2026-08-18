package engine

import (
	"os"
	"time"
)

// Resume occupies a released existing run after checks, classifies
// every reserved identity, executes reruns as new attempts, and
// persists decisions. A nil result means every supplied identity
// succeeded.
func Resume(req Request) []Defect {
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
	return s.loop(n)
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
	if d := unsupportedControlSchema(req.Workspace, run.SchemaVersion); len(d) > 0 {
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
	if d := planDrift(recordedDoc, req.Document); len(d) > 0 {
		return d
	}
	tasks, d := readInspectTasks(req.Workspace)
	if len(d) > 0 {
		return d
	}
	class := classifyResume(req.Workspace, recordedDoc, req.Document, tasks)
	return checkResumeOutputs(req.Workspace, req.Document, tasks, class)
}

func planDrift(recorded, supplied Document) []Defect {
	recIDs := taskIDSet(recorded.Tasks)
	supIDs := taskIDSet(supplied.Tasks)
	if len(recIDs) != len(supIDs) {
		return planDriftDefect()
	}
	for id := range recIDs {
		if !supIDs[id] {
			return planDriftDefect()
		}
	}
	recEdges := edgeKeySet(recorded.Edges)
	supEdges := edgeKeySet(supplied.Edges)
	if len(recEdges) != len(supEdges) {
		return planDriftDefect()
	}
	for k := range recEdges {
		if !supEdges[k] {
			return planDriftDefect()
		}
	}
	return nil
}

func taskIDSet(tasks []TaskPlan) map[string]bool {
	out := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		out[t.ID] = true
	}
	return out
}

func edgeKeySet(edges []Edge) map[string]bool {
	out := make(map[string]bool)
	for _, e := range edges {
		if e.FromTask == "" {
			continue
		}
		out[e.FromTask+"\x00"+e.FromPort+"\x00"+e.ToTask+"\x00"+e.ToPort] = true
	}
	return out
}

func planDriftDefect() []Defect {
	return []Defect{{
		Code:    DefectPlanDrift,
		Message: "plan drift",
		Paths:   []string{ControlDir + "/" + PlanFile},
	}}
}

func checkResumeOutputs(workspace string, doc Document, tasks []jsonTaskState, class remainingClass) []Defect {
	executed := executedIdentities(tasks)
	var defects []Defect
	for _, t := range doc.Tasks {
		applyReservedDefaults(&t)
		ident := reservedIdentity(t)
		if class.Decision[ident].Decision != reuseRerun {
			continue
		}
		for _, out := range t.Outputs {
			unit := bindUnit(t.ID, out.Name)
			for _, f := range namedIOFiles(out) {
				if !pathPresent(workspaceFile(workspace, f.path)) {
					continue
				}
				if executed[ident] {
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

func executedIdentities(tasks []jsonTaskState) map[string]bool {
	out := make(map[string]bool)
	for _, st := range tasks {
		applyTaskStateDefaults(&st)
		switch st.Status {
		case StatusSucceeded, StatusFailed, StatusIncomplete, StatusRunning:
			out[reservedIdentity(taskPlanFromState(st))] = true
		}
	}
	return out
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
	doc := req.Document
	for i := range doc.Tasks {
		applyReservedDefaults(&doc.Tasks[i])
	}
	class := classifyResume(req.Workspace, recordedDoc, doc, tasks)
	latest := latestAttempts(tasks)
	byIdent := make(map[string]jsonTaskState, len(latest))
	for _, st := range latest {
		byIdent[reservedIdentity(taskPlanFromState(st))] = st
	}
	s := &sched{
		workspace: req.Workspace,
		doc:       doc,
		run: jsonRun{
			SchemaVersion: SchemaVersion,
			ID:            existing.ID,
			Status:        StatusRunning,
			Started:       existing.Started,
			Occupancy: &jsonOccupancy{
				Active:  true,
				Host:    host,
				PID:     os.Getpid(),
				Started: now,
			},
		},
		tasks:    make(map[string]*jsonTaskState, len(doc.Tasks)),
		history:  priorAttempts(tasks),
		resume:   class.Decision,
		launched: make(map[string]bool),
		budget:   newBudget(readHostCapacity()),
	}
	if s.run.ID == "" {
		s.run.ID = runID(doc)
	}
	if s.run.Started == "" {
		s.run.Started = now
	}
	for _, t := range doc.Tasks {
		ident := reservedIdentity(t)
		if st, ok := byIdent[ident]; ok {
			cp := st
			if dec := class.Decision[ident]; dec.Decision == reuseReused {
				cp.Decision = dec.Decision
				cp.ReuseReason = dec.Reason
				cp.Differing = append([]string(nil), dec.Differing...)
			}
			s.tasks[t.ID] = &cp
			continue
		}
		st := initialTask(t)
		s.tasks[t.ID] = &st
	}
	root := workspaceFile(req.Workspace, ControlDir)
	lock, defects := claimOccupy(root)
	if len(defects) > 0 {
		return nil, defects
	}
	defer lock.Close()
	if current, found, err := readRunIdentity(req.Workspace); err != nil {
		return nil, pathDefects(err)
	} else if found && occupancyIsActive(current) {
		return nil, occupiedDefect()
	}
	plan, err := marshalControlPlan(doc)
	if err != nil {
		return nil, pathDefects(err)
	}
	if err := writeAtomic(workspaceFile(req.Workspace, ControlDir+"/"+PlanFile), plan); err != nil {
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
