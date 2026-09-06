// Package distribution provides the same Compose contract to downloads and
// newly generated projects. It has no dependency on a host launcher.
package distribution

import (
	_ "embed"
	"strconv"
	"strings"
)

//go:embed compose.yaml
var compose string

// Compose pins a generated project to the runtime that created it.
func Compose(image string) string {
	return strings.Replace(compose, "image: ghcr.io/hahyeonjeon/gobble:develop", "image: "+strconv.Quote(image), 1)
}
