package modules

import (
	"github.com/HahyeonJeon/gobble"
)

// ScatterFilePrelude returns shell setup for a command whose Gobble Scatter
// membership is one regular file staged below dir. The setup fails unless the
// task isolate contains exactly one member, then defines interval and stem.
func ScatterFilePrelude(unit string, dir gobble.Directory) (string, error) {
	if dir.IsZero() {
		return "", ComposeDefect(gobble.DefectInvalidPath, unit, "scatter member directory is empty")
	}
	root := ShellQuote(dir.String())
	return "set -eu\n" +
		"count=$(find " + root + " -maxdepth 1 -type f | wc -l)\n" +
		"[ \"$count\" -eq 1 ] || { echo \"expected one scatter member in " + dir.String() + "\" >&2; exit 2; }\n" +
		"interval=$(find " + root + " -maxdepth 1 -type f)\n" +
		"name=${interval##*/}\n" +
		"stem=${name%.*}\n", nil
}
