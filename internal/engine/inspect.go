package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	viewRun       = "run"
	viewInstances = "instances"
	viewErrors    = "errors"
	viewLogs      = "logs"
	viewTiming    = "timing"
	viewDAG       = "dag"
	viewLineage   = "lineage"
	viewRemaining = "remaining"
	viewReuse     = "reuse"

	inspectLogTail = 4096
)

// Inspect returns one read-only view of workspace. It does not occupy,
// create, or rewrite control files. Each control file is one atomic
// snapshot; cross-file combinations may be mid-update.
func Inspect(workspace, view, instance string) ([]byte, []Defect) {
	if d := inspectWorkspace(workspace); len(d) > 0 {
		return nil, d
	}
	if !knownInspectView(view) {
		return nil, []Defect{{
			Code:    DefectNotFound,
			Message: "view not found",
			Unit:    view,
		}}
	}
	run, d := readInspectRun(workspace)
	if len(d) > 0 {
		return nil, d
	}
	if d := unsupportedControlSchema(workspace, run.SchemaVersion); len(d) > 0 {
		return nil, d
	}
	plan, hasPlan, d := readInspectPlan(workspace)
	if len(d) > 0 {
		return nil, d
	}
	tasks, d := readInspectTasks(workspace)
	if len(d) > 0 {
		return nil, d
	}
	doc := Document{}
	if hasPlan {
		doc = documentFromPlan(plan)
	}
	latest := latestAttempts(tasks)
	selected, d := selectLatest(tasks, instance)
	if len(d) > 0 {
		return nil, d
	}
	switch view {
	case viewRun:
		return marshalInspect(inspectRunView(run))
	case viewInstances:
		return marshalJSONL(inspectInstanceRecords(selected, doc, run.SchemaVersion))
	case viewErrors:
		return marshalInspect(inspectErrorsView(selected, run.SchemaVersion))
	case viewLogs:
		return marshalInspect(inspectLogsView(workspace, selected, run.SchemaVersion))
	case viewTiming:
		return marshalInspect(inspectTimingView(run, selected))
	case viewDAG:
		return marshalInspect(inspectDAGView(plan, hasPlan, run.SchemaVersion))
	case viewLineage:
		return marshalInspect(inspectLineageView(latest, instance, run.SchemaVersion))
	case viewRemaining:
		return marshalJSONL(inspectRemainingRecords(workspace, doc, latest, instance, run.SchemaVersion))
	case viewReuse:
		return marshalJSONL(inspectReuseRecords(selected, run.SchemaVersion))
	default:
		return nil, []Defect{{
			Code:    DefectNotFound,
			Message: "view not found",
			Unit:    view,
		}}
	}
}

func knownInspectView(view string) bool {
	switch view {
	case viewRun, viewInstances, viewErrors, viewLogs, viewTiming, viewDAG, viewLineage, viewRemaining, viewReuse:
		return true
	default:
		return false
	}
}

func inspectWorkspace(workspace string) []Defect {
	if workspace == "" {
		return []Defect{{
			Code:    DefectNotFound,
			Message: "workspace not found",
		}}
	}
	info, err := os.Stat(workspace)
	if err != nil || !info.IsDir() {
		return []Defect{{
			Code:    DefectNotFound,
			Message: "workspace not found",
			Paths:   []string{workspace},
		}}
	}
	return nil
}

func readInspectRun(workspace string) (jsonRun, []Defect) {
	path := filepath.Join(workspace, ControlDir, RunIdentityFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonRun{}, []Defect{{
				Code:    DefectNotFound,
				Message: "run not found",
				Paths:   []string{ControlDir + "/" + RunIdentityFile},
			}}
		}
		return jsonRun{}, pathDefects(err)
	}
	var run jsonRun
	if err := json.Unmarshal(data, &run); err != nil {
		return jsonRun{}, []Defect{{
			Code:    DefectInvalidPath,
			Message: "run identity is not usable",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	return run, nil
}

func readInspectPlan(workspace string) (jsonPlan, bool, []Defect) {
	path := filepath.Join(workspace, ControlDir, PlanFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonPlan{}, false, nil
		}
		return jsonPlan{}, false, pathDefects(err)
	}
	var plan jsonPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return jsonPlan{}, false, []Defect{{
			Code:    DefectInvalidPath,
			Message: "plan is not usable",
			Paths:   []string{ControlDir + "/" + PlanFile},
		}}
	}
	return plan, true, nil
}

func readInspectTasks(workspace string) ([]jsonTaskState, []Defect) {
	path := filepath.Join(workspace, ControlDir, TasksFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, pathDefects(err)
	}
	var file jsonTasksFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, []Defect{{
			Code:    DefectInvalidPath,
			Message: "tasks are not usable",
			Paths:   []string{ControlDir + "/" + TasksFile},
		}}
	}
	applyLegacyTaskSlots(&file)
	return file.Tasks, nil
}

func selectLatest(tasks []jsonTaskState, instance string) ([]jsonTaskState, []Defect) {
	latest := latestAttempts(tasks)
	if instance == "" {
		return latest, nil
	}
	for _, st := range latest {
		if reservedIdentity(taskPlanFromState(st)) == instance {
			return []jsonTaskState{st}, nil
		}
	}
	return nil, []Defect{{
		Code:    DefectNotFound,
		Message: "instance not found",
		Unit:    instance,
	}}
}

type inspectRunDoc struct {
	ID            string           `json:"id"`
	Status        string           `json:"status"`
	SchemaVersion int              `json:"schema_version"`
	Occupancy     inspectOccupancy `json:"occupancy"`
	Started       string           `json:"started"`
	Ended         string           `json:"ended,omitempty"`
}

type inspectOccupancy struct {
	Active  bool   `json:"active"`
	Host    string `json:"host,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Started string `json:"started,omitempty"`
	Closed  string `json:"closed,omitempty"`
	Live    bool   `json:"live"`
}

func inspectRunView(run jsonRun) inspectRunDoc {
	occ := inspectOccupancy{Active: occupancyIsActive(run)}
	if run.Occupancy != nil {
		occ.Host = run.Occupancy.Host
		occ.PID = run.Occupancy.PID
		occ.Started = run.Occupancy.Started
		occ.Closed = run.Occupancy.Closed
	}
	occ.Live = ownerLooksLive(run)
	return inspectRunDoc{
		ID:            run.ID,
		Status:        run.Status,
		SchemaVersion: run.SchemaVersion,
		Occupancy:     occ,
		Started:       run.Started,
		Ended:         run.Ended,
	}
}

func ownerLooksLive(run jsonRun) bool {
	if !occupancyIsActive(run) || run.Occupancy == nil {
		return false
	}
	host, err := currentHost()
	if err != nil {
		return false
	}
	return run.Occupancy.Host == host && pidExists(run.Occupancy.PID)
}

type inspectInstanceDoc struct {
	SchemaVersion int               `json:"schema_version"`
	Identity      string            `json:"identity"`
	Status        string            `json:"status"`
	Executor      string            `json:"executor"`
	Image         string            `json:"image"`
	Command       []string          `json:"command"`
	Script        string            `json:"script,omitempty"`
	Params        []jsonParam       `json:"params"`
	Env           map[string]string `json:"env,omitempty"`
	Resources     jsonResources     `json:"resources"`
	Instance      string            `json:"instance"`
	ShardIndex    int               `json:"shard_index"`
	ShardCount    int               `json:"shard_count"`
	Attempt       int               `json:"attempt"`
	Stdout        string            `json:"stdout,omitempty"`
	Stderr        string            `json:"stderr,omitempty"`
	Decision      string            `json:"decision,omitempty"`
	Reason        string            `json:"reuse_reason,omitempty"`
}

func inspectInstanceRecords(tasks []jsonTaskState, doc Document, schema int) []inspectInstanceDoc {
	out := make([]inspectInstanceDoc, 0, len(tasks))
	for _, st := range tasks {
		applyTaskStateDefaults(&st)
		rec := inspectInstanceDoc{
			SchemaVersion: schema,
			Identity:      reservedIdentity(taskPlanFromState(st)),
			Status:        st.Status,
			Executor:      st.Executor,
			Image:         st.Image,
			Command:       jsonStrings(st.Command),
			Params:        st.Params,
			Resources:     st.Resources,
			Instance:      st.Instance,
			ShardIndex:    st.ShardIndex,
			ShardCount:    st.ShardCount,
			Attempt:       st.Attempt,
			Stdout:        st.Stdout,
			Stderr:        st.Stderr,
			Decision:      st.Decision,
			Reason:        reuseReasonOf(st),
		}
		if t, ok := planTaskByID(doc, st.ID); ok {
			rec.Script = t.Script
			rec.Env = t.Env
			if rec.Image == "" {
				rec.Image = t.Image
			}
			if len(rec.Command) == 0 && len(t.Command) > 0 {
				rec.Command = jsonStrings(t.Command)
			}
		}
		out = append(out, rec)
	}
	return out
}

func reuseReasonOf(st jsonTaskState) string {
	if st.Decision == "" {
		return ""
	}
	if st.ReuseReason != "" {
		return st.ReuseReason
	}
	return st.Reason
}

type inspectErrorsDoc struct {
	SchemaVersion int            `json:"schema_version"`
	Errors        []inspectError `json:"errors"`
}

type inspectError struct {
	Identity string `json:"identity"`
	Unit     string `json:"unit"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

func inspectErrorsView(tasks []jsonTaskState, schema int) inspectErrorsDoc {
	out := inspectErrorsDoc{SchemaVersion: schema, Errors: []inspectError{}}
	for _, st := range tasks {
		if st.Status == StatusSucceeded {
			continue
		}
		msg := st.Reason
		unit := st.ID
		if st.Error != nil {
			if st.Error.Message != "" {
				msg = st.Error.Message
			}
			if st.Error.Unit != "" {
				unit = st.Error.Unit
			}
		}
		out.Errors = append(out.Errors, inspectError{
			Identity: reservedIdentity(taskPlanFromState(st)),
			Unit:     unit,
			Status:   st.Status,
			Message:  msg,
		})
	}
	return out
}

type inspectLogsDoc struct {
	SchemaVersion int          `json:"schema_version"`
	Logs          []inspectLog `json:"logs"`
}

type inspectLog struct {
	Identity   string `json:"identity"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	StdoutSize int64  `json:"stdout_size"`
	StderrSize int64  `json:"stderr_size"`
	StdoutTail string `json:"stdout_tail,omitempty"`
	StderrTail string `json:"stderr_tail,omitempty"`
}

func inspectLogsView(workspace string, tasks []jsonTaskState, schema int) inspectLogsDoc {
	out := inspectLogsDoc{SchemaVersion: schema, Logs: []inspectLog{}}
	for _, st := range tasks {
		rec := inspectLog{
			Identity: reservedIdentity(taskPlanFromState(st)),
			Stdout:   st.Stdout,
			Stderr:   st.Stderr,
		}
		rec.StdoutSize, rec.StdoutTail = logPointer(workspace, st.Stdout)
		rec.StderrSize, rec.StderrTail = logPointer(workspace, st.Stderr)
		out.Logs = append(out.Logs, rec)
	}
	return out
}

func logPointer(workspace, rel string) (int64, string) {
	if rel == "" {
		return 0, ""
	}
	path := filepath.Join(workspace, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return 0, ""
	}
	return info.Size(), readTail(path, inspectLogTail)
}

func readTail(path string, limit int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	size := info.Size()
	start := int64(0)
	if size > limit {
		start = size - limit
	}
	if start > 0 {
		if _, err := f.Seek(start, 0); err != nil {
			return ""
		}
	}
	n := size - start
	buf := make([]byte, n)
	got, err := f.Read(buf)
	if err != nil && got == 0 {
		return ""
	}
	return string(buf[:got])
}

type inspectTimingDoc struct {
	SchemaVersion int             `json:"schema_version"`
	Started       string          `json:"started"`
	Ended         string          `json:"ended,omitempty"`
	Instances     []inspectTiming `json:"instances"`
}

type inspectTiming struct {
	Identity string `json:"identity"`
	Started  string `json:"started,omitempty"`
	Ended    string `json:"ended,omitempty"`
}

func inspectTimingView(run jsonRun, tasks []jsonTaskState) inspectTimingDoc {
	out := inspectTimingDoc{
		SchemaVersion: run.SchemaVersion,
		Started:       run.Started,
		Ended:         run.Ended,
		Instances:     []inspectTiming{},
	}
	for _, st := range tasks {
		out.Instances = append(out.Instances, inspectTiming{
			Identity: reservedIdentity(taskPlanFromState(st)),
			Started:  st.Started,
			Ended:    st.Ended,
		})
	}
	return out
}

type inspectDAGDoc struct {
	SchemaVersion int        `json:"schema_version"`
	Nodes         []string   `json:"nodes"`
	Edges         []jsonEdge `json:"edges"`
}

func inspectDAGView(plan jsonPlan, hasPlan bool, schema int) inspectDAGDoc {
	dag := jsonDAG{Nodes: []string{}, Edges: []jsonEdge{}}
	if hasPlan {
		dag = plan.DAG
		if dag.Nodes == nil {
			dag.Nodes = []string{}
		}
		if dag.Edges == nil {
			dag.Edges = []jsonEdge{}
		}
	}
	return inspectDAGDoc{SchemaVersion: schema, Nodes: dag.Nodes, Edges: dag.Edges}
}

type inspectLineageDoc struct {
	SchemaVersion int           `json:"schema_version"`
	Lineage       []jsonLineage `json:"lineage"`
}

func inspectLineageView(tasks []jsonTaskState, instance string, schema int) inspectLineageDoc {
	out := inspectLineageDoc{SchemaVersion: schema, Lineage: []jsonLineage{}}
	for _, st := range tasks {
		ident := reservedIdentity(taskPlanFromState(st))
		for _, lin := range st.Lineage {
			if instance != "" && lin.Producer != instance && lin.Consumer != instance && ident != instance {
				continue
			}
			out.Lineage = append(out.Lineage, lin)
		}
	}
	return out
}

type inspectRemainingDoc struct {
	SchemaVersion int      `json:"schema_version"`
	Identity      string   `json:"identity"`
	Remaining     bool     `json:"remaining"`
	Affected      bool     `json:"affected"`
	Status        string   `json:"status"`
	Reason        string   `json:"reason,omitempty"`
	Differing     []string `json:"differing,omitempty"`
}

func inspectRemainingRecords(workspace string, doc Document, latest []jsonTaskState, instance string, schema int) []inspectRemainingDoc {
	class := classifyRemaining(workspace, doc, latest)
	out := make([]inspectRemainingDoc, 0)
	for _, st := range latest {
		ident := reservedIdentity(taskPlanFromState(st))
		if instance != "" && ident != instance {
			continue
		}
		if !class.Remaining[ident] && !class.Affected[ident] {
			continue
		}
		rec := inspectRemainingDoc{
			SchemaVersion: schema,
			Identity:      ident,
			Remaining:     class.Remaining[ident],
			Affected:      class.Affected[ident],
			Status:        st.Status,
		}
		if dec, ok := class.Decision[ident]; ok {
			rec.Reason = dec.Reason
			rec.Differing = dec.Differing
		}
		out = append(out, rec)
	}
	return out
}

type inspectReuseDoc struct {
	SchemaVersion int      `json:"schema_version"`
	Identity      string   `json:"identity"`
	Decision      string   `json:"decision"`
	Reason        string   `json:"reason"`
	Differing     []string `json:"differing,omitempty"`
}

func inspectReuseRecords(tasks []jsonTaskState, schema int) []inspectReuseDoc {
	var out []inspectReuseDoc
	for _, st := range tasks {
		if st.Decision == "" {
			continue
		}
		out = append(out, inspectReuseDoc{
			SchemaVersion: schema,
			Identity:      reservedIdentity(taskPlanFromState(st)),
			Decision:      st.Decision,
			Reason:        reuseReasonOf(st),
			Differing:     st.Differing,
		})
	}
	return out
}

func documentFromPlan(plan jsonPlan) Document {
	doc := Document{Name: plan.Pipeline}
	ids := make([]string, 0, len(plan.Tasks))
	for _, t := range plan.Tasks {
		doc.Tasks = append(doc.Tasks, decodeTask(t))
		ids = append(ids, t.ID)
	}
	for _, e := range plan.DAG.Edges {
		fromTask, fromPort := splitTaskPort(e.From, ids)
		toTask, toPort := splitTaskPort(e.To, ids)
		doc.Edges = append(doc.Edges, Edge{
			FromTask: fromTask,
			FromPort: fromPort,
			ToTask:   toTask,
			ToPort:   toPort,
			Wait:     e.Wait,
		})
	}
	return doc
}

func decodeTask(t jsonTask) TaskPlan {
	return TaskPlan{
		ID:      t.ID,
		Name:    t.Name,
		Module:  t.Module,
		Branch:  t.Branch,
		Merge:   t.Merge,
		Command: t.Command,
		Script:  t.Script,
		Image:   t.Image,
		Backend: t.Backend,
		Resources: ResourcePlan{
			CPU:    t.Resources.CPU,
			Memory: t.Resources.Memory,
		},
		Params:  decodeParams(t.Params),
		Env:     t.Env,
		Inputs:  decodeIOs(t.Inputs),
		Outputs: decodeIOs(t.Outputs),
	}
}

func decodeIOs(in []jsonIO) []IO {
	out := make([]IO, 0, len(in))
	for _, b := range in {
		io := IO{
			Name:   b.Name,
			Path:   b.Path,
			Source: b.Source,
			Spec:   decodeSpec(b.Spec),
		}
		if b.Members != nil {
			io.Members = decodeMembers(b.Members)
		}
		out = append(out, io)
	}
	return out
}

func decodeMembers(in []jsonMember) []IOMember {
	out := make([]IOMember, 0, len(in))
	for _, m := range in {
		out = append(out, IOMember{
			Name:   m.Name,
			Path:   m.Path,
			Source: m.Source,
			Spec:   decodeSpec(m.Spec),
		})
	}
	return out
}

func decodeSpec(s jsonSpec) Path {
	return Path{
		Dir:      s.Dir,
		Prefix:   s.Prefix,
		Base:     s.Base,
		Suffixes: s.Suffixes,
		Ext:      s.Ext,
		Literal:  s.Literal,
	}
}

func splitTaskPort(ref string, ids []string) (task, port string) {
	best := ""
	for _, id := range ids {
		if ref == id || strings.HasPrefix(ref, id+".") {
			if len(id) > len(best) {
				best = id
			}
		}
	}
	if best == "" {
		i := strings.LastIndex(ref, ".")
		if i < 0 {
			return ref, ""
		}
		return ref[:i], ref[i+1:]
	}
	if ref == best {
		return best, ""
	}
	return best, strings.TrimPrefix(ref[len(best):], ".")
}

func marshalInspect(v any) ([]byte, []Defect) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, pathDefects(err)
	}
	return data, nil
}

func marshalJSONL[T any](records []T) ([]byte, []Defect) {
	if len(records) == 0 {
		return []byte{}, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, rec := range records {
		if err := enc.Encode(rec); err != nil {
			return nil, pathDefects(err)
		}
	}
	return buf.Bytes(), nil
}
