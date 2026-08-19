package gobble

import (
	"errors"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// Validate re-checks compose defects on g and rejects rendered-path
// conflicts, unsupported backends, non-finite or negative CPU, and
// unparseable Memory.
//
// On any defect it returns [*Error] with Op "validate". A clean graph
// returns a nil error.
func Validate(g *Graph) error {
	if g == nil {
		return &Error{Op: "validate", Defects: []Defect{{
			Code:    DefectInvalidRequest,
			Message: "nil graph",
		}}}
	}
	if err := composeDefects("validate", graphCheck(g)); err != nil {
		return err
	}
	doc, err := planDocument(g)
	if err != nil {
		return validateOp(err)
	}
	return publicError("validate", engine.Validate(doc))
}

func validateOp(err error) error {
	if err == nil {
		return nil
	}
	var ge *Error
	if !errors.As(err, &ge) {
		return &Error{Op: "validate", Defects: []Defect{{
			Code:    DefectInvalidPath,
			Message: err.Error(),
		}}}
	}
	out := *ge
	out.Op = "validate"
	if ge.Defects != nil {
		out.Defects = append([]Defect(nil), ge.Defects...)
	}
	return &out
}
