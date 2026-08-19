package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// ControlDir is the reserved workspace subtree for run identity.
// Authored plan paths must not start with this name.
const ControlDir = ".gobble"

// RunIdentityFile is the run identity document under ControlDir.
// Occupancy is the owner record inside that file, not mere presence.
const RunIdentityFile = "run.json"

// DefaultCap is the concurrency cap when the caller omits Cap.
const DefaultCap = 1

// MaxCap is the largest accepted concurrency cap.
const MaxCap = 64

// hostCapacity is a host CPU and memory snapshot. A false Known flag
// means that axis is unspecified and only the count cap binds on it.
type hostCapacity struct {
	CPU      float64
	Memory   int64
	CPUKnown bool
	MemKnown bool
}

// readHostCapacity is the injectable host snapshot used by Check and Run.
var readHostCapacity = defaultHostCapacity

func defaultHostCapacity() hostCapacity {
	n := runtime.NumCPU()
	cap := hostCapacity{CPU: float64(n), CPUKnown: n > 0}
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return cap
	}
	unit := uint64(info.Unit)
	if unit == 0 {
		unit = 1
	}
	total := info.Totalram * unit
	if total == 0 || total > uint64(int64(^uint64(0)>>1)) {
		return cap
	}
	cap.Memory = int64(total)
	cap.MemKnown = true
	return cap
}

// CheckResumeStart reports Resume preflight defects for workspace
// existence and cap. It does not occupy, inspect dests, or start work.
func CheckResumeStart(workspace string, cap int) []Defect {
	if d := checkWorkspace(workspace); len(d) > 0 {
		return d
	}
	return checkCap(cap)
}

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
	if d := checkImages(req.Document); len(d) > 0 {
		return d
	}
	if d := checkInputs(req.Workspace, req.Document); len(d) > 0 {
		return d
	}
	if d := checkOutputs(req.Workspace, req.Document); len(d) > 0 {
		return d
	}
	if d := checkCapacity(req.Document, readHostCapacity()); len(d) > 0 {
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
	run, exists, err := readRunIdentity(workspace)
	if err != nil {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "workspace occupancy is not usable",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	if !exists || !occupancyIsActive(run) {
		return nil
	}
	return occupiedDefect()
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
	if n > MaxCap {
		return []Defect{{
			Code:    DefectInvalidName,
			Message: "concurrency cap above 64",
		}}
	}
	return nil
}

func checkPlanPaths(doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, in := range t.Inputs {
			defects = append(defects, checkIOPaths(bindUnit(t.ID, in.Name), in)...)
		}
		for _, out := range t.Outputs {
			defects = append(defects, checkIOPaths(bindUnit(t.ID, out.Name), out)...)
		}
	}
	return defects
}

func checkIOPaths(unit string, io IO) []Defect {
	if io.Members != nil {
		var defects []Defect
		for _, m := range io.Members {
			found, ok := findIOMember(io.Members, m.Name)
			if !ok {
				continue
			}
			if d := checkPlanPath(bindUnit(unit, found.Name), found.Path); d != nil {
				defects = append(defects, *d)
			}
		}
		return defects
	}
	if d := checkPlanPath(unit, io.Path); d != nil {
		return []Defect{*d}
	}
	return nil
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
	if cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
		return &Defect{
			Code:    DefectInvalidPath,
			Unit:    unit,
			Message: "path is under .git",
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

func checkImages(doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		if msg := invalidImage(t.Image); msg != "" {
			defects = append(defects, Defect{
				Code:    DefectInvalidName,
				Unit:    t.ID,
				Message: msg,
			})
		}
	}
	return defects
}

func invalidImage(image string) string {
	if image == "" {
		return ""
	}
	trimmed := strings.TrimSpace(image)
	if trimmed == "" {
		return "empty image"
	}
	if strings.HasPrefix(trimmed, "-") || strings.ContainsAny(image, " \t\n\r") {
		return "invalid image"
	}
	return ""
}

func checkInputs(workspace string, doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, in := range t.Inputs {
			if hasUpstreamTask(doc, t.ID, in.Name) {
				continue
			}
			unit := bindUnit(t.ID, in.Name)
			if in.Members != nil {
				for _, f := range namedIOFiles(in) {
					src := fileSource(f)
					if regularFile(workspaceFile(workspace, src)) {
						continue
					}
					defects = append(defects, Defect{
						Code:    DefectMissingInput,
						Unit:    bindUnit(unit, f.name),
						Message: "missing input",
						Paths:   []string{src},
					})
				}
				continue
			}
			src := in.Path
			if in.Source != "" {
				src = in.Source
			}
			if regularFile(workspaceFile(workspace, src)) {
				continue
			}
			defects = append(defects, Defect{
				Code:    DefectMissingInput,
				Unit:    unit,
				Message: "missing input",
				Paths:   []string{src},
			})
		}
	}
	return defects
}

func checkOutputs(workspace string, doc Document) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, out := range t.Outputs {
			unit := bindUnit(t.ID, out.Name)
			if out.Members != nil {
				for _, f := range namedIOFiles(out) {
					if !pathPresent(workspaceFile(workspace, f.path)) {
						continue
					}
					defects = append(defects, Defect{
						Code:    DefectOutputExists,
						Unit:    bindUnit(unit, f.name),
						Message: "output exists",
						Paths:   []string{f.path},
					})
				}
				continue
			}
			if !pathPresent(workspaceFile(workspace, out.Path)) {
				continue
			}
			defects = append(defects, Defect{
				Code:    DefectOutputExists,
				Unit:    unit,
				Message: "output exists",
				Paths:   []string{out.Path},
			})
		}
	}
	return defects
}

func checkCapacity(doc Document, host hostCapacity) []Defect {
	var defects []Defect
	for _, t := range doc.Tasks {
		if host.CPUKnown && t.Resources.CPU > 0 && t.Resources.CPU > host.CPU {
			defects = append(defects, Defect{
				Code:    DefectInvalidName,
				Unit:    t.ID,
				Message: "cpu exceeds host capacity",
			})
		}
		bytes, ok := parseMemory(t.Resources.Memory)
		if !ok {
			defects = append(defects, Defect{
				Code:    DefectInvalidMemory,
				Unit:    t.ID,
				Message: "invalid memory " + strconv.Quote(t.Resources.Memory),
			})
			continue
		}
		if host.MemKnown && bytes > 0 && bytes > host.Memory {
			defects = append(defects, Defect{
				Code:    DefectInvalidName,
				Unit:    t.ID,
				Message: "memory exceeds host capacity",
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

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func pathPresent(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
