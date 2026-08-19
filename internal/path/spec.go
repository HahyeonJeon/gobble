// Package path owns PathSpec render, restage, classify, and field rules.
// Package gobble and package engine import it. This package must not
// import github.com/HahyeonJeon/gobble.
package path

import "strings"

// DeriveRule selects how a related-file bind derives a Spec.
// The zero value is DeriveAppend.
type DeriveRule int

const (
	// DeriveAppend appends a related extension. It is the default rule.
	DeriveAppend DeriveRule = iota
	// DeriveReplaceExt replaces the current extension.
	DeriveReplaceExt
)

// Spec is the shared PathSpec field model.
type Spec struct {
	Dir      string
	Prefix   string
	Base     string
	Suffixes []string
	Ext      string
	Literal  bool
	Opaque   string
	BadLit   bool
}

// Error is a render or field-rule failure.
type Error struct {
	Message string
	Paths   []string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Clone returns a Spec whose Suffixes slice does not alias s.
func (s Spec) Clone() Spec {
	if len(s.Suffixes) > 0 {
		suf := make([]string, len(s.Suffixes))
		copy(suf, s.Suffixes)
		s.Suffixes = suf
	}
	return s
}

// Render returns one comparable path string for a valid Spec.
func (s Spec) Render() (string, error) {
	if err := s.fieldError(); err != nil {
		return "", err
	}
	var filename string
	if s.Literal {
		filename = s.Opaque
	} else {
		filename = s.Prefix + s.Base
		for _, suf := range s.Suffixes {
			filename += "." + StripDot(suf)
		}
		if s.Ext != "" {
			if strings.HasPrefix(s.Ext, ".") {
				filename += s.Ext
			} else {
				filename += "." + s.Ext
			}
		}
	}
	raw := filename
	if s.Dir != "" {
		raw = Join(s.Dir, filename)
	}
	cleaned, escaped := Clean(raw)
	if escaped {
		return "", invalid("path escapes directory", raw)
	}
	return cleaned, nil
}

func (s Spec) fieldError() *Error {
	if s.BadLit {
		return invalid("literal does not allow this method")
	}
	if s.Literal {
		if s.Opaque == "" {
			return invalid("empty literal path")
		}
		if strings.HasSuffix(s.Opaque, ".") {
			return invalid("empty extension token", s.Opaque)
		}
		return nil
	}
	if s.Prefix == "" && s.Base == "" {
		return invalid("empty prefix and base")
	}
	if err := fieldChars("prefix", s.Prefix); err != nil {
		return err
	}
	if err := fieldChars("base", s.Base); err != nil {
		return err
	}
	if hasDotComponent(s.Base) {
		return invalid("base is a dot path component", s.Base)
	}
	for _, suf := range s.Suffixes {
		if StripDot(suf) == "" {
			return invalid("empty suffix", suf)
		}
		if err := fieldChars("suffix", suf); err != nil {
			return err
		}
		if hasDotComponent(suf) {
			return invalid("suffix is a dot path component", suf)
		}
	}
	if err := fieldChars("ext", s.Ext); err != nil {
		return err
	}
	if s.Ext != "" && (StripDot(s.Ext) == "" || hasDotComponent(s.Ext) || strings.HasSuffix(s.Ext, ".")) {
		return invalid("ext is a dot path component", s.Ext)
	}
	return nil
}

// IsZero reports whether s is a non-literal zero Spec.
func IsZero(s Spec) bool {
	return !s.Literal && s.Dir == "" && s.Prefix == "" && s.Base == "" && len(s.Suffixes) == 0 && s.Ext == ""
}

// IsRelated reports whether s is related-file sugar: only Ext, and
// optionally Dir, populated.
func IsRelated(s Spec) bool {
	return !s.Literal && s.Prefix == "" && s.Base == "" && len(s.Suffixes) == 0 && s.Ext != ""
}

// AppendExt returns a Spec with extra appended to Ext, or to a Literal
// opaque filename. One leading "." is stripped from extra, then "." is
// prepended.
func AppendExt(s Spec, extra string) Spec {
	out := s.Clone()
	token := "." + StripDot(extra)
	if out.Literal {
		out.Opaque += token
		return out
	}
	out.Ext += token
	return out
}

// ReplaceExt returns a copy of s with Ext replaced by ext. On a Literal
// the copy is marked invalid.
func ReplaceExt(s Spec, ext string) Spec {
	out := s.Clone()
	out.Ext = ext
	if out.Literal {
		out.BadLit = true
	}
	return out
}

// Classify applies inherit, related-file, and restage rules.
func Classify(spec, from Spec, rule DeriveRule) Spec {
	if IsZero(spec) {
		return from.Clone()
	}
	if IsRelated(spec) {
		var derived Spec
		if rule == DeriveReplaceExt {
			derived = ReplaceExt(from, spec.Ext)
		} else {
			derived = AppendExt(from, spec.Ext)
		}
		if spec.Dir != "" {
			derived.Dir = spec.Dir
		}
		return derived
	}
	return Restage(spec, from)
}

// Restage takes each set field from spec and inherits the rest from from.
func Restage(spec, from Spec) Spec {
	if spec.Literal {
		out := spec.Clone()
		if spec.Dir == "" {
			out.Dir = from.Dir
		}
		return out
	}
	out := from.Clone()
	if spec.Dir != "" {
		out.Dir = spec.Dir
	}
	if spec.Prefix != "" {
		out.Prefix = spec.Prefix
	}
	if spec.Base != "" {
		out.Base = spec.Base
	}
	if len(spec.Suffixes) > 0 {
		out.Suffixes = copyStrings(spec.Suffixes)
	}
	if spec.Ext != "" {
		out.Ext = spec.Ext
	}
	// Non-literal restage that sets Prefix, Base, Suffixes, or Ext must
	// not keep a Literal parent's opacity; Render would ignore the new fields.
	if spec.Prefix != "" || spec.Base != "" || len(spec.Suffixes) > 0 || spec.Ext != "" {
		out.Literal = false
		out.Opaque = ""
		out.BadLit = false
	}
	return out
}

func invalid(message string, paths ...string) *Error {
	return &Error{Message: message, Paths: paths}
}

func fieldChars(field, value string) *Error {
	if strings.ContainsAny(value, "/\\\x00") {
		return invalid(field + " contains an illegal character")
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

// StripDot removes one leading "." from s.
func StripDot(s string) string {
	if strings.HasPrefix(s, ".") {
		return s[1:]
	}
	return s
}

// Join joins parts with forward slashes. It does not collapse "." or "..".
func Join(parts ...string) string {
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

// Clean collapses "." and duplicate slashes. It applies ".." only while
// the first path component remains. escaped is true when ".." would
// leave that component.
func Clean(p string) (string, bool) {
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
