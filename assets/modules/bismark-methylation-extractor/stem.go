package bismarkmethylationextractor

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

func bismarkExtractorStem(spec gobble.PathSpec) string {
	path := modules.MustCommandPath(spec)
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	lower := strings.ToLower(base)
	for _, suf := range []string{".bam", ".sam"} {
		if strings.HasSuffix(lower, suf) {
			return base[:len(base)-len(suf)]
		}
	}
	return base
}
