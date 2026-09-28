package appservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

type creationScope struct {
	InputPath string         `json:"inputPath"`
	ScopeID   string         `json:"scopeId"`
	Draft     creationDraft  `json:"draft"`
	RuntimeID string         `json:"runtimeId"`
	Guidance  string         `json:"guidance"`
	Files     []proposalFile `json:"files"`
}
type creationManifest struct {
	InputPath     string           `json:"inputPath"`
	SchemaVersion int              `json:"schemaVersion"`
	Draft         creationDraft    `json:"draft"`
	Runtime       creationRuntime  `json:"runtime"`
	Files         []inspectionFile `json:"files"`
}

func (s *Service) creationDirectory(draftID, candidateID string) string {
	return filepath.Join(s.store.dir, "pipeline-creations", draftID, candidateID)
}
func (s *Service) creationScopeLocked(projectID, draftID string, generation int64) (creationScope, creationRuntime, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	d, err := s.store.mutableDraftLocked(projectID, draftID, generation)
	if err != nil {
		return creationScope{}, creationRuntime{}, err
	}
	if d.Input == nil {
		return creationScope{}, creationRuntime{}, problem("invalid_request", "Select a single-end read file first.")
	}
	p, err := s.store.draftProjectLocked(projectID)
	if err != nil {
		return creationScope{}, creationRuntime{}, err
	}
	observed, err := observeCreationInput(p, d.Input.ResourceID)
	if err != nil || observed != *d.Input {
		return creationScope{}, creationRuntime{}, problem("stale_revision", "The selected file changed. Review the selected file again.")
	}
	runtime, err := s.readCreationRuntime()
	if err != nil {
		return creationScope{}, runtime, err
	}
	source := runtime.Scaffold.Files[2]
	inputPath, err := pipelinereview.CreationPath(d.Input.RelativePath)
	if err != nil {
		return creationScope{}, runtime, err
	}
	scope := creationScope{inputPath, requestDigest([]any{d, runtime.ID}), d, runtime.ID, runtime.Scaffold.Guidance, []proposalFile{{source.Path, source.Content}}}
	return scope, runtime, nil
}
func (s *Service) creationAuthoring(projectID, draftID string, generation int64) (creationScope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	scope, _, err := s.creationScopeLocked(projectID, draftID, generation)
	return scope, err
}
func retainCreation(scope creationScope, runtime creationRuntime, files []proposalFile, dir string) (creationManifest, error) {
	if len(files) != 1 || files[0].Path != pipelinereview.CreationSourcePath || len(files[0].Content) == 0 || len(files[0].Content) > 32<<10 || !utf8.ValidString(files[0].Content) || strings.ContainsRune(files[0].Content, 0) {
		return creationManifest{}, problem("forbidden", "Provide only the complete bounded pipeline/pipe.go source. Setup and inputs cannot be edited.")
	}
	manifest := creationManifest{scope.InputPath, 1, cloneCreationDraft(scope.Draft), runtime, []inspectionFile{}}
	for _, f := range runtime.Scaffold.Files {
		data := []byte(f.Content)
		if f.Path == pipelinereview.CreationInputFile {
			var err error
			data, err = json.Marshal(pipelinereview.CreationInput{SchemaVersion: 1, Path: scope.InputPath, ReadLayout: "single-end"})
			if err != nil {
				return manifest, err
			}
		}
		if f.Path == files[0].Path {
			data = []byte(files[0].Content)
		}
		target := filepath.Join(dir, "project", filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return manifest, err
		}
		if err := atomicWrite(filepath.Dir(target), filepath.Base(target), data); err != nil {
			return manifest, err
		}
		if err := os.Chmod(target, 0444); err != nil {
			return manifest, err
		}
		manifest.Files = append(manifest.Files, inspectionFile{f.Path, digest(data), len(data)})
	}
	return manifest, writeInspectionJSON(filepath.Join(dir, "manifest.json"), manifest)
}
func verifyCreationSource(dir string, manifest creationManifest) error {
	allowed := map[string]inspectionFile{}
	for _, f := range manifest.Files {
		allowed[f.Path] = f
	}
	count := 0
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		f, ok := allowed[filepath.ToSlash(relative)]
		if !ok || entry.Type()&os.ModeSymlink != 0 {
			return problem("forbidden", "Unexpected creation source file.")
		}
		raw, err := readBoundedFile(path, 256<<10)
		if err != nil || len(raw) != f.Size || digest(raw) != f.SHA256 {
			return problem("stale_revision", "Retained creation source changed.")
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if count != len(manifest.Files) {
		return problem("stale_revision", "Retained creation source is incomplete.")
	}
	return nil
}
