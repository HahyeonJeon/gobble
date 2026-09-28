package appservice

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"time"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

type creationRuntime struct {
	SchemaVersion int                             `json:"schemaVersion"`
	ID            string                          `json:"id"`
	Binding       RuntimeBinding                  `json:"binding"`
	Scaffold      pipelinereview.CreationScaffold `json:"scaffold"`
}

// Native host only: this binding is never supplied by an Agent source edit.
// The profile owns it independently of any Project setup file.
func (s *Service) bindCreationRuntime(ctx context.Context, b RuntimeBinding) (creationRuntime, error) {
	if err := s.runtime.verify(ctx, b); err != nil {
		return creationRuntime{}, err
	}
	scaffold, err := s.runtime.creationScaffold(ctx, b)
	if err != nil {
		return creationRuntime{}, err
	}
	value := creationRuntime{1, requestDigest([]any{b, scaffold}), b, scaffold}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return value, problem("runtime_unavailable", "The service is closing.")
	}
	if err = s.runtime.verify(ctx, b); err != nil {
		return creationRuntime{}, err
	}
	if ctx.Err() != nil {
		return creationRuntime{}, ctx.Err()
	}
	if err = writeInspectionJSON(filepath.Join(s.store.dir, "creation-runtime.json"), value); err != nil {
		s.store.mu.Lock()
		s.store.failed = true
		s.store.mu.Unlock()
		return creationRuntime{}, err
	}
	return value, nil
}
func (s *Service) readCreationRuntime() (creationRuntime, error) {
	var value creationRuntime
	if err := readProposalFile(filepath.Join(s.store.dir, "creation-runtime.json"), &value); err != nil || value.SchemaVersion != 1 || !validBinding(value.Binding) || value.ID != requestDigest([]any{value.Binding, value.Scaffold}) || validateCreationScaffold(value.Scaffold) != nil {
		return creationRuntime{}, problem("incompatible_runtime", "Select a qualified local analysis engine before checking a new Pipeline.")
	}
	return value, nil
}
func validateCreationScaffold(v pipelinereview.CreationScaffold) error {
	if v.SchemaVersion != 1 || v.Scope != pipelinereview.CreationScope || v.Package != "pipeline" || v.InputPath != pipelinereview.CreationReadPath || len(v.Editable) != 1 || v.Editable[0] != pipelinereview.CreationSourcePath || len(v.Files) != 4 || len(v.Guidance) > 8000 {
		return problem("unsupported", "The engine's creation scaffold version is unsupported.")
	}
	expected := []string{"go.mod", "go.sum", pipelinereview.CreationSourcePath, pipelinereview.CreationInputFile}
	for i, f := range v.Files {
		if f.Path != expected[i] || len(f.Content) == 0 || len(f.Content) > 256<<10 {
			return problem("unsupported", "The engine returned an invalid scaffold manifest.")
		}
	}
	return nil
}
func (a runtimeAdapter) creationScaffold(ctx context.Context, b RuntimeBinding) (pipelinereview.CreationScaffold, error) {
	var value pipelinereview.CreationScaffold
	name := "gobble-scaffold-" + newID("req")
	args := []string{"run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=32", "--memory=128m", "--cpus=1", "--user", "10001:10001", "--workdir", "/tmp", "--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--entrypoint", "/usr/bin/timeout", b.ImageID, "--signal=KILL", "15s", "/usr/local/bin/gobble", "creation-scaffold"}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = a.commandAt(cleanup, b.Endpoint, "rm", "--force", name)
	}()
	raw, err := a.commandAt(ctx, b.Endpoint, args...)
	if err != nil {
		return value, err
	}
	if len(raw) > 512<<10 || decodeCreationJSON(raw, &value) != nil {
		return value, problem("unsupported", "The engine cannot export the creation scaffold.")
	}
	return value, validateCreationScaffold(value)
}

func decodeCreationJSON(raw []byte, target any) error {
	// Duplicate keys and unknown nested fields are refused at the native boundary.
	if validateJSON(raw) != nil {
		return problem("unsupported", "Invalid creation response.")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return problem("unsupported", "Trailing creation response.")
	}
	return nil
}
