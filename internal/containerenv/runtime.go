// Package containerenv prepares the shared Linux runtime for direct Docker or
// Compose invocation. Runtime identity comes from the connected daemon.
package containerenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const LockFile = ".gobble-runtime.json"

type Lock struct {
	Format    int    `json:"format"`
	Image     string `json:"image"`
	Daemon    string `json:"daemon"`
	Reference string `json:"reference,omitempty"`
}

func Host(daemon string) string {
	hash := sha256.Sum256([]byte(daemon))
	return "gobble-" + hex.EncodeToString(hash[:12])
}

func (l Lock) PullReference() string {
	if l.Reference != "" {
		return l.Reference
	}
	return l.Image
}

func Current() Lock {
	return Lock{Format: 1, Image: os.Getenv("GOBBLE_RUNTIME_IMAGE_ID"),
		Daemon: os.Getenv("GOBBLE_DAEMON_ID"), Reference: os.Getenv("GOBBLE_RUNTIME_REFERENCE")}
}

func docker(ctx context.Context, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// Prepare verifies the actual runtime, makes the shared project writable as its
// owner, and refuses to change an existing project's image or owning daemon.
func Prepare(ctx context.Context) error {
	endpoint := os.Getenv("DOCKER_HOST")
	if endpoint == "" {
		endpoint = "unix:///var/run/docker.sock"
	}
	if !strings.HasPrefix(endpoint, "unix:///") || os.Getenv("DOCKER_CONTEXT") != "" {
		return errors.New("mount the local Docker socket and use its unix:/// endpoint")
	}
	if err := os.Setenv("DOCKER_HOST", endpoint); err != nil {
		return err
	}
	controller := os.Getenv("GOBBLE_CONTROLLER")
	if controller == "" {
		var err error
		controller, err = os.Hostname()
		if err != nil {
			return err
		}
	}
	out, err := docker(ctx, "inspect", "--format", "{{json .}}", controller)
	if err != nil {
		return fmt.Errorf("discover runtime container (keep Docker's default hostname): %w", err)
	}
	var self struct {
		ID, Image string
		Mounts    []Mount
	}
	if err := json.Unmarshal(out, &self); err != nil {
		return err
	}
	out, err = docker(ctx, "info", "--format", "{{.ID}} {{.OSType}}")
	if err != nil {
		return err
	}
	info := strings.Fields(string(out))
	if len(info) != 2 || info[0] == "" || info[1] != "linux" {
		return errors.New("Gobble requires a local Docker engine running Linux containers")
	}
	var image struct {
		Os, Architecture string
		RepoDigests      []string
	}
	out, err = docker(ctx, "image", "inspect", "--format", "{{json .}}", self.Image)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, &image); err != nil {
		return err
	}
	if image.Os != "linux" || image.Architecture != "amd64" {
		return errors.New("this runtime requires linux/amd64; ARM hosts need Docker emulation")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := MapPath(cwd, self.Mounts); err != nil {
		return fmt.Errorf("share the project with a writable bind mount: %w", err)
	}
	if err := useProjectOwner(cwd, strings.TrimPrefix(endpoint, "unix://")); err != nil {
		return err
	}
	lock := Lock{Format: 1, Image: self.Image, Daemon: info[0]}
	if len(image.RepoDigests) > 0 {
		lock.Reference = image.RepoDigests[0]
	}
	if err := Pin(cwd, lock); err != nil {
		return err
	}
	for key, value := range map[string]string{
		"GOBBLE_CONTROLLER": self.ID, "GOBBLE_RUNTIME_IMAGE_ID": lock.Image,
		"GOBBLE_DAEMON_ID": lock.Daemon, "GOBBLE_RUNTIME_REFERENCE": lock.Reference,
		"HOME": "/tmp/gobble-home", "GOCACHE": filepath.Join(cwd, ".gobble-cache", "build"),
	} {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return os.MkdirAll(os.Getenv("HOME"), 0o755)
}

// Pin uses an atomic hard-link publication: concurrent first commands can only
// see a complete lock, and no command can overwrite an existing identity.
func Pin(root string, wanted Lock) error {
	path := filepath.Join(root, LockFile)
	data, err := os.ReadFile(path)
	if err == nil {
		var existing Lock
		if json.Unmarshal(data, &existing) != nil || existing.Format != 1 ||
			existing.Image != wanted.Image || existing.Daemon != wanted.Daemon {
			return fmt.Errorf("runtime/daemon mismatch in %s; use the project's pinned Compose image and original Docker engine; preserve this lock and run state", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	data, err = json.MarshalIndent(wanted, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(root, ".gobble-runtime-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(data, '\n')); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Link(f.Name(), path); os.IsExist(err) {
		return Pin(root, wanted)
	} else {
		return err
	}
}
