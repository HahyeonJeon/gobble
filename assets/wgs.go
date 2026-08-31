package assets

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

// WGS preserves the temporary top-level source name and now returns the lifted
// joint-germline default. Pre-lift WGS workspaces are not resumable with this
// graph. New code should import assets/pipelines/wgs directly.
//
// Deprecated: import assets/pipelines/wgs and call Pipeline.
func WGS() *gobble.Pipeline {
	return wgs.Pipeline()
}
