package gobble

import (
	"errors"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// preflight refuses an invalid start before occupy or execute.
// A nil error means the start is allowed, not that work ran.
func preflight(g *Graph, workspace string, cap int) error {
	if g == nil {
		return &Error{Op: "run", Defects: []Defect{{
			Code:    DefectInvalidName,
			Message: "nil graph",
		}}}
	}
	if pub := publicError("run", engine.Validate(snapshotGraph(g))); pub != nil {
		return pub
	}
	doc, err := planDocument(g)
	if err != nil {
		return runOp(err)
	}
	return publicError("run", engine.Check(engine.Request{
		Workspace: workspace,
		Cap:       cap,
		Document:  doc,
	}))
}

func runOp(err error) error {
	if err == nil {
		return nil
	}
	var ge *Error
	if !errors.As(err, &ge) {
		return &Error{Op: "run", Defects: []Defect{{
			Code:    DefectInvalidPath,
			Message: err.Error(),
		}}}
	}
	out := *ge
	out.Op = "run"
	if ge.Defects != nil {
		out.Defects = append([]Defect(nil), ge.Defects...)
	}
	return &out
}
