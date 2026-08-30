package assets

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
)

// MethylSeq preserves the top-level constructor at the graph-stable migration
// checkpoint. New code should import assets/pipelines/methylseq directly.
//
// Deprecated: import assets/pipelines/methylseq and call Pipeline.
func MethylSeq() *gobble.Pipeline {
	return methylseq.Pipeline()
}
