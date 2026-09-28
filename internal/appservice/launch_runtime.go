package appservice

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"github.com/HahyeonJeon/gobble/internal/preparation"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"
)

type admittedState struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Admission     preparation.Admission       `json:"admission"`
	Status        string                      `json:"status"`
	Snapshot      string                      `json:"snapshot"`
	OwnerActive   bool                        `json:"ownerActive"`
	OwnerLive     bool                        `json:"ownerLive"`
	Epoch         *preparation.ExecutionEpoch `json:"epoch,omitempty"`
}

func launchIntentDigest(in preparation.LaunchIntent) string { return requestDigest(in) }
func launchControllerName(rec launchRecord) string {
	return "gobble-launch-" + strings.TrimPrefix(launchIntentDigest(rec.Intent), "sha256:")[:32]
}
func writableMount(source, target string) string {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"type=bind", "src=" + source, "dst=" + target})
	w.Flush()
	return strings.TrimSpace(b.String())
}
func launchSocket(endpoint string) string {
	if runtime.GOOS == "linux" {
		return strings.TrimPrefix(endpoint, "unix://")
	}
	return "/var/run/docker.sock"
}

type executionController struct{ name, bundle, verb, digest string }

func (s *Service) launchController(rec launchRecord) executionController {
	return executionController{launchControllerName(rec), s.launchBundle(rec), "prepared-run", launchIntentDigest(rec.Intent)}
}
func (s *Service) createLaunchController(ctx context.Context, p projectRecord, rec launchRecord) error {
	return s.createExecutionController(ctx, p, rec, s.launchController(rec))
}
func (s *Service) createExecutionController(ctx context.Context, p projectRecord, rec launchRecord, spec executionController) error {
	if err := s.runtime.verify(ctx, rec.Runtime); err != nil {
		return err
	}
	name := spec.name
	target := filepath.Join(p.Root, filepath.FromSlash(rec.Value.OutputPath))
	input := filepath.Join(target, filepath.FromSlash(rec.Intent.Binding.InputPath))
	args := []string{"create", "--pull=never", "--platform", "linux/amd64", "--name", name, "--init", "--network=none", "--read-only", "--cap-drop=ALL", "--cap-add=SETUID", "--cap-add=SETGID", "--security-opt=no-new-privileges", "--user", "0:0", "--pids-limit=128", "--memory=512m", "--cpus=1", "--tmpfs", "/tmp:rw,size=64m", "--workdir", rec.Runtime.WorkspacePath, "--label", "io.gobble.launch=" + spec.digest, "--env", "GOBBLE_CONTAINER_BOOTSTRAP=1", "--env", "GOBBLE_CONTROLLER=" + name, "--env", "DOCKER_HOST=unix:///var/run/docker.sock", "--mount", writableMount(launchSocket(rec.Runtime.Endpoint), "/var/run/docker.sock"), "--mount", writableMount(target, rec.Runtime.WorkspacePath), "--mount", readOnlyMount(input, rec.Runtime.WorkspacePath+"/"+rec.Intent.Binding.InputPath), "--mount", readOnlyMount(spec.bundle, "/gobble/launch"), "--entrypoint", "/usr/local/bin/gobble", rec.Runtime.ImageID, spec.verb, "--workspace", rec.Runtime.WorkspacePath}
	_, err := s.runtime.commandAt(ctx, rec.Runtime.Endpoint, args...)
	return err
}
func (s *Service) startOrObserveController(ctx context.Context, p projectRecord, rec launchRecord) error {
	return s.startExecutionController(ctx, p, rec, s.launchController(rec))
}
func (s *Service) startExecutionController(ctx context.Context, p projectRecord, rec launchRecord, spec executionController) error {
	status, err := s.executionControllerState(ctx, p, rec, spec)
	if err != nil {
		return err
	}
	if status == "created" {
		_, err = s.runtime.commandAt(ctx, rec.Runtime.Endpoint, "start", spec.name)
	}
	return err
}

// Inspect validates the exact controller without starting or recreating it.
func (s *Service) executionControllerState(ctx context.Context, p projectRecord, rec launchRecord, spec executionController) (string, error) {
	name := spec.name
	raw, err := s.runtime.commandAt(ctx, rec.Runtime.Endpoint, "inspect", "--format", "{{json .}}", name)
	if err != nil {
		return "", problem("runtime_unavailable", "The original controller is not available. Check the same Run; a new controller will not be created.")
	}
	var c struct {
		Image  string
		Config struct {
			Image      string
			User       string
			Env        []string
			WorkingDir string
			Entrypoint []string
			Cmd        []string
			Labels     map[string]string
		}
		HostConfig struct {
			NetworkMode    string
			ReadonlyRootfs bool
			Privileged     bool
			CapAdd         []string
			CapDrop        []string
			SecurityOpt    []string
		}
		State  struct{ Status string }
		Mounts []struct {
			Source, Destination string
			RW                  bool
		}
	}
	if json.Unmarshal(raw, &c) != nil || c.Image != rec.Runtime.ImageID || c.Config.Image != rec.Runtime.ImageID || c.Config.Labels["io.gobble.launch"] != spec.digest || len(c.Config.Entrypoint) != 1 || c.Config.Entrypoint[0] != "/usr/local/bin/gobble" || strings.Join(c.Config.Cmd, "\n") != spec.verb+"\n--workspace\n"+rec.Runtime.WorkspacePath || c.HostConfig.NetworkMode != "none" || !c.HostConfig.ReadonlyRootfs || c.HostConfig.Privileged || c.Config.User != "0:0" || c.Config.WorkingDir != rec.Runtime.WorkspacePath || !slices.Equal(c.HostConfig.CapDrop, []string{"ALL"}) || len(c.HostConfig.CapAdd) != 2 || !slices.Contains(c.HostConfig.CapAdd, "CAP_SETUID") || !slices.Contains(c.HostConfig.CapAdd, "CAP_SETGID") || !slices.Contains(c.HostConfig.SecurityOpt, "no-new-privileges") {
		return "", problem("incompatible_runtime", "The original controller no longer matches the approved execution.")
	}
	for _, setting := range []string{"GOBBLE_CONTAINER_BOOTSTRAP=1", "GOBBLE_CONTROLLER=" + name, "DOCKER_HOST=unix:///var/run/docker.sock"} {
		key, _, _ := strings.Cut(setting, "=")
		count := 0
		for _, entry := range c.Config.Env {
			if strings.HasPrefix(entry, key+"=") {
				if entry != setting {
					return "", problem("incompatible_runtime", "The controller environment changed.")
				}
				count++
			}
		}
		if count != 1 {
			return "", problem("incompatible_runtime", "The controller environment is incomplete.")
		}
	}
	expected := map[string]struct {
		source string
		rw     bool
	}{rec.Runtime.WorkspacePath: {filepath.Join(p.Root, filepath.FromSlash(rec.Value.OutputPath)), true}, rec.Runtime.WorkspacePath + "/" + rec.Intent.Binding.InputPath: {filepath.Join(p.Root, filepath.FromSlash(rec.Value.OutputPath), filepath.FromSlash(rec.Intent.Binding.InputPath)), false}, "/gobble/launch": {spec.bundle, false}, "/var/run/docker.sock": {launchSocket(rec.Runtime.Endpoint), true}}
	binds := 0
	for _, m := range c.Mounts {
		if m.Destination == "/tmp" {
			continue
		}
		want, ok := expected[m.Destination]
		if !ok || m.Source != want.source || m.RW != want.rw {
			return "", problem("incompatible_runtime", "The original controller has different data access.")
		}
		delete(expected, m.Destination)
		binds++
	}
	if len(expected) != 0 || binds != 4 {
		return "", problem("incompatible_runtime", "The controller mounts are incomplete.")
	}
	if c.State.Status != "created" && c.State.Status != "running" && c.State.Status != "exited" {
		return "", problem("runtime_unavailable", "The original controller status is unconfirmed.")
	}
	return c.State.Status, nil
}
func (s *Service) launchQuery(ctx context.Context, p projectRecord, rec launchRecord, verb string, write bool) ([]byte, error) {
	if err := s.runtime.verify(ctx, rec.Runtime); err != nil {
		return nil, err
	}
	target := filepath.Join(p.Root, filepath.FromSlash(rec.Value.OutputPath))
	mount := readOnlyMount(target, rec.Runtime.WorkspacePath)
	if write {
		mount = writableMount(target, rec.Runtime.WorkspacePath)
	}
	name := "gobble-launch-query-" + newID("req")
	args := []string{"run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--user", "0:0", "--pids-limit=32", "--memory=256m", "--cpus=1", "--workdir", rec.Runtime.WorkspacePath, "--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--env", "GOBBLE_CONTROLLER=" + name, "--env", "GOBBLE_DAEMON_ID=" + rec.Runtime.DaemonID, "--env", "GOBBLE_RUNTIME_IMAGE_ID=" + rec.Runtime.ImageID, "--mount", mount}
	if write {
		args = append(args, "--mount", readOnlyMount(s.launchBundle(rec), "/gobble/launch"), "--mount", writableMount(launchSocket(rec.Runtime.Endpoint), "/var/run/docker.sock"), "--env", "DOCKER_HOST=unix:///var/run/docker.sock")
	}
	args = append(args, "--entrypoint", "/usr/local/bin/gobble", rec.Runtime.ImageID, verb, "--workspace", rec.Runtime.WorkspacePath)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = s.runtime.commandAt(cleanup, rec.Runtime.Endpoint, "rm", "--force", name)
	}()
	return s.runtime.commandAt(ctx, rec.Runtime.Endpoint, args...)
}
func (s *Service) queryLaunchState(ctx context.Context, p projectRecord, rec launchRecord) (admittedState, error) {
	var state admittedState
	raw, err := s.launchQuery(ctx, p, rec, "prepared-status", false)
	if err != nil {
		return state, err
	}
	if decodeCreationJSON(raw, &state) != nil || state.SchemaVersion != rec.Intent.SchemaVersion || state.Admission.SchemaVersion != rec.Intent.SchemaVersion || state.Admission.IntentDigest != launchIntentDigest(rec.Intent) || state.Admission.RequestID != rec.Intent.RequestID || state.Admission.PreparedDigest != rec.Intent.PreparedDigest || state.Admission.WorkspaceID != rec.Intent.WorkspaceID || !regexp.MustCompile(`^[a-f0-9]{32,128}$`).MatchString(state.Admission.Lease) || state.Snapshot == "" || len(state.Snapshot) > 256 || !slices.Contains([]string{"running", "stopping", "stopped", "interrupted", "succeeded", "failed"}, state.Status) {
		return state, problem("incompatible_runtime", "The observed Run does not match this exact launch intent.")
	}
	if (state.SchemaVersion == 1 && state.Epoch != nil) || (state.SchemaVersion == 2 && (state.Epoch == nil || !digestPattern.MatchString(state.Epoch.Head) || !regexp.MustCompile(`^[a-f0-9]{32,128}$`).MatchString(state.Epoch.Lease))) {
		return state, problem("incompatible_runtime", "The Run execution epoch is invalid.")
	}
	return state, nil
}
func (s *Service) sendLaunchStop(ctx context.Context, p projectRecord, rec launchRecord) (string, error) {
	data, _ := json.Marshal(map[string]string{"expectedLease": rec.Value.Operation.StopLease})
	if err := atomicWrite(s.launchBundle(rec), "stop.json", data); err != nil {
		return "requested", err
	}
	raw, err := s.launchQuery(ctx, p, rec, "prepared-stop", true)
	if err != nil {
		return "requested", err
	}
	var result struct {
		Status string `json:"status"`
		Lease  string `json:"lease,omitempty"`
	}
	if decodeCreationJSON(raw, &result) != nil {
		return "requested", problem("internal", "Stop response is unavailable.")
	}
	if result.Status != "requested" && result.Status != "settled" && result.Status != "owner-changed" && result.Status != "recovery-required" {
		return "requested", problem("internal", "Stop response is unsupported.")
	}
	if result.Lease != "" && result.Lease != rec.Value.Operation.StopLease {
		return "requested", problem("internal", "Stop response belongs to another owner.")
	}
	return result.Status, nil
}
