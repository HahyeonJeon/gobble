package gobble

import (
	"strings"
)

// Directory is placement for a PathSpec. It is not a directory output artifact.
type Directory struct {
	path string
}

// Dir returns a Directory for path. An empty path is the zero Directory.
func Dir(path string) Directory {
	return Directory{path: path}
}

// Join returns a Directory with elem appended using forward slashes.
// It does not collapse "." or "..". Render collapses internal ".." that
// stay under the first path component and returns DefectInvalidPath when
// ".." would leave that component.
func (d Directory) Join(elem ...string) Directory {
	parts := make([]string, 0, 1+len(elem))
	if d.path != "" {
		parts = append(parts, d.path)
	}
	for _, e := range elem {
		if e != "" {
			parts = append(parts, e)
		}
	}
	return Directory{path: joinSlash(parts...)}
}

// String returns the directory path.
func (d Directory) String() string {
	return d.path
}

// IsZero reports whether d has no directory prefix.
func (d Directory) IsZero() bool {
	return d.path == ""
}

// PathSpec is the public parameterized path model.
//
// Field names map from the locked concepts DirName, Prefix, BaseName,
// Suffixes, and Extension. Equality is field equality, not rendered-path
// equality. Methods use value receivers and return a new PathSpec.
type PathSpec struct {
	// Dir is directory placement (DirName).
	Dir Directory
	// Lead is the leading author tokens (Prefix).
	Lead string
	// Name is the stable name (BaseName).
	Name string
	// Steps is the ordered processing stack (Suffixes).
	Steps []string
	// Ext is the exact compound extension (Extension).
	Ext string

	literal bool
	opaque  string
	badLit  bool
}

// DeriveRule selects how a related-file bind derives a PathSpec.
// The zero value is DeriveAppend.
type DeriveRule int

const (
	// DeriveAppend appends a related extension. It is the default rule.
	DeriveAppend DeriveRule = iota
	// DeriveReplaceExt replaces the current extension.
	DeriveReplaceExt
)

// Literal returns a PathSpec that stores path as an opaque filename.
// If path has a directory, Dir is the parent and the last component is the
// opaque filename. Lead, Name, Steps, and Ext stay empty.
func Literal(path string) PathSpec {
	normalized := strings.ReplaceAll(path, `\`, "/")
	if normalized == "" {
		return PathSpec{literal: true}
	}
	i := strings.LastIndex(normalized, "/")
	if i < 0 {
		return PathSpec{literal: true, opaque: normalized}
	}
	dir, file := normalized[:i], normalized[i+1:]
	if dir == "" {
		dir = "/"
	}
	return PathSpec{Dir: Dir(dir), literal: true, opaque: file}
}

// Render returns one comparable path string for a valid PathSpec.
// Invalid specs return an [*Error] with Op "render" and DefectInvalidPath.
func (p PathSpec) Render() (string, error) {
	if err := p.fieldError(); err != nil {
		return "", err
	}
	var filename string
	if p.literal {
		filename = p.opaque
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
	if !p.Dir.IsZero() {
		raw = joinSlash(p.Dir.String(), filename)
	}
	cleaned, escaped := cleanPath(raw)
	if escaped {
		return "", renderInvalid("path escapes directory", raw)
	}
	return cleaned, nil
}

// Equal reports whether p and q have the same Dir string, Lead, Name, Steps
// elements, Ext, and literal opacity. It does not compare rendered strings.
func (p PathSpec) Equal(q PathSpec) bool {
	if p.Dir.String() != q.Dir.String() {
		return false
	}
	if p.Lead != q.Lead || p.Name != q.Name || p.Ext != q.Ext {
		return false
	}
	if p.literal != q.literal || p.opaque != q.opaque {
		return false
	}
	if len(p.Steps) != len(q.Steps) {
		return false
	}
	for i := range p.Steps {
		if p.Steps[i] != q.Steps[i] {
			return false
		}
	}
	return true
}

// WithDir returns a copy of p with Dir set to d.
func (p PathSpec) WithDir(d Directory) PathSpec {
	out := p.clone()
	out.Dir = d
	return out
}

// WithLead returns a copy of p with Lead replaced. On a Literal the copy is
// marked invalid and Render returns DefectInvalidPath.
func (p PathSpec) WithLead(lead string) PathSpec {
	out := p.clone()
	out.Lead = lead
	if out.literal {
		out.badLit = true
	}
	return out
}

// AppendStep returns a copy of p with token appended to Steps after stripping
// one leading ".". On a Literal the copy is marked invalid and Render returns
// DefectInvalidPath.
func (p PathSpec) AppendStep(token string) PathSpec {
	out := p.clone()
	out.Steps = append(out.Steps, stripOneLeadingDot(token))
	if out.literal {
		out.badLit = true
	}
	return out
}

// WithExt returns a copy of p with Ext replaced by ext. On a Literal the copy
// is marked invalid and Render returns DefectInvalidPath.
func (p PathSpec) WithExt(ext string) PathSpec {
	out := p.clone()
	out.Ext = ext
	if out.literal {
		out.badLit = true
	}
	return out
}

// Append returns a related PathSpec by appending extra to Ext, or to a Literal
// opaque filename. One leading "." is stripped from extra, then "." is prepended.
// An extra that is empty after strip therefore stores a trailing ".".
func (p PathSpec) Append(extra string) PathSpec {
	out := p.clone()
	token := "." + stripOneLeadingDot(extra)
	if out.literal {
		out.opaque += token
		return out
	}
	out.Ext += token
	return out
}

// ReplaceExtension returns a copy of p with Ext replaced by ext. On a Literal
// the copy is marked invalid and Render returns DefectInvalidPath.
func (p PathSpec) ReplaceExtension(ext string) PathSpec {
	out := p.clone()
	out.Ext = ext
	if out.literal {
		out.badLit = true
	}
	return out
}

func (p PathSpec) clone() PathSpec {
	if len(p.Steps) > 0 {
		steps := make([]string, len(p.Steps))
		copy(steps, p.Steps)
		p.Steps = steps
	}
	return p
}

func (p PathSpec) fieldError() *Error {
	if p.badLit {
		return renderInvalid("literal does not allow this method")
	}
	if p.literal {
		if p.opaque == "" {
			return renderInvalid("empty literal path")
		}
		return nil
	}
	if p.Lead == "" && p.Name == "" {
		return renderInvalid("empty lead and name")
	}
	if err := fieldChars("lead", p.Lead); err != nil {
		return err
	}
	if err := fieldChars("name", p.Name); err != nil {
		return err
	}
	for _, step := range p.Steps {
		if stripOneLeadingDot(step) == "" {
			return renderInvalid("empty step", step)
		}
		if err := fieldChars("step", step); err != nil {
			return err
		}
		if hasDotComponent(step) {
			return renderInvalid("step is a dot path component", step)
		}
	}
	if err := fieldChars("ext", p.Ext); err != nil {
		return err
	}
	if p.Ext != "" && (stripOneLeadingDot(p.Ext) == "" || hasDotComponent(p.Ext)) {
		return renderInvalid("ext is a dot path component", p.Ext)
	}
	return nil
}

func fieldChars(field, value string) *Error {
	if strings.ContainsAny(value, "/\\\x00") {
		return renderInvalid(field + " contains an illegal character")
	}
	return nil
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
