package engine

import (
	"encoding/json"
	"errors"
	"io"
)

// Document is the plan-time and execution view gobble translates from a Graph.
type Document struct {
	Name  string
	Tasks []TaskPlan
	Edges []Edge
}

// Request is a run check. Workspace and Cap do not belong on Document.
type Request struct {
	Workspace string
	Identity  *InstallIdentity
	// Cap is the caller concurrency limit. Zero means DefaultCap.
	Cap      int
	Document Document
}

// TaskPlan is one task in a plan Document.
type TaskPlan struct {
	ID                 string
	Name               string
	Instance           string
	ShardIndex         int
	ShardCount         int
	Attempt            int
	Module             string
	Branch             string
	Merge              string
	Scatter            string
	Gather             string
	When               string
	ScatterFromKind    string
	ScatterFromTask    string
	ScatterFromPort    string
	ScatterFromPath    string
	ScatterMembers     []string
	ScatterMemberPaths []string
	SkipIfMissingTask  string
	SkipIfMissingPort  string
	SkipIfMissingPath  string
	SkipIfFalse        string
	Command            []string
	Script             string
	Image              string
	Backend            string
	Resources          ResourcePlan
	Params             []ParamPlan
	Env                map[string]string
	EnvDigest          string
	ExecutablePath     string
	ExecutableSHA256   string
	Inputs             []IO
	Outputs            []IO
	// Replace selects authorized staged replace after isolate outputs.
	// Run publish ignores it and stays exclusive-create.
	Replace bool
}

// ResourcePlan is the recorded CPU and memory request.
type ResourcePlan struct {
	CPU    float64
	Memory string
}

// ParamPlan is one recorded name/value parameter.
type ParamPlan struct {
	Name  string
	Value string
}

// IO is one recorded input or output bind.
//
// Kind is file, group, or tree. Path may be empty on a Group IO.
// Members is omitted on single-file IOs. Manifest is the dest Tree
// .gobble-tree.json path. Source is the workspace path isolate copies
// from when it differs from Path. Empty Source means Path is both
// source and dest.
type IO struct {
	Name     string
	Kind     string
	Path     string
	Source   string
	Spec     Path
	Members  []IOMember
	Manifest string
}

// IOMember is one recorded Group member.
//
// Source is the workspace path isolate copies from when it differs from
// Path. Empty Source means Path is both source and dest.
type IOMember struct {
	Name   string
	Path   string
	Source string
	Spec   Path
}

// Edge is one directed bind edge. An empty FromTask is a pipeline input.
// Wait is the plan-relative paths that must exist before the downstream
// task may start.
type Edge struct {
	FromTask string
	FromPort string
	ToTask   string
	ToPort   string
	Wait     []string
}

// Plan is the encoded inspectable plan document.
type Plan struct {
	raw []byte
}

type jsonPlan struct {
	SchemaVersion int        `json:"schema_version,omitempty"`
	Snapshot      string     `json:"snapshot,omitempty"`
	Pipeline      string     `json:"pipeline"`
	Tasks         []jsonTask `json:"tasks"`
	DAG           jsonDAG    `json:"dag"`
}

type jsonTask struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Instance           string        `json:"instance"`
	ShardIndex         int           `json:"shard_index"`
	ShardCount         int           `json:"shard_count"`
	Attempt            int           `json:"attempt"`
	Module             string        `json:"module"`
	Branch             string        `json:"branch"`
	Merge              string        `json:"merge"`
	Scatter            string        `json:"scatter,omitempty"`
	Gather             string        `json:"gather,omitempty"`
	When               string        `json:"when,omitempty"`
	ScatterFromKind    string        `json:"scatter_from_kind,omitempty"`
	ScatterFromTask    string        `json:"scatter_from_task,omitempty"`
	ScatterFromPort    string        `json:"scatter_from_port,omitempty"`
	ScatterFromPath    string        `json:"scatter_from_path,omitempty"`
	ScatterMembers     []string      `json:"scatter_members,omitempty"`
	ScatterMemberPaths []string      `json:"scatter_member_paths,omitempty"`
	SkipIfMissingTask  string        `json:"skip_if_missing_task,omitempty"`
	SkipIfMissingPort  string        `json:"skip_if_missing_port,omitempty"`
	SkipIfMissingPath  string        `json:"skip_if_missing_path,omitempty"`
	SkipIfFalse        string        `json:"skip_if_false,omitempty"`
	Command            []string      `json:"command"`
	Script             string        `json:"script,omitempty"`
	Image              string        `json:"image"`
	Backend            string        `json:"backend"`
	Resources          jsonResources `json:"resources"`
	Params             []jsonParam   `json:"params"`
	EnvDigest          string        `json:"env_digest,omitempty"`
	Inputs             []jsonIO      `json:"inputs"`
	Outputs            []jsonIO      `json:"outputs"`
}

type jsonResources struct {
	CPU    float64 `json:"cpu"`
	Memory string  `json:"memory"`
}

type jsonParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type jsonIO struct {
	Name     string       `json:"name"`
	Kind     string       `json:"kind"`
	Path     string       `json:"path"`
	Source   string       `json:"source,omitempty"`
	Spec     jsonSpec     `json:"spec"`
	Members  []jsonMember `json:"members,omitempty"`
	Manifest string       `json:"manifest,omitempty"`
}

type jsonMember struct {
	Name   string   `json:"name"`
	Path   string   `json:"path"`
	Source string   `json:"source,omitempty"`
	Spec   jsonSpec `json:"spec"`
}

// jsonSpec uses locked PathSpec keys. Directory and PathSpec cannot use
// default json.Marshal: Directory.path is unexported and literal is not
// an exported field.
type jsonSpec struct {
	Dir      string   `json:"dir"`
	Prefix   string   `json:"prefix"`
	Base     string   `json:"base"`
	Suffixes []string `json:"suffixes"`
	Ext      string   `json:"ext"`
	Literal  bool     `json:"literal"`
}

type jsonDAG struct {
	Nodes []string   `json:"nodes"`
	Edges []jsonEdge `json:"edges"`
}

type jsonEdge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Wait []string `json:"wait"`
}

// BuildPlan validates doc, then encodes it. On defects it returns (nil, defects).
func BuildPlan(doc Document) (*Plan, []Defect) {
	if defects := Validate(doc); len(defects) > 0 {
		return nil, defects
	}
	raw, err := marshalPlan(doc)
	if err != nil {
		return nil, []Defect{{
			Code:    DefectInvalidValue,
			Message: "encode plan: " + err.Error(),
		}}
	}
	return &Plan{raw: raw}, nil
}

// WriteJSON writes the plan JSON, including a trailing newline.
func (p *Plan) WriteJSON(w io.Writer) error {
	if p == nil {
		return errors.New("nil plan")
	}
	_, err := w.Write(p.raw)
	return err
}

// MarshalJSON returns the plan JSON without the trailing newline.
func (p *Plan) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	raw := p.raw
	if n := len(raw); n > 0 && raw[n-1] == '\n' {
		raw = raw[:n-1]
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out, nil
}

func marshalPlan(doc Document) ([]byte, error) {
	return encodePlan(doc, SchemaVersion, false, "")
}

func marshalControlPlan(doc Document, snapshot string) ([]byte, error) {
	return encodePlan(doc, SchemaVersion, true, snapshot)
}

func encodePlan(doc Document, schema int, includeInputEdges bool, snapshot string) ([]byte, error) {
	jp := jsonPlan{
		SchemaVersion: schema,
		Snapshot:      snapshot,
		Pipeline:      doc.Name,
		Tasks:         make([]jsonTask, 0, len(doc.Tasks)),
		DAG: jsonDAG{
			Nodes: make([]string, 0, len(doc.Tasks)),
			Edges: make([]jsonEdge, 0),
		},
	}
	for _, t := range doc.Tasks {
		jp.Tasks = append(jp.Tasks, encodeTask(t))
		jp.DAG.Nodes = append(jp.DAG.Nodes, t.ID)
	}
	for _, e := range doc.Edges {
		if e.FromTask == "" && !includeInputEdges {
			continue
		}
		jp.DAG.Edges = append(jp.DAG.Edges, jsonEdge{
			From: e.FromTask + "." + e.FromPort,
			To:   e.ToTask + "." + e.ToPort,
			Wait: jsonStrings(e.Wait),
		})
	}
	data, err := json.MarshalIndent(jp, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func encodeTask(t TaskPlan) jsonTask {
	backend := t.Backend
	if backend == "" {
		backend = "local"
	}
	applyReservedDefaults(&t)
	return jsonTask{
		ID:                 t.ID,
		Name:               t.Name,
		Instance:           t.Instance,
		ShardIndex:         t.ShardIndex,
		ShardCount:         t.ShardCount,
		Attempt:            t.Attempt,
		Module:             t.Module,
		Branch:             t.Branch,
		Merge:              t.Merge,
		Scatter:            t.Scatter,
		Gather:             t.Gather,
		When:               t.When,
		ScatterFromKind:    t.ScatterFromKind,
		ScatterFromTask:    t.ScatterFromTask,
		ScatterFromPort:    t.ScatterFromPort,
		ScatterFromPath:    t.ScatterFromPath,
		ScatterMembers:     jsonOmitEmpty(t.ScatterMembers),
		ScatterMemberPaths: jsonOmitEmpty(t.ScatterMemberPaths),
		SkipIfMissingTask:  t.SkipIfMissingTask,
		SkipIfMissingPort:  t.SkipIfMissingPort,
		SkipIfMissingPath:  t.SkipIfMissingPath,
		SkipIfFalse:        t.SkipIfFalse,
		Command:            jsonStrings(t.Command),
		Script:             t.Script,
		Image:              t.Image,
		Backend:            backend,
		Resources: jsonResources{
			CPU:    t.Resources.CPU,
			Memory: t.Resources.Memory,
		},
		Params:    encodeParams(t.Params),
		EnvDigest: encodeEnvDigest(t),
		Inputs:    encodeIOs(t.Inputs),
		Outputs:   encodeIOs(t.Outputs),
	}
}

func encodeEnvDigest(t TaskPlan) string {
	if t.Env != nil {
		return envDigest(t.Env)
	}
	return t.EnvDigest
}

func encodeParams(in []ParamPlan) []jsonParam {
	out := make([]jsonParam, 0, len(in))
	for _, p := range in {
		out = append(out, jsonParam{Name: p.Name, Value: p.Value})
	}
	return out
}

func encodeIOs(in []IO) []jsonIO {
	out := make([]jsonIO, 0, len(in))
	for _, b := range in {
		out = append(out, jsonIO{
			Name:     b.Name,
			Kind:     ioKind(b),
			Path:     b.Path,
			Source:   b.Source,
			Spec:     encodeSpec(b.Spec),
			Members:  encodeMembers(b.Members),
			Manifest: b.Manifest,
		})
	}
	return out
}

func ioKind(b IO) string {
	if b.Kind != "" {
		return b.Kind
	}
	if b.Members != nil {
		return ArtifactGroup
	}
	if b.Manifest != "" {
		return ArtifactTree
	}
	return ArtifactFile
}

func encodeMembers(in []IOMember) []jsonMember {
	if in == nil {
		return nil
	}
	out := make([]jsonMember, 0, len(in))
	for _, m := range in {
		out = append(out, jsonMember{
			Name:   m.Name,
			Path:   m.Path,
			Source: m.Source,
			Spec:   encodeSpec(m.Spec),
		})
	}
	return out
}

func encodeSpec(p Path) jsonSpec {
	return jsonSpec{
		Dir:      p.Dir,
		Prefix:   p.Prefix,
		Base:     p.Base,
		Suffixes: jsonStrings(p.Suffixes),
		Ext:      p.Ext,
		Literal:  p.Literal,
	}
}

func jsonStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func jsonOmitEmpty(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return jsonStrings(in)
}
