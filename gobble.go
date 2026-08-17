// Package gobble is the Go pipeline library for composing bioinformatics pipelines.
//
// PathSpec is the public parameterized path model. Locked concepts map to
// exported fields and JSON keys:
//
//	DirName    → Dir    (JSON dir)
//	Prefix     → Lead   (JSON lead)
//	BaseName   → Name   (JSON name)
//	Suffixes   → Steps  (JSON steps)
//	Extension  → Ext    (JSON ext)
//
// Authors build a [Pipeline] and call [Compose] to obtain an immutable [Graph].
// [Validate] re-checks compose defects and rejects plan-time path conflicts
// and unsupported backends.
// The public surface is unsupported except these locked PathSpec concepts.
package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Validate re-checks compose defects on g and rejects rendered-path
// conflicts and unsupported backends.
//
// On any defect it returns [*Error] with Op "validate". A clean graph
// returns a nil error.
func Validate(g *Graph) error {
	if g == nil {
		return &Error{Op: "validate", Defects: []Defect{{
			Code:    DefectInvalidName,
			Message: "nil graph",
		}}}
	}
	return publicError("validate", engine.Validate(snapshotGraph(g)))
}
