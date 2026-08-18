package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Release closes occupancy on workspace when the owner is not live on
// this host. A live local owner and a foreign host are refused.
// Missing run is not-found. Already closed occupancy is already-released.
// Release does not execute, delete, cancel, or occupy.
func Release(workspace string) error {
	return publicError("release", engine.Release(workspace))
}
