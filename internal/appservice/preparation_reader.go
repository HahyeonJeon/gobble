package appservice

import (
	"context"
	"encoding/json"
	"github.com/HahyeonJeon/gobble/internal/preparation"
	"os"
	"path/filepath"
	"time"
)

type verifiedPreparation struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Digest        string              `json:"digest"`
	Binding       preparation.Binding `json:"binding"`
	Flow          json.RawMessage     `json:"flow"`
	Steps         []preparation.Step  `json:"steps"`
}

// Read through the trusted installed binary after the arbitrary Project evaluator
// has exited. No source is mounted or evaluated in this second container. Its
// safe projection must agree with the exact bytes that will reach P4 admission.
func (a runtimeAdapter) readPrepared(ctx context.Context, parent string, runtime RuntimeBinding, raw []byte, id string, b preparation.Binding) (verifiedPreparation, error) {
	var v verifiedPreparation
	dir := filepath.Join(parent, "validation")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return v, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	intent, err := json.Marshal(struct {
		Digest  string              `json:"digest"`
		Binding preparation.Binding `json:"binding"`
	}{id, b})
	if err != nil {
		return v, err
	}
	for name, data := range map[string][]byte{"payload.json": raw, "intent.json": intent} {
		if err = atomicWrite(dir, name, data); err != nil {
			return v, err
		}
		if err = os.Chmod(filepath.Join(dir, name), 0444); err != nil {
			return v, err
		}
	}
	name := "gobble-prepared-" + newID("req")
	args := []string{"run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=32", "--memory=128m", "--cpus=1", "--user", "10001:10001", "--workdir", "/gobble/prepared", "--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--mount", readOnlyMount(dir, "/gobble/prepared"), "--entrypoint", "/usr/bin/timeout", runtime.ImageID, "--signal=KILL", "15s", "/usr/local/bin/gobble", "prepared-review"}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = a.commandAt(cleanup, runtime.Endpoint, "rm", "--force", name)
	}()
	response, err := a.commandAt(ctx, runtime.Endpoint, args...)
	if err != nil || len(response) > maxFlowBytes || decodeCreationJSON(response, &v) != nil || v.SchemaVersion != 1 || v.Digest != id || v.Binding != b || len(v.Steps) != 2 {
		return v, problem("unsupported", "The selected engine could not independently verify this prepared plan. No analysis started.")
	}
	return v, nil
}
