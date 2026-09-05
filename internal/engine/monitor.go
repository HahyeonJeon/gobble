package engine

// This file projects existing controls. It never acquires occupancy or writes
// state. Presentation and aggregation belong to package monitor, not engine.

type monitorDoc struct {
	SchemaVersion int           `json:"schema_version"`
	Snapshot      string        `json:"snapshot"`
	Pipeline      string        `json:"pipeline"`
	Run           inspectRunDoc `json:"run"`
	Tasks         []monitorTask `json:"tasks"`
	Edges         []monitorEdge `json:"edges"`
	Logs          []inspectLog  `json:"logs"`
}

type monitorTask struct {
	Identity  string        `json:"identity"`
	TaskID    string        `json:"task_id"`
	Name      string        `json:"name"`
	Module    string        `json:"module"`
	Display   *Display      `json:"display,omitempty"`
	Status    string        `json:"status"`
	Attempt   int           `json:"attempt"`
	Executor  string        `json:"executor"`
	Image     string        `json:"image"`
	Command   []string      `json:"command"`
	Script    string        `json:"script,omitempty"`
	Resources jsonResources `json:"resources"`
	Started   string        `json:"started,omitempty"`
	Ended     string        `json:"ended,omitempty"`
	Reason    string        `json:"reason,omitempty"`
	Decision  string        `json:"decision,omitempty"`
	Template  bool          `json:"template,omitempty"`
	Expanded  bool          `json:"expanded,omitempty"`
}

type monitorEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func inspectMonitorView(workspace string, run jsonRun, doc Document, tasks []jsonTaskState, instance string) ([]byte, []Defect) {
	out := monitorDoc{
		SchemaVersion: run.SchemaVersion, Snapshot: run.Snapshot, Pipeline: doc.Name,
		Run: inspectRunView(workspace, run, tasks),
		Tasks: make([]monitorTask, 0, len(tasks)), Edges: []monitorEdge{}, Logs: []inspectLog{},
	}
	byID := make(map[string]TaskPlan, len(doc.Tasks))
	for _, task := range doc.Tasks {
		byID[task.ID] = task
	}
	for _, st := range tasks {
		applyTaskStateDefaults(&st)
		plan := byID[st.ID]
		task := monitorTask{
			Identity: reservedIdentity(taskPlanFromState(st)), TaskID: st.ID,
			Name: plan.Name, Module: plan.Module, Display: cloneDisplay(plan.Display),
			Status: st.Status, Attempt: st.Attempt, Executor: st.Executor,
			Image: st.Image, Command: jsonStrings(st.Command), Script: plan.Script,
			Resources: st.Resources, Started: st.Started, Ended: st.Ended,
			Reason: st.Reason, Decision: st.Decision,
			Template: isScatterTemplateState(&st), Expanded: st.Expansion != nil,
		}
		if st.Error != nil && st.Error.Message != "" {
			task.Reason = st.Error.Message
		}
		if task.Name == "" {
			task.Name = st.ID
		}
		out.Tasks = append(out.Tasks, task)
		if task.Identity == instance {
			logs, defects := inspectLogsView(workspace, []jsonTaskState{st}, run.SchemaVersion)
			if len(defects) > 0 {
				return nil, defects
			}
			out.Logs = logs.Logs
		}
	}
	seen := make(map[monitorEdge]bool)
	for _, edge := range doc.Edges {
		if edge.FromTask == "" || edge.ToTask == "" {
			continue
		}
		pair := monitorEdge{From: edge.FromTask, To: edge.ToTask}
		if !seen[pair] {
			out.Edges = append(out.Edges, pair)
			seen[pair] = true
		}
	}
	return marshalInspect(out)
}

// The scheduler knows a started attempt's canonical isolate before its final
// report publishes stdout/stderr pointers. Derive those pointers here, then
// subject them to logPointer's usual containment and regular-file checks.
func taskLogPaths(st jsonTaskState) (string, string) {
	stdout, stderr := st.Stdout, st.Stderr
	if st.Started != "" && !isScatterTemplateState(&st) {
		base := isolateRel(taskPlanFromState(st))
		if stdout == "" {
			stdout = base + "/stdout"
		}
		if stderr == "" {
			stderr = base + "/stderr"
		}
	}
	return stdout, stderr
}
