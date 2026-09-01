package wgsscenario

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	executor "github.com/HahyeonJeon/gobble/internal/engine/exec"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// Runtime executes the pipeline-owned WGS graph through the public lifecycle.
// Its Docker boundary creates declared outputs deterministically, so the WGS
// recovery scenario needs no network, fixture download, image pull, or Docker.
type Runtime struct {
	t           *testing.T
	graph       *gobble.Graph
	workspace   string
	identity    gobble.Identity
	docker      *fakeDocker
	started     chan struct{}
	startedOnce sync.Once
}

// NewRuntime composes config over the pipeline-owned WGS fixture, stages
// ordinary workspace inputs, and installs a deterministic Docker boundary.
func NewRuntime(t *testing.T, config wgs.Config) *Runtime {
	t.Helper()
	samples, _ := Samples(t)
	pipe := wgs.Build(samples, config)
	graph, err := gobble.Compose(pipe)
	if err != nil {
		t.Fatalf("Compose WGS runtime graph: %v", err)
	}
	raw := pc.MustPlanJSON(t, pipe)
	r := &Runtime{t: t, graph: graph, workspace: t.TempDir(), started: make(chan struct{})}
	r.identity, err = gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario")
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

// Run invokes the public WGS Run lifecycle.
func (r *Runtime) Run(ctx context.Context) error {
	return gobble.Run(ctx, r.graph, r.workspace, 8, gobble.WithIdentity(r.identity))
}

// Resume invokes the public WGS Resume lifecycle with the current graph.
func (r *Runtime) Resume(ctx context.Context) error {
	return gobble.Resume(ctx, r.graph, r.workspace, 8, gobble.WithIdentity(r.identity))
}

// FailCommand makes id return a contained nonzero command result with logs.
func (r *Runtime) FailCommand(id string) { r.docker.setMode(id, fakeFailCommand) }

// Block makes id remain active until lifecycle cancellation reaches Docker kill.
func (r *Runtime) Block(id string) { r.docker.setMode(id, fakeBlock) }

// Succeed clears a prior command-double fault for recovery evidence.
func (r *Runtime) Succeed(id string) { r.docker.setMode(id, fakeSuccess) }

// Started closes after a blocked command has started.
func (r *Runtime) Started() <-chan struct{} { return r.started }

// Workspace returns the caller-owned runtime workspace.
func (r *Runtime) Workspace() string { return r.workspace }

// ResumeWith composes changed typed samples and config, then resumes that
// graph in the existing workspace. Newly named fixture inputs are staged.
func (r *Runtime) ResumeWith(ctx context.Context, samples []wgs.Sample, config wgs.Config) error {
	r.t.Helper()
	pipe := wgs.Build(samples, config)
	graph, err := gobble.Compose(pipe)
	if err != nil {
		return fmt.Errorf("compose changed WGS runtime graph: %w", err)
	}
	raw := pc.MustPlanJSON(r.t, pipe)
	stageRuntimeInputs(r.t, r.workspace, samples, config)
	r.graph = graph
	r.docker.setTasks(pc.AllTasks(r.t, raw))
	return gobble.Resume(ctx, r.graph, r.workspace, 8, gobble.WithIdentity(r.identity))
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

func stageRuntimeInputs(t *testing.T, workspace string, samples []wgs.Sample, config wgs.Config) {
	t.Helper()
	paths := make([]string, 0, 3+2*len(config.Reference.KnownSites)+len(config.Reference.Intervals)+2*len(samples))
	for _, spec := range []gobble.PathSpec{config.Reference.FASTA, config.Reference.FAI, config.Reference.Dictionary} {
		paths = append(paths, renderRuntimePath(t, spec))
	}
	for _, site := range config.Reference.KnownSites {
		paths = append(paths, renderRuntimePath(t, site.VCF), renderRuntimePath(t, site.Index))
	}
	for _, interval := range config.Reference.Intervals {
		paths = append(paths, renderRuntimePath(t, interval.Spec))
	}
	for _, member := range config.Reference.BWAIndex.Members {
		paths = append(paths, renderRuntimePath(t, member.Spec))
	}
	for _, sample := range samples {
		for _, lane := range sample.Lanes {
			paths = append(paths, lane.Fastq1, lane.Fastq2)
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
		if err := os.WriteFile(path, []byte(rel+"\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}
}

func renderRuntimePath(t *testing.T, spec gobble.PathSpec) string {
	t.Helper()
	path, err := spec.Render()
	if err != nil {
		t.Fatalf("render WGS runtime path: %v", err)
	}
	return path
}

type fakeDocker struct {
	mu         sync.Mutex
	runtime    *Runtime
	tasks      map[string][]pc.Task
	modes      map[string]fakeMode
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
	fakeFailCommand
	fakeBlock
)

func newFakeDocker(tasks []pc.Task, runtime *Runtime) *fakeDocker {
	fake := &fakeDocker{
		runtime:    runtime,
		modes:      make(map[string]fakeMode),
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
				container.log = "simulated WGS cancellation\n"
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
		_, _ = fmt.Fprintf(stderr, "WGS fake Docker matched %d tasks: %#v", len(candidates), args)
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
	task := candidates[0]
	f.next++
	id := "wgs-fake-" + strconv.Itoa(f.next)
	container := fakeContainer{mount: mount, image: task.Image}
	switch f.modes[task.ID] {
	case fakeFailCommand:
		container.exit = 42
		container.log = "simulated WGS command failure for " + task.ID + "\n"
	case fakeBlock:
		container.running = true
		f.runtime.startedOnce.Do(func() { close(f.runtime.started) })
	default:
		if err := writeFakeOutputs(mount, task); err != nil {
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

func writeFakeOutputs(mount string, task pc.Task) error {
	member := scatterMember{}
	if task.Scatter != "" {
		var err error
		member, err = findScatterMember(mount)
		if err != nil {
			return fmt.Errorf("%s: %w", task.ID, err)
		}
	}
	for _, output := range task.Outputs {
		switch output.Kind {
		case "tree":
			path := output.Path
			if member.stem != "" {
				path = filepath.ToSlash(filepath.Join(path, member.stem))
			}
			dir := filepath.Join(mount, filepath.FromSlash(path))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "fixture.txt"), []byte(task.ID+"\n"), 0o644); err != nil {
				return err
			}
		case "group":
			for _, member := range output.Members {
				if err := writeFakeFile(mount, member.Path, task.ID); err != nil {
					return err
				}
			}
		default:
			path := output.Path
			if member.stem != "" && !output.Spec.Literal {
				var err error
				path, err = scatterOutputPath(mount, output.Spec, member)
				if err != nil {
					return err
				}
			}
			if path == "" {
				return fmt.Errorf("%s output %s has no path", task.ID, output.Name)
			}
			if err := writeFakeFile(mount, path, task.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

type scatterMember struct {
	stem string
	path string
}

func findScatterMember(mount string) (scatterMember, error) {
	var members []scatterMember
	err := filepath.WalkDir(mount, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".bed") {
			rel, relErr := filepath.Rel(mount, path)
			if relErr != nil {
				return relErr
			}
			members = append(members, scatterMember{
				stem: strings.TrimSuffix(entry.Name(), ".bed"),
				path: filepath.ToSlash(rel),
			})
		}
		return nil
	})
	if err != nil {
		return scatterMember{}, err
	}
	if len(members) != 1 {
		return scatterMember{}, fmt.Errorf("scatter isolate has %d BED members, want 1", len(members))
	}
	return members[0], nil
}

func scatterOutputPath(mount string, planSpec pc.Spec, member scatterMember) (string, error) {
	// The engine creates the declared output parent before invoking Docker.
	// Prefer that observable path when per-member specialization retains the
	// staged member path; otherwise use the ordinary stem-derived path.
	appended := filepath.ToSlash(filepath.Join(planSpec.Dir, member.path+"."+strings.TrimPrefix(planSpec.Ext, ".")))
	if info, err := os.Stat(filepath.Join(mount, filepath.FromSlash(filepath.Dir(appended)))); err == nil && info.IsDir() {
		return appended, nil
	}
	spec := gobble.PathSpec{
		Dir:      gobble.Dir(planSpec.Dir),
		Prefix:   planSpec.Prefix,
		Base:     member.stem,
		Suffixes: append([]string(nil), planSpec.Suffixes...),
		Ext:      planSpec.Ext,
	}
	return spec.Render()
}

func writeFakeFile(mount, rel, content string) error {
	path := filepath.Join(mount, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content+"\n"), 0o644)
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
