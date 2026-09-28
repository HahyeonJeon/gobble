package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type runCandidate struct {
	WorkspaceResourceID string `json:"workspaceResourceId"`
	Name                string `json:"name"`
}
type runsResult struct {
	ProjectID  string            `json:"projectId"`
	Runs       []RunRegistration `json:"runs"`
	Candidates []runCandidate    `json:"candidates"`
	Truncated  bool              `json:"truncated"`
}

type snapshotResult struct {
	ProjectID      string          `json:"projectId"`
	RunRef         string          `json:"runRef"`
	EngineRevision string          `json:"engineRevision"`
	ObservedAt     int64           `json:"observedAt"`
	RuntimeBinding RuntimeBinding  `json:"runtimeBinding"`
	Availability   string          `json:"availability"`
	Snapshot       json.RawMessage `json:"snapshot"`
}
type logsResult struct {
	ProjectID      string          `json:"projectId"`
	RunRef         string          `json:"runRef"`
	EngineRevision string          `json:"engineRevision"`
	ObservedAt     int64           `json:"observedAt"`
	Instance       string          `json:"instance"`
	Attempt        int             `json:"attempt"`
	TailLimitBytes int             `json:"tailLimitBytes"`
	Logs           json.RawMessage `json:"logs"`
}

// These headers validate the adapter contract; engine fields are passed through unchanged.
type monitorHeader struct {
	SchemaVersion int    `json:"schema_version"`
	Snapshot      string `json:"snapshot"`
	Run           struct {
		ID string `json:"id"`
	} `json:"run"`
	Tasks []struct {
		Identity string `json:"identity"`
		Attempt  int    `json:"attempt"`
	} `json:"tasks"`
	Logs json.RawMessage `json:"logs"`
}
type identityHeader struct {
	SchemaVersion int             `json:"schema_version"`
	View          string          `json:"view"`
	Match         bool            `json:"match"`
	Required      json.RawMessage `json:"required"`
}

func (s *store) listRuns(projectID string) (runsResult, error) {
	p, err := s.project(projectID)
	if err != nil {
		return runsResult{}, err
	}
	root, err := openProject(p)
	if err != nil {
		return runsResult{}, err
	}
	defer root.Close()
	out := runsResult{ProjectID: projectID, Runs: []RunRegistration{}, Candidates: []runCandidate{}}
	s.mu.Lock()
	for _, run := range s.data.Runs {
		if run.ProjectID == projectID {
			out.Runs = append(out.Runs, run.RunRegistration)
		}
	}
	s.mu.Unlock()
	paths := []string{"."}
	directory, err := openResource(root, "runs")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return runsResult{}, fileAccessError(err)
	}
	if err == nil {
		defer directory.Close()
		info, err := directory.Stat()
		if err != nil {
			return runsResult{}, fileAccessError(err)
		}
		if info.IsDir() {
			entries, readErr := directory.ReadDir(maxDirectoryEntries + 1)
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				return runsResult{}, fileAccessError(readErr)
			}
			if len(entries) > maxDirectoryEntries {
				entries = entries[:maxDirectoryEntries]
				out.Truncated = true
			}
			for _, entry := range entries {
				if entry.IsDir() {
					paths = append(paths, "runs/"+entry.Name())
				}
			}
		}
	}
	resources := map[string]string{}
	for _, path := range paths {
		info, err := root.Stat(filepath.Join(path, ".gobble"))
		if err != nil || !info.IsDir() {
			continue
		}
		id := resourceID(projectID, path)
		resources[id] = path
		attached := false
		for _, run := range out.Runs {
			if run.WorkspaceResourceID == id {
				attached = true
				break
			}
		}
		if !attached {
			name := filepath.Base(path)
			if path == "." {
				name = p.Name
			}
			out.Candidates = append(out.Candidates, runCandidate{id, name})
		}
	}
	if err := s.rememberResources(projectID, resources); err != nil {
		return runsResult{}, err
	}
	return out, nil
}

func (s *store) existingReceipt(requestID, digest string) (RunRegistration, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.data.Requests[requestID]
	if !ok {
		return RunRegistration{}, false, nil
	}
	if r.Digest != digest {
		return RunRegistration{}, false, problem("request_conflict", "This request ID belongs to another registration.")
	}
	for _, run := range s.data.Runs {
		if run.RunRef == r.RunRef {
			return run.RunRegistration, true, nil
		}
	}
	return RunRegistration{}, false, problem("request_conflict", "This request ID belongs to another operation.")
}

func (s *Service) attachRun(ctx context.Context, projectID, workspaceID, requestID string) (RunRegistration, error) {
	if !validID(requestID, "req") || !validID(workspaceID, "res") {
		return RunRegistration{}, problem("invalid_request", "A workspace resource and request ID are required.")
	}
	hash := requestDigest([]string{"attach", projectID, workspaceID})
	if old, ok, err := s.store.existingReceipt(requestID, hash); ok || err != nil {
		return old, err
	}
	p, err := s.store.project(projectID)
	if err != nil {
		return RunRegistration{}, err
	}
	workspace, ok := p.Resources[workspaceID]
	if !ok {
		return RunRegistration{}, problem("not_found", "Workspace not found in this Project.")
	}
	root, err := openProject(p)
	if err != nil {
		return RunRegistration{}, err
	}
	defer root.Close()
	info, err := root.Stat(workspace)
	if err != nil || !info.IsDir() {
		return RunRegistration{}, problem("not_found", "Choose an existing workspace directory.")
	}
	binding, err := s.runtime.selectBinding(ctx, p, workspace)
	if err != nil {
		return RunRegistration{}, err
	}
	identity, err := s.runtime.inspect(ctx, p, binding, "identity", "")
	if err != nil {
		return RunRegistration{}, err
	}
	required, err := readIdentity(identity)
	if err != nil {
		return RunRegistration{}, err
	}
	data, err := s.runtime.inspect(ctx, p, binding, "monitor", "")
	if err != nil {
		return RunRegistration{}, err
	}
	header, err := readMonitor(data)
	if err != nil {
		return RunRegistration{}, err
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	if old, ok := s.store.data.Requests[requestID]; ok {
		if old.Digest != hash {
			return RunRegistration{}, problem("request_conflict", "This request ID belongs to another registration.")
		}
		for _, run := range s.store.data.Runs {
			if run.RunRef == old.RunRef {
				return run.RunRegistration, nil
			}
		}
	}
	next := s.store.copyLocked()
	var selected RunRegistration
	for _, run := range next.Runs {
		if run.ProjectID == projectID && run.WorkspaceResourceID == workspaceID {
			if run.EngineRunID != header.Run.ID || run.Binding != binding || requestDigest(run.Identity) != requestDigest(required) {
				return RunRegistration{}, problem("request_conflict", "This workspace has a different registered run or runtime.")
			}
			selected = run.RunRegistration
			break
		}
	}
	if selected.RunRef == "" {
		if len(next.Runs) >= 1000 {
			return RunRegistration{}, problem("unsupported", "The Run registration limit has been reached.")
		}
		name := filepath.Base(workspace)
		if workspace == "." {
			name = p.Name
		}
		selected = RunRegistration{projectID, newID("run"), name, workspaceID, header.Run.ID}
		next.Runs = append(next.Runs, runRecord{selected, binding, required})
	}
	if len(next.Requests) >= 10000 {
		return RunRegistration{}, problem("unsupported", "The registration request limit has been reached.")
	}
	next.Requests[requestID] = receipt{Digest: hash, ProjectID: projectID, RunRef: selected.RunRef}
	if err := s.store.commitLocked(next); err != nil {
		return RunRegistration{}, err
	}
	return selected, nil
}

func readIdentity(raw json.RawMessage) (json.RawMessage, error) {
	var header identityHeader
	if json.Unmarshal(raw, &header) != nil || header.SchemaVersion != 2 || header.View != "identity" || !header.Match || len(header.Required) == 0 || string(header.Required) == "null" {
		return nil, problem("incompatible_runtime", "This runtime cannot read the workspace identity.")
	}
	return header.Required, nil
}
func readMonitor(raw json.RawMessage) (monitorHeader, error) {
	var header monitorHeader
	if json.Unmarshal(raw, &header) != nil || header.SchemaVersion != 2 || header.Snapshot == "" || header.Run.ID == "" || header.Tasks == nil || len(header.Logs) == 0 {
		return header, problem("incompatible_runtime", "The runtime returned an unsupported monitor snapshot.")
	}
	return header, nil
}

func (s *Service) queryRun(ctx context.Context, projectID, runRef, instance string, attempt int) (snapshotResult, error) {
	p, err := s.store.project(projectID)
	if err != nil {
		return snapshotResult{}, err
	}
	s.store.mu.Lock()
	var selected runRecord
	for _, run := range s.store.data.Runs {
		if run.RunRef == runRef && run.ProjectID == projectID {
			selected = run
			break
		}
	}
	s.store.mu.Unlock()
	if selected.RunRef == "" {
		return snapshotResult{}, problem("not_found", "Run not found in this Project.")
	}
	workspace := p.Resources[selected.WorkspaceResourceID]
	if selected.Binding.WorkspacePath != workspaceContainerPath(workspace) {
		return snapshotResult{}, problem("incompatible_runtime", "The workspace mapping has changed.")
	}
	root, err := openProject(p)
	if err != nil {
		return snapshotResult{}, err
	}
	_, err = root.Stat(workspace)
	root.Close()
	if err != nil {
		return snapshotResult{}, fileAccessError(err)
	}
	if err := s.runtime.verify(ctx, selected.Binding); err != nil {
		return snapshotResult{}, err
	}
	identity, err := s.runtime.inspect(ctx, p, selected.Binding, "identity", "")
	if err != nil {
		return snapshotResult{}, err
	}
	required, err := readIdentity(identity)
	if err != nil {
		return snapshotResult{}, err
	}
	if requestDigest(required) != requestDigest(selected.Identity) {
		return snapshotResult{}, problem("stale_revision", "The workspace engine identity has changed.")
	}
	raw, err := s.runtime.inspect(ctx, p, selected.Binding, "monitor", instance)
	if err != nil {
		return snapshotResult{}, err
	}
	header, err := readMonitor(raw)
	if err != nil {
		return snapshotResult{}, err
	}
	if header.Run.ID != selected.EngineRunID {
		return snapshotResult{}, problem("stale_revision", "The workspace now contains a different Run.")
	}
	if instance != "" {
		found := false
		for _, task := range header.Tasks {
			if task.Identity == instance && task.Attempt == attempt {
				found = true
				break
			}
		}
		if !found {
			return snapshotResult{}, problem("stale_revision", "The selected task attempt is no longer current.")
		}
	}
	return snapshotResult{projectID, runRef, header.Snapshot, time.Now().UnixMilli(), selected.Binding, "available", raw}, nil
}

func validInstance(instance string) bool {
	return len(instance) > 0 && len(instance) <= 256 && !strings.ContainsAny(instance, "\x00\r\n")
}
