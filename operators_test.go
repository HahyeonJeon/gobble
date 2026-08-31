package gobble_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestLOperatorsOnPipelineAndModule(t *testing.T) {
	p := gobble.NewPipeline("ops")
	if p.Scatter("each") == nil || p.Gather("all") == nil || p.When("opt") == nil {
		t.Fatalf("Pipeline Scatter/Gather/When returned nil")
	}
	m := gobble.NewPipeline("mod").AddModule("inner")
	if m.Scatter("each") == nil || m.Gather("all") == nil || m.When("opt") == nil {
		t.Fatalf("Module Scatter/Gather/When returned nil")
	}
}

func TestScatterBuildPlanAuthoredIDs(t *testing.T) {
	p := scatterGroupPipeline("true")
	raw := mustBuildPlanJSON(t, p)
	var decoded struct {
		Tasks []struct {
			ID         string `json:"id"`
			Instance   string `json:"instance"`
			ShardIndex int    `json:"shard_index"`
			ShardCount int    `json:"shard_count"`
			Scatter    string `json:"scatter"`
			Gather     string `json:"gather"`
		} `json:"tasks"`
		DAG struct {
			Nodes []string `json:"nodes"`
		} `json:"dag"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("plan JSON: %v", err)
	}
	if len(decoded.Tasks) != 2 {
		t.Fatalf("plan tasks got %d, want 2 authored", len(decoded.Tasks))
	}
	if decoded.Tasks[0].ID != "each.copy" || decoded.Tasks[0].Scatter != "each" {
		t.Fatalf("scatter task got %#v", decoded.Tasks[0])
	}
	if decoded.Tasks[0].Instance != "" || decoded.Tasks[0].ShardIndex != 0 || decoded.Tasks[0].ShardCount != 1 {
		t.Fatalf("scatter task identity got instance %q shard %d/%d", decoded.Tasks[0].Instance, decoded.Tasks[0].ShardIndex, decoded.Tasks[0].ShardCount)
	}
	if decoded.Tasks[1].ID != "all.join" || decoded.Tasks[1].Gather != "all" {
		t.Fatalf("gather task got %#v", decoded.Tasks[1])
	}
	for _, node := range decoded.DAG.Nodes {
		if strings.Contains(node, "/") {
			t.Fatalf("DAG node %q looks like a member identity", node)
		}
	}
	if bytes.Contains(raw, []byte("each.copy/s1/0")) {
		t.Fatalf("plan JSON exploded member identities")
	}
}

func TestScatterWithoutFromMissingInput(t *testing.T) {
	p := gobble.NewPipeline("no-from")
	p.Scatter("each").AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"true"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	_, err := gobble.Compose(p)
	requireComposeCode(t, err, gobble.DefectMissingInput, "each")
}

func TestScatterFileFromGroupCompose(t *testing.T) {
	p := scatterGroupPipeline("true")
	if _, err := gobble.Compose(p); err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
}

func TestAmbiguousGatherScatter(t *testing.T) {
	p := gobble.NewPipeline("ambig")
	a := p.AddInputGroup("a", gobble.Group{
		{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "a", Ext: ".txt"}},
	})
	b := p.AddInputGroup("b", gobble.Group{
		{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "b", Ext: ".txt"}},
	})
	left := p.Scatter("left").From(a).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"true"},
		Inputs:  []gobble.Bind{{Name: "in", From: a}},
		Outputs: []gobble.Bind{{Name: "out", From: a, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	right := p.Scatter("right").From(b).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"true"},
		Inputs:  []gobble.Bind{{Name: "in", From: b}},
		Outputs: []gobble.Bind{{Name: "out", From: b, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	p.Gather("all").AddTask(gobble.TaskSpec{
		Name:    "join",
		Command: []string{"true"},
		Inputs: []gobble.Bind{
			{Name: "left", From: left.Out("out")},
			{Name: "right", From: right.Out("out")},
		},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	_, err := gobble.Compose(p)
	requireComposeCode(t, err, gobble.DefectInvalidValue, "all.join")
}

func TestRunScatterMembersAndGather(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	g := mustCompose(func() *gobble.Pipeline {
		return scatterGroupPipeline(`f=$(find . -type f ! -name '*.out' | head -1); cp "$f" "$f.out"`)
	})(t)
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			t.Fatalf("Run() defects %#v", ge.Defects)
		}
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "in", "s1.txt.out")); err != nil {
		t.Fatalf("member dest s1: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "in", "s2.txt.out")); err != nil {
		t.Fatalf("member dest s2: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "all.txt"))
	if err != nil {
		t.Fatalf("gather dest: %v", err)
	}
	if !strings.Contains(string(got), "one") || !strings.Contains(string(got), "two") {
		t.Fatalf("gather dest got %q, want both members", got)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	var file struct {
		Tasks []struct {
			ID       string `json:"id"`
			Instance string `json:"instance"`
			Status   string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	idents := map[string]string{}
	for _, st := range file.Tasks {
		ident := st.ID
		if st.Instance != "" {
			ident = st.ID + "/" + st.Instance + "/0"
		}
		idents[ident] = st.Status
	}
	if idents["each.copy/s1/0"] != engine.StatusSucceeded || idents["each.copy/s2/0"] != engine.StatusSucceeded {
		t.Fatalf("member identities got %#v", idents)
	}
	if idents["all.join"] != engine.StatusSucceeded {
		t.Fatalf("gather status got %#v", idents)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect instances: %v", err)
	}
	if !bytes.Contains(rawInst, []byte(`"each.copy/s1/0"`)) {
		t.Fatalf("Inspect instances missing member identity: %s", rawInst)
	}
}

func TestRunScatterFileFromOneMember(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	p := gobble.NewPipeline("file-from")
	in := p.AddInput("sample", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	sc := p.Scatter("each").From(in)
	sc.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Script:  `f=$(find . -type f ! -name '*.out' | head -1); cp "$f" "$f.out"`,
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", From: in, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if !bytes.Contains(raw, []byte("in/sample.txt")) {
		t.Fatalf("tasks.json missing file dest key: %s", raw)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !bytes.Contains(rawInst, []byte("in/sample.txt")) {
		t.Fatalf("Inspect missing file member key: %s", rawInst)
	}
}

func TestRunGatherEmptyMembershipNeverReady(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tree"), 0o755); err != nil {
		t.Fatalf("mkdir tree: %v", err)
	}
	writeRunFile(t, filepath.Join(dir, "tree", ".gobble-tree.json"), `{"members":[]}`+"\n")
	p := gobble.NewPipeline("empty")
	tree := p.AddInputTree("items", gobble.DeclareTree(gobble.Dir("tree")))
	item := p.Scatter("each").From(tree).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"true"},
		Inputs:  []gobble.Bind{{Name: "in", From: tree, Tree: gobble.DeclareTree(gobble.Dir("work"))}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "item", Ext: ".txt"}}},
	})
	p.Gather("all").AddTask(gobble.TaskSpec{
		Name:    "join",
		Command: []string{"true"},
		Inputs:  []gobble.Bind{{Name: "in", From: item.Out("out")}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	err = gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
	requireRunError(t, "empty gather", err, gobble.DefectNeverReady, "each.copy")
}

func TestWhenSkipFalseParamAndRemaining(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	p := gobble.NewPipeline("when-false")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.When("opt").SkipIfFalse("keep").AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
		Params:  []gobble.Param{{Name: "keep", Value: "false"}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); !os.IsNotExist(err) {
		t.Fatalf("skipped task published dest, want absent")
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect instances: %v", err)
	}
	if !bytes.Contains(rawInst, []byte(`"skipped"`)) || !bytes.Contains(rawInst, []byte("false-param")) {
		t.Fatalf("Inspect instances skip facts got %s", rawInst)
	}
	rawRem, err := gobble.Inspect(dir, gobble.ViewRemaining, "")
	if err != nil {
		t.Fatalf("Inspect remaining: %v", err)
	}
	if bytes.Contains(rawRem, []byte("opt.copy")) {
		t.Fatalf("remaining listed skipped identity: %s", rawRem)
	}
}

func TestWhenSkipMissingFile(t *testing.T) {
	dir := t.TempDir()
	p := gobble.NewPipeline("when-miss")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.When("opt").SkipIfMissing(in).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Preflight(g, dir, 0); err != nil {
		t.Fatalf("Preflight() error = %v, want skip exemption", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !bytes.Contains(rawInst, []byte(`"skipped"`)) || !bytes.Contains(rawInst, []byte("missing-file")) {
		t.Fatalf("Inspect skip missing-file got %s", rawInst)
	}
}

func TestWhenSkipEmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "")
	p := gobble.NewPipeline("when-empty")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.When("opt").SkipIfMissing(in).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !bytes.Contains(rawInst, []byte("missing-file")) {
		t.Fatalf("Inspect empty file skip got %s", rawInst)
	}
}

func TestWhenUnknownProducerDoesNotSkip(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	p := gobble.NewPipeline("when-unknown")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	src := p.AddTask(gobble.TaskSpec{
		Name:    "prep",
		Command: []string{"false"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "prep", Ext: ".txt"}}},
	})
	p.When("opt").SkipIfMissing(src.Out("out")).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "out/prep.txt", "out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: src.Out("out")}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	err = gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
	requireRunError(t, "failed producer", err, gobble.DefectFailed, "prep")
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if bytes.Contains(rawInst, []byte(`"skipped"`)) {
		t.Fatalf("skipped while producer unsuccessful: %s", rawInst)
	}
}

func TestWhenFalseParamInvalidValue(t *testing.T) {
	p := gobble.NewPipeline("bad-bool")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.When("opt").SkipIfFalse("keep").AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"true"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{fileOut("out")},
		Params:  []gobble.Param{{Name: "keep", Value: "yes"}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	_, err = gobble.BuildPlan(g)
	requireComposeCode(t, err, gobble.DefectInvalidValue, "opt.copy")
}

func TestResumeUnchangedPreservesSkip(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	pipe := func() *gobble.Pipeline {
		p := gobble.NewPipeline("when-false")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		p.When("opt").SkipIfFalse("keep").AddTask(gobble.TaskSpec{
			Name:    "copy",
			Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
			Params:  []gobble.Param{{Name: "keep", Value: "false"}},
		})
		return p
	}
	g := mustCompose(pipe)(t)
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !bytes.Contains(rawInst, []byte(`"skipped"`)) {
		t.Fatalf("Unchanged resume lost skip: %s", rawInst)
	}
	if bytes.Contains(rawInst, []byte(`"attempt": 2`)) {
		t.Fatalf("Unchanged resume re-evaluated skip: %s", rawInst)
	}
}

func TestResumeUnchangedPreservesScatterMembers(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	g := mustCompose(func() *gobble.Pipeline {
		return scatterGroupPipeline(`f=$(find . -type f ! -name '*.out' | head -1); cp "$f" "$f.out"`)
	})(t)
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if !bytes.Contains(raw, []byte("each.copy/s1/0")) || !bytes.Contains(raw, []byte("each.copy/s2/0")) {
		t.Fatalf("Unchanged resume lost members: %s", raw)
	}
	if memberAttempt(raw, "s1") != 1 || memberAttempt(raw, "s2") != 1 {
		t.Fatalf("Unchanged resume relaunched members: %s", raw)
	}
}

func TestResumeIdentityChangedReexpands(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	g1 := mustCompose(func() *gobble.Pipeline {
		return scatterGroupPipeline(`f=$(find . -type f ! -name '*.out' | head -1); cp "$f" "$f.out"`)
	})(t)
	if err := gobble.Run(t.Context(), g1, dir, 2, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	g2 := mustCompose(func() *gobble.Pipeline {
		return scatterGroupPipeline(`f=$(find . -type f ! -name '*.out' | head -1); cp "$f" "$f.out"; true`)
	})(t)
	if err := gobble.Resume(t.Context(), g2, dir, 2, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			t.Fatalf("Resume() defects %#v", ge.Defects)
		}
		t.Fatalf("Resume() error = %v", err)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if memberAttempt(raw, "s1") < 2 && memberAttempt(raw, "s2") < 2 {
		t.Fatalf("IdentityChanged did not re-expand members: %s", raw)
	}
}

func TestRunScatterRestagedTreeExpandsProducer(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "tree", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "tree", "s2.txt"), "two")
	if err := engine.WriteTreeManifest(filepath.Join(dir, "tree")); err != nil {
		t.Fatalf("WriteTreeManifest() error = %v", err)
	}
	p := gobble.NewPipeline("tree-restage")
	tree := p.AddInputTree("items", gobble.DeclareTree(gobble.Dir("tree")))
	p.Scatter("each").From(tree).AddTask(gobble.TaskSpec{
		Name:   "copy",
		Script: `f=$(find . -type f ! -name '*.out' ! -name '.gobble-tree.json' | head -1); mkdir -p "$(dirname "$f")"; cp "$f" "$f.out"`,
		Inputs: []gobble.Bind{{
			Name: "in",
			From: tree,
			Tree: gobble.DeclareTree(gobble.Dir("work")),
		}},
		Outputs: []gobble.Bind{{Name: "out", From: tree, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			for _, d := range ge.Defects {
				if d.Code == gobble.DefectNeverReady {
					t.Fatalf("restaged Tree From never-ready: %#v", ge.Defects)
				}
			}
		}
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if !bytes.Contains(raw, []byte(`"s1.txt"`)) || !bytes.Contains(raw, []byte(`"s2.txt"`)) {
		t.Fatalf("restaged Tree expansion missing producer members: %s", raw)
	}
	if bytes.Contains(raw, []byte(`"members": []`)) {
		t.Fatalf("restaged Tree expansion empty: %s", raw)
	}
}

func TestResumeUnchangedRetriesFailedMember(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	g := mustCompose(func() *gobble.Pipeline {
		return scatterGroupPipeline("false")
	})(t)
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err == nil {
		t.Fatalf("Run() error = nil, want failed members")
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	rawRem, err := gobble.Inspect(dir, gobble.ViewRemaining, "")
	if err != nil {
		t.Fatalf("Inspect remaining: %v", err)
	}
	if !bytes.Contains(rawRem, []byte("each.copy/s1/0")) && !bytes.Contains(rawRem, []byte("each.copy/s2/0")) {
		t.Fatalf("failed members not remaining: %s", rawRem)
	}
	_ = gobble.Resume(t.Context(), g, dir, 2, testOccupyOption(t))
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if memberAttempt(raw, "s1") < 2 && memberAttempt(raw, "s2") < 2 {
		t.Fatalf("Unchanged resume did not retry failed members: %s", raw)
	}
}

func TestResumePredicateChangeReevaluates(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	first := func() *gobble.Pipeline {
		p := gobble.NewPipeline("when-pred")
		in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
		p.When("opt").SkipIfFalse("keep").AddTask(gobble.TaskSpec{
			Name:    "copy",
			Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
			Inputs:  []gobble.Bind{{Name: "in", From: in}},
			Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
			Params: []gobble.Param{
				{Name: "keep", Value: "false"},
				{Name: "run", Value: "true"},
			},
		})
		return p
	}
	g1 := mustCompose(first)(t)
	if err := gobble.Run(t.Context(), g1, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	p2 := gobble.NewPipeline("when-pred")
	in := p2.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p2.When("opt").SkipIfFalse("run").AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
		Params: []gobble.Param{
			{Name: "keep", Value: "false"},
			{Name: "run", Value: "true"},
		},
	})
	g2, err := gobble.Compose(p2)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g2, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); err != nil {
		t.Fatalf("predicate change did not run: %v", err)
	}
}

func TestRunScatterAddModuleSameMemberFlow(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	p := gobble.NewPipeline("mod-scatter")
	samples := p.AddInputGroup("samples", gobble.Group{
		{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}},
		{Name: "s2", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s2", Ext: ".txt"}},
	})
	mod := p.Scatter("each").From(samples).AddModule("mod")
	src := mod.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Script:  `f=$(find . -type f ! -name '*.out' ! -name '*.done' | head -1); cp "$f" "$f.out"`,
		Inputs:  []gobble.Bind{{Name: "in", From: samples}},
		Outputs: []gobble.Bind{{Name: "out", From: samples, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	mod.AddTask(gobble.TaskSpec{
		Name:    "mark",
		Script:  `f=$(find . -type f -name '*.out' | head -1); cp "$f" "$f.done"`,
		Inputs:  []gobble.Bind{{Name: "in", From: src.Out("out")}},
		Outputs: []gobble.Bind{{Name: "out", From: src.Out("out"), Spec: gobble.PathSpec{Ext: ".done"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			t.Fatalf("Run() defects %#v", ge.Defects)
		}
		t.Fatalf("Run() error = %v", err)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if !bytes.Contains(raw, []byte("each.mod.copy/s1/0")) || !bytes.Contains(raw, []byte("each.mod.mark/s1/0")) {
		t.Fatalf("AddModule members missing shared instanceSeg: %s", raw)
	}
	if _, err := os.Stat(filepath.Join(dir, "in", "s1.txt.out.done")); err != nil {
		t.Fatalf("same-member dest s1: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "in", "s2.txt.out.done")); err != nil {
		t.Fatalf("same-member dest s2: %v", err)
	}
}

func TestRunScatterAddModuleThreeTaskChain(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	p := gobble.NewPipeline("mod-chain")
	samples := p.AddInputGroup("samples", gobble.Group{
		{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}},
		{Name: "s2", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s2", Ext: ".txt"}},
	})
	mod := p.Scatter("each").From(samples).AddModule("mod")
	copyTask := mod.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Script:  `f=$(find . -type f ! -name '*.out' ! -name '*.done' ! -name '*.ok' | head -1); cp "$f" "$f.out"`,
		Inputs:  []gobble.Bind{{Name: "in", From: samples}},
		Outputs: []gobble.Bind{{Name: "out", From: samples, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	mark := mod.AddTask(gobble.TaskSpec{
		Name:    "mark",
		Script:  `f=$(find . -type f -name '*.out' ! -name '*.done' | head -1); cp "$f" "$f.done"`,
		Inputs:  []gobble.Bind{{Name: "in", From: copyTask.Out("out")}},
		Outputs: []gobble.Bind{{Name: "out", From: copyTask.Out("out"), Spec: gobble.PathSpec{Ext: ".done"}}},
	})
	mod.AddTask(gobble.TaskSpec{
		Name:    "check",
		Script:  `f=$(find . -type f -name '*.done' | head -1); cp "$f" "$f.ok"`,
		Inputs:  []gobble.Bind{{Name: "in", From: mark.Out("out")}},
		Outputs: []gobble.Bind{{Name: "out", From: mark.Out("out"), Spec: gobble.PathSpec{Ext: ".ok"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			t.Fatalf("Run() defects %#v", ge.Defects)
		}
		t.Fatalf("Run() error = %v", err)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if !bytes.Contains(raw, []byte("each.mod.check/s1/0")) || !bytes.Contains(raw, []byte("each.mod.check/s2/0")) {
		t.Fatalf("three-task AddModule missing check members: %s", raw)
	}
	if _, err := os.Stat(filepath.Join(dir, "in", "s1.txt.out.done.ok")); err != nil {
		t.Fatalf("third-hop dest s1: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "in", "s2.txt.out.done.ok")); err != nil {
		t.Fatalf("third-hop dest s2: %v", err)
	}
}

func TestResumeFromChangeDoesNotSubmitLeftoverMembers(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	writeRunFile(t, filepath.Join(dir, "in", "s3.txt"), "three")
	g1 := mustCompose(func() *gobble.Pipeline {
		return scatterNamedGroupPipeline(
			`f=$(find . -type f ! -name '*.out' | head -1); cp "$f" "$f.out"`,
			gobble.Group{
				{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}},
				{Name: "s2", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s2", Ext: ".txt"}},
			},
		)
	})(t)
	if err := gobble.Run(t.Context(), g1, dir, 2, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	g2 := mustCompose(func() *gobble.Pipeline {
		return scatterNamedGroupPipeline(
			`f=$(find . -type f ! -name '*.out' | head -1); cp "$f" "$f.out"`,
			gobble.Group{
				{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}},
				{Name: "s3", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s3", Ext: ".txt"}},
			},
		)
	})(t)
	if err := gobble.Resume(t.Context(), g2, dir, 2, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			t.Fatalf("Resume() defects %#v", ge.Defects)
		}
		t.Fatalf("Resume() error = %v", err)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	if !bytes.Contains(raw, []byte(`"s1"`)) || !bytes.Contains(raw, []byte(`"s3"`)) {
		t.Fatalf("From Change missing new expansion: %s", raw)
	}
	recs := decodeTaskRecords(t, raw)
	s2 := recordsByInstance(recs, "s2")
	if len(s2) != 1 || s2[0].Attempt != 1 || s2[0].Status != engine.StatusSucceeded {
		t.Fatalf("leftover s2 got %#v, want one succeeded attempt 1", s2)
	}
	if isolateExists(dir, "each.copy", "s2", 2) {
		t.Fatalf("From Change submitted leftover s2 isolate")
	}
	s1 := latestByInstance(recs, "s1")
	if s1.Attempt != 2 || s1.Status != engine.StatusSucceeded ||
		s1.Decision != "rerun" || s1.Change != "IdentityChanged" {
		t.Fatalf("retained s1 got %#v, want attempt 2 rerun IdentityChanged", s1)
	}
	for _, rec := range recordsByInstance(recs, "s1") {
		if rec.Status == engine.StatusNotStarted {
			t.Fatalf("retained s1 invented not-started attempt: %#v", rec)
		}
	}
	if isolateExists(dir, "each.copy", "s1", 3) {
		t.Fatalf("retained s1 invented attempt 3 isolate")
	}
	s3 := latestByInstance(recs, "s3")
	if s3.Attempt != 1 || s3.Status != engine.StatusSucceeded ||
		s3.Decision != "rerun" || s3.Change != "IdentityChanged" {
		t.Fatalf("new s3 got %#v, want attempt 1 rerun IdentityChanged", s3)
	}
	if len(recordsByInstance(recs, "s3")) != 1 {
		t.Fatalf("new s3 invented extra attempts: %#v", recordsByInstance(recs, "s3"))
	}
	if isolateExists(dir, "each.copy", "s3", 2) {
		t.Fatalf("new s3 invented attempt 2 isolate")
	}
	if _, err := os.Stat(filepath.Join(dir, "in", "s3.txt.out")); err != nil {
		t.Fatalf("new member dest: %v", err)
	}
}

func TestResumeTrueToFalseWhenSkips(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	first := func() *gobble.Pipeline {
		return whenPredWithAfter("run")
	}
	g1 := mustCompose(first)(t)
	if err := gobble.Run(t.Context(), g1, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	g2, err := gobble.Compose(whenPredWithAfter("keep"))
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g2, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	got := inspectLatestByID(t, rawInst)
	copyRec := got["opt.copy"]
	if copyRec.Status != engine.StatusSkipped || copyRec.Condition != "false-param" {
		t.Fatalf("true-to-false When did not skip: %s", rawInst)
	}
	after := got["after"]
	if after.Status != engine.StatusSkipped {
		t.Fatalf("downstream of skip stayed %q, want skipped: %s", after.Status, rawInst)
	}
}

func TestResumeTrueToFalseWhenSkipsPreviouslyFailedDownstream(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	g1 := mustCompose(func() *gobble.Pipeline {
		return whenPredWithAfterCmd("run", []string{"false"})
	})(t)
	if err := gobble.Run(t.Context(), g1, dir, 0, testOccupyOption(t)); err == nil {
		t.Fatalf("Run() error = nil, want failed downstream")
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	g2, err := gobble.Compose(whenPredWithAfterCmd("keep", []string{"false"}))
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g2, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v, want skipped downstream", err)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	got := inspectLatestByID(t, rawInst)
	if got["opt.copy"].Status != engine.StatusSkipped || got["opt.copy"].Condition != "false-param" {
		t.Fatalf("true-to-false When did not skip: %s", rawInst)
	}
	if got["after"].Status != engine.StatusSkipped {
		t.Fatalf("failed downstream of skip stayed %q, want skipped: %s", got["after"].Status, rawInst)
	}
}

func TestResumeTrueToFalseWhenSkipsReleasedCanceledDescendant(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	g1 := mustCompose(func() *gobble.Pipeline {
		return whenPredWithAfterCmd("run", []string{"sleep", "30"})
	})(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- gobble.Run(ctx, g1, dir, 0, testOccupyOption(t))
	}()
	waitTaskStatus(t, dir, "after", engine.StatusRunning)
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("Run() error = nil, want canceled")
		}
	case <-time.After(8 * time.Second):
		t.Fatal("timeout waiting for canceled Run")
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	rawRem, err := gobble.Inspect(dir, gobble.ViewRemaining, "after")
	if err != nil {
		t.Fatalf("Inspect remaining: %v", err)
	}
	if !bytes.Contains(rawRem, []byte(`"identity":"after"`)) || !bytes.Contains(rawRem, []byte(`"remaining":true`)) {
		t.Fatalf("released canceled descendant not remaining: %s", rawRem)
	}
	g2, err := gobble.Compose(whenPredWithAfterCmd("keep", []string{"sleep", "30"}))
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g2, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v, want skipped released descendant", err)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	got := inspectLatestByID(t, rawInst)
	if got["opt.copy"].Status != engine.StatusSkipped || got["opt.copy"].Condition != "false-param" {
		t.Fatalf("true-to-false When did not skip: %s", rawInst)
	}
	if got["after"].Status != engine.StatusSkipped {
		t.Fatalf("released canceled descendant stayed %q, want skipped: %s", got["after"].Status, rawInst)
	}
}

func TestResumeTrueToFalseWhenSkipsFailedScatterMember(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	writeRunFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeRunFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
	g1 := mustCompose(func() *gobble.Pipeline {
		return whenScatterFailPipeline("run")
	})(t)
	if err := gobble.Run(t.Context(), g1, dir, 2, testOccupyOption(t)); err == nil {
		t.Fatalf("Run() error = nil, want failed scatter members")
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	g2, err := gobble.Compose(whenScatterFailPipeline("keep"))
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g2, dir, 2, testOccupyOption(t)); err != nil {
		t.Fatalf("Resume() error = %v, want skipped failed members", err)
	}
	rawInst, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	got := inspectLatestByID(t, rawInst)
	if got["opt.gate"].Status != engine.StatusSkipped {
		t.Fatalf("true-to-false When gate stayed %q: %s", got["opt.gate"].Status, rawInst)
	}
	for _, ident := range []string{"each.copy/s1/0", "each.copy/s2/0"} {
		if got[ident].Status != engine.StatusSkipped {
			t.Fatalf("failed member %s stayed %q, want skipped: %s", ident, got[ident].Status, rawInst)
		}
	}
}

func memberAttempt(raw []byte, key string) int {
	var file struct {
		Tasks []struct {
			Instance string `json:"instance"`
			Attempt  int    `json:"attempt"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return 0
	}
	best := 0
	for _, st := range file.Tasks {
		if st.Instance == key && st.Attempt > best {
			best = st.Attempt
		}
	}
	return best
}

type taskRecord struct {
	ID          string `json:"id"`
	Instance    string `json:"instance"`
	Attempt     int    `json:"attempt"`
	Status      string `json:"status"`
	Decision    string `json:"decision"`
	Change      string `json:"change"`
	ReuseReason string `json:"reuse_reason"`
}

func decodeTaskRecords(t *testing.T, raw []byte) []taskRecord {
	t.Helper()
	var file struct {
		Tasks []taskRecord `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	return file.Tasks
}

func recordsByInstance(recs []taskRecord, key string) []taskRecord {
	var out []taskRecord
	for _, rec := range recs {
		if rec.Instance == key {
			out = append(out, rec)
		}
	}
	return out
}

func latestByInstance(recs []taskRecord, key string) taskRecord {
	var best taskRecord
	for _, rec := range recs {
		if rec.Instance == key && rec.Attempt >= best.Attempt {
			best = rec
		}
	}
	return best
}

func isolateExists(workspace, id, instance string, attempt int) bool {
	_, err := os.Stat(filepath.Join(workspace, engine.ControlDir, "tasks", id, instance, "0", strconv.Itoa(attempt)))
	return err == nil
}

type inspectInstRec struct {
	Identity  string `json:"identity"`
	Status    string `json:"status"`
	Attempt   int    `json:"attempt"`
	Condition string `json:"condition"`
}

func inspectLatestByID(t *testing.T, raw []byte) map[string]inspectInstRec {
	t.Helper()
	out := map[string]inspectInstRec{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	for dec.More() {
		var rec inspectInstRec
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("Inspect JSONL: %v", err)
		}
		out[rec.Identity] = rec
	}
	return out
}

func waitTaskStatus(t *testing.T, dir, id string, want ...string) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(filepath.Join(dir, engine.ControlDir, engine.TasksFile))
		if err == nil {
			var file struct {
				Tasks []taskRecord `json:"tasks"`
			}
			if json.Unmarshal(raw, &file) == nil {
				for _, rec := range file.Tasks {
					if rec.ID != id || rec.Instance != "" {
						continue
					}
					for _, w := range want {
						if rec.Status == w {
							return
						}
					}
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s in %v", id, want)
}

func whenPredWithAfter(skipParam string) *gobble.Pipeline {
	return whenPredWithAfterCmd(skipParam, []string{"cp", "out/sample.txt", "out/after.txt"})
}

func whenPredWithAfterCmd(skipParam string, afterCmd []string) *gobble.Pipeline {
	p := gobble.NewPipeline("when-pred")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	copyTask := p.When("opt").SkipIfFalse(skipParam).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}}},
		Params: []gobble.Param{
			{Name: "keep", Value: "false"},
			{Name: "run", Value: "true"},
		},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "after",
		Command: afterCmd,
		Inputs:  []gobble.Bind{{Name: "in", From: copyTask.Out("out")}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "after", Ext: ".txt"}}},
	})
	return p
}

func whenScatterFailPipeline(skipParam string) *gobble.Pipeline {
	p := gobble.NewPipeline("when-scatter")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	gate := p.When("opt").SkipIfFalse(skipParam).AddTask(gobble.TaskSpec{
		Name:    "gate",
		Command: []string{"cp", "in/sample.txt", "out/gate.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "gate", Ext: ".txt"}}},
		Params: []gobble.Param{
			{Name: "keep", Value: "false"},
			{Name: "run", Value: "true"},
		},
	})
	samples := p.AddInputGroup("samples", gobble.Group{
		{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}},
		{Name: "s2", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s2", Ext: ".txt"}},
	})
	p.Scatter("each").From(samples).AddTask(gobble.TaskSpec{
		Name:   "copy",
		Script: "false",
		Inputs: []gobble.Bind{
			{Name: "in", From: samples},
			{Name: "gate", From: gate.Out("out")},
		},
		Outputs: []gobble.Bind{{Name: "out", From: samples, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	return p
}

func scatterGroupPipeline(script string) *gobble.Pipeline {
	return scatterNamedGroupPipeline(script, gobble.Group{
		{Name: "s1", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}},
		{Name: "s2", Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s2", Ext: ".txt"}},
	})
}

func scatterNamedGroupPipeline(script string, members gobble.Group) *gobble.Pipeline {
	p := gobble.NewPipeline("scatter")
	samples := p.AddInputGroup("samples", members)
	item := p.Scatter("each").From(samples).AddTask(gobble.TaskSpec{
		Name:    "copy",
		Script:  script,
		Inputs:  []gobble.Bind{{Name: "in", From: samples}},
		Outputs: []gobble.Bind{{Name: "out", From: samples, Spec: gobble.PathSpec{Ext: ".out"}}},
	})
	p.Gather("all").AddTask(gobble.TaskSpec{
		Name:   "join",
		Script: "mkdir -p out; find . -name '*.out' -exec cat {} + > out/all.txt",
		Inputs: []gobble.Bind{{Name: "parts", From: item.Out("out")}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "all", Ext: ".txt"},
		}},
	})
	return p
}

func requireComposeCode(t *testing.T, err error, code gobble.DefectCode, unit string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %s", code)
	}
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("error %v, want *Error", err)
	}
	for _, d := range ge.Defects {
		if d.Code == code && (unit == "" || d.Unit == unit) {
			return
		}
	}
	t.Fatalf("error defects %#v, want %s unit %s", ge.Defects, code, unit)
}
