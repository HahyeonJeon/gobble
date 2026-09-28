package gobble

import (
	"fmt"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// PipelineInspection is a versioned, checked description of authored flow. It is
// not a Run, an executable serialization, or permission to execute. All returned
// slices are caller-owned. Version 2 adds module-authored settings; ports have no final-result roles.
type PipelineInspection struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Name          string               `json:"name"`
	Inputs        []PipelinePort       `json:"inputs"`
	Steps         []PipelineStep       `json:"steps"`
	Connections   []PipelineConnection `json:"connections"`
}

// PipelinePort describes declared data. Paths name declarations, not existing
// results; group members preserve their explicit identities.
type PipelinePort struct {
	Name    string           `json:"name"`
	Kind    string           `json:"kind"`
	Path    string           `json:"path"`
	Members []PipelineMember `json:"members"`
}

type PipelineMember struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// PipelineStep is an authored operation. Display is author-supplied presentation,
// and Control is Gobble's declared control structure, independent of canvas layout.
type PipelineStep struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Module   string           `json:"module"`
	Display  TaskDisplay      `json:"display"`
	Image    string           `json:"image"`
	Backend  string           `json:"backend"`
	CPU      float64          `json:"cpu"`
	Memory   string           `json:"memory"`
	Inputs   []PipelinePort   `json:"inputs"`
	Outputs  []PipelinePort   `json:"outputs"`
	Control  PipelineControl  `json:"control"`
	Settings []IntegerSetting `json:"settings"`
}

// PipelineControl retains exact control identities. Empty values mean the
// corresponding construct was not declared; an empty task names a pipeline input.
type PipelineControl struct {
	Branch             string   `json:"branch"`
	Merge              string   `json:"merge"`
	Scatter            string   `json:"scatter"`
	Gather             string   `json:"gather"`
	When               string   `json:"when"`
	ScatterFromKind    string   `json:"scatterFromKind"`
	ScatterFromTask    string   `json:"scatterFromTask"`
	ScatterFromPort    string   `json:"scatterFromPort"`
	ScatterFromPath    string   `json:"scatterFromPath"`
	ScatterMembers     []string `json:"scatterMembers"`
	ScatterMemberPaths []string `json:"scatterMemberPaths"`
	SkipIfMissingTask  string   `json:"skipIfMissingTask"`
	SkipIfMissingPort  string   `json:"skipIfMissingPort"`
	SkipIfMissingPath  string   `json:"skipIfMissingPath"`
	SkipIfFalse        string   `json:"skipIfFalse"`
}

// PipelineConnection identifies one directed port relationship in this artifact.
// ID is artifact-local, never correspondence across separately inspected graphs.
type PipelineConnection struct {
	ID       string   `json:"id"`
	FromTask string   `json:"fromTask"`
	FromPort string   `json:"fromPort"`
	ToTask   string   `json:"toTask"`
	ToPort   string   `json:"toPort"`
	Wait     []string `json:"wait"`
}

// InspectPipeline checks a composed graph using BuildPlan and returns complete
// authored flow, including pipeline input edges. It does not execute tasks or
// create a workspace. Compose and the caller's Pipeline function may perform I/O;
// consumers evaluating Project code must own that separate execution boundary.
// Existing Plan JSON and its compatibility contract are unchanged.
func InspectPipeline(g *Graph) (PipelineInspection, error) {
	plan, err := BuildPlan(g)
	if err != nil {
		return PipelineInspection{}, err
	}
	doc, err := planDocument(g)
	if err != nil {
		return PipelineInspection{}, err
	}
	out := PipelineInspection{2, g.name, []PipelinePort{}, []PipelineStep{}, []PipelineConnection{}}
	for _, in := range g.inputs {
		port, err := planIO(g, &graphTask{}, graphBind{name: in.name, spec: in.spec, tree: in.tree, members: in.members}, false)
		if err != nil {
			return PipelineInspection{}, planPathError(in.name, err)
		}
		out.Inputs = append(out.Inputs, inspectionPorts([]engine.IO{port})[0])
	}
	for i, task := range doc.Tasks {
		if err := validateIntegerSettings(g.tasks[i].inspectionSettings); err != nil {
			return PipelineInspection{}, fmt.Errorf("step %s: %w", task.ID, err)
		}
		backend := task.Backend
		if backend == "" {
			backend = "local"
		}
		out.Steps = append(out.Steps, PipelineStep{
			ID: task.ID, Name: task.Name, Module: task.Module, Display: cloneDisplay(g.tasks[i].display),
			Image: task.Image, Backend: backend, CPU: task.Resources.CPU, Memory: task.Resources.Memory,
			Settings: cloneIntegerSettings(g.tasks[i].inspectionSettings),
			Inputs:   inspectionPorts(task.Inputs), Outputs: inspectionPorts(task.Outputs),
			Control: PipelineControl{
				Branch: task.Branch, Merge: task.Merge, Scatter: task.Scatter, Gather: task.Gather, When: task.When,
				ScatterFromKind: task.ScatterFromKind, ScatterFromTask: task.ScatterFromTask,
				ScatterFromPort: task.ScatterFromPort, ScatterFromPath: task.ScatterFromPath,
				ScatterMembers: append([]string{}, task.ScatterMembers...), ScatterMemberPaths: append([]string{}, task.ScatterMemberPaths...),
				SkipIfMissingTask: task.SkipIfMissingTask, SkipIfMissingPort: task.SkipIfMissingPort,
				SkipIfMissingPath: task.SkipIfMissingPath, SkipIfFalse: task.SkipIfFalse,
			},
		})
	}
	for i, edge := range plan.Edges() {
		out.Connections = append(out.Connections, PipelineConnection{
			ID: fmt.Sprintf("edge-%d", i+1), FromTask: edge.FromTask, FromPort: edge.FromPort,
			ToTask: edge.ToTask, ToPort: edge.ToPort, Wait: append([]string{}, edge.Wait...),
		})
	}
	return out, nil
}

func inspectionPorts(ports []engine.IO) []PipelinePort {
	out := make([]PipelinePort, 0, len(ports))
	for _, port := range planBindsFromIO(ports) {
		value := PipelinePort{Name: port.name, Kind: port.kind, Path: port.path, Members: []PipelineMember{}}
		for _, member := range port.members {
			value.Members = append(value.Members, PipelineMember{member.name, member.path})
		}
		out = append(out, value)
	}
	return out
}
