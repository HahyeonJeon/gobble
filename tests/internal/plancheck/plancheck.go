// Package plancheck provides shared assertions for module and pipeline plans.
package plancheck

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

// Task is the plan surface used by asset evidence.
type Task struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Module    string    `json:"module"`
	Command   []string  `json:"command"`
	Script    string    `json:"script"`
	Image     string    `json:"image"`
	Resources Resources `json:"resources"`
	Params    []Param   `json:"params"`
	Inputs    []IO      `json:"inputs"`
	Outputs   []IO      `json:"outputs"`
}

// Resources is the plan resource surface used by parity evidence.
type Resources struct {
	CPU    float64         `json:"cpu"`
	Memory json.RawMessage `json:"memory"`
}

// Param is one plan parameter.
type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// IO is one plan bind.
type IO struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Path     string   `json:"path"`
	Source   string   `json:"source"`
	Manifest string   `json:"manifest"`
	Members  []Member `json:"members"`
}

// Member is one Group member.
type Member struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// MustPlanJSON composes p and returns its plan JSON.
func MustPlanJSON(t *testing.T, p *gobble.Pipeline) []byte {
	t.Helper()
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
	var buf bytes.Buffer
	if _, err := gobble.BuildPlan(g, gobble.WriteTo(&buf)); err != nil {
		t.Fatalf("BuildPlan() error = %v, want nil", err)
	}
	return buf.Bytes()
}

// AllTasks decodes all tasks from plan JSON.
func AllTasks(t *testing.T, raw []byte) []Task {
	t.Helper()
	var decoded struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal plan: %v", err)
	}
	return decoded.Tasks
}

// TaskByID returns the task with id or fails the test.
func TaskByID(t *testing.T, raw []byte, id string) Task {
	t.Helper()
	for _, task := range AllTasks(t, raw) {
		if task.ID == id {
			return task
		}
	}
	t.Fatalf("plan missing task %q in %#v", id, AllTasks(t, raw))
	return Task{}
}

// CountTasksNamed returns the number of tasks with name.
func CountTasksNamed(tasks []Task, name string) int {
	count := 0
	for _, task := range tasks {
		if task.Name == name {
			count++
		}
	}
	return count
}

// MustHaveTaskID fails when tasks omit id.
func MustHaveTaskID(t *testing.T, tasks []Task, id string) {
	t.Helper()
	for _, task := range tasks {
		if task.ID == id {
			return
		}
	}
	t.Fatalf("plan missing task id %q", id)
}

// AssertNoTaskName fails when tasks contain any named task.
func AssertNoTaskName(t *testing.T, tasks []Task, names ...string) {
	t.Helper()
	banned := make(map[string]bool, len(names))
	for _, name := range names {
		banned[name] = true
	}
	for _, task := range tasks {
		if banned[task.Name] {
			t.Fatalf("plan has task %s id %q, want none", task.Name, task.ID)
		}
	}
}

// AssertIOPath checks one named bind path.
func AssertIOPath(t *testing.T, ios []IO, name, path string) {
	t.Helper()
	for _, value := range ios {
		if value.Name == name {
			if value.Path != path {
				t.Fatalf("%s path = %q, want %q", name, value.Path, path)
			}
			return
		}
	}
	t.Fatalf("missing IO %q in %#v", name, ios)
}

// AssertIOSource checks one named input source.
func AssertIOSource(t *testing.T, ios []IO, name, source string) {
	t.Helper()
	for _, value := range ios {
		if value.Name == name {
			if value.Source != source {
				t.Fatalf("%s source = %q, want %q", name, value.Source, source)
			}
			return
		}
	}
	t.Fatalf("missing IO %q in %#v", name, ios)
}

// AssertTreeIO checks one Tree bind.
func AssertTreeIO(t *testing.T, ios []IO, name, dir string) {
	t.Helper()
	for _, value := range ios {
		if value.Name != name {
			continue
		}
		if value.Kind != "tree" || value.Path != dir || value.Manifest != dir+"/.gobble-tree.json" || len(value.Members) != 0 {
			t.Fatalf("%s tree = %#v, want dir %q", name, value, dir)
		}
		return
	}
	t.Fatalf("missing tree IO %q in %#v", name, ios)
}

// AssertGroupMembers checks one Group bind's ordered members.
func AssertGroupMembers(t *testing.T, ios []IO, name string, want []Member) {
	t.Helper()
	for _, value := range ios {
		if value.Name != name {
			continue
		}
		if value.Path != "" || len(value.Members) != len(want) {
			t.Fatalf("%s group = %#v, want %#v", name, value, want)
		}
		for i, member := range value.Members {
			if member != want[i] {
				t.Fatalf("%s member[%d] = %#v, want %#v", name, i, member, want[i])
			}
		}
		return
	}
	t.Fatalf("missing group IO %q in %#v", name, ios)
}

// AssertUniqueParamNames checks that a task has no duplicate parameter name.
func AssertUniqueParamNames(t *testing.T, params []Param) {
	t.Helper()
	seen := make(map[string]bool, len(params))
	for _, param := range params {
		if seen[param.Name] {
			t.Fatalf("duplicate Param name %q", param.Name)
		}
		seen[param.Name] = true
	}
}

// ContainsAll reports whether got contains each wanted token with multiplicity.
func ContainsAll(got []string, want ...string) bool {
	have := make(map[string]int, len(got))
	for _, value := range got {
		have[value]++
	}
	for _, value := range want {
		if have[value] == 0 {
			return false
		}
		have[value]--
	}
	return true
}

// ContainsSubstring reports whether any command token contains value.
func ContainsSubstring(command []string, value string) bool {
	for _, token := range command {
		if strings.Contains(token, value) {
			return true
		}
	}
	return false
}

// FlagValue returns the token after flag.
func FlagValue(command []string, flag string) (string, bool) {
	for i, token := range command {
		if token == flag && i+1 < len(command) {
			return command[i+1], true
		}
	}
	return "", false
}

// StageFile copies src into workspace at rel.
func StageFile(t *testing.T, workspace, rel, src string) {
	t.Helper()
	dst := filepath.Join(workspace, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dst, err)
	}
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatalf("copy %s: %v", dst, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("Close(%s) error = %v", dst, err)
	}
}
