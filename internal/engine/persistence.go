package engine

import (
	"encoding/json"
	"sort"
)

func (s *sched) persistControl() error {
	s.snapshot = newOccupancyID()
	return s.writeControl()
}

func (s *sched) writeControl() error {
	s.run.Snapshot = s.snapshot
	s.run.SchemaVersion = SchemaVersion
	plan, err := marshalControlPlan(s.doc, s.snapshot)
	if err != nil {
		return err
	}
	return s.writeCheckpoint(plan)
}

func (s *sched) writeCheckpoint(plan []byte) error {
	s.run.Snapshot = s.snapshot
	s.run.SchemaVersion = SchemaVersion
	tasks, err := s.marshalTasks()
	if err != nil {
		return err
	}
	run, err := json.MarshalIndent(s.run, "", "  ")
	if err != nil {
		return err
	}
	return commitCheckpoint(s.workspace, s.snapshot, plan, tasks, append(run, '\n'))
}

func (s *sched) marshalTasks() ([]byte, error) {
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
		return nil, err
	}
	return append(data, '\n'), nil
}
