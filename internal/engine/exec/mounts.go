package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type controllerMount struct {
	Type        string
	Source      string
	Destination string
	RW          bool
}

// Docker binds paths on the daemon host. Controller paths must be translated
// using the daemon's actual mount description, including Docker Desktop paths.
func daemonIsolate(ctx context.Context, isolate string) (string, error) {
	controller := os.Getenv("GOBBLE_CONTROLLER")
	if controller == "" {
		return isolate, nil
	}
	var out, stderr bytes.Buffer
	exit, err := dockerCLI(ctx, []string{"inspect", "--format", "{{json .Mounts}}", controller}, &out, &stderr)
	if err != nil {
		return "", err
	}
	if exit != 0 {
		return "", fmt.Errorf("inspect controller mounts: %s", strings.TrimSpace(stderr.String()))
	}
	var mounts []controllerMount
	if err := json.Unmarshal(out.Bytes(), &mounts); err != nil {
		return "", err
	}
	return mapControllerPath(isolate, mounts)
}

func mapControllerPath(path string, mounts []controllerMount) (string, error) {
	abs, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	var selected *controllerMount
	var relative string
	for i := range mounts {
		m := &mounts[i]
		rel, err := filepath.Rel(m.Destination, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if selected == nil || len(m.Destination) > len(selected.Destination) {
			selected, relative = m, rel
		}
	}
	if selected == nil || selected.Type != "bind" || !selected.RW || !filepath.IsAbs(selected.Source) {
		return "", errors.New("attempt directory requires a writable controller bind mount")
	}
	return filepath.Join(selected.Source, relative), nil
}
