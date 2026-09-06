package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	distribution "github.com/HahyeonJeon/gobble/distribution/runtime"
	"github.com/HahyeonJeon/gobble/internal/containerenv"
)

func sourceCheckout() (string, error) {
	source := os.Getenv("GOBBLE_SOURCE")
	if source == "" {
		_, file, _, ok := runtime.Caller(0)
		if ok {
			source = filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		}
	}
	if _, err := os.Stat(filepath.Join(source, "go.mod")); err != nil {
		return "", errors.New("set GOBBLE_SOURCE to the exact Gobble checkout used to build this command")
	}
	return filepath.Abs(source)
}

func runInit(req *request, stdout, stderr io.Writer) int {
	source, err := sourceCheckout()
	if err == nil {
		err = createProject(context.Background(), req.pkg, source, map[string]string{
			"pipeline.go": starterPipeline, "README.md": starterReadme,
			"runs/hello/inputs/sequences.fasta": ">sequence-one\nACGTACGT\n>sequence-two\nTGCATGCA\n",
		}, nil)
	}
	if err != nil {
		return writeErr(stderr, invalidRequest("init", err.Error()), 1)
	}
	return writeJSON(stdout, stderr, "init", map[string]any{"op": "init", "project": req.pkg, "next": "cd into the project, then gobble plan . and gobble run . --workspace runs/hello"})
}

// createProject reserves a fresh directory before writing anything. Failed
// scaffolds remain for diagnosis and can never overwrite an existing project.
func createProject(ctx context.Context, path, source string, files map[string]string, stage func(string) error) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return err
	}
	files["go.mod"] = "module gobble.local/pipeline\n\ngo 1.26\n\nrequire github.com/HahyeonJeon/gobble v0.0.0\n\nreplace github.com/HahyeonJeon/gobble => " + strconv.Quote(source) + "\n"
	files["AGENTS.md"] = starterAgentGuide
	files[".gitignore"] = "runs/\n.gobble-runtime.json\n.gobble-cache/\n"
	if image := os.Getenv("GOBBLE_RUNTIME_IMAGE_ID"); image != "" {
		config, err := json.MarshalIndent(containerenv.Current(), "", "  ")
		if err != nil {
			return err
		}
		files[".gobble-runtime.json"] = string(config) + "\n"
		files["compose.yaml"] = distribution.Compose(containerenv.Current().PullReference())
		for _, name := range []string{"README.md", "AGENTS.md"} {
			files[name] = composeGuide(files[name])
		}
	}
	for name, data := range files {
		dest := filepath.Join(path, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, []byte(data), 0o644); err != nil {
			return err
		}
	}
	if stage != nil {
		if err := stage(path); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	commands := [][]string{
		{"go", "mod", "tidy"}, {"git", "init", "--quiet"}, {"git", "add", "--all"},
		{"git", "-c", "user.name=Gobble", "-c", "user.email=gobble@localhost", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Initialize Gobble pipeline"},
	}
	for _, argv := range commands {
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Dir = path
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %w: %s", argv[0], err, out)
		}
	}
	return nil
}

func runDoctor(stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	checks := map[string]string{}
	for _, check := range []struct {
		name string
		args []string
	}{
		{"go", []string{"go", "version"}}, {"git", []string{"git", "--version"}},
		{"docker", []string{"docker", "info", "--format", "{{.ID}}"}},
	} {
		cmd := exec.CommandContext(ctx, check.args[0], check.args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return writeErr(stderr, invalidRequest("doctor", fmt.Sprintf("%s is unavailable: %v: %s", check.name, err, out)), 1)
		}
		checks[check.name] = strings.TrimSpace(string(out))
	}
	if controller := os.Getenv("GOBBLE_CONTROLLER"); controller != "" {
		if err := probeSiblingMount(ctx, controller); err != nil {
			return writeErr(stderr, invalidRequest("doctor", err.Error()), 1)
		}
		checks["workspace"] = "sibling container read and write verified"
	}
	return writeJSON(stdout, stderr, "doctor", map[string]any{"op": "doctor", "checks": checks})
}

func probeSiblingMount(ctx context.Context, controller string) error {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{json .Mounts}}", controller)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("inspect runtime mounts: %w", err)
	}
	var mounts []containerenv.Mount
	if err := json.Unmarshal(out, &mounts); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	source, err := containerenv.MapPath(cwd, mounts)
	if err != nil {
		return fmt.Errorf("runtime project mount: %w", err)
	}
	file, err := os.CreateTemp(".", ".gobble-probe-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer os.Remove(file.Name() + ".out")
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		file.Close()
		return err
	}
	token := hex.EncodeToString(random[:])
	_, err = file.WriteString(token)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	image := os.Getenv("GOBBLE_RUNTIME_IMAGE_ID")
	if image == "" {
		return errors.New("runtime image identity is missing")
	}
	cmd = exec.CommandContext(ctx, "docker", "run", "--platform", "linux/amd64", "--rm", "--network=none", "--user", strconv.Itoa(os.Getuid())+":"+strconv.Itoa(os.Getgid()), "--entrypoint", "sh", "--mount", probeBind(source), image,
		"-c", `test "$(cat "/probe/$1")" = "$2" && printf ok > "/probe/$1.out"`, "sh", filepath.Base(file.Name()), token)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sibling mount probe: %w: %s", err, out)
	}
	data, err := os.ReadFile(file.Name() + ".out")
	if err != nil || !bytes.Equal(data, []byte("ok")) {
		return errors.New("sibling write was not visible in the project")
	}
	return nil
}

const starterPipeline = `package pipeline

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("hello-gobble")
	input := p.AddInput("sequences", gobble.PathSpec{
		Dir: gobble.Dir("inputs"), Base: "sequences", Ext: ".fasta",
	})
	p.AddTask(gobble.TaskSpec{
		Name: "count-sequences",
		Command: []string{"sh", "-c", "awk '/^>/{n++} END {print n+0}' inputs/sequences.fasta > results/sequence-count.txt"},
		Inputs: []gobble.Bind{{Name: "sequences", From: input}},
		Outputs: []gobble.Bind{{Name: "count", Spec: gobble.PathSpec{
			Dir: gobble.Dir("results"), Base: "sequence-count", Ext: ".txt",
		}}},
	})
	return p
}
`

const starterReadme = `# My Gobble pipeline

This small example counts two FASTA sequences. From this directory:

    gobble doctor
    gobble plan .
    gobble run . --workspace runs/hello
    gobble inspect run --workspace runs/hello

The result is in runs/hello/results/sequence-count.txt (expected: 2).
For longer analyses, open another terminal in this directory:

    gobble watch --workspace runs/hello
    gobble stop --workspace runs/hello
    gobble resume . --workspace runs/hello

The example uses sh/awk provided by the execution environment. Ask your coding agent to adapt pipeline.go. Real analysis tools
should use explicit Docker images. Keep the runtime lock and pinned image for
existing runs. State and data remain under runs; removing a controller does not
remove them. This is foreground execution: closing the run terminal requires
recovery through Resume. Stop ends a task; Resume retries unfinished tasks.
`

const starterAgentGuide = `# Working on this pipeline

Use the installed gobble command from this directory. Go and Gobble are provided
by the selected Docker runtime, or by a direct Linux installation.
First ask for the analysis goal, samples, reference organism/build, available
resources, and expected outputs. Explain scientific choices before changing
them. Never guess the reference build or mix incompatible reference resources.

Edit pipeline.go, then validate and plan before executing. Use explicit tool
images, declared inputs/outputs, and resource budgets. Show the user the plan.
Start with the included tiny input before a large analysis. Use inspect for
machine-readable progress and errors; watch is interactive. Use stop to end an
active run and resume to reconcile and retry unfinished work. Never delete run
state to bypass an error, and never start another owner to bypass a run lock.
Do not edit or remove .gobble-runtime.json to upgrade an existing run.
`

func probeBind(source string) string {
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	_ = w.Write([]string{"type=bind", "src=" + source, "dst=/probe"})
	w.Flush()
	return strings.TrimSuffix(out.String(), "\n")
}

func composeGuide(text string) string {
	text = regexp.MustCompile(`(?m)^    gobble `).ReplaceAllString(text, "    docker compose run --rm gobble ")
	text = strings.ReplaceAll(text, "docker compose run --rm gobble run ", "docker compose run -d gobble run ")
	text = strings.ReplaceAll(text, "docker compose run --rm gobble resume ", "docker compose run -d gobble resume ")
	text = strings.ReplaceAll(text, "Use the installed gobble command from this directory.", "Use this project's pinned compose.yaml from this directory.")
	text = strings.ReplaceAll(text, "Go and Gobble are provided\nby the selected Docker runtime, or by a direct Linux installation.", "Go and Gobble are provided by the Docker runtime; no host Go or Gobble installation is needed.")
	text = strings.ReplaceAll(text, "Keep the run terminal open.", "Detached execution survives the calling terminal. Keep the returned container ID for docker logs and docker wait; a successful launch does not mean the analysis succeeded.")
	text = strings.ReplaceAll(text, "This is foreground execution: closing the run terminal requires\nrecovery through Resume.", "Detached execution survives closing the terminal. Use inspect to verify completion; use docker logs with the returned container ID if startup fails.")
	return text
}
