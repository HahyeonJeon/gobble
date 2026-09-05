package monitor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/monitor"
)

func TestMonitorFailureResumeReuseAndUnavailableLog(t *testing.T) {
	workspace := t.TempDir()
	gate := filepath.Join(t.TempDir(), "ready")
	identity, err := gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/monitor")
	if err != nil {
		t.Fatal(err)
	}
	option := gobble.WithIdentity(identity)
	output := func(name string) gobble.Bind {
		return gobble.Bind{Name: "out", Spec: gobble.PathSpec{Base: name, Ext: ".txt"}}
	}
	build := func(stage string) *gobble.Graph {
		p := gobble.NewPipeline("monitor-review")
		ref := p.AddTask(gobble.TaskSpec{Name: "reference", Display: gobble.TaskDisplay{Stage: stage, Scope: gobble.DisplayShared}, Command: []string{"sh", "-c", "printf reference > reference.txt"}, Outputs: []gobble.Bind{output("reference")}})
		sample := p.AddModule("S01").WithDisplay(gobble.TaskDisplay{Samples: []string{"S01"}}).AddTask(gobble.TaskSpec{Name: "work", Command: []string{"sh", "-c", `if [ ! -f "$1" ]; then printf 'example failure\n' >&2; exit 1; fi; cp reference.txt sample.txt`, "monitor-test", gate}, Inputs: []gobble.Bind{{Name: "in", From: ref.Out("out")}}, Outputs: []gobble.Bind{output("sample")}})
		p.AddModule("S010").WithDisplay(gobble.TaskDisplay{Samples: []string{"S010"}}).AddTask(gobble.TaskSpec{Name: "work", Command: []string{"sh", "-c", "printf healthy > healthy.txt"}, Outputs: []gobble.Bind{output("healthy")}})
		p.AddTask(gobble.TaskSpec{Name: "report", Display: gobble.TaskDisplay{Scope: gobble.DisplayCohort}, Command: []string{"cp", "sample.txt", "report.txt"}, Inputs: []gobble.Bind{{Name: "in", From: sample.Out("out")}}, Outputs: []gobble.Bind{output("report")}})
		g, err := gobble.Compose(p)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	if err := gobble.Run(t.Context(), build("Reference"), workspace, 2, option); err == nil {
		t.Fatal("fixture did not fail")
	}
	read := func(instance string) *monitor.Dashboard {
		s, err := monitor.Read(workspace, instance, option)
		if err != nil {
			t.Fatal(err)
		}
		d, err := monitor.Build(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	failed := read("S01.work")
	if c := failed.Total; c.Total != 4 || c.Succeeded != 2 || c.Failed != 1 || c.Blocked != 1 {
		t.Fatalf("failure counts: %+v", c)
	}
	if len(failed.SampleTasks("S01")) != 1 || len(failed.Attention) != 2 || failed.Shared.Total != 2 {
		t.Fatal("sample or attention membership incorrect")
	}
	if len(failed.Snapshot.Logs) != 1 || !strings.Contains(failed.Snapshot.Logs[0].StderrTail, "example failure") {
		t.Fatal("failed task log missing")
	}
	task, _ := failed.Task("S01.work")
	if task.Error == nil || task.Error.Unit == "" || task.Started == "" || task.Ended == "" {
		t.Fatalf("failure facts missing: %+v", task)
	}
	// A selected file that escapes the workspace stays unread, while global
	// monitoring remains available through an independently gated snapshot.
	logPath := filepath.Join(workspace, failed.Snapshot.Logs[0].Stderr)
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("PRIVATE OUTSIDE DATA"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, logPath); err != nil {
		t.Fatal(err)
	}
	if _, err := gobble.Inspect(workspace, gobble.ViewMonitor, "S01.work", option); err == nil {
		t.Fatal("raw inspect bypassed log containment")
	}
	unavailable := read("S01.work")
	if unavailable.Total != failed.Total || len(unavailable.Snapshot.Logs) != 1 || unavailable.Snapshot.Logs[0].Error == "" || unavailable.Snapshot.Logs[0].StderrTail != "" {
		t.Fatal("unreadable log froze global progress or disclosed bytes")
	}
	other := identity
	other.GobbleExecutableSHA256 = strings.Repeat("b", 64)
	if _, err := monitor.Read(workspace, "S01.work", gobble.WithIdentity(other)); err == nil {
		t.Fatal("fallback bypassed identity checks")
	}
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("example failure\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := gobble.Release(workspace, option); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gate, []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := gobble.Resume(t.Context(), build("Reference renamed for display"), workspace, 2, option); err != nil {
		t.Fatal(err)
	}
	done := read("S01.work")
	if done.Total.Total != 4 || !done.Total.Successful() || done.Total.Reused != 2 || len(done.Attention) != 0 {
		t.Fatalf("resume/reuse counts: %+v", done.Total)
	}
	task, _ = done.Task("S01.work")
	if task.Attempt != 2 {
		t.Fatalf("did not select latest attempt: %+v", task)
	}
	missing := read("retired-member")
	if missing.Total != done.Total || len(missing.Snapshot.Logs) != 0 {
		t.Fatal("retired selection froze the snapshot")
	}
}

func TestMonitorScatterMembersKeepLabelsAndCounts(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "in"), 0755); err != nil {
		t.Fatal(err)
	}
	members := gobble.Group{}
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(workspace, "in", name+".txt"), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		members = append(members, gobble.Member{Name: name, Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: name, Ext: ".txt"}})
	}
	p := gobble.NewPipeline("scatter-monitor")
	input := p.AddInputGroup("reads", members)
	m := p.AddModule("S01").WithDisplay(gobble.TaskDisplay{Samples: []string{"S01"}})
	m.Scatter("each").From(input).AddTask(gobble.TaskSpec{
		Name: "copy", Script: `for f in in/*.txt; do cp "$f" "$f.out"; done`,
		Inputs: []gobble.Bind{{Name: "in", From: input}}, Outputs: []gobble.Bind{{Name: "out", From: input, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	graph, err := gobble.Compose(p)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/monitor")
	if err != nil {
		t.Fatal(err)
	}
	option := gobble.WithIdentity(identity)
	if err := gobble.Run(t.Context(), graph, workspace, 2, option); err != nil {
		t.Fatal(err)
	}
	snapshot, err := monitor.Read(workspace, "", option)
	if err != nil {
		t.Fatal(err)
	}
	d, err := monitor.Build(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if d.Total.Total != 2 || d.Total.Succeeded != 2 || d.Total.Templates != 1 || d.Total.Unexpanded != 0 || len(d.Stages) != 1 || len(d.Samples) != 1 || d.Samples[0].Counts.Total != 2 {
		t.Fatalf("expanded members miscounted or lost labels: %+v", d)
	}
}
