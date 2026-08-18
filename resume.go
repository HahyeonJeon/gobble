package gobble

import (
	"errors"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// Resume occupies a released existing run in workspace and continues
// remaining work. A missing run is nothing-to-resume. Active occupy is
// occupied-workspace. Occupancy stays active when Resume returns.
//
// cap follows Run: zero means 1; values below 1 or above 64 are refused.
//
// A nil error means every instance in g succeeded. Contained failures
// return an [*Error] with Op "resume" that names the failed units.
func Resume(g *Graph, workspace string, cap int) error {
	if err := resumePreflight(g, workspace, cap); err != nil {
		return err
	}
	doc, err := planDocument(g)
	if err != nil {
		return resumeOp(err)
	}
	return publicError("resume", engine.Resume(engine.Request{
		Workspace: workspace,
		Cap:       cap,
		Document:  doc,
	}))
}

func resumePreflight(g *Graph, workspace string, cap int) error {
	if g == nil {
		return &Error{Op: "resume", Defects: []Defect{{
			Code:    DefectInvalidName,
			Message: "nil graph",
		}}}
	}
	if pub := publicError("resume", engine.Validate(snapshotGraph(g))); pub != nil {
		return pub
	}
	if _, err := planDocument(g); err != nil {
		return resumeOp(err)
	}
	return publicError("resume", engine.CheckResumeStart(workspace, cap))
}

func resumeOp(err error) error {
	if err == nil {
		return nil
	}
	var ge *Error
	if !errors.As(err, &ge) {
		return &Error{Op: "resume", Defects: []Defect{{
			Code:    DefectInvalidPath,
			Message: err.Error(),
		}}}
	}
	out := *ge
	out.Op = "resume"
	if ge.Defects != nil {
		out.Defects = append([]Defect(nil), ge.Defects...)
	}
	return &out
}
