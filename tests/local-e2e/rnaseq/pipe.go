package rnaseq

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets"
)

func Pipeline() *gobble.Pipeline {
	return assets.RNASeq()
}
