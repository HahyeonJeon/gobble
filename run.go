package gobble

import (
	"errors"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// Run executes g in workspace.
//
// The workspace must already exist. After pre-execution checks pass,
// Run occupies it by writing run identity under .gobble/. A second
// Run on that workspace is an occupied-workspace error. It does not
// resume, inspect, or delete.
//
// cap is the maximum number of tasks that may run at once. Zero means
// 1. Values below 1 or above 64 are refused.
//
// Empty Image runs as a host process. A nil error means every task
// succeeded. Contained task failure returns an [*Error] with Op "run"
// that names the failed units.
func Run(g *Graph, workspace string, cap int) error {
	if err := preflight(g, workspace, cap); err != nil {
		return err
	}
	doc, err := planDocument(g)
	if err != nil {
		return runOp(err)
	}
	return publicError("run", engine.Run(engine.Request{
		Workspace: workspace,
		Cap:       cap,
		Document:  doc,
	}))
}

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
