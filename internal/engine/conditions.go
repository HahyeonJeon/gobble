package engine

import (
	"os"
	"sort"
	"time"
)

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
		case StatusSucceeded, StatusSkipped, StatusNotStarted, StatusUnknown:
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
