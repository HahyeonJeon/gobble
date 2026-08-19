package gobble_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestReuseReasonTable(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testing.T) string
		act      func(*testing.T, string)
		view     string
		ident    string
		decision string
		reason   string
	}{
		{
			name: "reused-identity-matched",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processCopyPipeline)
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "reused",
			reason:   "reused-identity-matched",
		},
		{
			name: "identity-changed",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processCopyPipeline)
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processCopyIdentityPipeline("pwd > out/pwd.txt && cp in/sample.txt out/sample.txt && true", "slow"))(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "rerun",
			reason:   "identity-changed",
		},
		{
			name: "command-or-script-changed",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processScriptCopyPipeline("cp in/sample.txt out/sample.txt"))
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processScriptCopyPipeline("cp in/sample.txt out/sample.txt\n# v2"))(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "rerun",
			reason:   "command-or-script-changed",
		},
		{
			name: "params-changed",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processCopyParamsPipeline("fast"))
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processCopyParamsPipeline("slow"))(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "rerun",
			reason:   "params-changed",
		},
		{
			name: "env-changed",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processEnvCopyPipeline)
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processEnvCopyHomePipeline("/tmp/gobble-home-2"))(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "rerun",
			reason:   "env-changed",
		},
		{
			name: "image-changed",
			setup: func(t *testing.T) string {
				dir := readyReleasedRun(t, processCopyPipeline)
				patchLatestTaskField(t, dir, "copy", "image", "alpine:3.19.1")
				return dir
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "rerun",
			reason:   "image-changed",
		},
		{
			name: "input-fingerprint-changed",
			setup: func(t *testing.T) string {
				dir := readyReleasedRun(t, processCopyPipeline)
				in := filepath.Join(dir, "in", "sample.txt")
				if err := os.Chtimes(in, time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "rerun",
			reason:   "input-fingerprint-changed",
		},
		{
			name: "input-missing",
			setup: func(t *testing.T) string {
				dir := readyRunWorkspace(t)
				if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				if err := os.Remove(filepath.Join(dir, "in", "sample.txt")); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			view:   "remaining",
			ident:  "copy",
			reason: "input-missing",
		},
		{
			name: "output-missing",
			setup: func(t *testing.T) string {
				dir := readyRunWorkspace(t)
				if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				if err := os.Remove(filepath.Join(dir, "out", "sample.txt")); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			view:   "remaining",
			ident:  "copy",
			reason: "output-missing",
		},
		{
			name: "previous-incomplete",
			setup: func(t *testing.T) string {
				dir := readyRunWorkspace(t)
				g := mustCompose(processCopyPipeline)(t)
				if err := gobble.Run(t.Context(), g, dir, 0); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				forcePublicDeadOwner(t, dir)
				markCopyRunning(t, dir)
				if err := gobble.Release(dir); err != nil {
					t.Fatalf("Release() error = %v", err)
				}
				return dir
			},
			act: func(t *testing.T, dir string) {
				if err := gobble.Resume(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
					t.Fatalf("Resume() error = %v", err)
				}
			},
			view:     "reuse",
			ident:    "copy",
			decision: "rerun",
			reason:   "previous-incomplete",
		},
		{
			name: "previous-unsuccessful",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processContainPipeline)
			},
			act: func(t *testing.T, dir string) {
				err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2)
				requireResumeError(t, "contained resume", err, gobble.DefectFailed, "fail")
			},
			view:     "reuse",
			ident:    "fail",
			decision: "rerun",
			reason:   "previous-unsuccessful",
		},
		{
			name: "downstream-of-rerun",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processContainPipeline)
			},
			act: func(t *testing.T, dir string) {
				err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2)
				requireResumeError(t, "contained resume", err, gobble.DefectFailed, "fail")
			},
			view:     "reuse",
			ident:    "dep",
			decision: "blocked-upstream",
			reason:   "downstream-of-rerun",
		},
		{
			name: "blocked-upstream",
			setup: func(t *testing.T) string {
				return readyReleasedRun(t, processContainPipeline)
			},
			act: func(t *testing.T, dir string) {
				err := gobble.Resume(t.Context(), mustCompose(processContainPipeline)(t), dir, 2)
				requireResumeError(t, "contained resume", err, gobble.DefectFailed, "fail")
			},
			view:     "reuse",
			ident:    "dep",
			decision: "blocked-upstream",
			reason:   "downstream-of-rerun",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.setup(t)
			if tc.act != nil {
				tc.act(t, dir)
			}
			recs := instanceByID(mustInspectJSONL(t, dir, tc.view, ""))
			got := recs[tc.ident]
			if got == nil {
				t.Fatalf("%s missing from %s: %#v", tc.ident, tc.view, recs)
			}
			if got["reason"] != tc.reason {
				t.Fatalf("%s %s reason got %#v, want %s", tc.ident, tc.view, got, tc.reason)
			}
			if tc.decision != "" && got["decision"] != tc.decision {
				t.Fatalf("%s %s decision got %#v, want %s", tc.ident, tc.view, got, tc.decision)
			}
			if tc.view == "reuse" {
				inst := instanceByID(mustInspectJSONL(t, dir, "instances", ""))[tc.ident]
				if inst == nil || inst["reuse_reason"] != tc.reason {
					t.Fatalf("%s instance reuse_reason got %#v, want %s", tc.ident, inst, tc.reason)
				}
				if inst["decision"] != tc.decision {
					t.Fatalf("%s instance decision got %#v, want %s", tc.ident, inst, tc.decision)
				}
			}
		})
	}
}

func processCopyParamsPipeline(mode string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		p := gobble.NewPipeline("copy")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		p.AddTask(gobble.TaskSpec{
			Name:    "copy",
			Command: []string{"sh", "-c", "pwd > out/pwd.txt && cp in/sample.txt out/sample.txt"},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{
				{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
				{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "pwd", Ext: ".txt"}},
			},
			Params:    []gobble.Param{{Name: "mode", Value: mode}},
			Resources: gobble.Resources{CPU: 1, Memory: "512m"},
		})
		return p
	}
}

func processCopyIdentityPipeline(cmd, mode string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		p := gobble.NewPipeline("copy")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		p.AddTask(gobble.TaskSpec{
			Name:    "copy",
			Command: []string{"sh", "-c", cmd},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{
				{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
				{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "pwd", Ext: ".txt"}},
			},
			Params:    []gobble.Param{{Name: "mode", Value: mode}},
			Env:       map[string]string{"HOME": "/tmp/gobble-identity"},
			Resources: gobble.Resources{CPU: 1, Memory: "512m"},
		})
		return p
	}
}

func patchLatestTaskField(t *testing.T, dir, ident, key, value string) {
	t.Helper()
	path := filepath.Join(dir, engine.ControlDir, engine.TasksFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	old := []byte(`"` + key + `": ""`)
	if !bytes.Contains(data, old) {
		t.Fatalf("tasks.json missing empty %s for %s:\n%s", key, ident, data)
	}
	data = bytes.Replace(data, old, []byte(`"`+key+`": "`+value+`"`), 1)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
