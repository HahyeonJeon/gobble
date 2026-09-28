package exec

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

type logStream struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error // Published by closing done; read only after joining.
}

// The controller owns the collector. A monitor only reads attempt files.
// Starting once avoids repeatedly copying an ever-growing log on every poll.
func (d *Docker) followLogs(ctx context.Context, h Handle) {
	if h.Submission == nil || h.Isolate == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.streams == nil {
		d.streams = make(map[string]*logStream)
	}
	if d.streams[h.RuntimeID] != nil {
		return
	}
	out, err := createAttemptFile(filepath.Join(filepath.Dir(h.Isolate), "stdout"))
	if err != nil {
		return
	}
	stderr, err := createAttemptFile(filepath.Join(filepath.Dir(h.Isolate), "stderr"))
	if err != nil {
		out.Close()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	stream := &logStream{cancel: cancel, done: make(chan struct{})}
	d.streams[h.RuntimeID] = stream
	// Capture the client dependency and selected endpoint before starting work.
	cli, env := DockerCLI, dockerEnvForContext(ctx)
	go func() {
		defer close(stream.done)
		defer cancel()
		code, err := cli(ctx, []string{"logs", "--follow", h.RuntimeID}, env, out, stderr)
		if err == nil && code != 0 {
			err = fmt.Errorf("docker logs: exit %d", code)
		}
		stream.err = errors.Join(err, out.Close(), stderr.Close())
	}()
}

// finishLogs drains an owned collector after Docker proves the task stopped.
// Creating the same files again would reject the collector's own exclusive files.
// A failed stream is not treated as complete and never authorizes an overwrite.
func (d *Docker) finishLogs(ctx context.Context, h Handle) error {
	d.mu.Lock()
	stream := d.streams[h.RuntimeID]
	d.mu.Unlock()
	if stream == nil {
		return writeDockerLogs(ctx, h)
	}
	select {
	case <-stream.done:
		return stream.err
	case <-ctx.Done():
		stream.cancel()
		return ctx.Err()
	}
}

func (d *Docker) stopLogs(ctx context.Context, id string) error {
	d.mu.Lock()
	stream := d.streams[id]
	d.mu.Unlock()
	if stream == nil {
		return nil
	}
	stream.cancel()
	select {
	case <-stream.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
