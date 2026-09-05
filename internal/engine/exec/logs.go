package exec

import (
	"context"
	"path/filepath"
)

type logStream struct {
	cancel context.CancelFunc
	done   chan struct{}
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
		defer out.Close()
		defer stderr.Close()
		defer cancel()
		_, _ = cli(ctx, []string{"logs", "--follow", h.RuntimeID}, env, out, stderr)
	}()
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
