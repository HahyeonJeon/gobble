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
	ViewIdentity  View = "identity"
	// ViewMonitor combines run, task, timing, display, and DAG facts from one
	// coherent control snapshot. instance selects log tails only.
	ViewMonitor View = "monitor"
)

// Inspect returns one read-only JSON or JSONL view of workspace.
//
// view is run, instances, errors, logs, timing, dag, lineage,
// remaining, reuse, identity, or monitor. Unknown view is not-found. Empty instance
// selects every reserved identity. A non-empty instance selects one
// reserved identity. For monitor, instance selects logs while task and graph
// facts remain global. Each view or JSONL record includes schema_version.
// Inspect does not occupy, create, or rewrite control files.
// Omitted identity derives the module process identity from the workspace
// identity mode. A supplied [WithIdentity] option is used instead.
func Inspect(workspace string, view View, instance string, opts ...OccupyOption) ([]byte, error) {
	identity, err := parseOccupyOptions("inspect", false, opts)
	if err != nil {
		return nil, err
	}
	data, defects := engine.Inspect(workspace, string(view), instance, toEngineIdentity(identity))
	if err := publicError("inspect", defects); err != nil {
		return nil, err
	}
	return data, nil
}
