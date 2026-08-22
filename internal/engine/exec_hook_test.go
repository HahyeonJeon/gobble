package engine

import (
	"context"
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
	submit    func(context.Context, exec.Job) (exec.Handle, exec.Report, error)
	poll      func(context.Context, exec.Handle) (exec.Report, error)
	cancel    func(context.Context, exec.Handle) error
	reconcile func(context.Context, exec.Handle) (exec.Report, error)
}

func (f *fnExec) Submit(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
	return f.submit(ctx, job)
}

func (f *fnExec) Poll(ctx context.Context, h exec.Handle) (exec.Report, error) {
	if f.poll != nil {
		return f.poll(ctx, h)
	}
	if err := ctx.Err(); err != nil {
		return exec.Report{}, err
	}
	return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
}

func (f *fnExec) Cancel(ctx context.Context, h exec.Handle) error {
	if f.cancel != nil {
		return f.cancel(ctx, h)
	}
	return ctx.Err()
}

func (f *fnExec) Reconcile(ctx context.Context, h exec.Handle) (exec.Report, error) {
	if f.reconcile != nil {
		return f.reconcile(ctx, h)
	}
	if err := ctx.Err(); err != nil {
		return exec.Report{}, err
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
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
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
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			mu.Lock()
			defer mu.Unlock()
			if r, ok := done[h.Identity]; ok {
				return r, nil
			}
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
	}
}
