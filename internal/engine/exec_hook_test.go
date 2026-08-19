package engine

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func isolateWorkspace(isolate string) string {
	p := isolate
	for i := 0; i < 7; i++ {
		p = filepath.Dir(p)
	}
	return p
}

type fnExec struct {
	submit    func(exec.Job) (exec.Handle, exec.Report, error)
	poll      func(exec.Handle) (exec.Report, error)
	cancel    func(exec.Handle) error
	reconcile func(exec.Handle) (exec.Report, error)
}

func (f *fnExec) Submit(job exec.Job) (exec.Handle, exec.Report, error) {
	return f.submit(job)
}

func (f *fnExec) Poll(h exec.Handle) (exec.Report, error) {
	if f.poll != nil {
		return f.poll(h)
	}
	return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
}

func (f *fnExec) Cancel(h exec.Handle) error {
	if f.cancel != nil {
		return f.cancel(h)
	}
	return nil
}

func (f *fnExec) Reconcile(h exec.Handle) (exec.Report, error) {
	if f.reconcile != nil {
		return f.reconcile(h)
	}
	return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
}

func useExec(t *testing.T, ex exec.Executor) {
	t.Helper()
	orig := runExecutor
	t.Cleanup(func() { runExecutor = orig })
	runExecutor = ex
}

func blockingExec(tasks []TaskPlan, fn func(workspace string, task TaskPlan) report) exec.Executor {
	by := make(map[string]TaskPlan, len(tasks))
	for _, task := range tasks {
		cp := task
		applyReservedDefaults(&cp)
		by[reservedIdentity(cp)] = cp
	}
	var mu sync.Mutex
	done := map[string]exec.Report{}
	return &fnExec{
		submit: func(job exec.Job) (exec.Handle, exec.Report, error) {
			ws := isolateWorkspace(job.Isolate)
			task, ok := by[job.Identity]
			if !ok {
				task = TaskPlan{ID: job.Identity, Command: job.Argv, Image: job.Image, Env: job.Env}
			}
			r := fn(ws, task)
			rep := exec.Report{
				Identity:  job.Identity,
				RuntimeID: "1",
				Exit:      r.Exit,
				Message:   r.Message,
				Running:   false,
				Published: r.Published,
			}
			mu.Lock()
			done[job.Identity] = rep
			mu.Unlock()
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "1"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "1", Running: true}, nil
		},
		poll: func(h exec.Handle) (exec.Report, error) {
			mu.Lock()
			defer mu.Unlock()
			if r, ok := done[h.Identity]; ok {
				return r, nil
			}
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
	}
}
