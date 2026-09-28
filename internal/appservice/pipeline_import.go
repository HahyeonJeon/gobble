package appservice

import "path/filepath"

// importPipeline resolves a native chooser's folder inside an already registered
// Project. Recognition stays read-only; an import never copies or writes source.
func (s *store) importPipeline(projectID, requestID, path string) (PipelineDefinition, error) {
	project, err := s.project(projectID)
	if err != nil {
		return PipelineDefinition{}, err
	}
	if !filepath.IsAbs(path) {
		return PipelineDefinition{}, problem("invalid_request", "Choose an analysis folder inside this Project.")
	}
	// Native choosers may return macOS /var aliases for a canonical /private/var
	// Project. Compare resolved filesystem locations, then enforce containment.
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return PipelineDefinition{}, problem("invalid_request", "The chosen analysis folder is unavailable.")
	}
	relative, err := filepath.Rel(project.Root, path)
	relative = filepath.ToSlash(relative)
	if err != nil || !validRelative(relative) {
		return PipelineDefinition{}, problem("invalid_request", "Choose an analysis folder inside this Project.")
	}
	root, err := openProject(project)
	if err != nil {
		return PipelineDefinition{}, err
	}
	_, err = pipelineSource(root, relative)
	root.Close()
	if err != nil {
		return PipelineDefinition{}, problem("invalid_request", "No supported analysis was found in this folder. Ask the Agent to prepare or check the pipeline, then import it again.")
	}
	id := resourceID(projectID, relative)
	if err := s.rememberResources(projectID, map[string]string{id: relative}); err != nil {
		return PipelineDefinition{}, err
	}
	return s.registerPipeline(projectID, id, requestID, filepath.Base(path))
}
