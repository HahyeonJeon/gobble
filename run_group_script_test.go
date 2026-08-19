package gobble_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestRunGroupTable(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		fail    bool
		wantAmb bool
		wantAnn bool
	}{
		{
			name:    "publish both members",
			cmd:     "cp in/ref.amb out/out.amb && cp in/ref.ann out/out.ann",
			wantAmb: true,
			wantAnn: true,
		},
		{
			name:    "missing member unpublished",
			cmd:     "cp in/ref.amb out/out.amb",
			fail:    true,
			wantAmb: false,
			wantAnn: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeRunFile(t, filepath.Join(dir, "in", "ref.amb"), "amb")
			writeRunFile(t, filepath.Join(dir, "in", "ref.ann"), "ann")
			err := gobble.Run(t.Context(), mustCompose(processGroupPipeline(tc.cmd))(t), dir, 0)
			if tc.fail {
				requireRunError(t, tc.name, err, gobble.DefectFailed, "copy")
			} else if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			_, ambErr := os.Stat(filepath.Join(dir, "out", "out.amb"))
			_, annErr := os.Stat(filepath.Join(dir, "out", "out.ann"))
			if tc.wantAmb != (ambErr == nil) {
				t.Fatalf("out.amb exist=%v want %v err=%v", ambErr == nil, tc.wantAmb, ambErr)
			}
			if tc.wantAnn != (annErr == nil) {
				t.Fatalf("out.ann exist=%v want %v err=%v", annErr == nil, tc.wantAnn, annErr)
			}
			isolate := filepath.Join(dir, engine.ControlDir, "tasks", "copy", "_", "0", "1", "work")
			if _, err := os.Stat(filepath.Join(isolate, "in", "ref.amb")); err != nil {
				t.Fatalf("staged group member: %v", err)
			}
		})
	}
}

func TestRunScriptTable(t *testing.T) {
	tests := []struct {
		name   string
		script string
		fail   bool
		want   string
	}{
		{
			name:   "copy dest",
			script: "cp in/sample.txt out/sample.txt",
			want:   "reads",
		},
		{
			name:   "nounset fails",
			script: "echo $UNSET_GOBBLE_VAR > out/sample.txt",
			fail:   true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := readyRunWorkspace(t)
			err := gobble.Run(t.Context(), mustCompose(processScriptCopyPipeline(tc.script))(t), dir, 0)
			if tc.fail {
				requireRunError(t, tc.name, err, gobble.DefectFailed, "copy")
				if _, statErr := os.Stat(filepath.Join(dir, "out", "sample.txt")); !os.IsNotExist(statErr) {
					t.Fatalf("failed script published dest")
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			got, readErr := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
			if readErr != nil {
				t.Fatalf("published dest: %v", readErr)
			}
			if string(got) != tc.want {
				t.Fatalf("dest got %q, want %q", got, tc.want)
			}
		})
	}
}

func processGroupPipeline(cmd string) func() *gobble.Pipeline {
	return func() *gobble.Pipeline {
		p := gobble.NewPipeline("group")
		in := p.AddInputGroup("idx", gobble.Group{
			{Name: "amb", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "ref", Ext: ".amb"}},
			{Name: "ann", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "ref", Ext: ".ann"}},
		})
		p.AddTask(gobble.TaskSpec{
			Name:    "copy",
			Command: []string{"sh", "-c", cmd},
			Inputs: []gobble.Bind{{
				Name:  "idx",
				From:  in,
				Group: gobble.Group{{Name: "amb"}, {Name: "ann"}},
			}},
			Outputs: []gobble.Bind{{
				Name: "out",
				Group: gobble.Group{
					{Name: "amb", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "out", Ext: ".amb"}},
					{Name: "ann", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "out", Ext: ".ann"}},
				},
			}},
		})
		return p
	}
}
