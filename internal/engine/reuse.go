package engine

import (
	"os"
	"reflect"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

var lookupImageID = exec.LookupImageID

// Pre-execute reuse classification. Resume writes these; Inspect only reads.
const (
	reuseReused             = "reused"
	reuseRerun              = "rerun"
	decisionBlockedUpstream = "blocked-upstream"
)

// Closed graph-diff Change classes. Resume persists these; first Run omits them.
const (
	changeAdded           = "Added"
	changeRemoved         = "Removed"
	changeRewired         = "Rewired"
	changeRepathed        = "Repathed"
	changeIdentityChanged = "IdentityChanged"
	changeUnchanged       = "Unchanged"
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
	Change    string
}

type remainingClass struct {
	Remaining map[string]bool
	Affected  map[string]bool
	Decision  map[string]reuseDecision
}

func classifyReuse(workspace string, latest jsonTaskState, recorded, current TaskPlan) reuseDecision {
	return classifyReuseMode(workspace, latest, recorded, current, true)
}

func classifyReuseMode(workspace string, latest jsonTaskState, recorded, current TaskPlan, hashContent bool) reuseDecision {
	applyTaskStateDefaults(&latest)
	ident := reservedIdentity(taskPlanFromState(latest))
	dec := reuseDecision{Identity: ident, Decision: reuseRerun}
	switch latest.Status {
	case StatusIncomplete:
		dec.Reason = reasonPreviousIncomplete
		return dec
	case StatusSucceeded:
	case StatusSkipped:
		if !sameStrings(latest.Command, current.Command) || latest.Script != current.Script || recorded.Script != current.Script {
			dec.Reason = reasonCommandOrScriptChanged
			return dec
		}
		if !sameParams(decodeParams(latest.Params), current.Params) {
			dec.Reason = reasonParamsChanged
			return dec
		}
		if envIdentityChanged(latest, current) {
			dec.Reason = reasonEnvChanged
			return dec
		}
		dec.Decision = reuseReused
		dec.Reason = reasonReusedIdentityMatched
		return dec
	case StatusNotStarted:
		if current.Scatter != "" && current.Instance == "" {
			if !sameStrings(latest.Command, current.Command) || latest.Script != current.Script || recorded.Script != current.Script {
				dec.Reason = reasonCommandOrScriptChanged
				return dec
			}
			dec.Decision = reuseReused
			dec.Reason = reasonReusedIdentityMatched
			return dec
		}
		dec.Reason = reasonPreviousUnsuccessful
		return dec
	default:
		dec.Reason = reasonPreviousUnsuccessful
		return dec
	}

	var differ []string
	if !sameStrings(latest.Command, current.Command) || latest.Script != current.Script || recorded.Script != current.Script {
		differ = append(differ, "command-or-script")
	}
	if !sameParams(decodeParams(latest.Params), current.Params) {
		differ = append(differ, "params")
	}
	if envIdentityChanged(latest, current) {
		differ = append(differ, "env")
	}
	if execEnvChanged(workspace, latest, current, hashContent) {
		differ = append(differ, "image")
	}
	if current.Gather != "" {
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
		if destReuseMiss(workspace, latest, current, hashContent) {
			dec.Reason = reasonOutputMissing
			return dec
		}
		dec.Decision = reuseReused
		dec.Reason = reasonReusedIdentityMatched
		return dec
	}
	if inputReason, extra := compareInputIdentity(workspace, latest, current, hashContent); inputReason != "" {
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
	if destReuseMiss(workspace, latest, current, hashContent) {
		dec.Reason = reasonOutputMissing
		return dec
	}
	dec.Decision = reuseReused
	dec.Reason = reasonReusedIdentityMatched
	return dec
}

func envIdentityChanged(latest jsonTaskState, current TaskPlan) bool {
	stored := latest.EnvDigest
	cur := planEnvDigest(current)
	if stored == "" {
		stored = envDigest(nil)
	}
	return stored != cur
}

func execEnvChanged(_ string, latest jsonTaskState, current TaskPlan, _ bool) bool {
	if current.Image != "" || latest.Image != "" {
		if latest.Image != current.Image {
			return true
		}
		stored := latest.ImageDigest
		looked := lookupImageID(current.Image)
		if stored == "" || looked == "" || stored != looked {
			return true
		}
		return false
	}
	argv := current.Command
	if current.Script != "" {
		argv = executeArgv(current)
	}
	if len(argv) == 0 {
		return latest.ExecutableSHA256 != ""
	}
	resolved, err := exec.ResolveArgv0(argv[0], current.Env)
	if err != nil {
		return true
	}
	sum, err := sha256File(resolved)
	if err != nil {
		return true
	}
	if latest.ExecutableSHA256 == "" {
		return true
	}
	return latest.ExecutableSHA256 != sum
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

func compareInputIdentity(workspace string, latest jsonTaskState, current TaskPlan, hashContent bool) (string, []string) {
	files := identityFiles(workspace, current.Inputs)
	if len(latest.Fingerprints) == 0 {
		if len(files) == 0 {
			return "", nil
		}
		return reasonIdentityChanged, []string{fingerprintsAbsent}
	}
	recorded := recordByPath(latest.Fingerprints)
	missing := false
	changed := false
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		seen[f.path] = true
		path, present, err := containedRel(workspace, fileSource(f), false)
		if err != nil || !present || !regularFile(path) {
			missing = true
			continue
		}
		rec := recorded[f.path]
		if rec.SHA256 == "" {
			changed = true
			continue
		}
		curCheap, cheapErr := cheapKey(path)
		if !hashContent && cheapErr == nil && hasCheap(rec) && sameCheap(rec, curCheap) {
			continue
		}
		sum, err := sha256File(path)
		if err != nil || sum != rec.SHA256 {
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

func destReuseMiss(workspace string, latest jsonTaskState, current TaskPlan, hashContent bool) bool {
	if len(latest.Checksums) == 0 {
		return true
	}
	for _, h := range latest.Checksums {
		if h.Path == "" {
			continue
		}
		path, present, err := containedRel(workspace, h.Path, false)
		if err != nil || !present || !regularFile(path) {
			return true
		}
		if h.SHA256 == "" {
			return true
		}
		curCheap, cheapErr := cheapKey(path)
		if !hashContent && cheapErr == nil && hasCheap(h) && sameCheap(h, curCheap) {
			continue
		}
		sum, err := sha256File(path)
		if err != nil || sum != h.SHA256 {
			return true
		}
	}
	ident := reservedIdentity(taskPlanFromState(latest))
	known := make(map[string]bool, len(latest.Checksums)+len(latest.Lineage))
	for _, h := range latest.Checksums {
		if h.Path != "" {
			known[h.Path] = true
		}
	}
	for _, lin := range latest.Lineage {
		if lin.Producer == ident && lin.Path != "" {
			known[lin.Path] = true
		}
	}
	for _, f := range identityFiles(workspace, current.Outputs) {
		if !known[f.path] {
			return true
		}
	}
	return false
}

func identityFiles(workspace string, ios []IO) []namedFile {
	var out []namedFile
	for _, io := range ios {
		if isTreeIO(io) {
			probe := io
			probe.Path = treeSourceDir(io)
			out = append(out, treeDestMemberPaths(workspace, probe)...)
			continue
		}
		out = append(out, namedIOFiles(io)...)
	}
	return out
}

func declaredIOFiles(ios []IO) []namedFile {
	var out []namedFile
	for _, io := range ios {
		if isTreeIO(io) {
			out = append(out, namedFile{name: io.Name, path: io.Path})
			continue
		}
		out = append(out, namedIOFiles(io)...)
	}
	return out
}

func classifyRemaining(workspace string, doc Document, tasks []jsonTaskState) remainingClass {
	return classifyRemainingMode(workspace, doc, tasks, true)
}

func remainingAttempt(doc Document, st jsonTaskState, ident string) bool {
	if st.Status == StatusSucceeded || st.Status == StatusSkipped || st.Status == StatusPublishedUnfinalized {
		return false
	}
	if st.Scatter != "" && st.Instance == "" {
		return false
	}
	if t, ok := planTaskByID(doc, st.ID); ok && t.Scatter != "" && st.Instance == "" {
		return false
	}
	_ = ident
	return true
}

func classifyRemainingView(workspace string, doc Document, tasks []jsonTaskState) remainingClass {
	return classifyRemainingMode(workspace, doc, tasks, false)
}

func classifyRemainingMode(workspace string, doc Document, tasks []jsonTaskState, hashContent bool) remainingClass {
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
		if remainingAttempt(doc, st, ident) {
			out.Remaining[ident] = true
		}
		if st.Status == StatusPublishedUnfinalized {
			continue
		}
		recorded, current := reusePlans(doc, st)
		dec := classifyReuseMode(workspace, st, recorded, current, hashContent)
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

func classifyResume(workspace string, recorded, supplied Document, tasks []jsonTaskState) remainingClass {
	latest := latestAttempts(tasks)
	byIdent := make(map[string]jsonTaskState, len(latest))
	for _, st := range latest {
		byIdent[reservedIdentity(taskPlanFromState(st))] = st
	}
	recByIdent := taskPlanByIdentity(recorded)
	out := remainingClass{
		Remaining: make(map[string]bool, len(supplied.Tasks)+len(recByIdent)),
		Affected:  make(map[string]bool, len(supplied.Tasks)+len(recByIdent)),
		Decision:  make(map[string]reuseDecision, len(supplied.Tasks)+len(recByIdent)),
	}
	taskIDOf := make(map[string]string, len(supplied.Tasks))
	identsOfTask := make(map[string][]string)
	suppliedIdent := make(map[string]bool, len(supplied.Tasks))
	for _, t := range supplied.Tasks {
		applyReservedDefaults(&t)
		ident := reservedIdentity(t)
		suppliedIdent[ident] = true
		taskIDOf[ident] = t.ID
		identsOfTask[t.ID] = append(identsOfTask[t.ID], ident)
		rec, inRecorded := recByIdent[ident]
		if !inRecorded {
			out.Decision[ident] = reuseDecision{Identity: ident, Decision: reuseRerun, Change: changeAdded}
			out.Remaining[ident] = true
			out.Affected[ident] = true
			continue
		}
		st, ok := byIdent[ident]
		if !ok {
			st = jsonTaskState{
				ID:         t.ID,
				Instance:   t.Instance,
				ShardIndex: t.ShardIndex,
				ShardCount: t.ShardCount,
				Attempt:    t.Attempt,
				Status:     StatusNotStarted,
				Command:    jsonStrings(t.Command),
				Script:     t.Script,
				Image:      t.Image,
				Params:     encodeParams(t.Params),
				EnvDigest:  envDigest(t.Env),
			}
		}
		if st.Status == StatusPublishedUnfinalized {
			out.Decision[ident] = reuseDecision{Identity: ident, Change: changeUnchanged}
			continue
		}
		dec := classifyReuse(workspace, st, rec, t)
		if st.Status == StatusSkipped && t.When != "" && !whenStillSkips(workspace, supplied, t) {
			dec.Decision = reuseRerun
			dec.Reason = reasonPreviousUnsuccessful
			dec.Differing = nil
		}
		switch {
		case incomingEndpointsDiffer(recorded, supplied, t.ID):
			if dec.Decision == reuseReused {
				dec.Reason = ""
				dec.Differing = nil
			}
			dec.Decision = reuseRerun
			dec.Change = changeRewired
		case waitOrDestDiffer(recorded, supplied, rec, t):
			if dec.Decision == reuseReused {
				dec.Reason = ""
				dec.Differing = nil
			}
			dec.Decision = reuseRerun
			dec.Change = changeRepathed
		case operatorFieldsDiffer(rec, t):
			if dec.Decision == reuseReused {
				dec.Reason = ""
				dec.Differing = nil
			}
			dec.Decision = reuseRerun
			dec.Change = changeIdentityChanged
		case dec.Decision == reuseReused:
			dec.Change = changeUnchanged
		default:
			dec.Change = changeIdentityChanged
		}
		dec.Identity = ident
		out.Decision[ident] = dec
		if remainingAttempt(supplied, st, ident) {
			out.Remaining[ident] = true
		}
		if dec.Decision != reuseReused {
			out.Affected[ident] = true
		}
	}
	for ident := range recByIdent {
		if suppliedIdent[ident] {
			continue
		}
		out.Decision[ident] = reuseDecision{Identity: ident, Change: changeRemoved}
	}
	for ident, dec := range out.Decision {
		if dec.Decision != reuseRerun {
			continue
		}
		markDownstreamAffected(out.Affected, out.Decision, supplied, taskIDOf[ident], identsOfTask)
	}
	return out
}

func whenStillSkips(workspace string, doc Document, task TaskPlan) bool {
	if task.SkipIfFalse != "" {
		if value, ok := paramValue(task, task.SkipIfFalse); ok && value == "false" {
			return true
		}
	}
	if task.SkipIfMissingPort == "" && task.SkipIfMissingPath == "" {
		return false
	}
	path := task.SkipIfMissingPath
	if path == "" && task.SkipIfMissingTask != "" {
		if producer, ok := planTaskByID(doc, task.SkipIfMissingTask); ok {
			if out, ok := findProducerIO(producer, task.SkipIfMissingPort); ok {
				path = out.Path
			}
		}
	}
	if path == "" {
		return false
	}
	abs, present, err := containedRel(workspace, path, false)
	if err != nil || !present || !regularFile(abs) {
		return true
	}
	info, err := os.Lstat(abs)
	return err != nil || info.Size() == 0
}

func taskPlanByIdentity(doc Document) map[string]TaskPlan {
	out := make(map[string]TaskPlan, len(doc.Tasks))
	for _, t := range doc.Tasks {
		applyReservedDefaults(&t)
		out[reservedIdentity(t)] = t
	}
	return out
}

func incomingEndpointsDiffer(recorded, supplied Document, taskID string) bool {
	return !sameStringSet(incomingEndpointSet(recorded, taskID), incomingEndpointSet(supplied, taskID))
}

func waitOrDestDiffer(recorded, supplied Document, rec, cur TaskPlan) bool {
	if !sameStringMap(incomingWaitMap(recorded, rec.ID), incomingWaitMap(supplied, cur.ID)) {
		return true
	}
	return !sameStringSet(destPathSet(rec), destPathSet(cur))
}

func operatorFieldsDiffer(rec, cur TaskPlan) bool {
	return rec.Scatter != cur.Scatter ||
		rec.Gather != cur.Gather ||
		rec.When != cur.When ||
		rec.ScatterFromKind != cur.ScatterFromKind ||
		rec.ScatterFromTask != cur.ScatterFromTask ||
		rec.ScatterFromPort != cur.ScatterFromPort ||
		rec.ScatterFromPath != cur.ScatterFromPath ||
		!sameStrings(rec.ScatterMembers, cur.ScatterMembers) ||
		!sameStrings(rec.ScatterMemberPaths, cur.ScatterMemberPaths) ||
		!reflect.DeepEqual(encodeSpecs(rec.ScatterMemberSpecs), encodeSpecs(cur.ScatterMemberSpecs)) ||
		deferredBindsDiffer(rec, cur) ||
		rec.SkipIfMissingTask != cur.SkipIfMissingTask ||
		rec.SkipIfMissingPort != cur.SkipIfMissingPort ||
		rec.SkipIfMissingPath != cur.SkipIfMissingPath ||
		rec.SkipIfFalse != cur.SkipIfFalse
}

func deferredBindsDiffer(rec, cur TaskPlan) bool {
	if rec.Scatter == "" && cur.Scatter == "" {
		return false
	}
	return !reflect.DeepEqual(encodeIOs(rec.Inputs), encodeIOs(cur.Inputs)) ||
		!reflect.DeepEqual(encodeIOs(rec.Outputs), encodeIOs(cur.Outputs))
}

func incomingEndpointSet(doc Document, taskID string) map[string]bool {
	out := make(map[string]bool)
	for _, e := range doc.Edges {
		if e.ToTask != taskID {
			continue
		}
		out[endpointKey(e)] = true
	}
	return out
}

func incomingWaitMap(doc Document, taskID string) map[string]string {
	out := make(map[string]string)
	for _, e := range doc.Edges {
		if e.ToTask != taskID {
			continue
		}
		out[endpointKey(e)] = strings.Join(e.Wait, "\x01")
	}
	return out
}

func endpointKey(e Edge) string {
	return e.FromTask + "\x00" + e.FromPort + "\x00" + e.ToTask + "\x00" + e.ToPort
}

func destPathSet(t TaskPlan) map[string]bool {
	out := make(map[string]bool)
	for _, f := range declaredIOFiles(t.Outputs) {
		out[f.path] = true
	}
	return out
}

func sameStringSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func sameStringMap(a, b map[string]string) bool {
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

func reusePlans(doc Document, st jsonTaskState) (recorded, current TaskPlan) {
	recorded = taskPlanFromState(st)
	current = recorded
	if t, ok := planTaskByID(doc, st.ID); ok {
		current = t
		current.Instance = st.Instance
		current.ShardIndex = st.ShardIndex
		current.ShardCount = st.ShardCount
		current.Attempt = st.Attempt
	}
	return recorded, current
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

func priorAttempts(tasks []jsonTaskState) []jsonTaskState {
	latest := latestAttempts(tasks)
	keep := make(map[string]int, len(latest))
	for _, st := range latest {
		applyTaskStateDefaults(&st)
		keep[reservedIdentity(taskPlanFromState(st))] = st.Attempt
	}
	var out []jsonTaskState
	for _, st := range tasks {
		applyTaskStateDefaults(&st)
		ident := reservedIdentity(taskPlanFromState(st))
		if keep[ident] != st.Attempt {
			out = append(out, st)
		}
	}
	return out
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
		ID:               st.ID,
		Instance:         st.Instance,
		ShardIndex:       st.ShardIndex,
		ShardCount:       st.ShardCount,
		Attempt:          st.Attempt,
		Command:          st.Command,
		Script:           st.Script,
		Image:            st.Image,
		Params:           decodeParams(st.Params),
		EnvDigest:        st.EnvDigest,
		ExecutablePath:   st.ExecutablePath,
		ExecutableSHA256: st.ExecutableSHA256,
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
	if duplicateParamNames(a) || duplicateParamNames(b) {
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

func duplicateParamNames(params []ParamPlan) bool {
	seen := make(map[string]bool, len(params))
	for _, p := range params {
		if seen[p.Name] {
			return true
		}
		seen[p.Name] = true
	}
	return false
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func decodeParams(in []jsonParam) []ParamPlan {
	out := make([]ParamPlan, 0, len(in))
	for _, p := range in {
		out = append(out, ParamPlan{Name: p.Name, Value: p.Value})
	}
	return out
}
