package engine

import (
	"math"
	"strconv"
	"strings"

	intpath "github.com/HahyeonJeon/gobble/internal/path"
)

type rendered struct {
	task string
	unit string
	path string
	out  bool
}

// Validate reports Document defects: rendered-path conflicts,
// unsupported backends, non-finite or negative CPU, unparseable Memory,
// malformed image, env literals, and artifact XOR.
func Validate(doc Document) []Defect {
	var defects []Defect
	var paths []rendered
	for i := range doc.Tasks {
		t := &doc.Tasks[i]
		id := t.ID
		if id == "" {
			id = t.Name
		}
		if t.Backend != "" && t.Backend != "local" {
			defects = append(defects, Defect{
				Code:    DefectUnsupportedBackend,
				Unit:    id,
				Message: "unsupported backend",
			})
		}
		if !finiteCPU(t.Resources.CPU) {
			defects = append(defects, Defect{
				Code:    DefectInvalidValue,
				Unit:    id,
				Message: "non-finite cpu",
			})
		} else if t.Resources.CPU < 0 {
			defects = append(defects, Defect{
				Code:    DefectInvalidValue,
				Unit:    id,
				Message: "negative cpu",
			})
		}
		if _, ok := parseMemory(t.Resources.Memory); !ok {
			defects = append(defects, Defect{
				Code:    DefectInvalidMemory,
				Unit:    id,
				Message: "invalid memory " + strconv.Quote(t.Resources.Memory),
			})
		}
		if msg := invalidImage(t.Image); msg != "" {
			defects = append(defects, Defect{
				Code:    DefectInvalidValue,
				Unit:    id,
				Message: msg,
			})
		}
		defects = append(defects, checkEnv(id, t.Env)...)
		defects = append(defects, checkParams(id, t.Params)...)
		for _, b := range t.Inputs {
			defects = append(defects, checkArtifactXOR(bindUnit(id, b.Name), b)...)
			paths = append(paths, recordIOPaths(id, b, false)...)
		}
		for _, b := range t.Outputs {
			defects = append(defects, checkArtifactXOR(bindUnit(id, b.Name), b)...)
			paths = append(paths, recordIOPaths(id, b, true)...)
		}
	}
	defects = append(defects, checkConflicts(paths)...)
	return defects
}

func checkParams(id string, params []ParamPlan) []Defect {
	seen := make(map[string]bool, len(params))
	var defects []Defect
	for _, p := range params {
		if seen[p.Name] {
			defects = append(defects, Defect{
				Code:    DefectInvalidValue,
				Unit:    id,
				Message: "duplicate param name",
			})
			continue
		}
		seen[p.Name] = true
	}
	return defects
}

func checkEnv(id string, env map[string]string) []Defect {
	var defects []Defect
	for k, v := range env {
		if k == "" {
			defects = append(defects, Defect{
				Code:    DefectInvalidValue,
				Unit:    id,
				Message: "empty env key",
			})
		} else if strings.Contains(k, "=") {
			defects = append(defects, Defect{
				Code:    DefectInvalidValue,
				Unit:    id,
				Message: "env key contains =",
			})
		}
		if v == "" {
			defects = append(defects, Defect{
				Code:    DefectInvalidValue,
				Unit:    id,
				Message: "empty env value",
			})
		}
	}
	return defects
}

func checkArtifactXOR(unit string, b IO) []Defect {
	kind := ioKind(b)
	hasGroup := b.Members != nil
	hasTree := b.Manifest != ""
	hasFile := b.Path != "" || !isZeroPath(b.Spec)
	switch kind {
	case ArtifactGroup:
		if hasTree {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "group and tree both set"}}
		}
		if !isZeroPath(b.Spec) && b.Path != "" {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "group and spec both set"}}
		}
		if !hasGroup {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "empty group"}}
		}
		if len(b.Members) == 0 {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "empty group"}}
		}
	case ArtifactTree:
		if hasGroup {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "group and tree both set"}}
		}
		if !isZeroPath(b.Spec) && (b.Spec.Prefix != "" || b.Spec.Base != "" || len(b.Spec.Suffixes) > 0 || b.Spec.Ext != "" || b.Spec.Literal) {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "spec and tree both set"}}
		}
	case ArtifactFile, "":
		if hasGroup && hasFile {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "group and spec both set"}}
		}
		if hasGroup && hasTree {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "group and tree both set"}}
		}
		if hasTree && hasFile {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "spec and tree both set"}}
		}
		if hasGroup && len(b.Members) == 0 {
			return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "empty group"}}
		}
	default:
		return []Defect{{Code: DefectInvalidValue, Unit: unit, Message: "invalid artifact kind"}}
	}
	return nil
}

func recordIOPaths(task string, b IO, out bool) []rendered {
	unit := bindUnit(task, b.Name)
	if b.Members != nil {
		outp := make([]rendered, 0, len(b.Members))
		for _, m := range b.Members {
			if m.Path == "" {
				continue
			}
			outp = append(outp, rendered{
				task: task,
				unit: unit,
				path: comparablePath(m.Path),
				out:  out,
			})
		}
		return outp
	}
	if b.Path == "" {
		return nil
	}
	return []rendered{{
		task: task,
		unit: unit,
		path: comparablePath(b.Path),
		out:  out,
	}}
}

func checkConflicts(paths []rendered) []Defect {
	var defects []Defect
	byTaskOut := make(map[string]map[string]string)
	outputs := make(map[string]string)
	for _, r := range paths {
		if r.path == "" || !r.out {
			continue
		}
		if byTaskOut[r.task] == nil {
			byTaskOut[r.task] = make(map[string]string)
		}
		byTaskOut[r.task][r.path] = r.unit
		if _, ok := outputs[r.path]; ok {
			defects = append(defects, Defect{
				Code:    DefectConflict,
				Unit:    r.unit,
				Message: "conflict",
				Paths:   []string{r.path},
			})
		} else {
			outputs[r.path] = r.unit
		}
	}
	for _, r := range paths {
		if r.out {
			continue
		}
		outs := byTaskOut[r.task]
		if outs == nil {
			continue
		}
		if outUnit, ok := outs[r.path]; ok {
			defects = append(defects, Defect{
				Code:    DefectConflict,
				Unit:    outUnit,
				Message: "conflict",
				Paths:   []string{r.path},
			})
		}
	}
	return defects
}

func comparablePath(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	cleaned, escaped := intpath.Clean(p)
	if escaped || cleaned == "" {
		return p
	}
	return cleaned
}

func finiteCPU(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// parseMemory parses Docker --memory syntax. Empty is unspecified (0, true).
func parseMemory(s string) (int64, bool) {
	if s == "" {
		return 0, true
	}
	if n, err := strconv.ParseUint(s, 10, 63); err == nil {
		return int64(n), true
	}
	if len(s) < 2 {
		return 0, false
	}
	var mul int64
	switch s[len(s)-1] {
	case 'b', 'B':
		mul = 1
	case 'k', 'K':
		mul = 1024
	case 'm', 'M':
		mul = 1024 * 1024
	case 'g', 'G':
		mul = 1024 * 1024 * 1024
	default:
		return 0, false
	}
	num := s[:len(s)-1]
	if n, err := strconv.ParseUint(num, 10, 63); err == nil {
		if mul != 1 && n > uint64(math.MaxInt64)/uint64(mul) {
			return 0, false
		}
		return int64(n) * mul, true
	}
	f, err := strconv.ParseFloat(num, 64)
	if err != nil || f < 0 || !finiteCPU(f) {
		return 0, false
	}
	bytes := f * float64(mul)
	if bytes > float64(math.MaxInt64) {
		return 0, false
	}
	return int64(bytes), true
}

func bindUnit(taskID, port string) string {
	if taskID == "" {
		return port
	}
	if port == "" {
		return taskID
	}
	return taskID + "." + port
}
