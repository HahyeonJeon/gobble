package appservice

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
)

const schemaVersion = 1
const maxCatalogBytes = 8 << 20
const maxFileBytes = 8 << 20
const maxDirectoryEntries = 500

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Retry   string `json:"retry"`
}

func (e *apiError) Error() string { return e.Message }
func problem(code, message string) *apiError {
	retry := "never"
	if code == "runtime_unavailable" {
		retry = "after_reconnect"
	}
	if code == "stale_revision" {
		retry = "after_refresh"
	}
	return &apiError{code, message, retry}
}

// Project is presentation metadata. Native root paths stay inside the service.
type Project struct {
	ProjectID      string `json:"projectId"`
	Name           string `json:"name"`
	RootResourceID string `json:"rootResourceId"`
}

type projectRecord struct {
	Project
	Root         string            `json:"root"`
	RootIdentity string            `json:"rootIdentity"`
	Resources    map[string]string `json:"resources"`
}

// RuntimeBinding preserves the exact local daemon, image and container mapping.
type RuntimeBinding struct {
	Endpoint      string `json:"endpoint"`
	DaemonID      string `json:"daemonId"`
	ImageID       string `json:"imageId"`
	Platform      string `json:"platform"`
	ProjectPath   string `json:"projectPath"`
	WorkspacePath string `json:"workspacePath"`
}

// RunRegistration links a Project to one engine Run and execution directory.
// It stores no inferred execution state or Pipeline definition.
type RunRegistration struct {
	ProjectID           string `json:"projectId"`
	RunRef              string `json:"runRef"`
	Name                string `json:"name"`
	WorkspaceResourceID string `json:"workspaceResourceId"`
	EngineRunID         string `json:"engineRunId"`
}

type runRecord struct {
	RunRegistration
	Binding  RuntimeBinding  `json:"binding"`
	Identity json.RawMessage `json:"identity"`
}

type receipt struct {
	DraftID    string `json:"draftId,omitempty"`
	Digest     string `json:"digest"`
	ProjectID  string `json:"projectId"`
	RunRef     string `json:"runRef,omitempty"`
	PipelineID string `json:"pipelineId,omitempty"`
	ProposalID string `json:"proposalId,omitempty"`
}

type catalog struct {
	Drafts        map[string]creationDraft   `json:"drafts"`
	SchemaVersion int                        `json:"schemaVersion"`
	Projects      []projectRecord            `json:"projects"`
	Runs          []runRecord                `json:"runs"`
	Pipelines     []PipelineDefinition       `json:"pipelines"`
	Requests      map[string]receipt         `json:"requests"`
	Revisions     map[string]managedRevision `json:"revisions"`
}

var identifierPattern = regexp.MustCompile(`^[a-z]{3}_[A-Za-z0-9_-]{1,80}$`)
var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func validID(id, prefix string) bool {
	return identifierPattern.MatchString(id) && len(id) > 4 && id[:4] == prefix+"_"
}
func newID(prefix string) string { return prefix + "_" + rand.Text() }
func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func resourceID(projectID, path string) string {
	return "res_" + digest([]byte(projectID + "\x00" + path))[7:]
}
func requestDigest(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("invalid:%T", value)
	}
	return digest(data)
}
