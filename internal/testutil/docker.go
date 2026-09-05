package testutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DockerState is a one-container daemon model persisted across test processes.
// It models engine lifecycle contracts; it does not validate actual Docker.
type DockerState struct {
	DaemonID string
	ID       string
	Token    string
	Isolate  string
	Exists   bool
	Running  bool
	Starts   int
}

// Docker models create/start separately and refuses concurrent duplicate work.
// Hooks let tests interrupt the client before or after a daemon operation.
type Docker struct {
	Root            string
	RunContinuously bool
	Fail            string
	Before          func([]string)
	After           func([]string)
	mu              sync.Mutex
}

func (d *Docker) State() (DockerState, error) {
	data, err := os.ReadFile(filepath.Join(d.Root, "daemon.json"))
	if os.IsNotExist(err) {
		return DockerState{DaemonID: "test-daemon"}, nil
	}
	var st DockerState
	if err != nil {
		return st, err
	}
	err = json.Unmarshal(data, &st)
	return st, err
}

func (d *Docker) Save(st DockerState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d.Root, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d.Root, "daemon.json"), data, 0o600)
}

func (d *Docker) CLI(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Before != nil {
		d.Before(args)
	}
	if ctx.Err() != nil {
		return -1, ctx.Err()
	}
	st, err := d.State()
	if err != nil {
		return -1, err
	}
	if len(args) == 0 {
		return -1, errors.New("empty fake Docker command")
	}
	if args[0] == d.Fail {
		return -1, fmt.Errorf("injected %s failure", d.Fail)
	}
	value := func(flag string) string {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == flag {
				return args[i+1]
			}
		}
		return ""
	}
	switch args[0] {
	case "context":
		_, err = io.WriteString(stdout, "unix:///gobble-test.sock\n")
	case "info":
		_, err = io.WriteString(stdout, st.DaemonID+"\n")
	case "image", "pull":
		_, err = io.WriteString(stdout, "sha256:test-image\n")
	case "create":
		if st.Exists {
			return -1, errors.New("previous container was not reconciled")
		}
		st.Token = strings.TrimPrefix(value("--label"), "io.gobble.submission=")
		if value("--name") != "gobble-"+st.Token || st.Token == "" {
			return -1, errors.New("missing container ownership")
		}
		st.ID = "container-" + st.Token
		st.Isolate = strings.TrimSuffix(value("-v"), ":/work")
		st.Exists, st.Running = true, false
		_, err = io.WriteString(stdout, st.ID+"\n")
	case "start":
		if !st.Exists || st.ID != args[len(args)-1] || st.Running {
			return -1, errors.New("cannot start removed or already running container")
		}
		st.Starts++
		st.Running = d.RunContinuously
		// The engine fixture's single copy task exercises real staging,
		// publication, and checksums around the modeled backend.
		if input, e := os.ReadFile(filepath.Join(st.Isolate, "in", "sample.txt")); e == nil {
			if e = os.MkdirAll(filepath.Join(st.Isolate, "out"), 0o755); e == nil {
				err = os.WriteFile(filepath.Join(st.Isolate, "out", "sample.txt"), input, 0o644)
			} else {
				err = e
			}
		}
	case "container":
		filter := value("--filter")
		if st.Exists && (filter == "id="+st.ID || filter == "label=io.gobble.submission="+st.Token) {
			_, err = io.WriteString(stdout, st.ID+"\n")
		}
	case "inspect":
		if !st.Exists || st.ID != args[len(args)-1] {
			return -1, errors.New("container missing")
		}
		switch format := value("--format"); {
		case strings.Contains(format, "Config.Labels"):
			_, err = io.WriteString(stdout, st.Token+"\n")
		case strings.Contains(format, "State.Running"):
			_, err = fmt.Fprintf(stdout, "%t 0\n", st.Running)
		case format == "{{.Image}}":
			_, err = io.WriteString(stdout, "sha256:test-image\n")
		default:
			return -1, fmt.Errorf("unexpected inspect: %s", format)
		}
	case "logs":
		_, err = io.WriteString(stdout, "task log\n")
	case "rm":
		if !st.Exists || st.ID != args[len(args)-1] {
			return -1, errors.New("refusing removal of an unowned container")
		}
		st.Exists, st.Running = false, false
	case "kill":
		st.Running = false
	default:
		return -1, fmt.Errorf("unexpected fake Docker operation: %v", args)
	}
	if err != nil {
		return -1, err
	}
	if err := d.Save(st); err != nil {
		return -1, err
	}
	if d.After != nil {
		d.After(args)
	}
	return 0, nil
}
