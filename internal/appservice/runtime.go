package appservice

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxRuntimeOutput = 8 << 20

// commandRunner is the OS process boundary; no shell or model-supplied arguments enter it.
type commandRunner func(context.Context, []string, []string) ([]byte, error)
type runtimeAdapter struct{ command commandRunner }

func validBinding(b RuntimeBinding) bool {
	return len(b.Endpoint) <= 4096 && strings.HasPrefix(b.Endpoint, "unix:///") && !strings.ContainsAny(b.Endpoint, "\x00\r\n") && b.DaemonID != "" && len(b.DaemonID) <= 256 && digestPattern.MatchString(b.ImageID) && b.Platform == "linux/amd64" && b.ProjectPath == "/gobble/project" && strings.HasPrefix(b.WorkspacePath, "/gobble/project/") && validRelative(strings.TrimPrefix(b.WorkspacePath, "/gobble/project/"))
}

func runtimeEnvironment(endpoint string) []string {
	out := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key == "DOCKER_CONTEXT" || key == "DOCKER_HOST" || strings.HasPrefix(key, "GOBBLE_") {
			continue
		}
		out = append(out, entry)
	}
	return append(out, "DOCKER_HOST="+endpoint)
}

type boundedOutput struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		b.exceeded = true
		return 0, errors.New("runtime output limit exceeded")
	}
	return b.Buffer.Write(p)
}

// Diagnostics stay private; domain adapters may recognize only safe structured refusals.
type runtimeCommandError struct {
	cause      error
	diagnostic []byte
}

func (e *runtimeCommandError) Error() string { return e.cause.Error() }
func (e *runtimeCommandError) Unwrap() error { return e.cause }

func runDocker(ctx context.Context, args, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = env
	stdout := &boundedOutput{limit: maxRuntimeOutput}
	stderr := &boundedOutput{limit: 16 << 10}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil || stdout.exceeded || stderr.exceeded {
		failure := problem("runtime_unavailable", "The recorded Docker runtime could not complete this query.")
		var exited *exec.ExitError
		if ctx.Err() == nil && !stdout.exceeded && !stderr.exceeded && errors.As(err, &exited) {
			return nil, &runtimeCommandError{cause: failure, diagnostic: append([]byte(nil), stderr.Bytes()...)}
		}
		return nil, failure
	}
	return stdout.Bytes(), nil
}

func (a runtimeAdapter) commandAt(ctx context.Context, endpoint string, args ...string) ([]byte, error) {
	return a.command(ctx, args, runtimeEnvironment(endpoint))
}

func (a runtimeAdapter) selectBinding(ctx context.Context, p projectRecord, workspace string) (RuntimeBinding, error) {
	root, err := openProject(p)
	if err != nil {
		return RuntimeBinding{}, err
	}
	defer root.Close()
	f, err := openResource(root, ".gobble-runtime.json")
	if err != nil {
		return RuntimeBinding{}, problem("incompatible_runtime", "This Project has no readable pinned runtime record.")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return RuntimeBinding{}, problem("incompatible_runtime", "The runtime record is not a regular file.")
	}
	raw, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil || len(raw) > 8192 {
		return RuntimeBinding{}, problem("incompatible_runtime", "The runtime record is invalid.")
	}
	var lock struct {
		Format int    `json:"format"`
		Image  string `json:"image"`
		Daemon string `json:"daemon"`
	}
	if json.Unmarshal(raw, &lock) != nil || lock.Format != 1 || !digestPattern.MatchString(lock.Image) || lock.Daemon == "" {
		return RuntimeBinding{}, problem("incompatible_runtime", "The Project runtime record is unsupported.")
	}
	// Resolve once at attachment, then persist the endpoint and verify its daemon ID on every query.
	endpoint := os.Getenv("DOCKER_HOST")
	if endpoint == "" || os.Getenv("DOCKER_CONTEXT") != "" {
		out, err := a.command(ctx, []string{"context", "inspect", "--format", "{{.Endpoints.docker.Host}}"}, os.Environ())
		if err != nil {
			return RuntimeBinding{}, problem("runtime_unavailable", "Docker is unavailable. Start the recorded local Docker engine.")
		}
		endpoint = strings.TrimSpace(string(out))
	}
	b := RuntimeBinding{endpoint, lock.Daemon, lock.Image, "linux/amd64", "/gobble/project", "/gobble/project/" + workspace}
	if !validBinding(b) {
		return RuntimeBinding{}, problem("incompatible_runtime", "Only a pinned local Linux Docker runtime is supported.")
	}
	if err := a.verify(ctx, b); err != nil {
		return RuntimeBinding{}, err
	}
	return b, nil
}

func (a runtimeAdapter) verify(ctx context.Context, b RuntimeBinding) error {
	if !validBinding(b) {
		return problem("incompatible_runtime", "The recorded runtime binding is invalid.")
	}
	info, err := a.commandAt(ctx, b.Endpoint, "info", "--format", "{{.ID}} {{.OSType}}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(info)) != b.DaemonID+" linux" {
		return problem("incompatible_runtime", "The Docker daemon differs from the registered runtime.")
	}
	image, err := a.commandAt(ctx, b.Endpoint, "image", "inspect", "--format", "{{.Id}} {{.Os}}/{{.Architecture}}", b.ImageID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(image)) != b.ImageID+" linux/amd64" {
		return problem("incompatible_runtime", "Restore the exact registered Linux runtime image.")
	}
	return nil
}

func readOnlyMount(source, target string) string {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"type=bind", "src=" + source, "dst=" + target, "readonly"})
	writer.Flush()
	return strings.TrimSuffix(buffer.String(), "\n")
}

func (a runtimeAdapter) inspect(ctx context.Context, p projectRecord, b RuntimeBinding, view, instance string) (json.RawMessage, error) {
	if view != "identity" && view != "monitor" {
		return nil, problem("unsupported", "Only identity and monitor queries are supported.")
	}
	if !validBinding(b) {
		return nil, problem("incompatible_runtime", "The recorded runtime binding is invalid.")
	}
	name := "gobble-query-" + newID("req")
	args := []string{"run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=64", "--memory=256m", "--cpus=1",
		"--workdir", b.ProjectPath, "--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--mount", readOnlyMount(p.Root, b.ProjectPath), "--entrypoint", "/usr/local/bin/gobble", b.ImageID, "inspect", view, "--workspace", b.WorkspacePath}
	if instance != "" {
		args = append(args, "--instance", instance)
	}
	// A cancelled docker client may leave its query container; this unique name is owned by this call.
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = a.commandAt(cleanup, b.Endpoint, "rm", "--force", name)
	}()
	raw, err := a.commandAt(ctx, b.Endpoint, args...)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxRuntimeOutput || !json.Valid(raw) {
		return nil, problem("incompatible_runtime", "The runtime returned an invalid query response.")
	}
	return raw, nil
}

func workspaceContainerPath(relative string) string {
	return "/gobble/project/" + filepath.ToSlash(relative)
}

func (s *Service) removeRuntimeQuery(b RuntimeBinding, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = s.runtime.commandAt(ctx, b.Endpoint, "rm", "--force", name)
}
