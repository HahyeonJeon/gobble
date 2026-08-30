package rnaseq

import (
	"github.com/HahyeonJeon/gobble"
	product "github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

func Pipeline() *gobble.Pipeline {
	return product.Pipeline()
}
