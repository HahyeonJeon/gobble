package appservice

import (
	"context"
	"encoding/json"
	"time"
)

// pipelineFlow is a bounded evaluator, distinct from the Run query adapter. It
// uses a container-local deadline even if the host client disappears, and
// mounts only retained declared inputs; no Project, Docker socket, credentials,
// network, host process access or writable shared cache enters the container.
func (a runtimeAdapter) pipelineFlow(ctx context.Context, retained string, source inspectionSource) (json.RawMessage, error) {
	return a.pipelineEvaluation(ctx, retained, source, "flow")
}

func (a runtimeAdapter) pipelineEvaluation(ctx context.Context, retained string, source inspectionSource, verb string) (json.RawMessage, error) {
	name := "gobble-flow-" + newID("req")
	b := source.Runtime
	args := []string{"run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=128", "--memory=768m", "--cpus=1", "--user", "10001:10001",
		"--tmpfs", "/tmp:rw,exec,nosuid,nodev,size=512m,mode=1777", "--workdir", "/gobble/project",
		"--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--env", "GOTOOLCHAIN=local", "--env", "GOPROXY=off", "--env", "GOSUMDB=off", "--env", "GOWORK=off", "--env", "GOENV=off", "--env", "GOFLAGS=-mod=readonly -buildvcs=false", "--env", "GOMAXPROCS=1", "--env", "GOCACHE=/tmp/go-cache", "--env", "HOME=/tmp",
		"--mount", readOnlyMount(retained, "/gobble/project"), "--entrypoint", "/usr/bin/timeout", b.ImageID, "--signal=KILL", "110s", "/usr/local/bin/gobble", verb, "./" + source.Package}
	if source.Sample != "" {
		args = append(args, "--sample", source.Sample)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = a.commandAt(cleanup, b.Endpoint, "rm", "--force", name)
	}()
	raw, err := a.commandAt(ctx, b.Endpoint, args...)
	if err != nil {
		if ctx.Err() != nil {
			return nil, problem("runtime_unavailable", "The analysis check was interrupted or exceeded two minutes.")
		}
		return nil, problem("invalid_request", "The analysis could not be checked. Ask the Agent to check its inputs and confirm that the pinned runtime supports pipeline inspection.")
	}
	if verb == "review" || verb == "creation-review" || verb == "prepare" {
		return raw, nil
	}
	// The service owns bounds and wire identity; the App's closed flow contract
	// additionally validates every field and endpoint before presenting these bytes.
	var header struct {
		SchemaVersion int               `json:"schemaVersion"`
		Name          string            `json:"name"`
		Steps         []json.RawMessage `json:"steps"`
		Inputs        []json.RawMessage `json:"inputs"`
		Connections   []json.RawMessage `json:"connections"`
	}
	if len(raw) > maxFlowBytes || validateJSON(raw) != nil || json.Unmarshal(raw, &header) != nil || (header.SchemaVersion != 1 && header.SchemaVersion != 2) || header.Name == "" || header.Steps == nil || header.Inputs == nil || header.Connections == nil || len(header.Steps) > 500 || len(header.Inputs) > 500 || len(header.Connections) > 2000 {
		return nil, problem("unsupported", "This flow is unsupported or exceeds the inspection limit of 500 steps and 2,000 connections.")
	}
	return raw, nil
}
