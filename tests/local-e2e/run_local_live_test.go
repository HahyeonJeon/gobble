//go:build live

package local_e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestRunLocalFixture(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	copyRunLocalInput(t, dir)
	g := mustCompose(runLocalFixturePipeline)(t)
	if err := gobble.Run(t.Context(), g, dir, 2); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	gotImage, err := os.ReadFile(filepath.Join(dir, "out", "docker", "sample.txt"))
	if err != nil {
		t.Fatalf("published docker output: %v", err)
	}
	if string(gotImage) != "fixture\n" && string(gotImage) != "fixture" {
		t.Fatalf("published docker output got %q, want fixture", gotImage)
	}
	gotHost, err := os.ReadFile(filepath.Join(dir, "out", "process", "sample.txt"))
	if err != nil {
		t.Fatalf("published process output: %v", err)
	}
	if string(gotHost) != "fixture\n" && string(gotHost) != "fixture" {
		t.Fatalf("published process output got %q, want fixture", gotHost)
	}
	pwd, err := os.ReadFile(filepath.Join(dir, "out", "docker", "pwd.txt"))
	if err != nil {
		t.Fatalf("published container cwd: %v", err)
	}
	if strings.TrimSpace(string(pwd)) != "/work" {
		t.Fatalf("container cwd got %q, want /work", pwd)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	var file struct {
		Tasks []struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Executor  string `json:"executor"`
			Image     string `json:"image"`
			Resources struct {
				CPU    float64 `json:"cpu"`
				Memory string  `json:"memory"`
			} `json:"resources"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	byID := map[string]struct {
		Status   string
		Executor string
		Image    string
		CPU      float64
		Memory   string
	}{}
	for _, st := range file.Tasks {
		byID[st.ID] = struct {
			Status   string
			Executor string
			Image    string
			CPU      float64
			Memory   string
		}{st.Status, st.Executor, st.Image, st.Resources.CPU, st.Resources.Memory}
	}
	image := byID["image"]
	if image.Status != engine.StatusSucceeded || image.Executor != "docker" || image.Image != runLocalImage {
		t.Fatalf("image task got %#v", image)
	}
	if image.CPU != 1 || image.Memory != "256m" {
		t.Fatalf("image resources got cpu %v memory %q", image.CPU, image.Memory)
	}
	host := byID["host"]
	if host.Status != engine.StatusSucceeded || host.Executor != "process" || host.Image != "" {
		t.Fatalf("host task got %#v", host)
	}
	recoverAfterSuccessAPI(t, g, dir, 2)
}

func TestRunLocalBadImage(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	copyRunLocalInput(t, dir)
	err := gobble.Run(t.Context(), mustCompose(runLocalBadImagePipeline)(t), dir, 2)
	ge := requireRunError(t, "bad image", err, gobble.DefectFailed, "image")
	if ge == nil || strings.Contains(strings.ToLower(ge.Error()), "skip") {
		t.Fatalf("bad image error = %v, want named failure not skip", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "out", "docker", "sample.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("failed docker output was published")
	}
	got, readErr := os.ReadFile(filepath.Join(dir, "out", "process", "sample.txt"))
	if readErr != nil {
		t.Fatalf("independent process output: %v", readErr)
	}
	if string(got) != "fixture\n" && string(got) != "fixture" {
		t.Fatalf("independent process output got %q, want fixture", got)
	}
	if _, statErr := os.Stat(filepath.Join(dir, engine.ControlDir, "tasks", "image", "_", "0", "1", "work")); statErr != nil {
		t.Fatalf("failed work directory: %v", statErr)
	}
	raw := mustJSONFile(t, filepath.Join(dir, engine.ControlDir, engine.TasksFile))
	var file struct {
		Tasks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	byID := map[string]string{}
	for _, st := range file.Tasks {
		byID[st.ID] = st.Status
		if st.ID == "image" && (st.Error == nil || st.Error.Message == "" || !strings.Contains(st.Error.Message, "docker")) {
			t.Fatalf("image error got %#v, want named docker failure", st.Error)
		}
	}
	if byID["image"] != engine.StatusFailed {
		t.Fatalf("image status got %q, want failed", byID["image"])
	}
	if byID["host"] != engine.StatusSucceeded {
		t.Fatalf("host status got %q, want succeeded", byID["host"])
	}
}

func runLocalFixturePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("run-local")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "image",
		Image:   runLocalImage,
		Command: []string{"sh", "-c", "pwd > out/docker/pwd.txt && cp in/sample.txt out/docker/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{
			{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "sample", Ext: ".txt"}},
			{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "pwd", Ext: ".txt"}},
		},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Resources: gobble.Resources{CPU: 1, Memory: "256m"},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "host",
		Command: []string{"cp", "in/sample.txt", "out/process/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out/process"), Base: "sample", Ext: ".txt"},
		}},
	})
	return p
}

func runLocalBadImagePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("run-local-bad")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "image",
		Image:   "gobble-missing-image:not-a-tag",
		Command: []string{"cp", "in/sample.txt", "out/docker/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "sample", Ext: ".txt"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "host",
		Command: []string{"cp", "in/sample.txt", "out/process/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out/process"), Base: "sample", Ext: ".txt"},
		}},
	})
	return p
}
