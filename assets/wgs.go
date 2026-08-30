package assets

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

// WGS preserves the top-level constructor at the graph-stable migration
// checkpoint. New code should import assets/pipelines/wgs directly.
//
// Deprecated: import assets/pipelines/wgs and call Pipeline.
func WGS() *gobble.Pipeline {
	return wgs.Pipeline()
}
