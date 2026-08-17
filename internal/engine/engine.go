// Package engine owns validation walks, plan construction,
// pre-execution run checks, and process and Docker execution.
//
// Package gobble translates Pipeline and Graph values into the snapshot
// and Document types defined here. Snapshot is the validate view.
// Document and TaskPlan are the execution view. This package must not
// import github.com/HahyeonJeon/gobble or any cmd path.
//
// Path render and restage rules are copied from package gobble so this
// package stays import-free. Tests in package gobble compare both
// renderers. Do not change one copy without the other.
package engine

import "strings"

// Defect codes match the public gobble.DefectCode strings.
const (
	DefectCycle              = "cycle"
	DefectMissingInput       = "missing-input"
	DefectMissingOutput      = "missing-output"
	DefectMissingCommand     = "missing-command"
	DefectInvalidName        = "invalid-name"
	DefectInvalidPath        = "invalid-path"
	DefectConflict           = "conflict"
	DefectUnsupportedBackend = "unsupported-backend"
	DefectOccupiedWorkspace  = "occupied-workspace"
	DefectOutputExists       = "output-exists"
	DefectFailed             = "failed"
)

// Defect is one named failure found by a validation walk.
type Defect struct {
	Code    string
	Unit    string
	Message string
	Paths   []string
}

// DeriveRule selects how a related-file bind derives a Path.
// The zero value is DeriveAppend.
type DeriveRule int

const (
	// DeriveAppend appends a related extension. It is the default rule.
	DeriveAppend DeriveRule = iota
	// DeriveReplaceExt replaces the current extension.
	DeriveReplaceExt
)

// FromKind is the kind of bind source recorded on a snapshot Bind.
type FromKind int

const (
	// FromZero means From was never set.
	FromZero FromKind = iota
	// FromInput names a pipeline input.
	FromInput
	// FromOut names a task output port.
	FromOut
	// FromIn names a task input port.
	FromIn
)

// NodeKind is a snapshot tree node kind used for sibling-name checks.
type NodeKind int

const (
	// NodeModule is a named module.
	NodeModule NodeKind = iota
	// NodeBranch is a named branch.
	NodeBranch
	// NodeMerge is a named merge.
	NodeMerge
	// NodeTask is a task leaf.
	NodeTask
)

// Path is an engine-owned snapshot of a PathSpec, including unexported
// literal state that package gobble copies across the seam.
type Path struct {
	Dir     string
	Lead    string
	Name    string
	Steps   []string
	Ext     string
	Literal bool
	Opaque  string
	BadLit  bool
}

// Bind is an engine-owned bind snapshot.
//
// When Resolved is true, Spec is the final path and is not classified
// against From. Graph translation sets Resolved.
type Bind struct {
	Name     string
	Spec     Path
	FromKind FromKind
	FromName string
	FromTask string
	Rule     DeriveRule
	Resolved bool
}

// Input is a pipeline input snapshot.
type Input struct {
	Name string
	Spec Path
}

// Task is an engine-owned task snapshot.
type Task struct {
	ID       string
	Name     string
	Command  []string
	Backend  string
	CPU      float64
	Inputs   []Bind
	Outputs  []Bind
	OutCalls []string
	InCalls  []string
}

// Node is a tree node for sibling-name checks on a Pipeline snapshot.
type Node struct {
	Kind     NodeKind
	Name     string
	Children []Node
	TaskID   string
}

// Snapshot is the engine-owned Pipeline or Graph view used by the walks.
type Snapshot struct {
	Name   string
	Inputs []Input
	Tasks  []Task
	Nodes  []Node
}

func (p Path) clone() Path {
	if len(p.Steps) > 0 {
		steps := make([]string, len(p.Steps))
		copy(steps, p.Steps)
		p.Steps = steps
	}
	return p
}

// Render returns one comparable path string, or a DefectInvalidPath.
func (p Path) Render() (string, *Defect) {
	if d := p.fieldError(); d != nil {
		return "", d
	}
	var filename string
	if p.Literal {
		filename = p.Opaque
	} else {
		filename = p.Lead + p.Name
		for _, step := range p.Steps {
			filename += "." + stripOneLeadingDot(step)
		}
		if p.Ext != "" {
			if strings.HasPrefix(p.Ext, ".") {
				filename += p.Ext
			} else {
				filename += "." + p.Ext
			}
		}
	}
	raw := filename
	if p.Dir != "" {
		raw = joinSlash(p.Dir, filename)
	}
	cleaned, escaped := cleanPath(raw)
	if escaped {
		return "", invalidPath("path escapes directory", raw)
	}
	return cleaned, nil
}

func (p Path) fieldError() *Defect {
	if p.BadLit {
		return invalidPath("literal does not allow this method")
	}
	if p.Literal {
		if p.Opaque == "" {
			return invalidPath("empty literal path")
		}
		if strings.HasSuffix(p.Opaque, ".") {
			return invalidPath("empty extension token", p.Opaque)
		}
		return nil
	}
	if p.Lead == "" && p.Name == "" {
		return invalidPath("empty lead and name")
	}
	if d := fieldChars("lead", p.Lead); d != nil {
		return d
	}
	if d := fieldChars("name", p.Name); d != nil {
		return d
	}
	if hasDotComponent(p.Name) {
		return invalidPath("name is a dot path component", p.Name)
	}
	for _, step := range p.Steps {
		if stripOneLeadingDot(step) == "" {
			return invalidPath("empty step", step)
		}
		if d := fieldChars("step", step); d != nil {
			return d
		}
		if hasDotComponent(step) {
			return invalidPath("step is a dot path component", step)
		}
	}
	if d := fieldChars("ext", p.Ext); d != nil {
		return d
	}
	if p.Ext != "" && (stripOneLeadingDot(p.Ext) == "" || hasDotComponent(p.Ext) || strings.HasSuffix(p.Ext, ".")) {
		return invalidPath("ext is a dot path component", p.Ext)
	}
	return nil
}

func (p Path) appendExt(extra string) Path {
	out := p.clone()
	token := "." + stripOneLeadingDot(extra)
	if out.Literal {
		out.Opaque += token
		return out
	}
	out.Ext += token
	return out
}

func (p Path) replaceExt(ext string) Path {
	out := p.clone()
	out.Ext = ext
	if out.Literal {
		out.BadLit = true
	}
	return out
}

func isZeroPath(p Path) bool {
	return !p.Literal && p.Dir == "" && p.Lead == "" && p.Name == "" && len(p.Steps) == 0 && p.Ext == ""
}

func isRelatedFile(p Path) bool {
	return !p.Literal && p.Lead == "" && p.Name == "" && len(p.Steps) == 0 && p.Ext != ""
}

// Classify applies inherit, related-file, and restage rules on snapshot Paths.
func Classify(spec, from Path, rule DeriveRule) Path {
	return classifyPath(spec, from, rule)
}

func classifyPath(spec, from Path, rule DeriveRule) Path {
	if isZeroPath(spec) {
		return from.clone()
	}
	if isRelatedFile(spec) {
		var derived Path
		if rule == DeriveReplaceExt {
			derived = from.replaceExt(spec.Ext)
		} else {
			derived = from.appendExt(spec.Ext)
		}
		if spec.Dir != "" {
			derived.Dir = spec.Dir
		}
		return derived
	}
	return restagePath(spec, from)
}

func restagePath(spec, from Path) Path {
	if spec.Literal {
		out := spec.clone()
		if spec.Dir == "" {
			out.Dir = from.Dir
		}
		return out
	}
	out := from.clone()
	if spec.Dir != "" {
		out.Dir = spec.Dir
	}
	if spec.Lead != "" {
		out.Lead = spec.Lead
	}
	if spec.Name != "" {
		out.Name = spec.Name
	}
	if len(spec.Steps) > 0 {
		out.Steps = copyStrings(spec.Steps)
	}
	if spec.Ext != "" {
		out.Ext = spec.Ext
	}
	// Non-literal restage that sets Lead, Name, Steps, or Ext must not
	// keep a Literal parent's opacity; Render would ignore the new fields.
	if spec.Lead != "" || spec.Name != "" || len(spec.Steps) > 0 || spec.Ext != "" {
		out.Literal = false
		out.Opaque = ""
		out.BadLit = false
	}
	return out
}

func invalidPath(message string, paths ...string) *Defect {
	return &Defect{
		Code:    DefectInvalidPath,
		Message: message,
		Paths:   paths,
	}
}

func fieldChars(field, value string) *Defect {
	if containsIllegal(value) {
		return invalidPath(field + " contains an illegal character")
	}
	return nil
}

func containsIllegal(value string) bool {
	return strings.ContainsAny(value, "/\\\x00")
}

func hasDotComponent(s string) bool {
	if s == "." || s == ".." {
		return true
	}
	s = strings.ReplaceAll(s, `\`, "/")
	for _, part := range strings.Split(s, "/") {
		if part == "." || part == ".." {
			return true
		}
	}
	return false
}

func stripOneLeadingDot(s string) string {
	if strings.HasPrefix(s, ".") {
		return s[1:]
	}
	return s
}

func joinSlash(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		p = strings.ReplaceAll(p, `\`, "/")
		if b.Len() == 0 {
			b.WriteString(p)
			continue
		}
		if !strings.HasSuffix(b.String(), "/") {
			b.WriteByte('/')
		}
		b.WriteString(strings.TrimPrefix(p, "/"))
	}
	return b.String()
}

// cleanPath collapses "." and duplicate slashes. It applies ".." only
// while the first path component remains. escaped is true when ".."
// would leave that component.
func cleanPath(p string) (string, bool) {
	p = strings.ReplaceAll(p, `\`, "/")
	if p == "" {
		return "", false
	}
	abs := strings.HasPrefix(p, "/")
	out := make([]string, 0, strings.Count(p, "/")+1)
	for i, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." {
			if i == 0 && abs {
				out = append(out, "")
			}
			continue
		}
		if seg == ".." {
			if abs {
				if len(out) <= 2 {
					return "", true
				}
				out = out[:len(out)-1]
				continue
			}
			if len(out) <= 1 {
				return "", true
			}
			out = out[:len(out)-1]
			continue
		}
		out = append(out, seg)
	}
	if abs {
		if len(out) == 0 || (len(out) == 1 && out[0] == "") {
			return "/", false
		}
		return strings.Join(out, "/"), false
	}
	if len(out) == 0 {
		return ".", false
	}
	return strings.Join(out, "/"), false
}

func copyStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
