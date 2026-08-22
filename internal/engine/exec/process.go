package exec

import (
	"context"
	"errors"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
)

// Process runs each job in its own process group.
type Process struct {
	mu   sync.Mutex
	live map[string]*proc
}

type proc struct {
	cmd  *osexec.Cmd
	done chan struct{}
	exit int
	err  error
}

// NewProcess returns a process-group adapter.
func NewProcess() *Process {
	return &Process{live: make(map[string]*proc)}
}

// Submit starts argv in Isolate with declared Env only.
func (p *Process) Submit(ctx context.Context, job Job) (Handle, Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Handle{}, Report{}, err
	}
	if len(job.Argv) == 0 {
		return Handle{}, Report{}, errors.New("empty command")
	}
	cmd := osexec.Command(job.Argv[0], job.Argv[1:]...)
	cmd.Dir = job.Isolate
	cmd.Env = processEnv(job.Env)
	stdoutPath := filepath.Join(filepath.Dir(job.Isolate), "stdout")
	stderrPath := filepath.Join(filepath.Dir(job.Isolate), "stderr")
	outf, err := os.Create(stdoutPath)
	if err != nil {
		return Handle{}, Report{}, err
	}
	errf, err := os.Create(stderrPath)
	if err != nil {
		outf.Close()
		return Handle{}, Report{}, err
	}
	cmd.Stdout = outf
	cmd.Stderr = errf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		outf.Close()
		errf.Close()
		return Handle{}, Report{}, err
	}
	pr := &proc{cmd: cmd, done: make(chan struct{})}
	go func() {
		waitErr := cmd.Wait()
		outf.Close()
		errf.Close()
		if waitErr == nil {
			pr.exit = 0
		} else {
			var ee *osexec.ExitError
			if errors.As(waitErr, &ee) {
				pr.exit = ee.ExitCode()
			} else {
				pr.err = waitErr
				pr.exit = -1
			}
		}
		close(pr.done)
	}()
	pid := strconv.Itoa(cmd.Process.Pid)
	p.mu.Lock()
	p.live[pid] = pr
	p.mu.Unlock()
	h := Handle{Identity: job.Identity, Backend: BackendProcess, RuntimeID: pid}
	return h, Report{Identity: job.Identity, RuntimeID: pid, Running: true}, nil
}

// Poll reports whether the process has exited.
func (p *Process) Poll(ctx context.Context, h Handle) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	p.mu.Lock()
	pr, ok := p.live[h.RuntimeID]
	p.mu.Unlock()
	if !ok {
		return Report{}, errors.New("unproved process identity")
	}
	select {
	case <-ctx.Done():
		return Report{}, ctx.Err()
	case <-pr.done:
		msg := ""
		if pr.err != nil {
			msg = pr.err.Error()
		} else if pr.exit != 0 {
			msg = "exit " + strconv.Itoa(pr.exit)
		}
		return Report{
			Identity:  h.Identity,
			RuntimeID: h.RuntimeID,
			Exit:      pr.exit,
			Message:   msg,
			Running:   false,
		}, nil
	default:
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
	}
}

// Cancel kills the job's process group when the PID is proved to be
// this adapter's Gobble job.
func (p *Process) Cancel(ctx context.Context, h Handle) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	pr, ok := p.live[h.RuntimeID]
	p.mu.Unlock()
	if !ok || pr == nil || pr.cmd == nil || pr.cmd.Process == nil {
		return errors.New("unproved process identity")
	}
	pid := pr.cmd.Process.Pid
	err1 := syscall.Kill(-pid, syscall.SIGKILL)
	err2 := syscall.Kill(pid, syscall.SIGKILL)
	if err1 != nil && !errors.Is(err1, syscall.ESRCH) && err2 != nil && !errors.Is(err2, syscall.ESRCH) {
		return err2
	}
	return nil
}

// Reconcile uses in-memory wait state. An unproved PID is unknown.
func (p *Process) Reconcile(ctx context.Context, h Handle) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	p.mu.Lock()
	_, ok := p.live[h.RuntimeID]
	p.mu.Unlock()
	if !ok {
		return Report{}, errors.New("unproved process identity")
	}
	return p.Poll(ctx, h)
}

func processEnv(env map[string]string) []string {
	env = copyEnv(env)
	out := make([]string, 0, 1+len(env))
	if _, ok := env["PATH"]; !ok {
		out = append(out, "PATH=/usr/bin:/bin")
	}
	for k, v := range env {
		if k == "" || v == "" {
			continue
		}
		out = append(out, k+"="+v)
	}
	return out
}
