//go:build linux

package tui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/monitor"
	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

func terminalPair(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { master.Close() })
	if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	n, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { slave.Close() })
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 30, Col: 110}); err != nil {
		t.Fatal(err)
	}
	return master, slave
}

func TestWatchQuitRestoresTerminalAndLeavesPipelineRunning(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	workspace := t.TempDir()
	gate := filepath.Join(t.TempDir(), "finish")
	identity, err := gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/monitor/tui")
	if err != nil {
		t.Fatal(err)
	}
	option := gobble.WithIdentity(identity)
	p := gobble.NewPipeline("watch-integration")
	p.AddModule("S01").WithDisplay(gobble.TaskDisplay{Samples: []string{"S01"}}).AddTask(gobble.TaskSpec{
		Name:    "work",
		Command: []string{"sh", "-c", `printf 'live output\n'; while [ ! -f "$1" ]; do sleep 0.02; done; printf done > result.txt`, "gobble-test", gate},
		Outputs: []gobble.Bind{{Name: "result", Spec: gobble.PathSpec{Base: "result", Ext: ".txt"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- gobble.Run(ctx, g, workspace, 1, option) }()
	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-runDone:
			t.Fatalf("pipeline ended before watch: %v", err)
		case <-deadline:
			t.Fatal("pipeline did not produce live output")
		case <-ticker.C:
			s, err := monitor.Read(workspace, "", option)
			if err != nil || len(s.Tasks) == 0 || s.Tasks[0].Status != "running" {
				continue
			}
			s, err = monitor.Read(workspace, s.Tasks[0].Identity, option)
			if err == nil && len(s.Logs) == 1 && strings.Contains(s.Logs[0].StdoutTail, "live output") {
				goto ready
			}
		}
	}
ready:
	master, slave := terminalPair(t)
	before, err := term.GetState(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	painted := make(chan struct{})
	go func() {
		var seen bytes.Buffer
		buf := make([]byte, 4096)
		sent := false
		for {
			n, err := master.Read(buf)
			if !sent {
				seen.Write(buf[:n])
				if bytes.Contains(seen.Bytes(), []byte("pipeline monitor")) {
					close(painted)
					sent = true
				}
			}
			if err != nil {
				return
			}
		}
	}()
	watchCtx, watchCancel := context.WithCancel(t.Context())
	defer watchCancel()
	watchDone := make(chan error, 1)
	go func() { watchDone <- Watch(watchCtx, workspace, slave, slave, option) }()
	select {
	case <-painted:
	case err := <-watchDone:
		t.Fatalf("watch exited before rendering: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("watch did not render")
	}
	if _, err := master.Write([]byte("q")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-watchDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("q did not exit watch")
	}
	after, err := term.GetState(slave.Fd())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("terminal state was not restored: %v", err)
	}
	select {
	case err := <-runDone:
		t.Fatalf("quitting watch stopped pipeline: %v", err)
	default:
	}
	if err := os.WriteFile(gate, []byte("finish"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pipeline did not finish after gate release")
	}
}
