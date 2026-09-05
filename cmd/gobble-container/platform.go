package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func supportedHost(goos, arch string) bool {
	return (goos == "darwin" && (arch == "amd64" || arch == "arm64")) ||
		((goos == "linux" || goos == "windows") && arch == "amd64")
}

func controllerSocket(goos, endpoint string) string {
	if goos == "linux" {
		return strings.TrimPrefix(endpoint, "unix://")
	}
	// Desktop's client socket lives on the host; bind mounts are resolved in
	// its Linux VM. Do not mount ~/.docker/run/docker.sock into the controller.
	return "/var/run/docker.sock"
}

func findDocker() error {
	if _, err := exec.LookPath("docker"); err == nil {
		return nil
	}
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		for _, dir := range []string{filepath.Join(home, ".docker", "bin"), "/usr/local/bin", "/Applications/Docker.app/Contents/Resources/bin"} {
			info, err := os.Stat(filepath.Join(dir, "docker"))
			if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
				return os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
		}
	}
	return errors.New("Docker CLI is unavailable; install Docker Desktop (macOS/Windows) or Docker Engine (Linux) and start it")
}

func inspectRuntime(ctx context.Context, image string) (string, error) {
	value, err := dockerOutput(ctx, "image", "inspect", "--format", "{{.Id}} {{.Os}}/{{.Architecture}}", image)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(value)
	if len(fields) != 2 || !validImageID(fields[0]) {
		return "", errors.New("Docker returned invalid runtime image metadata")
	}
	if fields[1] != runtimePlatform {
		return "", fmt.Errorf("runtime image is %s; rebuild with --platform %s (Apple Silicon uses Docker Desktop emulation)", fields[1], runtimePlatform)
	}
	return fields[0], nil
}
