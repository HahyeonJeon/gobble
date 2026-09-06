package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

"github.com/HahyeonJeon/gobble/internal/containerenv"
)

type controllerMount = containerenv.Mount

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
return containerenv.MapPath(path, mounts)
}
