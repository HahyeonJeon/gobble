package main

import (
	"context"
	"io"
	"os"

	"github.com/HahyeonJeon/gobble/monitor/tui"
)

func runWatch(req *request, stderr io.Writer) int {
	terminal, ok := stderr.(*os.File)
	if !ok {
		return writeErr(stderr, invalidRequest("watch", "watch requires terminal input and stderr; use inspect monitor for JSON"), 1)
	}
	if err := tui.Watch(context.Background(), req.workspace, os.Stdin, terminal); err != nil {
		return writeLibraryErr(stderr, err)
	}
	return 0
}
