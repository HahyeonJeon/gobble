package gobble

import (
	"context"
	"errors"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// Resume occupies a released existing run in workspace and continues
// remaining work. Topology and identity edits classify as Change:
// Added, Removed, Rewired, Repathed, IdentityChanged, or Unchanged.
// change is persisted on each classified identity after Resume.
// A missing run is nothing-to-resume. Active occupy is
// occupied-workspace. Occupancy stays active when Resume returns.
//
// cap follows Run: zero means 1; values below 1 or above 64 are refused.
//
// A nil error means every instance in g succeeded. Contained failures
// return an [*Error] with Op "resume" that names the failed units.
// When ctx is done, in-flight work is canceled and the error is
// [*Error] with Op "resume" and DefectCanceled.
// Resume requires exactly one effective [WithIdentity] option. Zero-value
// OccupyOption values are ignored.
func Resume(ctx context.Context, g *Graph, workspace string, cap int, opts ...OccupyOption) error {
	identity, err := parseOccupyOptions("resume", true, opts)
	if err != nil {
		return err
	}
	if err := resumePreflight(g, workspace, cap); err != nil {
		return err
	}
	doc, err := planDocument(g)
	if err != nil {
		return resumeOp(err)
	}
	return publicError("resume", engine.Resume(ctx, engine.Request{
		Workspace: workspace,
		Cap:       cap,
		Document:  doc,
		Identity:  toEngineIdentity(identity),
	}))
}

func resumePreflight(g *Graph, workspace string, cap int) error {
	if g == nil {
		return &Error{Op: "resume", Defects: []Defect{{
			Code:    DefectInvalidRequest,
			Message: "nil graph",
		}}}
	}
	if err := composeDefects("resume", graphCheck(g)); err != nil {
		return err
	}
	doc, err := planDocument(g)
	if err != nil {
		return resumeOp(err)
	}
	if pub := publicError("resume", engine.Validate(doc)); pub != nil {
		return pub
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
