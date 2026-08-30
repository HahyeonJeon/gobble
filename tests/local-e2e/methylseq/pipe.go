package methylseq

import (
	"github.com/HahyeonJeon/gobble"
	product "github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
)

func Pipeline() *gobble.Pipeline {
	return product.Pipeline()
}
