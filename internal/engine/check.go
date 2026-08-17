package engine

import (
	"os"
	"path/filepath"
	"strings"
)

// ControlDir is the reserved workspace subtree for run identity.
// Authored plan paths must not start with this name.
const ControlDir = ".gobble"

// RunIdentityFile is the run identity document under ControlDir.
// Its presence occupies the workspace.
const RunIdentityFile = "run.json"

// DefaultCap is the concurrency cap when the caller omits Cap.
const DefaultCap = 1

// Check reports pre-execution defects on req. It does not occupy the
// workspace, create directories, or start a task.
func Check(req Request) []Defect {
	if d := checkWorkspace(req.Workspace); len(d) > 0 {
		return d
	}
	if d := checkOccupied(req.Workspace); len(d) > 0 {
		return d
	}
	if d := checkCap(req.Cap); len(d) > 0 {
		return d
	}
	if d := checkPlanPaths(req.Document); len(d) > 0 {
		return d
	}
	if d := checkBackends(req.Document); len(d) > 0 {
		return d
	}
	if d := checkInputs(req.Workspace, req.Document); len(d) > 0 {
		return d
	}
	if d := checkOutputs(req.Workspace, req.Document); len(d) > 0 {
		return d
	}
	return nil
}

func checkWorkspace(workspace string) []Defect {
	if workspace == "" {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "missing workspace",
		}}
	}
	info, err := os.Stat(workspace)
	if err != nil {
		if os.IsNotExist(err) {
			return []Defect{{
				Code:    DefectInvalidPath,
				Message: "missing workspace",
				Paths:   []string{workspace},
			}}
		}
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "workspace is not usable",
			Paths:   []string{workspace},
		}}
	}
	if !info.IsDir() {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "workspace is not a directory",
			Paths:   []string{workspace},
		}}
	}
	return nil
}

func checkOccupied(workspace string) []Defect {
	ident := filepath.Join(workspace, ControlDir, RunIdentityFile)
	if _, err := os.Stat(ident); err != nil {
		return nil
	}
	return []Defect{{
		Code:    DefectOccupiedWorkspace,
		Message: "occupied workspace",
		Paths:   []string{ControlDir + "/" + RunIdentityFile},
	}}
}

func checkCap(cap int) []Defect {
	n := cap
	if n == 0 {
		n = DefaultCap
	}
	if n < 1 {
		return []Defect{{
			Code:    DefectInvalidName,
			Message: "concurrency cap below 1",
		}}
	}
	return nil
}

func checkPlanPaths(doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, in := range t.Inputs {
			if d := checkPlanPath(bindUnit(t.ID, in.Name), in.Path); d != nil {
				defects = append(defects, *d)
			}
		}
		for _, out := range t.Outputs {
			if d := checkPlanPath(bindUnit(t.ID, out.Name), out.Path); d != nil {
				defects = append(defects, *d)
			}
		}
	}
	return defects
}

func checkPlanPath(unit, path string) *Defect {
	if path == "" {
		return &Defect{
			Code:    DefectInvalidPath,
			Unit:    unit,
			Message: "empty plan path",
		}
	}
	normalized := strings.ReplaceAll(path, `\`, "/")
	if filepath.IsAbs(path) || filepath.IsAbs(normalized) || strings.HasPrefix(normalized, "/") {
		return &Defect{
			Code:    DefectInvalidPath,
			Unit:    unit,
			Message: "absolute plan path",
			Paths:   []string{path},
		}
	}
	cleaned, escaped := cleanPath(normalized)
	if escaped {
		return &Defect{
			Code:    DefectInvalidPath,
			Unit:    unit,
			Message: "path escapes directory",
			Paths:   []string{path},
		}
	}
	if cleaned == ControlDir || strings.HasPrefix(cleaned, ControlDir+"/") {
		return &Defect{
			Code:    DefectInvalidPath,
			Unit:    unit,
			Message: "path is under .gobble",
			Paths:   []string{path},
		}
	}
	return nil
}

func checkBackends(doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		if t.Backend != "" && t.Backend != "local" {
			defects = append(defects, Defect{
				Code:    DefectUnsupportedBackend,
				Unit:    t.ID,
				Message: "unsupported backend",
			})
		}
	}
	return defects
}

func checkInputs(workspace string, doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, in := range t.Inputs {
			if hasUpstreamTask(doc, t.ID, in.Name) {
				continue
			}
			if fileExists(workspaceFile(workspace, in.Path)) {
				continue
			}
			defects = append(defects, Defect{
				Code:    DefectMissingInput,
				Unit:    bindUnit(t.ID, in.Name),
				Message: "missing input",
				Paths:   []string{in.Path},
			})
		}
	}
	return defects
}

func checkOutputs(workspace string, doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, out := range t.Outputs {
			if !fileExists(workspaceFile(workspace, out.Path)) {
				continue
			}
			defects = append(defects, Defect{
				Code:    DefectOutputExists,
				Unit:    bindUnit(t.ID, out.Name),
				Message: "output exists",
				Paths:   []string{out.Path},
			})
		}
	}
	return defects
}

func hasUpstreamTask(doc Document, taskID, port string) bool {
	for _, e := range doc.Edges {
		if e.ToTask == taskID && e.ToPort == port && e.FromTask != "" {
			return true
		}
	}
	return false
}

func workspaceFile(workspace, planPath string) string {
	return filepath.Join(workspace, filepath.FromSlash(planPath))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
