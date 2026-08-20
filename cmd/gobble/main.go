// Command gobble is the product CLI for the Gobble pipeline loop.
package main

import (
	"io"
	"os"

	"github.com/HahyeonJeon/gobble"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	req, err := parse(args)
	if err != nil {
		return writeErr(stderr, err, 2)
	}
	if req.help {
		return writeHelp(stdout, stderr, req.command)
	}
	if req.version {
		return writeVersion(stdout, stderr)
	}
	switch req.command {
	case "inspect":
		return runInspect(req, stdout, stderr)
	case "release":
		return runRelease(req, stdout, stderr)
	case "compose", "validate", "plan", "run", "resume":
		return runDriver(req, stdout, stderr)
	default:
		return writeErr(stderr, invalidRequest("cli", "unknown command"), 2)
	}
}

func runInspect(req *request, stdout, stderr io.Writer) int {
	data, err := gobble.Inspect(req.workspace, gobble.View(req.view), req.instance)
	if err != nil {
		return writeLibraryErr(stderr, err)
	}
	if _, werr := stdout.Write(data); werr != nil {
		return writeErr(stderr, invalidRequest("inspect", "stdout write failed"), 1)
	}
	return 0
}

func runRelease(req *request, stdout, stderr io.Writer) int {
	if err := gobble.Release(req.workspace); err != nil {
		return writeLibraryErr(stderr, err)
	}
	return writeJSON(stdout, stderr, "release", struct {
		Op string `json:"op"`
	}{Op: "release"})
}
