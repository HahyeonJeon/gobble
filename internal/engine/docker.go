package engine

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const containerWorkDir = "/work"

// dockerCLI runs docker argv without the docker token. Tests replace it.
var dockerCLI = runDockerCLI

func dockerRunner(task TaskPlan) runner {
	return func(cwd string, argv []string, stdout, stderr io.Writer) (int, error) {
		return runDocker(cwd, task, argv, stdout, stderr)
	}
}

func runDocker(cwd string, task TaskPlan, argv []string, stdout, stderr io.Writer) (int, error) {
	if task.Image == "" {
		return -1, errors.New("empty image")
	}
	if msg := invalidImage(task.Image); msg != "" {
		return -1, errors.New(msg)
	}
	if len(argv) == 0 {
		return -1, errors.New("empty command")
	}
	if err := ensureImage(task.Image, stdout, stderr); err != nil {
		return -1, err
	}
	return dockerCLI(dockerRunArgs(cwd, task, argv), stdout, stderr)
}

// dockerRunArgs is docker run after the docker token. Command is the
// complete container argv. Image ENTRYPOINT and CMD are unused.
// Non-zero CPU and Memory become --cpus and --memory. Declared Env
// becomes -e KEY=VALUE.
func dockerRunArgs(cwd string, task TaskPlan, argv []string) []string {
	abs := cwd
	if resolved, err := filepath.Abs(cwd); err == nil {
		abs = resolved
	}
	args := []string{
		"run", "--rm",
		"--user", strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid()),
		"--network=none",
		"--entrypoint", argv[0],
		"-v", abs + ":" + containerWorkDir,
		"-w", containerWorkDir,
	}
	if task.Resources.CPU != 0 {
		args = append(args, "--cpus", strconv.FormatFloat(task.Resources.CPU, 'f', -1, 64))
	}
	if n, ok := parseMemory(task.Resources.Memory); ok && n > 0 {
		args = append(args, "--memory", task.Resources.Memory)
	}
	keys := make([]string, 0, len(task.Env))
	for k, v := range task.Env {
		if k == "" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-e", k+"="+task.Env[k])
	}
	args = append(args, task.Image)
	return append(args, argv[1:]...)
}

func ensureImage(image string, stdout, stderr io.Writer) error {
	if exit, err := dockerCLI([]string{"image", "inspect", image}, io.Discard, io.Discard); err == nil && exit == 0 {
		return nil
	} else if err != nil {
		return errors.New("docker: " + err.Error())
	}
	var buf bytes.Buffer
	exit, err := dockerCLI([]string{"pull", image}, stdout, io.MultiWriter(stderr, &buf))
	if err != nil {
		return errors.New("docker pull: " + err.Error())
	}
	if exit != 0 {
		msg := strings.TrimSpace(buf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(exit)
		}
		return errors.New("docker pull: " + msg)
	}
	return nil
}

func runDockerCLI(args []string, stdout, stderr io.Writer) (int, error) {
	return runProcess("", append([]string{"docker"}, args...), stdout, stderr)
}
