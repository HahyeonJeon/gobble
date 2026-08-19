package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// View names one Inspect workspace view.
type View string

const (
	ViewRun       View = "run"
	ViewInstances View = "instances"
	ViewErrors    View = "errors"
	ViewLogs      View = "logs"
	ViewTiming    View = "timing"
	ViewDAG       View = "dag"
	ViewLineage   View = "lineage"
	ViewRemaining View = "remaining"
	ViewReuse     View = "reuse"
)

// Inspect returns one read-only JSON or JSONL view of workspace.
//
// view is run, instances, errors, logs, timing, dag, lineage,
// remaining, or reuse. Unknown view is not-found. Empty instance
// selects every reserved identity. A non-empty instance selects one
// reserved identity. Each object or JSONL record includes schema_version.
// Inspect does not occupy, create, or rewrite control files.
func Inspect(workspace string, view View, instance string) ([]byte, error) {
	data, defects := engine.Inspect(workspace, string(view), instance)
	if err := publicError("inspect", defects); err != nil {
		return nil, err
	}
	return data, nil
}
