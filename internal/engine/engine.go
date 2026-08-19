// Package engine owns validation walks, plan construction,
// pre-execution run checks, occupancy release, inspect, resume,
// and process and Docker execution.
//
// Package gobble renders PathSpec through internal/path, runs
// compose-time checks, and translates Graph values into Document.
// Document and TaskPlan are the only engine payload. This package
// must not import github.com/HahyeonJeon/gobble or any cmd path.
//
// Path render, restage, and classify live in package internal/path.
package engine

import (
	"errors"

	intpath "github.com/HahyeonJeon/gobble/internal/path"
)

// Defect codes match the public gobble.DefectCode strings.
const (
	DefectCycle              = "cycle"
	DefectMissingInput       = "missing-input"
	DefectMissingOutput      = "missing-output"
	DefectMissingCommand     = "missing-command"
	DefectInvalidName        = "invalid-name"
	DefectInvalidValue       = "invalid-value"
	DefectInvalidRequest     = "invalid-request"
	DefectInvalidPath        = "invalid-path"
	DefectConflict           = "conflict"
	DefectUnsupportedBackend = "unsupported-backend"
	DefectOccupiedWorkspace  = "occupied-workspace"
	DefectOutputExists       = "output-exists"
	DefectFailed             = "failed"
	DefectNeverReady         = "never-ready"
	DefectInvalidMemory      = "invalid-memory"
	DefectNotFound           = "not-found"
	DefectAlreadyReleased    = "already-released"
	DefectLiveOccupancy      = "live-occupancy"
	DefectForeignHost        = "foreign-host"
	DefectUnsupportedSchema  = "unsupported-schema"
	DefectNothingToResume    = "nothing-to-resume"
	DefectPlanDrift          = "plan-drift"
)

// SchemaVersion is the control-document version this engine writes.
// Schema 0 and 1 documents are unsupported-schema. There is no migration.
const SchemaVersion = 2

// Artifact kinds recorded on Document IO. Tree is unused until later work.
const (
	ArtifactFile  = "file"
	ArtifactGroup = "group"
	ArtifactTree  = "tree"
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

// Path is the plan-document record of a PathSpec, including unexported
// literal state that package gobble copies across the seam.
type Path struct {
	Dir      string
	Prefix   string
	Base     string
	Suffixes []string
	Ext      string
	Literal  bool
	Opaque   string
	BadLit   bool
}

func (p Path) clone() Path {
	if len(p.Suffixes) > 0 {
		suf := make([]string, len(p.Suffixes))
		copy(suf, p.Suffixes)
		p.Suffixes = suf
	}
	return p
}

func (p Path) spec() intpath.Spec {
	return intpath.Spec{
		Dir:      p.Dir,
		Prefix:   p.Prefix,
		Base:     p.Base,
		Suffixes: copyStrings(p.Suffixes),
		Ext:      p.Ext,
		Literal:  p.Literal,
		Opaque:   p.Opaque,
		BadLit:   p.BadLit,
	}
}

func pathFromSpec(s intpath.Spec) Path {
	return Path{
		Dir:      s.Dir,
		Prefix:   s.Prefix,
		Base:     s.Base,
		Suffixes: copyStrings(s.Suffixes),
		Ext:      s.Ext,
		Literal:  s.Literal,
		Opaque:   s.Opaque,
		BadLit:   s.BadLit,
	}
}

// Render returns one comparable path string, or a DefectInvalidPath.
func (p Path) Render() (string, *Defect) {
	s, err := p.spec().Render()
	if err != nil {
		var pe *intpath.Error
		if errors.As(err, &pe) {
			return "", invalidPath(pe.Message, pe.Paths...)
		}
		return "", invalidPath(err.Error())
	}
	return s, nil
}

func isZeroPath(p Path) bool {
	return intpath.IsZero(p.spec())
}

// Classify applies inherit, related-file, and restage rules on Path records.
func Classify(spec, from Path, rule DeriveRule) Path {
	return pathFromSpec(intpath.Classify(spec.spec(), from.spec(), intpath.DeriveRule(rule)))
}

func invalidPath(message string, paths ...string) *Defect {
	return &Defect{
		Code:    DefectInvalidPath,
		Message: message,
		Paths:   paths,
	}
}

func copyStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
