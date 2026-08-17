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
	// Cap is the caller concurrency limit. Zero means DefaultCap.
	Cap      int
	Document Document
}

// TaskPlan is one task in a plan Document.
type TaskPlan struct {
	ID        string
	Name      string
	Module    string
	Branch    string
	Merge     string
	Command   []string
	Image     string
	Backend   string
	Resources ResourcePlan
	Params    []ParamPlan
	Inputs    []IO
	Outputs   []IO
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
type IO struct {
	Name string
	Path string
	Spec Path
}

// Edge is one directed bind edge. An empty FromTask is a pipeline input.
type Edge struct {
	FromTask string
	FromPort string
	ToTask   string
	ToPort   string
}

// Plan is the encoded inspectable plan document.
type Plan struct {
	raw []byte
}

type jsonPlan struct {
	Pipeline string     `json:"pipeline"`
	Tasks    []jsonTask `json:"tasks"`
	DAG      jsonDAG    `json:"dag"`
}

type jsonTask struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Module    string        `json:"module"`
	Branch    string        `json:"branch"`
	Merge     string        `json:"merge"`
	Command   []string      `json:"command"`
	Image     string        `json:"image"`
	Backend   string        `json:"backend"`
	Resources jsonResources `json:"resources"`
	Params    []jsonParam   `json:"params"`
	Inputs    []jsonIO      `json:"inputs"`
	Outputs   []jsonIO      `json:"outputs"`
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
	Name string   `json:"name"`
	Path string   `json:"path"`
	Spec jsonSpec `json:"spec"`
}

// jsonSpec uses locked PathSpec keys. Directory and PathSpec cannot use
// default json.Marshal: Directory.path is unexported and literal is not
// an exported field.
type jsonSpec struct {
	Dir     string   `json:"dir"`
	Lead    string   `json:"lead"`
	Name    string   `json:"name"`
	Steps   []string `json:"steps"`
	Ext     string   `json:"ext"`
	Literal bool     `json:"literal"`
}

type jsonDAG struct {
	Nodes []string   `json:"nodes"`
	Edges []jsonEdge `json:"edges"`
}

type jsonEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// BuildPlan validates s, then encodes doc. On defects it returns (nil, defects).
func BuildPlan(s Snapshot, doc Document) (*Plan, []Defect) {
	if defects := Validate(s); len(defects) > 0 {
		return nil, defects
	}
	raw, err := marshalPlan(doc)
	if err != nil {
		return nil, []Defect{{
			Code:    DefectInvalidName,
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
	jp := jsonPlan{
		Pipeline: doc.Name,
		Tasks:    make([]jsonTask, 0, len(doc.Tasks)),
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
		if e.FromTask == "" {
			continue
		}
		jp.DAG.Edges = append(jp.DAG.Edges, jsonEdge{
			From: e.FromTask + "." + e.FromPort,
			To:   e.ToTask + "." + e.ToPort,
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
	return jsonTask{
		ID:      t.ID,
		Name:    t.Name,
		Module:  t.Module,
		Branch:  t.Branch,
		Merge:   t.Merge,
		Command: jsonStrings(t.Command),
		Image:   t.Image,
		Backend: backend,
		Resources: jsonResources{
			CPU:    t.Resources.CPU,
			Memory: t.Resources.Memory,
		},
		Params:  encodeParams(t.Params),
		Inputs:  encodeIOs(t.Inputs),
		Outputs: encodeIOs(t.Outputs),
	}
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
			Name: b.Name,
			Path: b.Path,
			Spec: jsonSpec{
				Dir:     b.Spec.Dir,
				Lead:    b.Spec.Lead,
				Name:    b.Spec.Name,
				Steps:   jsonStrings(b.Spec.Steps),
				Ext:     b.Spec.Ext,
				Literal: b.Spec.Literal,
			},
		})
	}
	return out
}

func jsonStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
