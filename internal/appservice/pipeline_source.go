package appservice

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const maxInspectionFiles = 1000
const maxInspectionBytes = 32 << 20

// inspectionSetup declares the entire evaluator-visible Project input set.
// Paths are Agent-maintained setup, never renderer-supplied execution arguments.
type inspectionSetup struct {
	SchemaVersion int      `json:"schemaVersion"`
	Files         []string `json:"files"`
	Sample        string   `json:"sample"`
}
type inspectionFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}
type inspectionSource struct {
	SchemaVersion int              `json:"schemaVersion"`
	ProjectID     string           `json:"projectId"`
	PipelineID    string           `json:"pipelineId"`
	Package       string           `json:"package"`
	Sample        string           `json:"sample"`
	Files         []inspectionFile `json:"files"`
	Runtime       RuntimeBinding   `json:"runtime"`
}

func (s *store) pipelineProject(projectID, pipelineID string) (projectRecord, PipelineDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pipeline := range s.data.Pipelines {
		if pipeline.ProjectID != projectID || pipeline.PipelineID != pipelineID {
			continue
		}
		for _, project := range s.data.Projects {
			if project.ProjectID == projectID {
				// Do not alias the catalog map across the mutex boundary.
				copy := project
				copy.Resources = make(map[string]string, len(project.Resources))
				for id, path := range project.Resources {
					copy.Resources[id] = path
				}
				return copy, pipeline, nil
			}
		}
	}
	return projectRecord{}, PipelineDefinition{}, problem("not_found", "Pipeline not found in this Project.")
}

func readInspectionFile(root *os.Root, path string) ([]byte, error) {
	file, err := openResource(root, path)
	if err != nil {
		return nil, problem("invalid_request", "A declared analysis input is unavailable. Ask the Agent to check the analysis setup.")
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() > maxFileBytes {
		return nil, problem("unsupported", "Inspection inputs must be regular files of at most 8 MiB. Large research datasets are not copied for flow inspection.")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil || len(data) > maxFileBytes {
		return nil, problem("invalid_request", "An analysis input could not be retained.")
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, problem("stale_revision", "Analysis inputs changed during inspection. Check the flow again.")
	}
	return data, nil
}

// retainInspection copies only declared files through the Project root boundary.
// The evaluator receives this retained tree, never the mutable research Project.
func retainInspection(project projectRecord, pipeline PipelineDefinition, binding RuntimeBinding, destination string) (inspectionSource, error) {
	if pipeline.Origin.Kind != "imported" {
		return inspectionSource{}, problem("unsupported", "Managed source must be read from its retained Current version.")
	}
	root, err := openProject(project)
	if err != nil {
		return inspectionSource{}, err
	}
	defer root.Close()
	setupPath := filepath.ToSlash(filepath.Join(project.Resources[pipeline.Origin.PackageResourceID], ".gobble-inspection.json"))
	raw, err := readInspectionFile(root, setupPath)
	if err != nil {
		return inspectionSource{}, problem("invalid_request", "This analysis needs inspection setup. Ask the Agent to declare its source and small configuration inputs.")
	}
	var setup inspectionSetup
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if validateJSON(raw) != nil || decoder.Decode(&setup) != nil || setup.SchemaVersion != 1 || len(setup.Files) == 0 || len(setup.Files) > maxInspectionFiles || (setup.Sample != "" && !validRelative(setup.Sample)) {
		return inspectionSource{}, problem("invalid_request", "The analysis inspection setup is unsupported. Ask the Agent to update it.")
	}
	paths := map[string]bool{setupPath: true, ".gobble-runtime.json": true, "go.mod": true}
	for _, path := range setup.Files {
		if !validRelative(path) || path == "." || filepath.ToSlash(filepath.Clean(path)) != path {
			return inspectionSource{}, problem("invalid_request", "Inspection inputs must stay inside this Project.")
		}
		paths[path] = true
	}
	if setup.Sample != "" {
		paths[setup.Sample] = true
	}
	if !paths[project.Resources[pipeline.Origin.SourceResourceID]] {
		return inspectionSource{}, problem("invalid_request", "The analysis entry source is missing from its inspection setup.")
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	manifest := inspectionSource{1, project.ProjectID, pipeline.PipelineID, project.Resources[pipeline.Origin.PackageResourceID], setup.Sample, []inspectionFile{}, binding}
	total := 0
	for _, path := range ordered {
		data, err := readInspectionFile(root, path)
		if err != nil {
			return inspectionSource{}, err
		}
		if path == setupPath && !bytes.Equal(raw, data) {
			return inspectionSource{}, problem("stale_revision", "The analysis setup changed. Check the flow again.")
		}
		if path == ".gobble-runtime.json" {
			var lock struct {
				Format int    `json:"format"`
				Image  string `json:"image"`
				Daemon string `json:"daemon"`
			}
			if json.Unmarshal(data, &lock) != nil || lock.Format != 1 || lock.Image != binding.ImageID || lock.Daemon != binding.DaemonID {
				return inspectionSource{}, problem("stale_revision", "The analysis runtime changed. Check the flow again.")
			}
		}
		total += len(data)
		if total > maxInspectionBytes {
			return inspectionSource{}, problem("unsupported", "Inspection inputs exceed 32 MiB. Ask the Agent to narrow the setup inputs.")
		}
		manifest.Files = append(manifest.Files, inspectionFile{path, digest(data), len(data)})
		if destination != "" {
			target := filepath.Join(destination, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return inspectionSource{}, err
			}
			if err := os.WriteFile(target, data, 0o444); err != nil {
				return inspectionSource{}, err
			}
		}
	}
	return manifest, nil
}
