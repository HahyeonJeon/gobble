package appservice

import (
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"
)

type store struct {
	mu        sync.Mutex
	dir       string
	lock      *os.File
	data      catalog
	failed    bool
	migration []byte
	// Per-store persistence boundary, also used to exercise uncertain durable writes.
	writeFile func(string, string, []byte) error
}

func openStore(dir string) (*store, error) {
	if !filepath.IsAbs(dir) {
		return nil, errors.New("catalog directory must be absolute")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "catalog.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = lockCatalog(lock); err != nil {
		lock.Close()
		return nil, errors.New("another service owns this catalog")
	}
	s := &store{writeFile: atomicWrite, dir: dir, lock: lock, data: catalog{SchemaVersion: catalogVersion, Projects: []projectRecord{}, Runs: []runRecord{}, Pipelines: []PipelineDefinition{}, Requests: map[string]receipt{}, Revisions: map[string]managedRevision{}, Drafts: map[string]creationDraft{}}}
	data, err := readBoundedFile(filepath.Join(dir, "catalog.json"), maxCatalogBytes)
	if errors.Is(err, os.ErrNotExist) {
		// A missing primary with a retained backup needs explicit recovery, not a new catalog.
		for _, name := range []string{"catalog.backup.json", "catalog.v1.backup.json", "catalog.v2.backup.json", "catalog.v3.backup.json", "catalog.v4.backup.json"} {
			if _, backupErr := os.Stat(filepath.Join(dir, name)); !errors.Is(backupErr, os.ErrNotExist) {
				lock.Close()
				return nil, errors.New("catalog missing; preserve the backup and restore explicitly")
			}
		}
		return s, nil
	}
	if err == nil {
		err = validateJSON(data)
	}
	if err == nil {
		s.data, s.migration, err = decodeCatalog(data)
	}
	if err != nil {
		lock.Close()
		return nil, errors.New("catalog is invalid or unsupported; preserve it and restore explicitly")
	}
	return s, nil
}

func validateCatalog(c catalog) error {
	if c.SchemaVersion != catalogVersion || c.Projects == nil || c.Runs == nil || c.Requests == nil || len(c.Projects) > 100 || len(c.Runs) > 1000 || len(c.Requests) > 10000 {
		return errors.New("invalid catalog")
	}
	projects := map[string]projectRecord{}
	for _, p := range c.Projects {
		if !validID(p.ProjectID, "prj") || !filepath.IsAbs(p.Root) || p.RootIdentity == "" || strings.TrimSpace(p.Name) == "" || len(p.Resources) > 10000 || p.Resources[p.RootResourceID] != "." {
			return errors.New("invalid project")
		}
		if _, ok := projects[p.ProjectID]; ok {
			return errors.New("duplicate project")
		}
		for _, other := range projects {
			if overlaps(p.Root, other.Root) || p.RootIdentity == other.RootIdentity {
				return errors.New("overlapping projects")
			}
		}
		for id, path := range p.Resources {
			if id != resourceID(p.ProjectID, path) || !validRelative(path) {
				return errors.New("invalid resource")
			}
		}
		projects[p.ProjectID] = p
	}
	runs := map[string]bool{}
	for _, run := range c.Runs {
		p, ok := projects[run.ProjectID]
		if !ok || !validID(run.RunRef, "run") || runs[run.RunRef] || p.Resources[run.WorkspaceResourceID] == "" || run.EngineRunID == "" || !json.Valid(run.Identity) || !validBinding(run.Binding) {
			return errors.New("invalid run")
		}
		runs[run.RunRef] = true
	}
	pipelines, err := validatePipelines(c, projects)
	if err != nil {
		return err
	}
	if err := validateManagedRevisions(c, projects, pipelines); err != nil {
		return err
	}
	if err := validateCreationDrafts(c, projects); err != nil {
		return err
	}
	for id, r := range c.Requests {
		if r.DraftID != "" && (c.Drafts[r.DraftID].ProjectID != r.ProjectID || r.PipelineID != "" || r.RunRef != "" || r.ProposalID != "") {
			return errors.New("invalid draft receipt")
		}
		if !validID(id, "req") || !digestPattern.MatchString(r.Digest) || projects[r.ProjectID].ProjectID == "" || (r.RunRef != "" && !runs[r.RunRef]) || (r.PipelineID != "" && pipelines[r.PipelineID] != r.ProjectID) || (r.PipelineID != "" && r.RunRef != "") || (r.ProposalID != "" && (!validID(r.ProposalID, "req") || r.PipelineID == "")) {
			return errors.New("invalid receipt")
		}
	}
	return nil
}

func (s *store) close() error { return s.lock.Close() }

func (s *store) copyLocked() catalog {
	c := s.data
	c.Projects = append([]projectRecord{}, c.Projects...)
	for i := range c.Projects {
		c.Projects[i].Resources = maps.Clone(c.Projects[i].Resources)
	}
	c.Runs = append([]runRecord{}, c.Runs...)
	c.Pipelines = append([]PipelineDefinition{}, c.Pipelines...)
	c.Requests = maps.Clone(c.Requests)
	c.Revisions = maps.Clone(c.Revisions)
	for id, revision := range c.Revisions {
		if revision.Creation != nil {
			birth := *revision.Creation
			revision.Creation = &birth
		}
		revision.Artifact.Flow = append(json.RawMessage(nil), revision.Artifact.Flow...)
		c.Revisions[id] = revision
	}
	c.Drafts = maps.Clone(c.Drafts)
	for id, draft := range c.Drafts {
		c.Drafts[id] = cloneCreationDraft(draft)
	}
	return c
}

func (s *store) commitLocked(next catalog) error {
	if s.failed {
		return problem("internal", "Catalog storage is unavailable. Restart the app before making further changes.")
	}
	if err := validateCatalog(next); err != nil {
		return problem("internal", "Catalog update was rejected.")
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil || len(data) > maxCatalogBytes {
		return problem("unsupported", "The Project catalog has reached its size limit.")
	}
	previous, err := readBoundedFile(filepath.Join(s.dir, "catalog.json"), maxCatalogBytes)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return problem("internal", "The Project catalog could not be read.")
	}
	if len(s.migration) > 0 {
		if err := archiveHistoricalCatalog(s.dir, s.migration); err != nil {
			s.failed = true
			return problem("internal", "The original catalog could not be archived. No update was accepted.")
		}
	}
	if err == nil {
		if err = s.writeFile(s.dir, "catalog.backup.json", previous); err != nil {
			s.failed = true
			return problem("internal", "The Project catalog backup could not be saved.")
		}
	}
	if err = s.writeFile(s.dir, "catalog.json", data); err != nil {
		s.failed = true
		return problem("internal", "The Project catalog could not be saved.")
	}
	s.data = next
	s.migration = nil
	return nil
}

func atomicWrite(dir, name string, data []byte) error {
	f, err := os.CreateTemp(dir, ".catalog-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(f.Name(), filepath.Join(dir, name)); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func readBoundedFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if len(data) > int(limit) {
		return nil, errors.New("file exceeds size limit")
	}
	return data, err
}

func validRelative(path string) bool {
	return utf8.ValidString(path) && len(path) <= 4096 && !strings.ContainsAny(path, "\\\x00") && filepath.IsLocal(path) && filepath.ToSlash(filepath.Clean(path)) == path
}
func overlaps(a, b string) bool {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		a = strings.ToLower(a)
		b = strings.ToLower(b)
	}
	inside := func(root, path string) bool {
		rel, err := filepath.Rel(root, path)
		return err == nil && (rel == "." || filepath.IsLocal(rel))
	}
	return inside(a, b) || inside(b, a)
}

func (s *store) registerProject(requestID, rootPath, name string) (Project, error) {
	if !validID(requestID, "req") || !filepath.IsAbs(rootPath) || len(rootPath) > 4096 || len(name) > 200 {
		return Project{}, problem("invalid_request", "Choose an existing absolute project folder and a valid request ID.")
	}
	digest := requestDigest([]string{"register", rootPath, name})
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.data.Requests[requestID]; ok {
		if old.Digest != digest {
			return Project{}, problem("request_conflict", "This request ID belongs to another registration.")
		}
		for _, p := range s.data.Projects {
			if p.ProjectID == old.ProjectID {
				return p.Project, nil
			}
		}
	}
	canonical, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return Project{}, problem("not_found", "The selected project folder is unavailable.")
	}
	root, err := os.OpenRoot(canonical)
	if err != nil {
		return Project{}, problem("not_found", "Choose an existing project folder.")
	}
	defer root.Close()
	info, err := root.Stat(".")
	if err != nil || rootIdentity(info) == "" {
		return Project{}, problem("unsupported", "This folder cannot be registered on this host.")
	}
	next := s.copyLocked()
	var selected Project
	for i, p := range next.Projects {
		if p.RootIdentity == rootIdentity(info) {
			selected = p.Project
			next.Projects[i].Root = canonical
			continue
		}
		if overlaps(p.Root, canonical) {
			return Project{}, problem("request_conflict", "This folder overlaps an existing Project.")
		}
	}
	if selected.ProjectID == "" {
		if len(next.Projects) >= 100 {
			return Project{}, problem("unsupported", "The Project limit has been reached.")
		}
		name = strings.TrimSpace(name)
		if name == "" {
			name = filepath.Base(canonical)
		}
		if len(name) > 200 {
			return Project{}, problem("invalid_request", "Use a shorter Project name.")
		}
		id := newID("prj")
		rootID := resourceID(id, ".")
		selected = Project{id, name, rootID}
		next.Projects = append(next.Projects, projectRecord{selected, canonical, rootIdentity(info), map[string]string{rootID: "."}})
	}
	if len(next.Requests) >= 10000 {
		return Project{}, problem("unsupported", "The registration request limit has been reached.")
	}
	next.Requests[requestID] = receipt{Digest: digest, ProjectID: selected.ProjectID}
	if err := s.commitLocked(next); err != nil {
		return Project{}, err
	}
	return selected, nil
}

func (s *store) projects() []Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Project, 0, len(s.data.Projects))
	for _, p := range s.data.Projects {
		out = append(out, p.Project)
	}
	return out
}

func (s *store) project(id string) (projectRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.data.Projects {
		if p.ProjectID == id {
			p.Resources = maps.Clone(p.Resources)
			return p, nil
		}
	}
	return projectRecord{}, problem("not_found", "Project not found.")
}

func (s *store) rememberResources(projectID string, resources map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.copyLocked()
	for i, p := range next.Projects {
		if p.ProjectID == projectID {
			changed := false
			for id, path := range resources {
				if p.Resources[id] != path {
					p.Resources[id] = path
					changed = true
				}
			}
			if !changed {
				return nil
			}
			if len(p.Resources) > 10000 {
				return problem("unsupported", "The Project resource limit has been reached.")
			}
			next.Projects[i] = p
			return s.commitLocked(next)
		}
	}
	return problem("not_found", "Project not found.")
}

func openProject(p projectRecord) (*os.Root, error) {
	root, err := os.OpenRoot(p.Root)
	if err != nil {
		return nil, problem("not_found", "The Project folder is unavailable.")
	}
	info, err := root.Stat(".")
	if err != nil || rootIdentity(info) != p.RootIdentity {
		root.Close()
		return nil, problem("stale_revision", "The Project folder has been replaced. Restore the original folder before continuing.")
	}
	return root, nil
}
