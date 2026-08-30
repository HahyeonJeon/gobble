package assets

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

// RNASeq preserves the top-level constructor at the graph-stable migration
// checkpoint. New code should import assets/pipelines/rnaseq directly.
//
// Deprecated: import assets/pipelines/rnaseq and call Pipeline.
func RNASeq() *gobble.Pipeline {
	return rnaseq.Pipeline()
}
