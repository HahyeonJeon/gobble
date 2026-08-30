package wgs

import (
	"github.com/HahyeonJeon/gobble"
	product "github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

func Pipeline() *gobble.Pipeline {
	return product.Pipeline()
}
