package assets

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

// RNASeq preserves the temporary top-level source name and now returns the
// lifted STAR-Salmon default. Pre-lift RNA workspaces are not resumable with
// this graph. New code should import assets/pipelines/rnaseq directly.
//
// Deprecated: import assets/pipelines/rnaseq and call Pipeline.
func RNASeq() *gobble.Pipeline {
	return rnaseq.Pipeline()
}
