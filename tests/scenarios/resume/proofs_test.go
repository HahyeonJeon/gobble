package resume

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/HahyeonJeon/gobble"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

var testIdentityOnce sync.Once
var testIdentity gobble.Identity
var testIdentityErr error

func testOccupyOption(t *testing.T) gobble.OccupyOption {
	t.Helper()
	testIdentityOnce.Do(func() {
		testIdentity, testIdentityErr = gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/assets")
	})
	if testIdentityErr != nil {
		t.Fatalf("IdentityFromBuildInfo() error = %v", testIdentityErr)
	}
	return gobble.WithIdentity(testIdentity)
}

type inspectRec struct {
	Identity  string `json:"identity"`
	Status    string `json:"status"`
	Attempt   int    `json:"attempt"`
	Decision  string `json:"decision"`
	Reason    string `json:"reuse_reason"`
	Condition string `json:"condition"`
	Instance  string `json:"instance"`
}

func TestScatterGatherRunAndPlan(t *testing.T) {
	dir := t.TempDir()
	writeProofFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
	writeProofFile(t, filepath.Join(dir, "in", "s2.txt"), "two")

	raw := pc.MustPlanJSON(t, ScatterGather())
	tasks := pc.AllTasks(t, raw)
	if len(tasks) != 2 {
		t.Fatalf("plan tasks got %d, want 2 authored", len(tasks))
	}
	pc.MustHaveTaskID(t, tasks, "each.copy")
	pc.MustHaveTaskID(t, tasks, "all.join")
	if bytes.Contains(raw, []byte("each.copy/s1/0")) {
		t.Fatalf("plan JSON exploded member identities")
	}

	mustRunProof(t, mustComposeProof(t, ScatterGather()), dir, 2)
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
	inst := inspectByIdentity(t, dir, gobble.ViewInstances)
	if inst["each.copy/s1/0"].Status != "succeeded" || inst["each.copy/s2/0"].Status != "succeeded" {
		t.Fatalf("scatter members got %#v", inst)
	}
	if inst["all.join"].Status != "succeeded" {
		t.Fatalf("gather status got %#v", inst)
	}
}

func TestConditionalSkipFalseAndTrue(t *testing.T) {
	t.Run("false", func(t *testing.T) {
		dir := t.TempDir()
		writeProofFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
		mustRunProof(t, mustComposeProof(t, ConditionalSkip("false")), dir, 0)
		if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); !os.IsNotExist(err) {
			t.Fatalf("skipped task published dest, want absent")
		}
		inst := inspectByIdentity(t, dir, gobble.ViewInstances)
		if inst["opt.copy"].Status != "skipped" || inst["opt.copy"].Condition != "false-param" {
			t.Fatalf("skip facts got %#v", inst["opt.copy"])
		}
		rawRem, err := gobble.Inspect(dir, gobble.ViewRemaining, "")
		if err != nil {
			t.Fatalf("Inspect remaining: %v", err)
		}
		if bytes.Contains(rawRem, []byte("opt.copy")) {
			t.Fatalf("remaining listed skipped identity: %s", rawRem)
		}
	})
	t.Run("true", func(t *testing.T) {
		dir := t.TempDir()
		writeProofFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
		mustRunProof(t, mustComposeProof(t, ConditionalSkip("true")), dir, 0)
		assertFile(t, filepath.Join(dir, "out", "sample.txt"), "reads")
		inst := inspectByIdentity(t, dir, gobble.ViewInstances)
		if inst["opt.copy"].Status != "succeeded" {
			t.Fatalf("true branch status = %q, want succeeded", inst["opt.copy"].Status)
		}
	})
}

func TestDynamicExpansionRunAndPlan(t *testing.T) {
	dir := t.TempDir()
	raw := pc.MustPlanJSON(t, DynamicExpansion())
	tasks := pc.AllTasks(t, raw)
	pc.MustHaveTaskID(t, tasks, "split")
	pc.MustHaveTaskID(t, tasks, "each.copy")
	pc.MustHaveTaskID(t, tasks, "all.join")
	if bytes.Contains(raw, []byte("each.copy/s1.txt/0")) {
		t.Fatalf("plan JSON exploded member identities")
	}

	mustRunProof(t, mustComposeProof(t, DynamicExpansion()), dir, 2)
	inst := inspectByIdentity(t, dir, gobble.ViewInstances)
	if inst["split"].Status != "succeeded" {
		t.Fatalf("producer status = %q, want succeeded", inst["split"].Status)
	}
	if inst["each.copy/s1.txt/0"].Status != "succeeded" || inst["each.copy/s2.txt/0"].Status != "succeeded" {
		t.Fatalf("expanded members got %#v", inst)
	}
	if inst["all.join"].Status != "succeeded" {
		t.Fatalf("gather status got %#v", inst)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "all.txt"))
	if err != nil {
		t.Fatalf("gather dest: %v", err)
	}
	if !strings.Contains(string(got), "one") || !strings.Contains(string(got), "two") {
		t.Fatalf("gather dest got %q, want both expanded members", got)
	}
}

func TestTreeGroupPublication(t *testing.T) {
	dir := t.TempDir()
	writeProofFile(t, filepath.Join(dir, "in", "ref.amb"), "amb")
	writeProofFile(t, filepath.Join(dir, "in", "ref.ann"), "ann")
	mustRunProof(t, mustComposeProof(t, TreeGroup()), dir, 2)
	assertFile(t, filepath.Join(dir, "out", "out.amb"), "amb")
	assertFile(t, filepath.Join(dir, "out", "out.ann"), "ann")
	assertFile(t, filepath.Join(dir, "out", "ok.txt"), "ok\n")
	man, err := os.ReadFile(filepath.Join(dir, "work", "idx", ".gobble-tree.json"))
	if err != nil {
		t.Fatalf("tree manifest: %v", err)
	}
	if !bytes.Contains(man, []byte(`"path": "SA"`)) || !bytes.Contains(man, []byte(`"path": "ann"`)) {
		t.Fatalf("manifest %s, want SA and ann", man)
	}
	if bytes.Contains(man, []byte(".gobble-tree.json")) {
		t.Fatalf("manifest listed itself: %s", man)
	}
	inst := inspectByIdentity(t, dir, gobble.ViewInstances)
	for _, id := range []string{"copy_group", "make_tree", "use_tree"} {
		if inst[id].Status != "succeeded" {
			t.Fatalf("%s status = %q, want succeeded", id, inst[id].Status)
		}
	}
}

func TestFanoutResumeReusesScatterAndModules(t *testing.T) {
	t.Run("scatter shards", func(t *testing.T) {
		dir := t.TempDir()
		writeProofFile(t, filepath.Join(dir, "in", "s1.txt"), "one")
		writeProofFile(t, filepath.Join(dir, "in", "s2.txt"), "two")
		g := mustComposeProof(t, ScatterGather())
		mustRunProof(t, g, dir, 2)
		mustReleaseProof(t, dir)
		mustResumeProof(t, g, dir, 2)
		inst := inspectByIdentity(t, dir, gobble.ViewInstances)
		for _, id := range []string{"each.copy/s1/0", "each.copy/s2/0", "all.join"} {
			if inst[id].Status != "succeeded" || inst[id].Attempt != 1 {
				t.Fatalf("%s after resume got %#v, want succeeded attempt 1", id, inst[id])
			}
		}
		reuse := inspectReuseByIdentity(t, dir)
		if reuse["all.join"].Decision != "reused" {
			t.Fatalf("gather reuse got %#v", reuse["all.join"])
		}
		if _, err := os.Stat(filepath.Join(dir, "in", "s1.txt.out")); err != nil {
			t.Fatalf("scatter resume lost s1 dest: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "in", "s2.txt.out")); err != nil {
			t.Fatalf("scatter resume lost s2 dest: %v", err)
		}
	})
	t.Run("module fan-out", func(t *testing.T) {
		dir := t.TempDir()
		writeProofFile(t, filepath.Join(dir, "in", "a.txt"), "a")
		writeProofFile(t, filepath.Join(dir, "in", "b.txt"), "b")
		g := mustComposeProof(t, ModuleFanout())
		mustRunProof(t, g, dir, 2)
		mustReleaseProof(t, dir)
		mustResumeProof(t, g, dir, 2)
		reuse := inspectReuseByIdentity(t, dir)
		for _, id := range []string{"left.copy", "right.copy"} {
			if reuse[id].Decision != "reused" || reuse[id].Reason != "reused-identity-matched" {
				t.Fatalf("%s reuse got %#v", id, reuse[id])
			}
		}
		inst := inspectByIdentity(t, dir, gobble.ViewInstances)
		if inst["left.copy"].Attempt != 1 || inst["right.copy"].Attempt != 1 {
			t.Fatalf("module fan-out resume relaunched: %#v", inst)
		}
		assertFile(t, filepath.Join(dir, "out", "a.txt"), "a")
		assertFile(t, filepath.Join(dir, "out", "b.txt"), "b")
	})
}

func TestSyntheticCohortRun(t *testing.T) {
	dir := t.TempDir()
	raw := pc.MustPlanJSON(t, SyntheticCohort())
	tasks := pc.AllTasks(t, raw)
	if len(tasks) != syntheticCohortSize {
		t.Fatalf("cohort plan tasks = %d, want %d", len(tasks), syntheticCohortSize)
	}
	for i := 1; i <= syntheticCohortSize; i++ {
		name := fmt.Sprintf("s%02d", i)
		writeProofFile(t, filepath.Join(dir, "in", name+".txt"), name)
		pc.MustHaveTaskID(t, tasks, name+".copy")
		if pc.TaskByID(t, raw, name+".copy").Image != "" {
			t.Fatalf("%s.copy used an image, want process", name)
		}
	}
	mustRunProof(t, mustComposeProof(t, SyntheticCohort()), dir, 4)
	inst := inspectByIdentity(t, dir, gobble.ViewInstances)
	if len(inst) != syntheticCohortSize {
		t.Fatalf("cohort instances = %d, want %d", len(inst), syntheticCohortSize)
	}
	for i := 1; i <= syntheticCohortSize; i++ {
		name := fmt.Sprintf("s%02d", i)
		id := name + ".copy"
		if inst[id].Status != "succeeded" {
			t.Fatalf("%s status = %q, want succeeded", id, inst[id].Status)
		}
		assertFile(t, filepath.Join(dir, "work", name, "out.txt"), name)
	}
}

func writeProofFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func mustComposeProof(t *testing.T, p *gobble.Pipeline) *gobble.Graph {
	t.Helper()
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
	return g
}

func mustRunProof(t *testing.T, g *gobble.Graph, dir string, cap int) {
	t.Helper()
	if err := gobble.Run(t.Context(), g, dir, cap, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			t.Fatalf("Run() defects %#v", ge.Defects)
		}
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func mustReleaseProof(t *testing.T, dir string) {
	t.Helper()
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
}

func mustResumeProof(t *testing.T, g *gobble.Graph, dir string, cap int) {
	t.Helper()
	if err := gobble.Resume(t.Context(), g, dir, cap, testOccupyOption(t)); err != nil {
		var ge *gobble.Error
		if errors.As(err, &ge) {
			t.Fatalf("Resume() defects %#v", ge.Defects)
		}
		t.Fatalf("Resume() error = %v, want nil", err)
	}
}

func inspectByIdentity(t *testing.T, dir string, view gobble.View) map[string]inspectRec {
	t.Helper()
	raw, err := gobble.Inspect(dir, view, "")
	if err != nil {
		t.Fatalf("Inspect(%s): %v", view, err)
	}
	out := map[string]inspectRec{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	for dec.More() {
		var rec inspectRec
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("Inspect %s JSONL: %v", view, err)
		}
		out[rec.Identity] = rec
	}
	return out
}

func inspectReuseByIdentity(t *testing.T, dir string) map[string]inspectRec {
	t.Helper()
	raw, err := gobble.Inspect(dir, gobble.ViewReuse, "")
	if err != nil {
		t.Fatalf("Inspect(reuse): %v", err)
	}
	out := map[string]inspectRec{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	for dec.More() {
		var rec struct {
			Identity string `json:"identity"`
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("Inspect reuse JSONL: %v", err)
		}
		out[rec.Identity] = inspectRec{Identity: rec.Identity, Decision: rec.Decision, Reason: rec.Reason}
	}
	return out
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func assertNoTaskID(t *testing.T, tasks []pc.Task, id string) {
	t.Helper()
	for _, task := range tasks {
		if task.ID == id {
			t.Fatalf("plan has task id %q, want none", id)
		}
	}
}
