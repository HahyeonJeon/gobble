package methylseqscenario

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	executor "github.com/HahyeonJeon/gobble/internal/engine/exec"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// Runtime executes the real composed Methyl graph through the public
// lifecycle. Its Docker boundary creates declared outputs deterministically,
// so scenario tests need no network, fixture download, or container runtime.
type Runtime struct {
	t           *testing.T
	graph       *gobble.Graph
	workspace   string
	identity    gobble.Identity
	docker      *fakeDocker
	started     chan struct{}
	startedOnce sync.Once
}

// NewRuntime composes config over the official Methyl fixture and installs a
// deterministic Docker boundary.
func NewRuntime(t *testing.T, config methylseq.Config) *Runtime {
	t.Helper()
	samples, _ := Samples(t)
	pipe := methylseq.Build(samples, config)
	graph, err := gobble.Compose(pipe)
	if err != nil {
		t.Fatalf("Compose Methyl runtime graph: %v", err)
	}
	raw := pc.MustPlanJSON(t, pipe)
	r := &Runtime{t: t, graph: graph, workspace: t.TempDir(), started: make(chan struct{})}
	r.identity, err = gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario")
	if err != nil {
		t.Fatalf("IdentityFromBuildInfo: %v", err)
	}
	r.docker = newFakeDocker(pc.AllTasks(t, raw), r)
	previous := executor.DockerCLI
	executor.DockerCLI = r.docker.call
	t.Cleanup(func() { executor.DockerCLI = previous })
	stageRuntimeInputs(t, r.workspace, samples, config)
	return r
}

// Workspace returns the caller-owned runtime workspace.
func (r *Runtime) Workspace() string { return r.workspace }

// Fail makes id exit with a command failure.
func (r *Runtime) Fail(id string) { r.docker.setMode(id, fakeFail) }

// Block makes id remain active until context cancellation reaches Docker kill.
func (r *Runtime) Block(id string) { r.docker.setMode(id, fakeBlock) }

// Succeed clears a prior fault for recovery evidence.
func (r *Runtime) Succeed(id string) { r.docker.setMode(id, fakeSuccess) }

// Started closes when a blocked task has started.
func (r *Runtime) Started() <-chan struct{} { return r.started }

// Run invokes the public Methyl Run lifecycle.
func (r *Runtime) Run(ctx context.Context) error {
	return gobble.Run(ctx, r.graph, r.workspace, 8, gobble.WithIdentity(r.identity))
}

// Resume invokes the public Methyl Resume lifecycle.
func (r *Runtime) Resume(ctx context.Context) error {
	return gobble.Resume(ctx, r.graph, r.workspace, 8, gobble.WithIdentity(r.identity))
}

// ResumeWith composes changed typed samples and config, then resumes that
// graph in the existing workspace. It stages only newly named fixture inputs.
func (r *Runtime) ResumeWith(ctx context.Context, samples []methylseq.Sample, config methylseq.Config) error {
	r.t.Helper()
	pipe := methylseq.Build(samples, config)
	graph, err := gobble.Compose(pipe)
	if err != nil {
		return fmt.Errorf("compose changed Methyl runtime graph: %w", err)
	}
	raw := pc.MustPlanJSON(r.t, pipe)
	stageRuntimeInputs(r.t, r.workspace, samples, config)
	r.graph = graph
	r.docker.setTasks(pc.AllTasks(r.t, raw))
	return r.Resume(ctx)
}

// Release invokes the public recovery release.
func (r *Runtime) Release() error {
	return gobble.Release(r.workspace, gobble.WithIdentity(r.identity))
}

// InspectObject decodes one object-shaped Inspect view.
func (r *Runtime) InspectObject(view gobble.View) map[string]any {
	r.t.Helper()
	raw, err := gobble.Inspect(r.workspace, view, "", gobble.WithIdentity(r.identity))
	if err != nil {
		r.t.Fatalf("Inspect(%s): %v", view, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		r.t.Fatalf("Inspect(%s) JSON: %v\n%s", view, err, raw)
	}
	return out
}

// InspectRecords decodes one JSONL Inspect view.
func (r *Runtime) InspectRecords(view gobble.View) []map[string]any {
	r.t.Helper()
	raw, err := gobble.Inspect(r.workspace, view, "", gobble.WithIdentity(r.identity))
	if err != nil {
		r.t.Fatalf("Inspect(%s): %v", view, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	var out []map[string]any
	for decoder.More() {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			r.t.Fatalf("Inspect(%s) JSONL: %v\n%s", view, err, raw)
		}
		out = append(out, record)
	}
	return out
}

func stageRuntimeInputs(t *testing.T, workspace string, samples []methylseq.Sample, config methylseq.Config) {
	t.Helper()
	var paths []string
	if config.Reference.BismarkIndex.IsZero() {
		path, err := config.Reference.FASTA.Render()
		if err != nil {
			t.Fatalf("render Methyl reference: %v", err)
		}
		paths = append(paths, path)
	} else {
		t.Fatal("hermetic lifecycle runtime expects generated-index defaults")
	}
	for _, sample := range samples {
		for _, run := range sample.Runs {
			paths = append(paths, run.Fastq1)
			if run.Fastq2 != "" {
				paths = append(paths, run.Fastq2)
			}
		}
	}
	for _, rel := range paths {
		path := filepath.Join(workspace, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			t.Fatalf("Stat(%s): %v", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("fixture\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}
}

type fakeMode int

const (
	fakeSuccess fakeMode = iota
	fakeFail
	fakeBlock
)

type fakeDocker struct {
	mu         sync.Mutex
	runtime    *Runtime
	tasks      map[string]pc.Task
	modes      map[string]fakeMode
	containers map[string]*fakeContainer
	next       int
}

type fakeContainer struct {
	mount   string
	image   string
	running bool
	exit    int
}

func newFakeDocker(tasks []pc.Task, runtime *Runtime) *fakeDocker {
	fake := &fakeDocker{runtime: runtime, modes: make(map[string]fakeMode), containers: make(map[string]*fakeContainer)}
	fake.setTasks(tasks)
	return fake
}

func (f *fakeDocker) setTasks(tasks []pc.Task) {
	indexed := make(map[string]pc.Task, len(tasks))
	for _, task := range tasks {
		indexed[fakeTaskKey(task.Image, taskArgv(task))] = task
	}
	f.mu.Lock()
	f.tasks = indexed
	f.mu.Unlock()
}

func (f *fakeDocker) setMode(id string, mode fakeMode) {
	f.mu.Lock()
	f.modes[id] = mode
	f.mu.Unlock()
}

func (f *fakeDocker) call(ctx context.Context, args, _ []string, stdout, stderr io.Writer) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return -1, err
	}
	if len(args) == 0 {
		return 1, nil
	}
	switch args[0] {
	case "image":
		if strings.Contains(strings.Join(args, " "), "{{.Id}}") {
			_, digest, _ := strings.Cut(args[len(args)-1], "@")
			_, _ = io.WriteString(stdout, digest+"\n")
		}
		return 0, nil
	case "run":
		return f.run(args, stdout, stderr)
	case "inspect":
		return f.inspect(args, stdout)
	case "logs":
		if len(args) > 1 {
			if container := f.containers[args[1]]; container != nil && container.exit != 0 {
				_, _ = io.WriteString(stderr, "simulated Methyl command failure\n")
			}
		}
		return 0, nil
	case "rm":
		return 0, nil
	case "kill":
		if len(args) > 1 {
			if container := f.containers[args[1]]; container != nil {
				container.running = false
				container.exit = 137
			}
		}
		return 0, nil
	default:
		_, _ = io.WriteString(stderr, "unsupported fake docker call")
		return 1, nil
	}
}

func (f *fakeDocker) run(args []string, stdout, stderr io.Writer) (int, error) {
	var task pc.Task
	found := false
	entrypoint := ""
	for i := 1; i+1 < len(args); i++ {
		if args[i] == "--entrypoint" {
			entrypoint = args[i+1]
			break
		}
	}
	for i := 1; i < len(args); i++ {
		argv := append([]string{entrypoint}, args[i+1:]...)
		candidate, ok := f.tasks[fakeTaskKey(args[i], argv)]
		if ok {
			task, found = candidate, true
			break
		}
	}
	if !found {
		_, _ = fmt.Fprintf(stderr, "unmatched Methyl task: %#v", args)
		return 1, nil
	}
	mount := ""
	for i := 1; i+1 < len(args); i++ {
		if args[i] == "-v" {
			mount, _, _ = strings.Cut(args[i+1], ":")
			break
		}
	}
	if mount == "" {
		_, _ = io.WriteString(stderr, "missing work mount")
		return 1, nil
	}
	f.next++
	id := "methyl-fake-" + strconv.Itoa(f.next)
	mode := f.modes[task.ID]
	container := &fakeContainer{mount: mount, image: task.Image}
	switch mode {
	case fakeBlock:
		container.running = true
		f.runtime.startedOnce.Do(func() { close(f.runtime.started) })
	case fakeFail:
		container.exit = 42
	default:
		if err := writeFakeOutputs(mount, task.Outputs); err != nil {
			_, _ = io.WriteString(stderr, err.Error())
			return 1, nil
		}
	}
	f.containers[id] = container
	_, _ = io.WriteString(stdout, id+"\n")
	return 0, nil
}

func (f *fakeDocker) inspect(args []string, stdout io.Writer) (int, error) {
	if len(args) < 2 {
		return 1, nil
	}
	id := args[len(args)-1]
	container := f.containers[id]
	if container == nil {
		return 1, nil
	}
	format := strings.Join(args, " ")
	switch {
	case strings.Contains(format, ".State.Running"):
		_, _ = fmt.Fprintf(stdout, "%t %d\n", container.running, container.exit)
	case strings.Contains(format, ".Mounts"):
		_, _ = io.WriteString(stdout, container.mount+"\n")
	case strings.Contains(format, ".Image"):
		_, digest, _ := strings.Cut(container.image, "@")
		_, _ = io.WriteString(stdout, digest+"\n")
	default:
		return 1, nil
	}
	return 0, nil
}

func writeFakeOutputs(mount string, outputs []pc.IO) error {
	for _, output := range outputs {
		switch output.Kind {
		case "tree":
			dir := filepath.Join(mount, filepath.FromSlash(output.Path))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "fixture.txt"), []byte("fixture\n"), 0o644); err != nil {
				return err
			}
		case "group":
			for _, member := range output.Members {
				if err := writeFakeFile(mount, member.Path); err != nil {
					return err
				}
			}
		default:
			if err := writeFakeFile(mount, output.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFakeFile(mount, rel string) error {
	path := filepath.Join(mount, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("fixture\n"), 0o644)
}

func fakeTaskKey(image string, argv []string) string {
	return image + "\x00" + strings.Join(argv, "\x00")
}

func taskArgv(task pc.Task) []string {
	if task.Script != "" {
		return []string{"sh", "-c", "set -eu\n" + task.Script}
	}
	return task.Command
}
