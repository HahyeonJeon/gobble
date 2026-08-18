package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Release closes occupancy on workspace when the owner is not live on
// this host. It does not execute, delete, cancel, or occupy.
func Release(workspace string) []Defect {
	ident := filepath.Join(workspace, ControlDir, RunIdentityFile)
	data, err := os.ReadFile(ident)
	if err != nil {
		if os.IsNotExist(err) {
			return []Defect{{
				Code:    DefectNotFound,
				Message: "run not found",
				Paths:   []string{ControlDir + "/" + RunIdentityFile},
			}}
		}
		return pathDefects(err)
	}
	var run jsonRun
	if err := json.Unmarshal(data, &run); err != nil {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "run identity is not usable",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	if d := unsupportedControlSchema(workspace, run.SchemaVersion); len(d) > 0 {
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
	occ := run.Occupancy
	if occ != nil && occ.Host != "" && occ.Host != host {
		return []Defect{{
			Code:    DefectForeignHost,
			Message: "foreign host",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	if occ != nil && occ.Host == host && pidExists(occ.PID) {
		return []Defect{{
			Code:    DefectLiveOccupancy,
			Message: "live occupancy",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tasks, marked, found, err := markIncomplete(workspace, now)
	if err != nil {
		return pathDefects(err)
	}
	if occ == nil {
		occ = &jsonOccupancy{}
	}
	occ.Active = false
	occ.Closed = now
	occ.Incomplete = marked
	run.Occupancy = occ
	run.SchemaVersion = SchemaVersion
	run.Status = runStatusFromTasks(tasks)
	if run.Ended == "" {
		run.Ended = now
	}
	s := &sched{workspace: workspace, run: run, tasks: make(map[string]*jsonTaskState, len(tasks))}
	for i := range tasks {
		st := tasks[i]
		s.tasks[st.ID] = &st
		s.doc.Tasks = append(s.doc.Tasks, TaskPlan{ID: st.ID})
	}
	if found {
		if err := s.writeTasks(); err != nil {
			return pathDefects(err)
		}
	}
	if err := s.writeRun(); err != nil {
		return pathDefects(err)
	}
	return nil
}

func unsupportedControlSchema(workspace string, runSchema int) []Defect {
	if schemaUnsupported(runSchema) {
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

func markIncomplete(workspace, now string) ([]jsonTaskState, []string, bool, error) {
	path := filepath.Join(workspace, ControlDir, TasksFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, false, nil
		}
		return nil, nil, false, err
	}
	var doc jsonTasksFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, false, err
	}
	var marked []string
	for i := range doc.Tasks {
		st := &doc.Tasks[i]
		if st.Status != StatusRunning {
			continue
		}
		st.Status = StatusIncomplete
		st.Ended = now
		st.Reason = "released"
		marked = append(marked, reservedIdentity(TaskPlan{
			ID:         st.ID,
			Instance:   st.Instance,
			ShardIndex: st.ShardIndex,
		}))
	}
	return doc.Tasks, marked, true, nil
}

func runStatusFromTasks(tasks []jsonTaskState) string {
	if len(tasks) == 0 {
		return StatusFailed
	}
	for _, st := range tasks {
		if st.Status != StatusSucceeded {
			return StatusFailed
		}
	}
	return StatusSucceeded
}
