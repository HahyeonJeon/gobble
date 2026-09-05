// Command gobble-container is distributed as "gobble" for Docker installations.
// It deliberately imports no Linux engine code and runs natively on Linux, Windows, and macOS.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const runtimeLock = ".gobble-runtime.json"
const runtimePlatform = "linux/amd64"

type runtimeConfig struct {
	Format int    `json:"format"`
	Image  string `json:"image"`
	Daemon string `json:"daemon"`
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 || (args[0] == "help" && len(args) == 1) || args[0] == "--help" {
		fmt.Println("Gobble · local pipelines with Docker\n\nUsage: gobble <command> [arguments]\n\n  init DIR     create a runnable project for your coding agent\n  demo NAME DIR prepare an existing assay with verified test data\n  doctor       check Docker and the runtime\n  plan         show the project pipeline\n  run          execute the pipeline (--workspace DIR)\n  watch        monitor a run (--workspace DIR)\n  stop         stop a run (--workspace DIR)\n  resume       reconcile and resume work (--workspace DIR)\n\nRun inside your project. Other arguments are passed to the runtime CLI.\nThe first invocation requires GOBBLE_RUNTIME_IMAGE; the exact local image ID\nis then saved per project. Keep that image for Stop and Resume.")
		return 0
	}
	if !supportedHost(runtime.GOOS, runtime.GOARCH) {
		return fail(errors.New("supported launchers: Linux x64, Windows x64, and macOS Intel/Apple Silicon"))
	}
	if err := findDocker(); err != nil {
		return fail(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fail(err)
	}
	root, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	endpoint, err := dockerOutput(ctx, "context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
	if os.Getenv("DOCKER_CONTEXT") == "" && os.Getenv("DOCKER_HOST") != "" {
		endpoint, err = os.Getenv("DOCKER_HOST"), nil
	}
	if err != nil {
		return fail(err)
	}
	if !strings.HasPrefix(endpoint, "unix:///") && !(runtime.GOOS == "windows" && strings.HasPrefix(endpoint, "npipe://")) {
		return fail(errors.New("select a local Docker engine; remote TCP/SSH contexts cannot mount local files"))
	}
	// Bind every subsequent command to the selected endpoint, even if the user's
	// default Docker context changes while the controller is running.
	if err := os.Unsetenv("DOCKER_CONTEXT"); err != nil {
		return fail(err)
	}
	if err := os.Setenv("DOCKER_HOST", endpoint); err != nil {
		return fail(err)
	}
	daemon, err := dockerOutput(ctx, "info", "--format", "{{.ID}}")
	if err != nil {
		return fail(fmt.Errorf("Docker is unavailable; start Docker Desktop or Docker Engine: %w", err))
	}
	if daemon == "" {
		return fail(errors.New("Docker returned an empty daemon identity"))
	}
	osType, err := dockerOutput(ctx, "info", "--format", "{{.OSType}}")
	if err != nil || osType != "linux" {
		return fail(errors.New("switch Docker Desktop to Linux containers"))
	}
	config, err := selectRuntime(ctx, root, daemon)
	if err != nil {
		return fail(err)
	}
	forwarded, mounts, err := translateArgs(root, args)
	if err != nil {
		return fail(err)
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fail(err)
	}
	name := "gobble-controller-" + hex.EncodeToString(random[:])
	host := sha256.Sum256([]byte(daemon))
	socket := controllerSocket(runtime.GOOS, endpoint)
	projectID := sha256.Sum256([]byte(root))
	create := []string{"create", "--platform", runtimePlatform, "--name", name, "--hostname", "gobble-" + hex.EncodeToString(host[:12]),
		"--label", "io.gobble.project=" + hex.EncodeToString(projectID[:]),
		"--label", "io.gobble.command=" + args[0],
		"--init", "--stop-timeout", "40", "--workdir", "/gobble/project",
		"--env", "GOBBLE_CONTROLLER=" + name, "--env", "DOCKER_HOST=unix:///var/run/docker.sock",
		"--env", "GOBBLE_RUNTIME_IMAGE_ID=" + config.Image,
		"--env", "GOBBLE_DAEMON_ID=" + daemon,
		"--env", "HOME=/tmp/gobble-home", "--env", "GOCACHE=/gobble/project/.gobble-cache/build",
		"--mount", bindMount(socket, "/var/run/docker.sock"),
		"--mount", bindMount(root, "/gobble/project")}
	create = append(create, localPermissions(socket)...)
	create = append(create, mounts...)
	if args[0] == "watch" {
		create = append(create, "-it")
	}
	create = append(create, config.Image)
	create = append(create, forwarded...)
	if _, err := dockerOutput(ctx, create...); err != nil {
		return fail(err)
	}
	// Cleanup has its own bound. No analysis containers are removed here;
	// their checkpoint records belong to the runtime recovery protocol.
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = dockerOutput(cleanup, "rm", name)
	}()
	attach := exec.Command("docker", "start", "--attach", "--interactive", name)
	attach.Stdin, attach.Stdout, attach.Stderr = os.Stdin, os.Stdout, os.Stderr
	stopSignals := forwardInterrupt(name)
	defer stopSignals()
	err = attach.Run()
	if err == nil {
		// docker start --attach propagates the container's exit code.
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() >= 0 {
		return exit.ExitCode()
	}
	return fail(err)
}

var dockerOutput = runDockerOutput

func runDockerOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func selectRuntime(ctx context.Context, root, daemon string) (runtimeConfig, error) {
	path := filepath.Join(root, runtimeLock)
	data, err := os.ReadFile(path)
	if err == nil {
		var config runtimeConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return config, err
		}
		if config.Format != 1 || !validImageID(config.Image) || config.Daemon != daemon {
			return config, errors.New("runtime lock does not match this Docker engine; restore the recorded engine and image")
		}
		if _, err := inspectRuntime(ctx, config.Image); err != nil {
			return config, fmt.Errorf("restore the project's pinned runtime image %s: %w", config.Image, err)
		}
		return config, nil
	}
	if !os.IsNotExist(err) {
		return runtimeConfig{}, err
	}
	image := os.Getenv("GOBBLE_RUNTIME_IMAGE")
	if image == "" {
		return runtimeConfig{}, errors.New("set GOBBLE_RUNTIME_IMAGE to the runtime image you built or downloaded; this preview has no published default image")
	}
	id, err := inspectRuntime(ctx, image)
	if err != nil {
		return runtimeConfig{}, fmt.Errorf("prepare runtime image %s with docker pull or docker build first: %w", image, err)
	}
	config := runtimeConfig{Format: 1, Image: id, Daemon: daemon}
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return config, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if os.IsExist(err) {
		return selectRuntime(ctx, root, daemon)
	}
	if err != nil {
		return config, err
	}
	_, err = f.Write(append(data, '\n'))
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return config, err
}

func validImageID(id string) bool {
	if !strings.HasPrefix(id, "sha256:") || len(id) != 71 {
		return false
	}
	_, err := hex.DecodeString(id[7:])
	return err == nil
}

func bindMount(source, target string) string {
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	_ = w.Write([]string{"type=bind", "src=" + source, "dst=" + target})
	w.Flush()
	return strings.TrimSuffix(out.String(), "\n")
}

func translateArgs(root string, args []string) ([]string, []string, error) {
	var err error
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, nil, err
	}
	out := append([]string(nil), args...)
	var mounts []string
	projectIndex := projectOperandIndex(args)
	for i := 1; i < len(out); i++ {
		name, value, equals := strings.Cut(out[i], "=")
		projectDir := i == projectIndex
		if projectDir {
			name, value, equals = "project directory", out[i], false
		} else if name != "--workspace" && name != "--sample" && name != "--output" {
			if filepath.IsAbs(out[i]) {
				return nil, nil, errors.New("use a package path relative to the project directory")
			}
			continue
		}
		if !equals && !projectDir {
			if i+1 >= len(out) {
				return nil, nil, fmt.Errorf("%s requires a path", name)
			}
			i++
			value = out[i]
		}
		abs := value
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(root, abs)
		}
		// Resolve parents too: init/demo/output can name a new file under an
		// existing symlink. It must not escape the mounted project directory.
		abs, err = resolveExistingParent(abs)
		if err != nil {
			return nil, nil, err
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if name != "--workspace" {
				return nil, nil, fmt.Errorf("%s must be inside the project", name)
			}
			info, err := os.Stat(abs)
			if err != nil || !info.IsDir() {
				return nil, nil, errors.New("external workspace must be an existing directory")
			}
			mounts = append(mounts, "--mount", bindMount(abs, "/gobble/workspace"))
			value = "/gobble/workspace"
		} else {
			value = "/gobble/project/" + filepath.ToSlash(rel)
		}
		if equals {
			out[i] = name + "=" + value
		} else {
			out[i] = value
		}
	}
	return out, mounts, nil
}

func projectOperandIndex(args []string) int {
	if len(args) == 0 || (args[0] != "init" && args[0] != "demo") {
		return -1
	}
	want := 1
	if args[0] == "demo" {
		want = 2
	}
	operands, endOpts := 0, false
	for i := 1; i < len(args); i++ {
		if args[i] == "--" && !endOpts {
			endOpts = true
			continue
		}
		if !endOpts && strings.HasPrefix(args[i], "-") {
			continue
		}
		operands++
		if operands == want {
			return i
		}
	}
	return -1
}

func resolveExistingParent(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil || !os.IsNotExist(err) {
		return resolved, err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", err
	}
	resolved, err = resolveExistingParent(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(path)), nil
}

func fail(err error) int { fmt.Fprintln(os.Stderr, "gobble:", err); return 1 }
