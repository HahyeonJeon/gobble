package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Release always Reconciles first, then closes occupancy when leftovers
// are known stopped. The occupying process may Release after Run
// returns. A later process while that owner is live is live-occupancy.
// Occupancy does not close while any identity remains unknown
// (unknown-backend). A foreign host is foreign-host. Missing run is
// not-found. Already closed occupancy is already-released. Empty or
// missing workspace is invalid-path. Release does not execute, delete,
// or occupy a new run.
// Omitted identity derives the module process identity from the workspace
// identity mode. A supplied [WithIdentity] option is used instead.
func Release(workspace string, opts ...OccupyOption) error {
	identity, err := parseOccupyOptions("release", false, opts)
	if err != nil {
		return err
	}
	return publicError("release", engine.Release(workspace, toEngineIdentity(identity)))
}
