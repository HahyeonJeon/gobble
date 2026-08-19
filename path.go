package gobble

import (
	"errors"
	"strings"

	intpath "github.com/HahyeonJeon/gobble/internal/path"
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
	return Directory{path: intpath.Join(parts...)}
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
// Fields are Dir, Prefix, Base, Suffixes, and Ext. Equality is field
// equality, not rendered-path equality. Methods use value receivers
// and return a new PathSpec.
type PathSpec struct {
	// Dir is directory placement.
	Dir Directory
	// Prefix is the leading author tokens.
	Prefix string
	// Base is the stable name. "." and ".." are invalid.
	Base string
	// Suffixes is the ordered processing stack.
	Suffixes []string
	// Ext is the exact compound extension.
	Ext string

	literal bool
	opaque  string
	badLit  bool
}

// DeriveRule selects how a related-file bind derives a PathSpec.
// A related-file bind has From set and Spec with only Ext, and optionally
// Dir, populated. The zero value is DeriveAppend.
type DeriveRule int

const (
	// DeriveAppend appends a related extension. It is the default rule.
	DeriveAppend DeriveRule = iota
	// DeriveReplaceExt replaces the current extension.
	DeriveReplaceExt
)

// Literal returns a PathSpec that stores path as an opaque filename.
// If path has a directory, Dir is the parent and the last component is the
// opaque filename. Prefix, Base, Suffixes, and Ext stay empty.
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
	s, err := specFrom(p).Render()
	if err != nil {
		var pe *intpath.Error
		if errors.As(err, &pe) {
			return "", renderInvalid(pe.Message, pe.Paths...)
		}
		return "", renderInvalid(err.Error())
	}
	return s, nil
}

// Equal reports whether p and q have the same Dir string, Prefix, Base,
// Suffixes elements, Ext, and literal opacity. It does not compare
// rendered strings or the invalid-method marker set when a Literal
// method is refused.
func (p PathSpec) Equal(q PathSpec) bool {
	if p.Dir.String() != q.Dir.String() {
		return false
	}
	if p.Prefix != q.Prefix || p.Base != q.Base || p.Ext != q.Ext {
		return false
	}
	if p.literal != q.literal || p.opaque != q.opaque {
		return false
	}
	if len(p.Suffixes) != len(q.Suffixes) {
		return false
	}
	for i := range p.Suffixes {
		if p.Suffixes[i] != q.Suffixes[i] {
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

// WithPrefix returns a copy of p with Prefix replaced. On a Literal the
// copy is marked invalid and Render returns DefectInvalidPath.
func (p PathSpec) WithPrefix(prefix string) PathSpec {
	out := p.clone()
	out.Prefix = prefix
	if out.literal {
		out.badLit = true
	}
	return out
}

// WithBase returns a copy of p with Base replaced. On a Literal the copy
// is marked invalid and Render returns DefectInvalidPath.
func (p PathSpec) WithBase(base string) PathSpec {
	out := p.clone()
	out.Base = base
	if out.literal {
		out.badLit = true
	}
	return out
}

// AppendSuffix returns a copy of p with token appended to Suffixes after
// stripping one leading ".". On a Literal the copy is marked invalid and
// Render returns DefectInvalidPath.
func (p PathSpec) AppendSuffix(token string) PathSpec {
	out := p.clone()
	out.Suffixes = append(out.Suffixes, intpath.StripDot(token))
	if out.literal {
		out.badLit = true
	}
	return out
}

// WithExt returns a copy of p with Ext replaced by ext. On a Literal the
// copy is marked invalid and Render returns DefectInvalidPath.
func (p PathSpec) WithExt(ext string) PathSpec {
	out := p.clone()
	out.Ext = ext
	if out.literal {
		out.badLit = true
	}
	return out
}

// AppendExt returns a related PathSpec by appending extra to Ext, or to a
// Literal opaque filename. One leading "." is stripped from extra, then
// "." is prepended. An extra that is empty after strip makes Render
// return DefectInvalidPath.
func (p PathSpec) AppendExt(extra string) PathSpec {
	return specTo(intpath.AppendExt(specFrom(p), extra))
}

func (p PathSpec) clone() PathSpec {
	if len(p.Suffixes) > 0 {
		suf := make([]string, len(p.Suffixes))
		copy(suf, p.Suffixes)
		p.Suffixes = suf
	}
	return p
}

func specFrom(p PathSpec) intpath.Spec {
	return intpath.Spec{
		Dir:      p.Dir.String(),
		Prefix:   p.Prefix,
		Base:     p.Base,
		Suffixes: copyStrings(p.Suffixes),
		Ext:      p.Ext,
		Literal:  p.literal,
		Opaque:   p.opaque,
		BadLit:   p.badLit,
	}
}

func specTo(s intpath.Spec) PathSpec {
	return PathSpec{
		Dir:      Dir(s.Dir),
		Prefix:   s.Prefix,
		Base:     s.Base,
		Suffixes: copyStrings(s.Suffixes),
		Ext:      s.Ext,
		literal:  s.Literal,
		opaque:   s.Opaque,
		badLit:   s.BadLit,
	}
}
