package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Inspect returns one read-only JSON or JSONL view of workspace.
//
// view is run, instances, errors, logs, timing, DAG, lineage,
// remaining, or reuse. Empty instance selects every reserved
// identity. A non-empty instance selects one reserved identity.
// Inspect does not occupy, create, or rewrite control files.
func Inspect(workspace, view, instance string) ([]byte, error) {
	data, defects := engine.Inspect(workspace, view, instance)
	if err := publicError("inspect", defects); err != nil {
		return nil, err
	}
	return data, nil
}
