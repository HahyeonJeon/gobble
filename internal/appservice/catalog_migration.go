package appservice

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const catalogVersion = 5

// Frozen pre-origin registrations and receipts: new fields cannot enter old catalogs.
type catalogPipelineV3 struct {
	ProjectID         string `json:"projectId"`
	PipelineID        string `json:"pipelineId"`
	Name              string `json:"name"`
	PackageResourceID string `json:"packageResourceId"`
	SourceResourceID  string `json:"sourceResourceId"`
}
type catalogReceiptV3 struct {
	Digest     string `json:"digest"`
	ProjectID  string `json:"projectId"`
	RunRef     string `json:"runRef,omitempty"`
	PipelineID string `json:"pipelineId,omitempty"`
	ProposalID string `json:"proposalId,omitempty"`
}
type catalogV3 struct {
	SchemaVersion int                          `json:"schemaVersion"`
	Projects      []projectRecord              `json:"projects"`
	Runs          []runRecord                  `json:"runs"`
	Pipelines     []catalogPipelineV3          `json:"pipelines"`
	Requests      map[string]catalogReceiptV3  `json:"requests"`
	Revisions     map[string]catalogRevisionV4 `json:"revisions"`
}

func migratePipelines(old []catalogPipelineV3) []PipelineDefinition {
	if old == nil {
		return nil
	}
	out := make([]PipelineDefinition, 0, len(old))
	for _, p := range old {
		out = append(out, PipelineDefinition{p.ProjectID, p.PipelineID, p.Name, PipelineOrigin{"imported", p.PackageResourceID, p.SourceResourceID}})
	}
	return out
}

// Frozen v2 input; new revision and adoption fields cannot enter old documents.
type catalogV2 struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Projects      []projectRecord             `json:"projects"`
	Runs          []runRecord                 `json:"runs"`
	Pipelines     []catalogPipelineV3         `json:"pipelines"`
	Requests      map[string]catalogReceiptV2 `json:"requests"`
}
type catalogReceiptV2 struct {
	Digest     string `json:"digest"`
	ProjectID  string `json:"projectId"`
	RunRef     string `json:"runRef,omitempty"`
	PipelineID string `json:"pipelineId,omitempty"`
}

// These structs retain the v1 decoding boundary. New fields belong on catalog,
// not on this historical input shape.
type catalogV1 struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Projects      []projectRecord             `json:"projects"`
	Runs          []runRecord                 `json:"runs"`
	Requests      map[string]catalogReceiptV1 `json:"requests"`
}
type catalogReceiptV1 struct {
	Digest    string `json:"digest"`
	ProjectID string `json:"projectId"`
	RunRef    string `json:"runRef,omitempty"`
}

func decodeCatalog(data []byte) (catalog, []byte, error) {
	var header struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return catalog{}, nil, err
	}
	decode := func(value any) error {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(value); err != nil {
			return err
		}
		if decoder.Decode(new(any)) != io.EOF {
			return errors.New("trailing catalog content")
		}
		return nil
	}
	var current catalog
	var original []byte
	switch header.SchemaVersion {
	case 1:
		var old catalogV1
		if err := decode(&old); err != nil {
			return catalog{}, nil, err
		}
		current = catalog{SchemaVersion: catalogVersion, Projects: old.Projects, Runs: old.Runs, Pipelines: []PipelineDefinition{}, Revisions: map[string]managedRevision{}}
		if old.Requests != nil {
			current.Requests = make(map[string]receipt, len(old.Requests))
			for id, r := range old.Requests {
				current.Requests[id] = receipt{Digest: r.Digest, ProjectID: r.ProjectID, RunRef: r.RunRef}
			}
		}
		original = bytes.Clone(data)
	case 2:
		var old catalogV2
		if err := decode(&old); err != nil {
			return catalog{}, nil, err
		}
		current = catalog{SchemaVersion: catalogVersion, Projects: old.Projects, Runs: old.Runs, Pipelines: migratePipelines(old.Pipelines), Revisions: map[string]managedRevision{}}
		if old.Requests != nil {
			current.Requests = map[string]receipt{}
			for id, v := range old.Requests {
				current.Requests[id] = receipt{Digest: v.Digest, ProjectID: v.ProjectID, RunRef: v.RunRef, PipelineID: v.PipelineID}
			}
		}
		original = bytes.Clone(data)
	case 3:
		var old catalogV3
		if err := decode(&old); err != nil {
			return catalog{}, nil, err
		}
		current = catalog{SchemaVersion: catalogVersion, Projects: old.Projects, Runs: old.Runs, Pipelines: migratePipelines(old.Pipelines)}
		if old.Requests != nil {
			current.Requests = map[string]receipt{}
			for id, r := range old.Requests {
				current.Requests[id] = receipt{Digest: r.Digest, ProjectID: r.ProjectID, RunRef: r.RunRef, PipelineID: r.PipelineID, ProposalID: r.ProposalID}
			}
		}
		if old.Revisions != nil && len(old.Revisions) <= 100 {
			current.Revisions = map[string]managedRevision{}
			for projectID, revision := range old.Revisions {
				owner := ""
				for _, p := range old.Pipelines {
					if p.PipelineID == revision.PipelineID {
						owner = p.ProjectID
					}
				}
				if owner == "" || owner != projectID {
					return catalog{}, nil, errors.New("invalid legacy revision owner")
				}
				current.Revisions[revision.PipelineID] = managedRevision{PipelineID: revision.PipelineID, ProposalID: revision.ProposalID, Artifact: revision.Artifact}
			}
		}
		original = bytes.Clone(data)
	case 4:
		var old catalogV4
		if err := decode(&old); err != nil {
			return catalog{}, nil, err
		}
		current = catalog{SchemaVersion: catalogVersion, Projects: old.Projects, Runs: old.Runs, Pipelines: old.Pipelines, Requests: old.Requests}
		if old.Revisions != nil {
			current.Revisions = map[string]managedRevision{}
			for id, r := range old.Revisions {
				current.Revisions[id] = managedRevision{PipelineID: r.PipelineID, ProposalID: r.ProposalID, Artifact: r.Artifact}
			}
		}
		if old.Drafts != nil {
			current.Drafts = map[string]creationDraft{}
			for id, d := range old.Drafts {
				if d.State != "draft" && d.State != "discarded" {
					return catalog{}, nil, errors.New("invalid historical draft state")
				}
				current.Drafts[id] = creationDraft{ProjectID: d.ProjectID, DraftID: d.DraftID, Generation: d.Generation, State: d.State, Brief: d.Brief, Input: d.Input}
			}
		}
		original = bytes.Clone(data)
	case catalogVersion:
		if err := decode(&current); err != nil {
			return catalog{}, nil, err
		}
	default:
		return catalog{}, nil, errors.New("unsupported catalog version")
	}
	if header.SchemaVersion < 4 {
		current.Drafts = map[string]creationDraft{}
	}
	if err := validateCatalog(current); err != nil {
		return catalog{}, nil, err
	}
	return current, original, nil
}

func validatePipelines(c catalog, projects map[string]projectRecord) (map[string]string, error) {
	if c.Pipelines == nil || len(c.Pipelines) > maxPipelines {
		return nil, errors.New("invalid pipelines")
	}
	owners := make(map[string]string, len(c.Pipelines))
	packages := make(map[string]bool, len(c.Pipelines))
	for _, p := range c.Pipelines {
		project, ok := projects[p.ProjectID]
		if !ok || !validID(p.PipelineID, "pip") || !utf8.ValidString(p.Name) || strings.TrimSpace(p.Name) == "" || len(p.Name) > 200 {
			return nil, errors.New("invalid pipeline")
		}
		if owners[p.PipelineID] != "" {
			return nil, errors.New("duplicate pipeline")
		}
		switch p.Origin.Kind {
		case "imported":
			pkg, pkgOK := project.Resources[p.Origin.PackageResourceID]
			src, srcOK := project.Resources[p.Origin.SourceResourceID]
			key := p.ProjectID + ":" + p.Origin.PackageResourceID
			if !pkgOK || !srcOK || filepath.ToSlash(filepath.Dir(src)) != pkg || !strings.HasSuffix(src, ".go") || strings.HasSuffix(src, "_test.go") || packages[key] {
				return nil, errors.New("invalid imported pipeline binding")
			}
			packages[key] = true
		case "managed":
			if p.Origin.PackageResourceID != "" || p.Origin.SourceResourceID != "" || c.Revisions[p.PipelineID].PipelineID != p.PipelineID {
				return nil, errors.New("invalid managed pipeline binding")
			}
		default:
			return nil, errors.New("invalid pipeline origin")
		}
		owners[p.PipelineID] = p.ProjectID
	}
	return owners, nil
}

// Archive exactly once; a different existing archive is preserved, never replaced.
func archiveHistoricalCatalog(dir string, original []byte) error {
	var header struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if json.Unmarshal(original, &header) != nil || (header.SchemaVersion < 1 || header.SchemaVersion > 4) {
		return errors.New("invalid migration archive")
	}
	name := fmt.Sprintf("catalog.v%d.backup.json", header.SchemaVersion)
	path := filepath.Join(dir, name)
	old, err := readBoundedFile(path, maxCatalogBytes)
	if err == nil {
		if !bytes.Equal(old, original) {
			return errors.New("existing catalog archive differs")
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// The catalog lock owns this directory; publication is atomic as for primary writes.
	return atomicWrite(dir, name, original)
}

// Frozen v4 storage shapes. Creation birth ownership must not enter older catalogs.
type catalogRevisionV4 struct {
	PipelineID string           `json:"pipelineId"`
	ProposalID string           `json:"proposalId"`
	Artifact   pipelineArtifact `json:"artifact"`
}
type catalogDraftV4 struct {
	ProjectID  string         `json:"projectId"`
	DraftID    string         `json:"draftId"`
	Generation int64          `json:"generation"`
	State      string         `json:"state"`
	Brief      string         `json:"brief"`
	Input      *creationInput `json:"input,omitempty"`
}
type catalogV4 struct {
	SchemaVersion int                          `json:"schemaVersion"`
	Projects      []projectRecord              `json:"projects"`
	Runs          []runRecord                  `json:"runs"`
	Pipelines     []PipelineDefinition         `json:"pipelines"`
	Requests      map[string]receipt           `json:"requests"`
	Revisions     map[string]catalogRevisionV4 `json:"revisions"`
	Drafts        map[string]catalogDraftV4    `json:"drafts"`
}
