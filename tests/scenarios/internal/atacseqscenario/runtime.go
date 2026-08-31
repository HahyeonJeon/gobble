package atacseqscenario

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	executor "github.com/HahyeonJeon/gobble/internal/engine/exec"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// Runtime executes the pipeline-owned ATAC graph through the public lifecycle.
// Its Docker boundary creates declared outputs deterministically, so recovery
// evidence needs no network, fixture download, image pull, or Docker daemon.
type Runtime struct {
	t           *testing.T
	graph       *gobble.Graph
	workspace   string
	identity    gobble.Identity
	docker      *fakeDocker
	started     chan struct{}
	startedOnce sync.Once
}

// ConsumedInput is one regular input byte opened by a command double.
type ConsumedInput struct {
	Path   string
	SHA256 string
}

// NewRuntime composes config over the pipeline-owned fixture, stages ordinary
// workspace inputs, and installs a deterministic Docker boundary.
func NewRuntime(t *testing.T, config atacseq.Config) *Runtime {
	t.Helper()
	return newRuntime(t, config, t.TempDir(), true)
}

// NewRuntimeWithWorkspaceInputs composes the ATAC graph over inputs already
// staged in workspace. It is used by live fixture evidence after exact fetch.
func NewRuntimeWithWorkspaceInputs(t *testing.T, config atacseq.Config, workspace string) *Runtime {
	t.Helper()
	return newRuntime(t, config, workspace, false)
}

func newRuntime(t *testing.T, config atacseq.Config, workspace string, stage bool) *Runtime {
	t.Helper()
	samples, _ := Samples(t)
	pipe := atacseq.Build(samples, config)
	graph, err := gobble.Compose(pipe)
	if err != nil {
		t.Fatalf("Compose ATAC runtime graph: %v", err)
	}
	r := &Runtime{t: t, graph: graph, workspace: workspace, started: make(chan struct{})}
	r.identity, err = gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario")
	if err != nil {
		t.Fatalf("IdentityFromBuildInfo: %v", err)
	}
	r.docker = newFakeDocker(pc.AllTasks(t, pc.MustPlanJSON(t, pipe)), r)
	previous := executor.DockerCLI
	executor.DockerCLI = r.docker.call
	t.Cleanup(func() { executor.DockerCLI = previous })
	if stage {
		stageInputs(t, r.workspace, samples, config)
	}
	return r
}

// Run invokes the public ATAC Run lifecycle.
func (r *Runtime) Run(ctx context.Context) error {
	return gobble.Run(ctx, r.graph, r.workspace, 16, gobble.WithIdentity(r.identity))
}

// Resume invokes the public ATAC Resume lifecycle with the current graph.
func (r *Runtime) Resume(ctx context.Context) error {
	return gobble.Resume(ctx, r.graph, r.workspace, 16, gobble.WithIdentity(r.identity))
}

// FailInput makes id reject its staged inputs as a contained command failure.
func (r *Runtime) FailInput(id string) { r.docker.setMode(id, fakeFailInput) }

// Block makes id remain active until Run cancellation reaches Docker kill.
func (r *Runtime) Block(id string) { r.docker.setMode(id, fakeBlock) }

// Succeed clears a prior command-double fault for recovery evidence.
func (r *Runtime) Succeed(id string) { r.docker.setMode(id, fakeSuccess) }

// Started closes after a blocked command has consumed its inputs and started.
func (r *Runtime) Started() <-chan struct{} { return r.started }

// ResumeWith composes changed typed values and resumes in the same workspace.
func (r *Runtime) ResumeWith(ctx context.Context, samples []atacseq.Sample, config atacseq.Config) error {
	pipe := atacseq.Build(samples, config)
	graph, err := gobble.Compose(pipe)
	if err != nil {
		return fmt.Errorf("compose changed ATAC graph: %w", err)
	}
	stageInputs(r.t, r.workspace, samples, config)
	r.graph = graph
	r.docker.setTasks(pc.AllTasks(r.t, pc.MustPlanJSON(r.t, pipe)))
	return gobble.Resume(ctx, graph, r.workspace, 16, gobble.WithIdentity(r.identity))
}

// Release invokes the public recovery release.
func (r *Runtime) Release() error {
	return gobble.Release(r.workspace, gobble.WithIdentity(r.identity))
}

// Workspace returns the caller-created test workspace.
func (r *Runtime) Workspace() string { return r.workspace }

// ConsumedInputs returns a copy of the regular input identities opened by each
// selected command double.
func (r *Runtime) ConsumedInputs() map[string][]ConsumedInput {
	return r.docker.consumedInputs()
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
			r.t.Fatalf("Inspect(%s) JSONL: %v", view, err)
		}
		out = append(out, record)
	}
	return out
}

func stageInputs(t *testing.T, workspace string, samples []atacseq.Sample, config atacseq.Config) {
	t.Helper()
	paths := []string{render(t, config.Reference.FASTA), render(t, config.Reference.Annotation)}
	if config.Filters.RemoveBlacklist {
		paths = append(paths, render(t, config.Reference.Blacklist))
	}
	for _, member := range config.Reference.BWAIndex.Members {
		paths = append(paths, render(t, member.Spec))
	}
	for _, sample := range samples {
		for _, replicate := range sample.Replicates {
			for _, run := range replicate.Runs {
				paths = append(paths, run.Fastq1)
				if run.Fastq2 != "" {
					paths = append(paths, run.Fastq2)
				}
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
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(rel+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func render(t *testing.T, spec gobble.PathSpec) string {
	t.Helper()
	path, err := spec.Render()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

type fakeDocker struct {
	mu         sync.Mutex
	runtime    *Runtime
	tasks      map[string][]pc.Task
	modes      map[string]fakeMode
	consumed   map[string][]ConsumedInput
	containers map[string]fakeContainer
	next       int
}

type fakeContainer struct {
	mount   string
	image   string
	running bool
	exit    int
	log     string
}

type fakeMode int

const (
	fakeSuccess fakeMode = iota
	fakeFailInput
	fakeBlock
)

func newFakeDocker(tasks []pc.Task, runtime *Runtime) *fakeDocker {
	fake := &fakeDocker{
		runtime:    runtime,
		modes:      make(map[string]fakeMode),
		consumed:   make(map[string][]ConsumedInput),
		containers: make(map[string]fakeContainer),
	}
	fake.setTasks(tasks)
	return fake
}

func (f *fakeDocker) setTasks(tasks []pc.Task) {
	indexed := make(map[string][]pc.Task, len(tasks))
	for _, task := range tasks {
		key := fakeTaskKey(task.Image, taskArgv(task))
		indexed[key] = append(indexed[key], task)
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

func (f *fakeDocker) consumedInputs() map[string][]ConsumedInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string][]ConsumedInput, len(f.consumed))
	for id, inputs := range f.consumed {
		out[id] = append([]ConsumedInput(nil), inputs...)
	}
	return out
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
			if container, ok := f.containers[args[1]]; ok && container.log != "" {
				_, _ = io.WriteString(stderr, container.log)
			}
		}
		return 0, nil
	case "rm":
		return 0, nil
	case "kill":
		if len(args) > 1 {
			if container, ok := f.containers[args[1]]; ok {
				container.running = false
				container.exit = 137
				container.log = "simulated ATAC cancellation\n"
				f.containers[args[1]] = container
			}
		}
		return 0, nil
	default:
		_, _ = io.WriteString(stderr, "unsupported fake Docker call")
		return 1, nil
	}
}

func (f *fakeDocker) run(args []string, stdout, stderr io.Writer) (int, error) {
	entrypoint := ""
	for i := 1; i+1 < len(args); i++ {
		if args[i] == "--entrypoint" {
			entrypoint = args[i+1]
			break
		}
	}
	var candidates []pc.Task
	for i := 1; i < len(args); i++ {
		argv := append([]string{entrypoint}, args[i+1:]...)
		if tasks := f.tasks[fakeTaskKey(args[i], argv)]; len(tasks) > 0 {
			candidates = tasks
			break
		}
	}
	if len(candidates) != 1 {
		_, _ = fmt.Fprintf(stderr, "ATAC fake Docker matched %d tasks", len(candidates))
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
		return 1, nil
	}
	task := candidates[0]
	consumed, err := consumeInputs(mount, task.Inputs)
	if err != nil {
		_, _ = io.WriteString(stderr, err.Error())
		return 1, nil
	}
	f.consumed[task.ID] = append([]ConsumedInput(nil), consumed...)
	f.next++
	id := "atac-fake-" + strconv.Itoa(f.next)
	container := fakeContainer{mount: mount, image: task.Image}
	switch f.modes[task.ID] {
	case fakeFailInput:
		container.exit = 42
		container.log = "simulated ATAC input rejection for " + task.ID + "\n"
	case fakeBlock:
		container.running = true
		f.runtime.startedOnce.Do(func() { close(f.runtime.started) })
	default:
		if err := writeOutputs(mount, task, consumed); err != nil {
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
	container, ok := f.containers[args[len(args)-1]]
	if !ok {
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

func consumeInputs(mount string, inputs []pc.IO) ([]ConsumedInput, error) {
	var consumed []ConsumedInput
	for _, input := range inputs {
		switch input.Kind {
		case "group":
			for _, member := range input.Members {
				record, err := consumeFile(mount, member.Path)
				if err != nil {
					return nil, err
				}
				consumed = append(consumed, record)
			}
		case "tree":
			root := filepath.Join(mount, filepath.FromSlash(input.Path))
			err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if !entry.Type().IsRegular() {
					return nil
				}
				rel, err := filepath.Rel(mount, path)
				if err != nil {
					return err
				}
				record, err := consumeFile(mount, filepath.ToSlash(rel))
				if err != nil {
					return err
				}
				consumed = append(consumed, record)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("consume Tree input %s: %w", input.Path, err)
			}
		default:
			record, err := consumeFile(mount, input.Path)
			if err != nil {
				return nil, err
			}
			consumed = append(consumed, record)
		}
	}
	sort.Slice(consumed, func(i, j int) bool { return consumed[i].Path < consumed[j].Path })
	return consumed, nil
}

func consumeFile(mount, rel string) (ConsumedInput, error) {
	if rel == "" {
		return ConsumedInput{}, errors.New("ATAC command input has no path")
	}
	path := filepath.Join(mount, filepath.FromSlash(rel))
	f, err := os.Open(path)
	if err != nil {
		return ConsumedInput{}, fmt.Errorf("open ATAC command input %s: %w", rel, err)
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, f)
	closeErr := f.Close()
	if copyErr != nil {
		return ConsumedInput{}, fmt.Errorf("read ATAC command input %s: %w", rel, copyErr)
	}
	if closeErr != nil {
		return ConsumedInput{}, fmt.Errorf("close ATAC command input %s: %w", rel, closeErr)
	}
	return ConsumedInput{Path: filepath.ToSlash(rel), SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

func writeOutputs(mount string, task pc.Task, consumed []ConsumedInput) error {
	content := outputContent(task.ID, consumed)
	for _, output := range task.Outputs {
		switch output.Kind {
		case "tree":
			dir := filepath.Join(mount, filepath.FromSlash(output.Path))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "fixture.txt"), content, 0o644); err != nil {
				return err
			}
		case "group":
			for _, member := range output.Members {
				if err := writeFile(mount, member.Path, content); err != nil {
					return err
				}
			}
		default:
			if output.Path == "" {
				return fmt.Errorf("%s output %s has no path", task.ID, output.Name)
			}
			if err := writeFile(mount, output.Path, content); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFile(mount, rel string, content []byte) error {
	path := filepath.Join(mount, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func outputContent(taskID string, consumed []ConsumedInput) []byte {
	var out strings.Builder
	out.WriteString(taskID)
	out.WriteByte('\n')
	for _, input := range consumed {
		out.WriteString(input.Path)
		out.WriteByte('=')
		out.WriteString(input.SHA256)
		out.WriteByte('\n')
	}
	return []byte(out.String())
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
