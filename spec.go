package gobble

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
// input. A Bind is either a single Spec or a Group, not both. A non-nil
// Group, including an empty list, is a Group bind. When From is set, Spec
// is classified as follows: a zero Spec inherits the spec Compose resolves
// for that port, which may itself be inherited, not Handle.Spec; a Spec
// that has only Ext set, and optionally Dir, is a related file of From
// using Rule; any other Spec restages field by field, taking each set field
// from Spec and inheriting the rest from From. A Literal restage keeps
// opacity and merges Dir unless Spec.Dir is set. From should name another
// task or a pipeline input. A From that points at the same task is a cycle.
// A Group From must name another Group port with the same member-name set.
// Rule is used when this Bind is a related file of From. The zero Rule is
// DeriveAppend.
type Bind struct {
	Name  string
	Spec  PathSpec
	Group Group
	From  Handle
	Rule  DeriveRule
}

// Group is an ordered list of named regular-file members on one Bind.
// A nil Group is a single-file bind. A non-nil empty Group is invalid.
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

func isZeroSpec(p PathSpec) bool {
	return !p.literal && p.Dir.IsZero() && p.Lead == "" && p.Name == "" && len(p.Steps) == 0 && p.Ext == ""
}

func isRelatedFile(p PathSpec) bool {
	return !p.literal && p.Lead == "" && p.Name == "" && len(p.Steps) == 0 && p.Ext != ""
}

// classifySpec resolves spec against from: zero inherit, related-file sugar,
// then restage.
func classifySpec(spec, from PathSpec, rule DeriveRule) PathSpec {
	if isZeroSpec(spec) {
		return from.clone()
	}
	if isRelatedFile(spec) {
		var derived PathSpec
		if rule == DeriveReplaceExt {
			derived = from.ReplaceExtension(spec.Ext)
		} else {
			derived = from.Append(spec.Ext)
		}
		if !spec.Dir.IsZero() {
			return derived.WithDir(spec.Dir)
		}
		return derived
	}
	return restageSpec(spec, from)
}

func restageSpec(spec, from PathSpec) PathSpec {
	if spec.literal {
		out := spec.clone()
		if spec.Dir.IsZero() {
			out.Dir = from.Dir
		}
		return out
	}
	out := from.clone()
	if !spec.Dir.IsZero() {
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
		out.literal = false
		out.opaque = ""
		out.badLit = false
	}
	return out
}
