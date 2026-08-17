package engine

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

const containerWorkDir = "/work"

// dockerCLI runs docker argv without the docker token. Tests replace it.
var dockerCLI = runDockerCLI

func dockerRunner(image string) runner {
	return func(cwd string, argv []string, stdout, stderr io.Writer) (int, error) {
		return runDocker(cwd, image, argv, stdout, stderr)
	}
}

func runDocker(cwd, image string, argv []string, stdout, stderr io.Writer) (int, error) {
	if image == "" {
		return -1, errors.New("empty image")
	}
	if len(argv) == 0 {
		return -1, errors.New("empty command")
	}
	if err := ensureImage(image, stdout, stderr); err != nil {
		return -1, err
	}
	return dockerCLI(dockerRunArgs(cwd, image, argv), stdout, stderr)
}

// dockerRunArgs is docker run after the docker token. Command is the
// complete container argv. Image ENTRYPOINT and CMD are unused.
// --cpus and --memory are not applied.
func dockerRunArgs(cwd, image string, argv []string) []string {
	abs := cwd
	if resolved, err := filepath.Abs(cwd); err == nil {
		abs = resolved
	}
	args := []string{
		"run", "--rm",
		"--entrypoint", argv[0],
		"-v", abs + ":" + containerWorkDir,
		"-w", containerWorkDir,
		image,
	}
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
