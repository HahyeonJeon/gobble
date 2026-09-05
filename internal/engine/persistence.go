package engine

import (
	"encoding/json"
	"path/filepath"
	"sort"
)

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
