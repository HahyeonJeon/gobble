// Command gobble is the product CLI for the Gobble pipeline loop.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/containerenv"
)

func main() {
	if os.Getenv("GOBBLE_CONTAINER_BOOTSTRAP") == "1" && needsContainer(os.Args[1:]) {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		err := containerenv.Prepare(ctx)
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobble runtime:", err)
			os.Exit(1)
		}
	}
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
	case "demo":
		return runDemo(req, stdout, stderr)
	case "init":
		return runInit(req, stdout, stderr)
	case "doctor":
		return runDoctor(stdout, stderr)
	case "watch":
		return runWatch(req, stderr)
	case "pack":
		return runPack(req, stderr)
	case "inspect":
		return runInspect(req, stdout, stderr)
	case "stop":
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		result, err := gobble.Stop(ctx, req.workspace)
		if err != nil {
			return writeLibraryErr(stderr, err)
		}
		return writeJSON(stdout, stderr, "stop", result)
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

func needsContainer(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return false
		}
	}
	return args[0] != "help" && args[0] != "version" && args[0] != "--version"
}
