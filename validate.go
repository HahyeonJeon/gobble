package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Validate re-checks compose defects on g and rejects rendered-path
// conflicts, unsupported backends, non-finite or negative CPU, and
// unparseable Memory.
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
