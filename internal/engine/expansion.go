package engine

import (
	intpath "github.com/HahyeonJeon/gobble/internal/path"
	"strings"
)

func isScatterTemplate(t TaskPlan) bool {
	return t.Scatter != "" && t.Instance == ""
}

func isScatterTemplateState(st *jsonTaskState) bool {
	return st != nil && st.Scatter != "" && st.Instance == ""
}

func isNonFailureState(st *jsonTaskState) bool {
	if st == nil {
		return false
	}
	switch st.Status {
	case StatusSucceeded, StatusSkipped, StatusBlocked, StatusPublishedUnfinalized:
		return true
	case StatusNotStarted:
		return isScatterTemplateState(st)
	default:
		return isScatterTemplateState(st)
	}
}

func isKnownTerminal(status string) bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusSkipped, StatusBlocked, StatusIncomplete, StatusPublishedUnfinalized:
		return true
	default:
		return false
	}
}

func (s *sched) failureOf(ident, unit string, st *jsonTaskState) []Defect {
	if st == nil {
		return []Defect{{Code: DefectFailed, Unit: unit, Message: "task failed"}}
	}
	switch st.Status {
	case StatusSucceeded, StatusBlocked, StatusSkipped, StatusPublishedUnfinalized:
		return nil
	case StatusUnknown:
		return []Defect{{
			Code:    DefectUnknownBackend,
			Unit:    ident,
			Message: "unknown backend",
		}}
	case StatusNotStarted:
		if isScatterTemplateState(st) {
			return nil
		}
		return []Defect{{
			Code:    DefectFailed,
			Unit:    unit,
			Message: "not started",
		}}
	default:
		msg := "task failed"
		if st.Error != nil && st.Error.Message != "" {
			msg = st.Error.Message
		}
		code := DefectFailed
		if st.Reason == "path escapes directory" || msg == "path escapes directory" {
			code = DefectInvalidPath
		}
		if st.Reason == "never-ready" || msg == "never-ready" {
			code = DefectNeverReady
		}
		return []Defect{{
			Code:    code,
			Unit:    unit,
			Message: msg,
		}}
	}
}

func (s *sched) isScatterTaskID(id string) bool {
	t, ok := s.taskByID(id)
	return ok && isScatterTemplate(t)
}

func (s *sched) scatterTemplateState(id string) *jsonTaskState {
	t, ok := s.taskByID(id)
	if !ok {
		return nil
	}
	return s.tasks[reservedIdentity(t)]
}

func (s *sched) sameScatterID(a, b string) bool {
	ta, oka := s.taskByID(a)
	tb, okb := s.taskByID(b)
	return oka && okb && sameScatter(ta, tb)
}

func sameScatter(a, b TaskPlan) bool {
	return a.Scatter != "" && a.Scatter == b.Scatter &&
		a.ScatterFromTask == b.ScatterFromTask &&
		a.ScatterFromPort == b.ScatterFromPort
}

func (s *sched) scatterMemberReady(id, key string) bool {
	t, ok := s.taskByID(id)
	if !ok {
		return false
	}
	t.Instance = key
	t.ShardIndex = DefaultShardIndex
	applyReservedDefaults(&t)
	ms := s.tasks[reservedIdentity(t)]
	return ms != nil && ms.Status == StatusSucceeded
}

func (s *sched) scatterMembersReady(id string) (bool, *Defect) {
	st := s.scatterTemplateState(id)
	if st == nil || st.Expansion == nil {
		return false, nil
	}
	if len(st.Expansion.Members) == 0 {
		d := Defect{Code: DefectNeverReady, Unit: id, Message: "never-ready"}
		return false, &d
	}
	allSucceeded := true
	for _, key := range st.Expansion.Members {
		member, ok := s.taskByID(id)
		if !ok {
			return false, nil
		}
		member.Instance = key
		member.ShardIndex = DefaultShardIndex
		applyReservedDefaults(&member)
		ident := reservedIdentity(member)
		ms := s.tasks[ident]
		if ms == nil {
			return false, nil
		}
		if ms.Status == StatusRunning || ms.Status == StatusUnknown || ms.Status == StatusNotStarted {
			return false, nil
		}
		if !isKnownTerminal(ms.Status) {
			return false, nil
		}
		if ms.Status != StatusSucceeded {
			allSucceeded = false
		}
	}
	if !allSucceeded {
		return false, nil
	}
	return true, nil
}

func (s *sched) maybeExpand() *Defect {
	for _, t := range s.doc.Tasks {
		if !isScatterTemplate(t) {
			continue
		}
		st := s.tasks[reservedIdentity(t)]
		if st == nil || st.Status == StatusSkipped {
			continue
		}
		if st.Expansion != nil {
			if d := s.seedMembers(t, st.Expansion.Members); d != nil {
				return d
			}
			continue
		}
		keys, producer, ready, d := s.expansionKeys(t)
		if d != nil {
			return d
		}
		if !ready {
			continue
		}
		st.Expansion = &jsonExpansion{Producer: producer, Members: keys}
		if st.Expansion.Members == nil {
			st.Expansion.Members = []string{}
		}
		s.notePersist(s.persistControl())
		if d := s.seedMembers(t, st.Expansion.Members); d != nil {
			return d
		}
	}
	return nil
}

func (s *sched) seedMembers(t TaskPlan, keys []string) *Defect {
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if seen[key] {
			return &Defect{Code: DefectInvalidValue, Unit: t.ID, Message: "invalid-value"}
		}
		seen[key] = true
		member := cloneTaskPlan(t)
		member.Instance = key
		member.ShardIndex = DefaultShardIndex
		applyReservedDefaults(&member)
		ident := reservedIdentity(member)
		if _, _, err := containedRel(s.workspace, isolateRel(member), false); err != nil {
			d := escapeDefect(t.ID, key)
			return &d
		}
		if s.tasks[ident] != nil {
			continue
		}
		st := initialTask(member)
		if s.resume != nil {
			parentIdent := reservedIdentity(t)
			if pdec, ok := s.resume[parentIdent]; ok && pdec.Decision == reuseRerun {
				dec := reuseDecision{
					Identity:  ident,
					Decision:  reuseRerun,
					Change:    pdec.Change,
					Reason:    pdec.Reason,
					Differing: append([]string(nil), pdec.Differing...),
				}
				s.resume[ident] = dec
				applyResumeDecision(&st, dec, true)
			}
		}
		s.tasks[ident] = &st
	}
	return nil
}

func (s *sched) expansionKeys(t TaskPlan) ([]string, string, bool, *Defect) {
	producer := t.ScatterFromTask
	if producer == "" {
		producer = t.ScatterFromPort
	}
	if t.ScatterFromTask != "" {
		up := s.stateByTaskID(t.ScatterFromTask)
		if up == nil || up.Status != StatusSucceeded {
			return nil, producer, false, nil
		}
		if s.resume != nil {
			upTask, ok := s.taskByID(t.ScatterFromTask)
			if !ok {
				return nil, producer, false, nil
			}
			ident := reservedIdentity(upTask)
			if s.resume[ident].Decision == reuseRerun && !s.succeededThisResume(ident) {
				return nil, producer, false, nil
			}
		}
		keys, d := s.producerMemberKeys(t)
		return keys, producer, d == nil, d
	}
	if t.ScatterFromKind == ArtifactTree {
		keys, d := s.staticTreeKeys(t)
		return keys, producer, d == nil, d
	}
	if t.ScatterMembers != nil {
		return append([]string(nil), t.ScatterMembers...), producer, true, nil
	}
	return []string{}, producer, true, nil
}

func (s *sched) producerMemberKeys(t TaskPlan) ([]string, *Defect) {
	src, ok := s.taskByID(t.ScatterFromTask)
	if !ok {
		return nil, &Defect{Code: DefectMissingInput, Unit: t.ID, Message: "missing input"}
	}
	io, ok := findProducerIO(src, t.ScatterFromPort)
	if !ok {
		return nil, &Defect{Code: DefectMissingInput, Unit: t.ID, Message: "missing input"}
	}
	switch t.ScatterFromKind {
	case ArtifactGroup:
		if io.Members == nil {
			return []string{}, nil
		}
		keys := make([]string, 0, len(io.Members))
		for _, m := range io.Members {
			keys = append(keys, m.Name)
		}
		return keys, nil
	case ArtifactTree:
		return treeMemberKeys(s.workspace, io, t.ID)
	default:
		path := io.Path
		if path == "" {
			return []string{}, nil
		}
		return []string{path}, nil
	}
}

func (s *sched) staticTreeKeys(t TaskPlan) ([]string, *Defect) {
	if t.ScatterFromPath != "" {
		io := IO{Kind: ArtifactTree, Path: t.ScatterFromPath, Source: t.ScatterFromPath}
		return treeMemberKeys(s.workspace, io, t.ID)
	}
	var io IO
	found := false
	for _, in := range t.Inputs {
		if s.ioFromScatterProducer(t, in.Name) || in.Name == t.ScatterFromPort {
			io = in
			found = true
			break
		}
	}
	if !found {
		return []string{}, nil
	}
	return treeMemberKeys(s.workspace, io, t.ID)
}

func treeMemberKeys(workspace string, io IO, unit string) ([]string, *Defect) {
	files := treeSourceMemberPaths(workspace, io)
	keys := make([]string, 0, len(files))
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		if seen[f.name] {
			return nil, &Defect{Code: DefectInvalidValue, Unit: unit, Message: "invalid-value"}
		}
		seen[f.name] = true
		keys = append(keys, f.name)
	}
	return keys, nil
}

func findProducerIO(t TaskPlan, port string) (IO, bool) {
	for _, out := range t.Outputs {
		if out.Name == port {
			return out, true
		}
	}
	for _, in := range t.Inputs {
		if in.Name == port {
			return in, true
		}
	}
	return IO{}, false
}

func (s *sched) specializeGatherIO(t *TaskPlan) {
	if t.Gather == "" {
		return
	}
	for i := range t.Inputs {
		fromTask, fromPort := s.edgeFrom(t.ID, t.Inputs[i].Name)
		if fromTask == "" || !s.isScatterTaskID(fromTask) {
			continue
		}
		st := s.scatterTemplateState(fromTask)
		if st == nil || st.Expansion == nil {
			continue
		}
		src, ok := s.taskByID(fromTask)
		if !ok {
			continue
		}
		members := make([]IOMember, 0, len(st.Expansion.Members))
		for _, key := range st.Expansion.Members {
			member := cloneTaskPlan(src)
			member.Instance = key
			member.ShardIndex = DefaultShardIndex
			applyReservedDefaults(&member)
			s.specializeMemberIO(&member, key)
			io, ok := findProducerIO(member, fromPort)
			if !ok || io.Path == "" {
				continue
			}
			members = append(members, IOMember{Name: key, Path: io.Path, Source: io.Path, Spec: io.Spec})
		}
		if len(members) == 0 {
			continue
		}
		t.Inputs[i].Kind = ArtifactGroup
		t.Inputs[i].Path = ""
		t.Inputs[i].Source = ""
		t.Inputs[i].Members = members
		t.Inputs[i].Manifest = ""
	}
}

func (s *sched) edgeFrom(toTask, toPort string) (string, string) {
	for _, e := range s.doc.Edges {
		if e.ToTask == toTask && e.ToPort == toPort {
			return e.FromTask, e.FromPort
		}
	}
	return "", ""
}

func (s *sched) specializeMemberIO(t *TaskPlan, key string) {
	s.specializeMemberChain(t, key, make(map[string]bool))
}

func (s *sched) specializeMemberChain(t *TaskPlan, key string, seen map[string]bool) {
	if seen[t.ID] {
		return
	}
	seen[t.ID] = true
	defer delete(seen, t.ID)
	memberPath, memberSpec := s.memberSource(t, key)
	if memberPath != "" || !isZeroPath(memberSpec) {
		for i := range t.Inputs {
			if s.ioFromScatterProducer(*t, t.Inputs[i].Name) {
				s.applyMemberIO(&t.Inputs[i], memberPath, memberSpec, key, true, false)
			}
		}
		for i := range t.Outputs {
			if s.ioFromScatterProducer(*t, t.Outputs[i].Name) {
				s.applyMemberIO(&t.Outputs[i], memberPath, memberSpec, key, false, true)
			}
		}
	}
	s.specializeSiblingChain(t, key, true, seen)
	s.specializeSiblingChain(t, key, false, seen)
}

func (s *sched) specializeSiblingChain(t *TaskPlan, key string, inputs bool, seen map[string]bool) {
	ports := t.Outputs
	if inputs {
		ports = t.Inputs
	}
	for i := range ports {
		fromTask, fromPort := s.edgeFrom(t.ID, ports[i].Name)
		if fromTask == "" || fromTask == t.ID {
			continue
		}
		src, ok := s.taskByID(fromTask)
		if !ok || !sameScatter(*t, src) {
			continue
		}
		sibling := cloneTaskPlan(src)
		sibling.Instance = key
		sibling.ShardIndex = DefaultShardIndex
		applyReservedDefaults(&sibling)
		s.specializeMemberChain(&sibling, key, seen)
		io, ok := findProducerIO(sibling, fromPort)
		if !ok || io.Path == "" {
			continue
		}
		path := io.Path
		spec := io.Spec
		if inputs {
			s.applyMemberIO(&t.Inputs[i], path, spec, key, true, true)
			continue
		}
		s.applyMemberIO(&t.Outputs[i], path, spec, key, false, true)
	}
}

func (s *sched) ioFromScatterProducer(t TaskPlan, port string) bool {
	if t.Scatter == "" {
		return false
	}
	for _, e := range s.doc.Edges {
		if e.ToTask != t.ID || e.ToPort != port {
			continue
		}
		if e.FromTask == t.ScatterFromTask && e.FromPort == t.ScatterFromPort {
			return true
		}
	}
	return false
}

func (s *sched) memberSource(t *TaskPlan, key string) (string, Path) {
	if t.ScatterFromTask != "" {
		src, ok := s.taskByID(t.ScatterFromTask)
		if !ok {
			return key, literalPath(key)
		}
		io, ok := findProducerIO(src, t.ScatterFromPort)
		if !ok {
			return key, literalPath(key)
		}
		switch t.ScatterFromKind {
		case ArtifactGroup:
			for _, m := range io.Members {
				if m.Name == key {
					path := m.Path
					if m.Source != "" {
						path = m.Source
					}
					return path, m.Spec
				}
			}
		case ArtifactTree:
			dir := treeSourceDir(io)
			path := key
			if dir != "" {
				path = strings.TrimSuffix(strings.ReplaceAll(dir, `\`, "/"), "/") + "/" + key
			}
			return path, literalPath(path)
		default:
			path := io.Path
			if path == "" {
				path = key
			}
			return path, io.Spec
		}
		return key, literalPath(key)
	}
	for i, name := range t.ScatterMembers {
		if name != key {
			continue
		}
		path := key
		if i < len(t.ScatterMemberPaths) && t.ScatterMemberPaths[i] != "" {
			path = t.ScatterMemberPaths[i]
		}
		return path, literalPath(path)
	}
	if t.ScatterFromKind == ArtifactFile && len(t.ScatterMembers) == 1 {
		return t.ScatterMembers[0], literalPath(t.ScatterMembers[0])
	}
	for _, in := range t.Inputs {
		if !s.ioFromScatterProducer(*t, in.Name) {
			continue
		}
		if in.Members != nil {
			for _, m := range in.Members {
				if m.Name == key {
					path := m.Path
					if m.Source != "" {
						path = m.Source
					}
					return path, m.Spec
				}
			}
		}
		if t.ScatterFromKind == ArtifactTree {
			dir := treeSourceDir(in)
			if dir == "" {
				dir = t.ScatterFromPath
			}
			path := key
			if dir != "" {
				path = strings.TrimSuffix(strings.ReplaceAll(dir, `\`, "/"), "/") + "/" + key
			}
			return path, literalPath(path)
		}
	}
	if t.ScatterFromKind == ArtifactTree && t.ScatterFromPath != "" {
		dir := strings.TrimSuffix(strings.ReplaceAll(t.ScatterFromPath, `\`, "/"), "/")
		path := key
		if dir != "" {
			path = dir + "/" + key
		}
		return path, literalPath(path)
	}
	return key, literalPath(key)
}

func (s *sched) applyMemberIO(io *IO, memberPath string, memberSpec Path, key string, asInput, treeMember bool) {
	if io.Kind == ArtifactTree && treeMember {
		root := strings.TrimSuffix(strings.ReplaceAll(io.Path, `\`, "/"), "/")
		path := key
		if root != "" {
			path = root + "/" + key
		}
		io.Path = path
		io.Manifest = path + "/" + treeManifestName
		io.Members = nil
		if asInput && memberPath != "" && memberPath != path {
			io.Source = memberPath
		} else {
			io.Source = ""
		}
		return
	}
	from := memberSpec
	if isZeroPath(from) {
		from = literalPath(memberPath)
	}
	classified := pathFromSpec(intpath.Classify(io.Spec.spec(), from.spec(), intpath.DeriveAppend))
	path, d := classified.Render()
	if d != nil || path == "" {
		path = memberPath
		classified = from
	}
	io.Kind = ArtifactFile
	io.Members = nil
	io.Manifest = ""
	io.Spec = classified
	if asInput {
		io.Source = memberPath
		io.Path = path
		if io.Path == io.Source {
			io.Source = ""
		}
		return
	}
	io.Path = path
	io.Source = ""
}

func literalPath(path string) Path {
	return Path{Literal: true, Opaque: path}
}

func (s *sched) latestHistoryAttempt(ident string) int {
	best := 0
	for _, st := range s.history {
		h := reservedIdentity(taskPlanFromState(st))
		if h == ident && st.Attempt > best {
			best = st.Attempt
		}
	}
	return best
}
