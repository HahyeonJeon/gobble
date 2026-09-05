package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWatchInvocationAndRedirectedOutput(t *testing.T) {
	for _, args := range [][]string{{"watch", "--workspace", "run"}, {"--workspace=run", "watch"}} {
		req, err := parse(args)
		if err != nil || req.command != "watch" || req.workspace != "run" { t.Fatalf("parse %v: %v %v", args, req, err) }
	}
	for _, args := range [][]string{{"watch"}, {"watch", "pkg", "--workspace=run"}, {"watch", "--workspace=run", "--instance=a"}, {"watch", "--workspace=run", "--cap=2"}} {
		if _, err := parse(args); err == nil { t.Fatalf("accepted %v", args) }
	}
	var stderr bytes.Buffer
	if code := runWatch(&request{command: "watch", workspace: "run"}, &stderr); code == 0 || !strings.Contains(stderr.String(), "terminal") { t.Fatalf("redirected watch: %d %s", code, &stderr) }
}
