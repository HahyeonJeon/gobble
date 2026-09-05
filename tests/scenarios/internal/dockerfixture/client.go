// Package dockerfixture shares Docker lifecycle mechanics across assay fixtures.
// Each assay still owns its command matching, input facts, and output fixtures.
package dockerfixture

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"strings"
	"sync"
)

type Call func(context.Context, []string, []string, io.Writer, io.Writer) (int, error)

type container struct {
	args       []string
	token      string
	image      string
	underlying string
}

// Lifecycle creates stopped containers and defers assay execution until start.
// execute handles the assay-specific run/inspect/log/kill behavior.
func Lifecycle(execute Call) Call {
	var mu sync.Mutex
	containers := map[string]*container{}
	return func(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		if err := ctx.Err(); err != nil {
			return -1, err
		}
		if len(args) == 0 {
			return -1, errors.New("empty Docker fixture command")
		}
		switch args[0] {
		case "context":
			_, err := io.WriteString(stdout, "unix:///gobble-scenario.sock\n")
			return 0, err
		case "info":
			_, err := io.WriteString(stdout, "gobble-scenario-daemon\n")
			return 0, err
		case "image", "pull":
			return execute(ctx, args, env, stdout, stderr)
		case "create":
			c := &container{args: append([]string(nil), args...)}
			var name string
			for i := 1; i < len(args); i++ {
				if args[i] == "--network=none" {
					continue
				}
				if strings.HasPrefix(args[i], "-") && i+1 < len(args) {
					switch args[i] {
					case "--label":
						c.token = strings.TrimPrefix(args[i+1], "io.gobble.submission=")
					case "--name":
						name = args[i+1]
					}
					i++
					continue
				}
				c.image = args[i]
				break
			}
			if name != "gobble-"+c.token || c.token == "" || c.image == "" {
				return -1, errors.New("incomplete Docker fixture creation")
			}
			id := "fixture-" + c.token
			if containers[id] != nil {
				return -1, errors.New("duplicate Docker fixture creation")
			}
			containers[id] = c
			_, err := io.WriteString(stdout, id+"\n")
			return 0, err
		case "container":
			filter := ""
			for i := 1; i+1 < len(args); i++ {
				if args[i] == "--filter" {
					filter = args[i+1]
				}
			}
			var ids []string
			for id, c := range containers {
				if filter == "id="+id || filter == "label=io.gobble.submission="+c.token {
					ids = append(ids, id)
				}
			}
			sort.Strings(ids)
			_, err := io.WriteString(stdout, strings.Join(ids, "\n"))
			return 0, err
		}
		id := args[len(args)-1]
		c := containers[id]
		if c == nil {
			return -1, errors.New("Docker fixture container not found")
		}
		switch args[0] {
		case "start":
			if c.underlying != "" {
				return -1, errors.New("Docker fixture already started")
			}
			var out bytes.Buffer
			run := append([]string(nil), c.args...)
			run[0] = "run"
			code, err := execute(ctx, run, env, &out, stderr)
			if err != nil || code != 0 {
				return code, err
			}
			c.underlying = strings.TrimSpace(out.String())
			if c.underlying == "" {
				return -1, errors.New("assay fixture did not return an execution identity")
			}
			_, err = io.WriteString(stdout, id+"\n")
			return 0, err
		case "inspect":
			format := strings.Join(args, " ")
			if strings.Contains(format, "Config.Labels") {
				_, err := io.WriteString(stdout, c.token+"\n")
				return 0, err
			}
			if strings.Contains(format, "{{.Image}}") {
				_, digest, _ := strings.Cut(c.image, "@")
				_, err := io.WriteString(stdout, digest+"\n")
				return 0, err
			}
			if c.underlying == "" && strings.Contains(format, "State.Running") {
				_, err := io.WriteString(stdout, "false 0\n")
				return 0, err
			}
		case "rm":
			if c.underlying != "" {
				if code, err := execute(ctx, []string{"kill", c.underlying}, env, io.Discard, stderr); err != nil || code != 0 {
					return code, err
				}
			}
			delete(containers, id)
			return 0, nil
		case "logs":
			if c.underlying == "" {
				return 0, nil
			}
		}
		translated := append([]string(nil), args...)
		translated[len(translated)-1] = c.underlying
		return execute(ctx, translated, env, stdout, stderr)
	}
}
