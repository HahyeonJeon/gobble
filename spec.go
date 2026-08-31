package gobble

import intpath "github.com/HahyeonJeon/gobble/internal/path"

// TaskSpec is the declaration of one pipeline task.
//
// Image non-empty means a container task. Image empty means a local process.
// Backend empty or "local" is allowed. Any other backend is unsupported at
// validate and plan time, not at builder time.
type TaskSpec struct {
	Name      string
	Command   []string
	Script    string
	Image     string
	Backend   string
	Inputs    []Bind
	Outputs   []Bind
	Params    []Param
	Env       map[string]string
	Resources Resources
}

// Bind is one named input or output port on a task.
//
// Name is the local port name only. It never implicitly matches a pipeline
// input. A Bind is exactly one of a File Spec, a Group, or a Tree.
// A non-nil Group, including an empty list, is a Group bind. A Tree with
// IsZero false is a Tree bind. When From is set, Spec is classified as
// follows: a zero Spec inherits the spec Compose resolves for that port,
// which may itself be inherited, not Handle.Spec; a Spec that has only Ext
// set, and optionally Dir, is a related file of From using Rule; any other
// Spec restages field by field, taking each set field from Spec and
// inheriting the rest from From. A Literal restage keeps opacity and merges
// Dir unless Spec.Dir is set. Tree has no related-file sugar: a present
// Tree with a zero Dir inherits the From directory; a non-zero Dir uses
// that directory. A Tree output inside a Scatter may instead use that Scatter
// membership as From and a non-zero Dir as its parent; each runtime member
// derives one child Tree named by the stable member key. From should
// name another task or a pipeline input. A From that points at the same task is
// a cycle. A Group From must name another
// Group port or Group pipeline input with the same member-name set. A Tree
// From must name another Tree port or Tree pipeline input.
// Rule is used when this Bind is a related file of From. The zero Rule is
// DeriveAppend.
type Bind struct {
	Name  string
	Spec  PathSpec
	Group Group
	Tree  Tree
	From  Handle
	Rule  DeriveRule
}

// Tree is a declared directory artifact. Dir is placement inside the
// isolate work root. IsZero is false only for a Tree constructed with
// DeclareTree, including a zero Dir that inherits From.
type Tree struct {
	Dir     Directory
	present bool
}

// DeclareTree returns a Tree bind for dir. A zero dir is still a Tree
// bind: Compose inherits From or reports invalid-path.
func DeclareTree(dir Directory) Tree {
	return Tree{Dir: dir, present: true}
}

// IsZero reports whether t is not a Tree bind. A present Tree with a
// zero Dir is not zero.
func (t Tree) IsZero() bool {
	return !t.present
}

// Group is an ordered list of named regular-file members on one Bind.
// A nil Group is a non-Group bind. A non-nil empty Group is invalid.
type Group []Member

// Member is one named regular-file path in a Group.
type Member struct {
	Name string
	Spec PathSpec
}

// Param is a named string parameter on a task.
type Param struct {
	Name  string
	Value string
}

// Resources records requested compute.
//
// CPU is cores. Zero means unspecified. Negative and non-finite CPU are
// invalid. Memory is Docker --memory syntax: integer bytes, or a number
// plus one suffix b, k, m, or g, case-insensitive. Empty Memory is
// unspecified.
type Resources struct {
	CPU    float64
	Memory string
}

func copyStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func copyParams(in []Param) []Param {
	if in == nil {
		return nil
	}
	out := make([]Param, len(in))
	copy(out, in)
	return out
}

func copyEnv(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func findBind(binds []Bind, name string) (Bind, bool) {
	for _, b := range binds {
		if b.Name == name {
			return b, true
		}
	}
	return Bind{}, false
}

func hasBind(binds []Bind, name string) bool {
	_, ok := findBind(binds, name)
	return ok
}

func isZeroSpec(p PathSpec) bool {
	return intpath.IsZero(specFrom(p))
}

func isRelatedFile(p PathSpec) bool {
	return intpath.IsRelated(specFrom(p))
}

// classifySpec resolves spec against from: zero inherit, related-file sugar,
// then restage.
func classifySpec(spec, from PathSpec, rule DeriveRule) PathSpec {
	return specTo(intpath.Classify(specFrom(spec), specFrom(from), intpath.DeriveRule(rule)))
}
