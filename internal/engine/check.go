package engine

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	intpath "github.com/HahyeonJeon/gobble/internal/path"
)

var errInvalidPath = errors.New("invalid-path")

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
	if d := checkControlContainment(workspace); len(d) > 0 {
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
	if d := emptyGraphDefects(req.Document); len(d) > 0 {
		return d
	}
	if d := checkControlContainment(req.Workspace); len(d) > 0 {
		return d
	}
	if d := checkControlSchema(req.Workspace); len(d) > 0 {
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
	if d := checkWaitPaths(req.Workspace, req.Document); len(d) > 0 {
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

func checkControlSchema(workspace string) []Defect {
	run, exists, err := readRunIdentity(workspace)
	if err != nil {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "workspace occupancy is not usable",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	if !exists {
		root := workspaceFile(workspace, ControlDir)
		for _, name := range []string{PlanFile, TasksFile} {
			ver, found, err := readSchemaFile(workspaceFile(root, name))
			if err != nil {
				return pathDefects(err)
			}
			if found && schemaUnsupported(ver) {
				return schemaDefect(ControlDir + "/" + name)
			}
		}
		return nil
	}
	return unsupportedControlSchema(workspace, run)
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
			Code:    DefectInvalidValue,
			Message: "concurrency cap below 1",
		}}
	}
	if n > MaxCap {
		return []Defect{{
			Code:    DefectInvalidValue,
			Message: "concurrency cap above 64",
		}}
	}
	return nil
}

func checkPlanPaths(doc Document) []Defect {
	scatterIDs := make(map[string]bool, len(doc.Tasks))
	for _, t := range doc.Tasks {
		if t.Scatter != "" {
			scatterIDs[t.ID] = true
		}
	}
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, in := range t.Inputs {
			if scatterRelatedEmptyIO(doc, t, in.Name, in.Path, scatterIDs) {
				continue
			}
			defects = append(defects, checkIOPaths(bindUnit(t.ID, in.Name), in)...)
		}
		for _, out := range t.Outputs {
			if scatterRelatedEmptyIO(doc, t, out.Name, out.Path, scatterIDs) {
				continue
			}
			defects = append(defects, checkIOPaths(bindUnit(t.ID, out.Name), out)...)
		}
	}
	return defects
}

func scatterRelatedEmptyIO(doc Document, t TaskPlan, port, path string, scatterIDs map[string]bool) bool {
	if path != "" {
		return false
	}
	for _, e := range doc.Edges {
		if e.ToTask != t.ID || e.ToPort != port {
			continue
		}
		if t.Scatter != "" && e.FromTask == t.ScatterFromTask && e.FromPort == t.ScatterFromPort {
			return true
		}
		if scatterIDs[e.FromTask] {
			return true
		}
	}
	return false
}

func checkIOPaths(unit string, io IO) []Defect {
	if isTreeIO(io) {
		var defects []Defect
		if d := checkPlanPath(unit, io.Path); d != nil {
			defects = append(defects, *d)
		}
		if io.Source != "" {
			if d := checkPlanPath(unit, io.Source); d != nil {
				defects = append(defects, *d)
			}
		}
		if io.Manifest != "" {
			if d := checkPlanPath(unit, io.Manifest); d != nil {
				defects = append(defects, *d)
			}
		}
		return defects
	}
	if io.Members != nil {
		var defects []Defect
		for _, m := range io.Members {
			found, ok := findIOMember(io.Members, m.Name)
			if !ok {
				continue
			}
			memberUnit := bindUnit(unit, found.Name)
			if d := checkPlanPath(memberUnit, found.Path); d != nil {
				defects = append(defects, *d)
			}
			if found.Source != "" {
				if d := checkPlanPath(memberUnit, found.Source); d != nil {
					defects = append(defects, *d)
				}
			}
		}
		return defects
	}
	var defects []Defect
	if d := checkPlanPath(unit, io.Path); d != nil {
		defects = append(defects, *d)
	}
	if io.Source != "" {
		if d := checkPlanPath(unit, io.Source); d != nil {
			defects = append(defects, *d)
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
	cleaned, escaped := intpath.Clean(normalized)
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
				Code:    DefectInvalidValue,
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

func skipIfMissingExempt(t TaskPlan, in IO) bool {
	if t.SkipIfMissingPath == "" && t.SkipIfMissingPort == "" {
		return false
	}
	src := in.Path
	if in.Source != "" {
		src = in.Source
	}
	if t.SkipIfMissingPath != "" && src == t.SkipIfMissingPath {
		return true
	}
	return t.SkipIfMissingTask == "" && in.Name == t.SkipIfMissingPort
}

func checkInputs(workspace string, doc Document) []Defect {
	scatterIDs := make(map[string]bool, len(doc.Tasks))
	for _, t := range doc.Tasks {
		if t.Scatter != "" {
			scatterIDs[t.ID] = true
		}
	}
	var defects []Defect
	for _, t := range doc.Tasks {
		for _, in := range t.Inputs {
			if hasUpstreamTask(doc, t.ID, in.Name) {
				continue
			}
			src := in.Path
			if in.Source != "" {
				src = in.Source
			}
			if scatterRelatedEmptyIO(doc, t, in.Name, src, scatterIDs) {
				continue
			}
			unit := bindUnit(t.ID, in.Name)
			exempt := skipIfMissingExempt(t, in)
			if isTreeIO(in) {
				src := treeSourceDir(in)
				abs, present, err := containedRel(workspace, src, false)
				if err != nil {
					defects = append(defects, escapeDefect(unit, src))
					continue
				}
				if present && isDir(abs) {
					continue
				}
				defects = append(defects, Defect{
					Code:    DefectMissingInput,
					Unit:    unit,
					Message: "missing input",
					Paths:   []string{src},
				})
				continue
			}
			if in.Members != nil {
				for _, f := range namedIOFiles(in) {
					src := fileSource(f)
					abs, present, err := containedRel(workspace, src, false)
					if err != nil {
						defects = append(defects, escapeDefect(bindUnit(unit, f.name), src))
						continue
					}
					if present && regularFile(abs) {
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
			src = in.Path
			if in.Source != "" {
				src = in.Source
			}
			abs, present, err := containedRel(workspace, src, false)
			if err != nil {
				defects = append(defects, escapeDefect(unit, src))
				continue
			}
			if present && regularFile(abs) {
				continue
			}
			if exempt {
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
			if !isTreeIO(out) && out.Members == nil && out.Path == "" {
				continue
			}
			if isTreeIO(out) {
				_, present, err := containedRel(workspace, out.Path, true)
				if err != nil {
					defects = append(defects, escapeDefect(unit, out.Path))
					continue
				}
				if !present {
					continue
				}
				defects = append(defects, Defect{
					Code:    DefectOutputExists,
					Unit:    unit,
					Message: "output exists",
					Paths:   []string{out.Path},
				})
				continue
			}
			if out.Members != nil {
				for _, f := range namedIOFiles(out) {
					_, present, err := containedRel(workspace, f.path, true)
					if err != nil {
						defects = append(defects, escapeDefect(bindUnit(unit, f.name), f.path))
						continue
					}
					if !present {
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
			_, present, err := containedRel(workspace, out.Path, true)
			if err != nil {
				defects = append(defects, escapeDefect(unit, out.Path))
				continue
			}
			if !present {
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
				Code:    DefectInvalidValue,
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
				Code:    DefectInvalidValue,
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

func escapeDefect(unit, path string) Defect {
	return Defect{
		Code:    DefectInvalidPath,
		Unit:    unit,
		Message: "path escapes directory",
		Paths:   []string{path},
	}
}

func checkWaitPaths(workspace string, doc Document) []Defect {
	var defects []Defect
	for _, e := range doc.Edges {
		for _, p := range e.Wait {
			if p == "" {
				continue
			}
			if d := checkPlanPath(e.ToTask, p); d != nil {
				defects = append(defects, *d)
				continue
			}
			if _, _, err := containedRel(workspace, p, false); err != nil {
				defects = append(defects, escapeDefect(e.ToTask, p))
			}
		}
	}
	return defects
}

func checkControlContainment(workspace string) []Defect {
	abs, present, err := containedRel(workspace, ControlDir, false)
	if err != nil {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "path escapes directory",
			Paths:   []string{ControlDir},
		}}
	}
	if !present {
		return nil
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "workspace occupancy is not usable",
			Paths:   []string{ControlDir},
		}}
	}
	if !info.IsDir() {
		return []Defect{{
			Code:    DefectInvalidPath,
			Message: "path escapes directory",
			Paths:   []string{ControlDir},
		}}
	}
	children := []string{
		ControlDir + "/" + RunIdentityFile,
		ControlDir + "/" + PlanFile,
		ControlDir + "/" + TasksFile,
		ControlDir + "/" + occupyLockFile,
		ControlDir + "/tasks",
	}
	for _, rel := range children {
		if _, _, err := containedRel(workspace, rel, false); err != nil {
			return []Defect{{
				Code:    DefectInvalidPath,
				Message: "path escapes directory",
				Paths:   []string{rel},
			}}
		}
	}
	return nil
}

func workspaceRoot(workspace string) (string, error) {
	if workspace == "" {
		return "", errInvalidPath
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return "", err
		}
		abs = resolved
		info, err = os.Lstat(abs)
		if err != nil {
			return "", err
		}
	}
	if !info.IsDir() {
		return "", errInvalidPath
	}
	return abs, nil
}

func proveInside(root, candidate string) error {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return errInvalidPath
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errInvalidPath
	}
	return nil
}

// containedRel walks rel from the real workspace root with Lstat.
// Symlink components are rejected except a final dest leaf when probe
// is true (dangling dest stays output-exists). Missing suffix is
// allowed only after the deepest existing ancestor is contained.
func containedRel(workspace, rel string, probe bool) (string, bool, error) {
	if rel == "" || strings.IndexByte(rel, 0) >= 0 {
		return "", false, errInvalidPath
	}
	root, err := workspaceRoot(workspace)
	if err != nil {
		return "", false, err
	}
	normalized := strings.ReplaceAll(rel, `\`, "/")
	if filepath.IsAbs(rel) || filepath.IsAbs(normalized) || strings.HasPrefix(normalized, "/") {
		return "", false, errInvalidPath
	}
	cleaned, escaped := intpath.Clean(normalized)
	if escaped || cleaned == "" || cleaned == "." {
		return "", false, errInvalidPath
	}
	parts := strings.Split(cleaned, "/")
	cur := root
	for i, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", false, errInvalidPath
		}
		next := filepath.Join(cur, part)
		info, err := os.Lstat(next)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", false, err
			}
			cand := cur
			for _, p := range parts[i:] {
				cand = filepath.Join(cand, p)
			}
			if err := proveInside(root, cur); err != nil {
				return "", false, err
			}
			if err := proveInside(root, cand); err != nil {
				return "", false, err
			}
			return cand, false, nil
		}
		last := i == len(parts)-1
		if info.Mode()&os.ModeSymlink != 0 {
			if last && probe {
				if err := proveInside(root, next); err != nil {
					return "", false, err
				}
				return next, true, nil
			}
			return "", false, errInvalidPath
		}
		if !last && !info.IsDir() {
			if probe {
				cand := next
				for _, p := range parts[i+1:] {
					cand = filepath.Join(cand, p)
				}
				if err := proveInside(root, cand); err != nil {
					return "", false, err
				}
				return cand, false, nil
			}
			return "", false, errInvalidPath
		}
		cur = next
	}
	if err := proveInside(root, cur); err != nil {
		return "", false, err
	}
	return cur, true, nil
}

func containedFile(workspace, rel string) (string, error) {
	abs, present, err := containedRel(workspace, rel, false)
	if err != nil {
		return "", err
	}
	if !present {
		return "", os.ErrNotExist
	}
	return abs, nil
}
