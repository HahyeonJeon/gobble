// Package exec is the scheduler-to-backend seam.
//
// Executor implementations start and observe jobs. They do not write
// run.json, plan.json, or tasks.json.
package exec

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
)

// ErrEscapedPath means a log or dest path is a symlink or otherwise untrusted.
var ErrEscapedPath = errors.New("invalid-path")

func createAttemptFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrEscapedPath
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, ErrEscapedPath
		}
		return nil, err
	}
	return f, nil
}

// Backend names recorded on a Handle.
const (
	BackendProcess = "process"
	BackendDocker  = "docker"
)

// Job is one task invocation. Isolate is an absolute work directory.
type Job struct {
	Identity    string
	Isolate     string
	Argv        []string
	Env         map[string]string
	Image       string
	CPU         float64
	Memory      string
	MemoryBytes int64
}

// Handle identifies a submitted backend job.
type Handle struct {
	Identity  string
	Backend   string
	RuntimeID string
}

// Report is an executor observation. Adapters leave Published false.
// A test executor may set Published when dest files already exist.
type Report struct {
	Identity    string
	RuntimeID   string
	ImageDigest string
	Exit        int
	Message     string
	Running     bool
	Published   bool
}

// Executor submits, observes, cancels, and reconciles backend jobs.
// ctx bounds that one call. It is not a public occupancy Cancel verb.
type Executor interface {
	Submit(ctx context.Context, job Job) (Handle, Report, error)
	Poll(ctx context.Context, h Handle) (Report, error)
	Cancel(ctx context.Context, h Handle) error
	Reconcile(ctx context.Context, h Handle) (Report, error)
}

// Local selects process or docker by Image (R3). Empty Image is process.
func Local() Executor {
	return &local{
		process: NewProcess(),
		docker:  NewDocker(),
	}
}

type local struct {
	process *Process
	docker  *Docker
}

func (l *local) Submit(ctx context.Context, job Job) (Handle, Report, error) {
	return l.pickImage(job.Image).Submit(ctx, job)
}

func (l *local) Poll(ctx context.Context, h Handle) (Report, error) {
	return l.pickBackend(h.Backend).Poll(ctx, h)
}

func (l *local) Cancel(ctx context.Context, h Handle) error {
	return l.pickBackend(h.Backend).Cancel(ctx, h)
}

func (l *local) Reconcile(ctx context.Context, h Handle) (Report, error) {
	return l.pickBackend(h.Backend).Reconcile(ctx, h)
}

func (l *local) pickImage(image string) Executor {
	if image != "" {
		return l.docker
	}
	return l.process
}

func (l *local) pickBackend(backend string) Executor {
	if backend == BackendDocker {
		return l.docker
	}
	return l.process
}

func copyEnv(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = v
	}
	return out
}

func discard() io.Writer {
	return io.Discard
}
