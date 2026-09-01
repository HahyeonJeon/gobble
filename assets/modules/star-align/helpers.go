package staralign

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
)

func defaultSTARAlignDir() gobble.Directory {
	return gobble.Dir("work/star-align")
}

func starAlignPrefix(dir gobble.Directory) string {
	s := dir.String()
	if s == "" || strings.HasSuffix(s, "/") {
		return s
	}
	return s + "/"
}
