package exec

import (
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
func (p *Process) Submit(job Job) (Handle, Report, error) {
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
func (p *Process) Poll(h Handle) (Report, error) {
	p.mu.Lock()
	pr, ok := p.live[h.RuntimeID]
	p.mu.Unlock()
	if !ok {
		return p.Reconcile(h)
	}
	select {
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

// Cancel kills the job's process group.
func (p *Process) Cancel(h Handle) error {
	pid, err := strconv.Atoi(h.RuntimeID)
	if err != nil || pid <= 0 {
		return nil
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
	return nil
}

// Reconcile uses in-memory wait state, then pid liveness.
func (p *Process) Reconcile(h Handle) (Report, error) {
	p.mu.Lock()
	_, ok := p.live[h.RuntimeID]
	p.mu.Unlock()
	if ok {
		return p.Poll(h)
	}
	pid, err := strconv.Atoi(h.RuntimeID)
	if err != nil || pid <= 0 {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
	}
	err = syscall.Kill(pid, 0)
	live := err == nil || errors.Is(err, syscall.EPERM)
	return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: live}, nil
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
