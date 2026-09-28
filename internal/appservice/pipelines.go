package appservice

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxPipelines = 1000
const maxPipelineSourceBytes = 1 << 20

// PipelineDefinition is registered analysis identity, independent of View lifetime.
// A creation draft is not a PipelineDefinition.
type PipelineDefinition struct {
	ProjectID  string         `json:"projectId"`
	PipelineID string         `json:"pipelineId"`
	Name       string         `json:"name"`
	Origin     PipelineOrigin `json:"origin"`
}

type pipelinesResult struct {
	ProjectID string               `json:"projectId"`
	Pipelines []PipelineDefinition `json:"pipelines"`
}

func (s *store) listPipelines(projectID string) (pipelinesResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for _, p := range s.data.Projects {
		if p.ProjectID == projectID {
			found = true
			break
		}
	}
	if !found {
		return pipelinesResult{}, problem("not_found", "Project not found.")
	}
	out := pipelinesResult{projectID, []PipelineDefinition{}}
	for _, p := range s.data.Pipelines {
		if p.ProjectID == projectID {
			out.Pipelines = append(out.Pipelines, p)
		}
	}
	return out, nil
}

// registerPipeline serializes metadata publication with other catalog mutations.
// Recognition parses contained source only; it does not compile or execute Go.
func (s *store) registerPipeline(projectID, packageID, requestID, name string) (PipelineDefinition, error) {
	if !validID(projectID, "prj") || !validID(packageID, "res") || !validID(requestID, "req") ||
		!utf8.ValidString(name) || len(name) > 200 || strings.TrimSpace(name) == "" {
		return PipelineDefinition{}, problem("invalid_request", "Choose a Project folder and a Pipeline name of 1–200 bytes.")
	}
	intent := requestDigest([]string{"register_pipeline", projectID, packageID, name})
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.data.Requests[requestID]; ok {
		if old.Digest != intent {
			return PipelineDefinition{}, problem("request_conflict", "This request ID belongs to another registration.")
		}
		for _, p := range s.data.Pipelines {
			if p.PipelineID == old.PipelineID && p.ProjectID == projectID {
				return p, nil
			}
		}
		return PipelineDefinition{}, problem("internal", "The Pipeline registration receipt is unavailable.")
	}
	next := s.copyLocked()
	projectIndex := -1
	for i, p := range next.Projects {
		if p.ProjectID == projectID {
			projectIndex = i
			break
		}
	}
	if projectIndex < 0 {
		return PipelineDefinition{}, problem("not_found", "Project not found.")
	}
	project := &next.Projects[projectIndex]
	path, ok := project.Resources[packageID]
	if !ok {
		return PipelineDefinition{}, problem("not_found", "Folder not found in this Project.")
	}
	if len(next.Requests) >= 10000 {
		return PipelineDefinition{}, problem("unsupported", "The registration request limit has been reached.")
	}
	var selected PipelineDefinition
	for _, p := range next.Pipelines {
		if p.ProjectID == projectID && p.Origin.PackageResourceID == packageID {
			selected = p
			break
		}
	}
	if selected.PipelineID == "" {
		if len(next.Pipelines) >= maxPipelines {
			return PipelineDefinition{}, problem("unsupported", "The Pipeline limit has been reached.")
		}
		root, err := openProject(*project)
		if err != nil {
			return PipelineDefinition{}, err
		}
		defer root.Close()
		source, err := pipelineSource(root, path)
		if err != nil {
			return PipelineDefinition{}, err
		}
		sourceID := resourceID(projectID, source)
		project.Resources[sourceID] = source
		if len(project.Resources) > 10000 {
			return PipelineDefinition{}, problem("unsupported", "The Project resource limit has been reached.")
		}
		selected = PipelineDefinition{projectID, newID("pip"), strings.TrimSpace(name), PipelineOrigin{"imported", packageID, sourceID}}
		next.Pipelines = append(next.Pipelines, selected)
	}
	next.Requests[requestID] = receipt{Digest: intent, ProjectID: projectID, PipelineID: selected.PipelineID}
	if err := s.commitLocked(next); err != nil {
		return PipelineDefinition{}, err
	}
	return selected, nil
}

func pipelineSource(root *os.Root, path string) (string, error) {
	directory, err := openResource(root, path)
	if err != nil {
		return "", fileAccessError(err)
	}
	defer directory.Close()
	info, err := directory.Stat()
	if err != nil || !info.IsDir() {
		return "", problem("invalid_request", "Choose the folder containing the Go Pipeline() entry point.")
	}
	entries, err := directory.ReadDir(maxDirectoryEntries + 1)
	if err != nil && err != io.EOF {
		return "", fileAccessError(err)
	}
	if len(entries) > maxDirectoryEntries {
		return "", problem("unsupported", "Pipeline registration supports folders with up to 500 entries.")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	selected, total := "", 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(path, entry.Name()))
		data, err := readPipelineSource(root, rel)
		if err != nil {
			return "", err
		}
		total += len(data)
		if total > maxFileBytes {
			return "", problem("unsupported", "Pipeline registration supports up to 8 MiB of Go source per folder.")
		}
		source, err := parser.ParseFile(token.NewFileSet(), entry.Name(), data, parser.SkipObjectResolution)
		if err != nil {
			return "", problem("invalid_request", "The folder contains invalid Go source. Fix it before registering the Pipeline.")
		}
		for _, decl := range source.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "Pipeline" {
				continue
			}
			if fn.Body == nil || fn.Type.TypeParams.NumFields() != 0 || fn.Type.Params.NumFields() != 0 || fn.Type.Results.NumFields() != 1 {
				return "", problem("invalid_request", "The Go entry point must be a non-generic Pipeline() function with one result.")
			}
			if selected != "" {
				return "", problem("unsupported", "This folder has multiple Pipeline() declarations. Register a package with one unambiguous entry point.")
			}
			selected = rel
		}
	}
	if selected == "" {
		return "", problem("invalid_request", "No Pipeline() entry point was found in this folder's Go source.")
	}
	return selected, nil
}

func readPipelineSource(root *os.Root, path string) ([]byte, error) {
	file, err := openResource(root, path)
	if err != nil {
		return nil, fileAccessError(err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return nil, fileAccessError(err)
	}
	if !before.Mode().IsRegular() || before.Size() > maxPipelineSourceBytes {
		return nil, problem("unsupported", "Pipeline source must be a regular Go file of at most 1 MiB.")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPipelineSourceBytes+1))
	if err != nil {
		return nil, fileAccessError(err)
	}
	if len(data) > maxPipelineSourceBytes {
		return nil, problem("unsupported", "Pipeline source exceeds the 1 MiB limit.")
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, problem("stale_revision", "Pipeline source changed while registering. Refresh and try again.")
	}
	return data, nil
}
