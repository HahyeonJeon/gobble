package panicpipe

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	panic("author-abort")
}
