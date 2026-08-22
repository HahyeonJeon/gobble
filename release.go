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
func Release(workspace string) error {
	return publicError("release", engine.Release(workspace))
}
