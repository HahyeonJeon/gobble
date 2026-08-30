package assets

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
)

// MethylSeq preserves the temporary top-level source name and now returns the
// lifted directional Bismark default. Pre-lift Methyl workspaces are not
// resumable with this graph. New code should import assets/pipelines/methylseq
// directly.
//
// Deprecated: import assets/pipelines/methylseq and call Pipeline.
func MethylSeq() *gobble.Pipeline {
	return methylseq.Pipeline()
}
