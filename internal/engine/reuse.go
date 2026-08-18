package engine

// Pre-execute reuse classification. Resume writes these; Inspect only reads.
const (
	reuseReused = "reused"
	reuseRerun  = "rerun"
)

// Closed reuse reasons shared by Inspect and Resume.
const (
	reasonReusedIdentityMatched   = "reused-identity-matched"
	reasonIdentityChanged         = "identity-changed"
	reasonCommandOrScriptChanged  = "command-or-script-changed"
	reasonParamsChanged           = "params-changed"
	reasonEnvChanged              = "env-changed"
	reasonImageChanged            = "image-changed"
	reasonInputFingerprintChanged = "input-fingerprint-changed"
	reasonInputMissing            = "input-missing"
	reasonOutputMissing           = "output-missing"
	reasonPreviousIncomplete      = "previous-incomplete"
	reasonPreviousUnsuccessful    = "previous-unsuccessful"
	reasonDownstreamOfRerun       = "downstream-of-rerun"

	fingerprintsAbsent = "recorded fingerprints were absent"
)

// reuseDecision is the shared Inspect/Resume reuse check result.
// It is not written by Inspect.
type reuseDecision struct {
	Identity  string
	Decision  string
	Reason    string
	Differing []string
}

type remainingClass struct {
	Remaining map[string]bool
	Affected  map[string]bool
	Decision  map[string]reuseDecision
}

func classifyReuse(workspace string, latest jsonTaskState, recorded, current TaskPlan) reuseDecision {
	applyTaskStateDefaults(&latest)
	ident := reservedIdentity(taskPlanFromState(latest))
	dec := reuseDecision{Identity: ident, Decision: reuseRerun}
	switch latest.Status {
	case StatusIncomplete:
		dec.Reason = reasonPreviousIncomplete
		return dec
	case StatusSucceeded:
	default:
		dec.Reason = reasonPreviousUnsuccessful
		return dec
	}

	var differ []string
	if !sameStrings(latest.Command, current.Command) || recorded.Script != current.Script {
		differ = append(differ, "command-or-script")
	}
	if !sameParams(decodeParams(latest.Params), current.Params) {
		differ = append(differ, "params")
	}
	if !sameEnv(recorded.Env, current.Env) {
		differ = append(differ, "env")
	}
	if latest.Image != current.Image {
		differ = append(differ, "image")
	}
	if inputReason, extra := compareInputIdentity(workspace, latest, current); inputReason != "" {
		if inputReason == reasonIdentityChanged {
			differ = append(differ, extra...)
		} else {
			differ = append(differ, inputReason)
		}
	}

	if len(differ) > 1 {
		dec.Reason = reasonIdentityChanged
		dec.Differing = differ
		return dec
	}
	if len(differ) == 1 {
		dec.Differing = differ
		dec.Reason = reuseReasonFor(differ[0])
		return dec
	}
	if publishedMissing(workspace, current.Outputs) {
		dec.Reason = reasonOutputMissing
		return dec
	}
	dec.Decision = reuseReused
	dec.Reason = reasonReusedIdentityMatched
	return dec
}

func reuseReasonFor(component string) string {
	switch component {
	case "command-or-script":
		return reasonCommandOrScriptChanged
	case "params":
		return reasonParamsChanged
	case "env":
		return reasonEnvChanged
	case "image":
		return reasonImageChanged
	case fingerprintsAbsent:
		return reasonIdentityChanged
	case reasonInputFingerprintChanged:
		return reasonInputFingerprintChanged
	case reasonInputMissing:
		return reasonInputMissing
	default:
		return reasonIdentityChanged
	}
}

func compareInputIdentity(workspace string, latest jsonTaskState, current TaskPlan) (string, []string) {
	files := declaredIOFiles(current.Inputs)
	if len(latest.Fingerprints) == 0 {
		if len(files) == 0 {
			return "", nil
		}
		return reasonIdentityChanged, []string{fingerprintsAbsent}
	}
	recorded := hashByPath(latest.Fingerprints)
	missing := false
	changed := false
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		seen[f.path] = true
		path := workspaceFile(workspace, f.path)
		if !regularFile(path) {
			missing = true
			continue
		}
		sum, err := sha256File(path)
		if err != nil || recorded[f.path] != sum {
			changed = true
		}
	}
	for _, h := range latest.Fingerprints {
		if !seen[h.Path] {
			changed = true
		}
	}
	if missing && changed {
		return reasonIdentityChanged, []string{reasonInputMissing, reasonInputFingerprintChanged}
	}
	if missing {
		return reasonInputMissing, []string{reasonInputMissing}
	}
	if changed {
		return reasonInputFingerprintChanged, []string{reasonInputFingerprintChanged}
	}
	return "", nil
}

func publishedMissing(workspace string, outputs []IO) bool {
	for _, out := range outputs {
		for _, f := range namedIOFiles(out) {
			if !regularFile(workspaceFile(workspace, f.path)) {
				return true
			}
		}
	}
	return false
}

func declaredIOFiles(ios []IO) []namedFile {
	var out []namedFile
	for _, io := range ios {
		out = append(out, namedIOFiles(io)...)
	}
	return out
}

func classifyRemaining(workspace string, doc Document, tasks []jsonTaskState) remainingClass {
	latest := latestAttempts(tasks)
	out := remainingClass{
		Remaining: make(map[string]bool, len(latest)),
		Affected:  make(map[string]bool, len(latest)),
		Decision:  make(map[string]reuseDecision, len(latest)),
	}
	byIdent := make(map[string]jsonTaskState, len(latest))
	taskIDOf := make(map[string]string, len(latest))
	identsOfTask := make(map[string][]string)
	for _, st := range latest {
		ident := reservedIdentity(taskPlanFromState(st))
		byIdent[ident] = st
		taskIDOf[ident] = st.ID
		identsOfTask[st.ID] = append(identsOfTask[st.ID], ident)
		if st.Status != StatusSucceeded {
			out.Remaining[ident] = true
		}
		recorded, current := reusePlans(doc, st)
		dec := classifyReuse(workspace, st, recorded, current)
		out.Decision[ident] = dec
		if dec.Decision != reuseReused {
			out.Affected[ident] = true
		}
	}
	for ident, dec := range out.Decision {
		if dec.Decision == reuseReused {
			continue
		}
		markDownstreamAffected(out.Affected, out.Decision, doc, taskIDOf[ident], identsOfTask)
	}
	return out
}

func reusePlans(doc Document, st jsonTaskState) (recorded, current TaskPlan) {
	current = taskPlanFromState(st)
	if t, ok := planTaskByID(doc, st.ID); ok {
		current = t
		current.Instance = st.Instance
		current.ShardIndex = st.ShardIndex
		current.ShardCount = st.ShardCount
		current.Attempt = st.Attempt
	}
	return current, current
}

func markDownstreamAffected(affected map[string]bool, decisions map[string]reuseDecision, doc Document, fromTask string, identsOfTask map[string][]string) {
	if fromTask == "" {
		return
	}
	seen := map[string]bool{fromTask: true}
	stack := []string{fromTask}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, e := range doc.Edges {
			if e.FromTask != id || e.ToTask == "" || seen[e.ToTask] {
				continue
			}
			seen[e.ToTask] = true
			stack = append(stack, e.ToTask)
			for _, ident := range identsOfTask[e.ToTask] {
				if affected[ident] {
					continue
				}
				affected[ident] = true
				dec := decisions[ident]
				dec.Identity = ident
				dec.Decision = reuseRerun
				dec.Reason = reasonDownstreamOfRerun
				decisions[ident] = dec
			}
		}
	}
}

func latestAttempts(tasks []jsonTaskState) []jsonTaskState {
	type slot struct {
		idx     int
		attempt int
	}
	best := make(map[string]slot, len(tasks))
	order := make([]string, 0, len(tasks))
	for i := range tasks {
		st := tasks[i]
		applyTaskStateDefaults(&st)
		ident := reservedIdentity(taskPlanFromState(st))
		cur, ok := best[ident]
		if !ok {
			best[ident] = slot{idx: i, attempt: st.Attempt}
			order = append(order, ident)
			continue
		}
		if st.Attempt >= cur.attempt {
			best[ident] = slot{idx: i, attempt: st.Attempt}
		}
	}
	out := make([]jsonTaskState, 0, len(order))
	for _, ident := range order {
		out = append(out, tasks[best[ident].idx])
	}
	return out
}

func taskPlanFromState(st jsonTaskState) TaskPlan {
	applyTaskStateDefaults(&st)
	return TaskPlan{
		ID:         st.ID,
		Instance:   st.Instance,
		ShardIndex: st.ShardIndex,
		ShardCount: st.ShardCount,
		Attempt:    st.Attempt,
		Command:    st.Command,
		Image:      st.Image,
		Params:     decodeParams(st.Params),
		Resources: ResourcePlan{
			CPU:    st.Resources.CPU,
			Memory: st.Resources.Memory,
		},
	}
}

func planTaskByID(doc Document, id string) (TaskPlan, bool) {
	for _, t := range doc.Tasks {
		if t.ID == id {
			return t, true
		}
	}
	return TaskPlan{}, false
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameParams(a, b []ParamPlan) bool {
	if len(a) != len(b) {
		return false
	}
	left := make(map[string]string, len(a))
	for _, p := range a {
		left[p.Name] = p.Value
	}
	for _, p := range b {
		if left[p.Name] != p.Value {
			return false
		}
		delete(left, p.Name)
	}
	return len(left) == 0
}

func sameEnv(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func decodeParams(in []jsonParam) []ParamPlan {
	out := make([]ParamPlan, 0, len(in))
	for _, p := range in {
		out = append(out, ParamPlan{Name: p.Name, Value: p.Value})
	}
	return out
}
