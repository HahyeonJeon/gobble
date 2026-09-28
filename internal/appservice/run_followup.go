package appservice

import (
	"encoding/json"
	"regexp"
)

// Evidence contains sent manifests, never log/file bytes. Main owns validation
// and content-addressed storage; native owns the Pipeline/Run association.
type runFollowUpInput struct {
	LaunchReviewID string            `json:"launchReviewId"`
	SubmissionID   string            `json:"submissionId"`
	Evidence       []json.RawMessage `json:"evidence"`
}
type runFollowUp struct {
	runFollowUpInput
	SchemaVersion int    `json:"schemaVersion"`
	ProjectID     string `json:"projectId"`
	PipelineID    string `json:"pipelineId"`
	RunRef        string `json:"runRef"`
	RunName       string `json:"runName"`
	PreparationID string `json:"preparationId"`
	ArtifactID    string `json:"artifactId"`
	InputSHA256   string `json:"inputSHA256"`
}

var followUpAttachmentID = regexp.MustCompile(`^att_[A-Za-z0-9_-]{1,128}$`)

// Called under Service.mu before persisting the immutable proposal. Replaying a
// saved proposal returns its original origin, independently of later Current.
func (s *Service) resolveRunFollowUpLocked(project, pipeline string, input *runFollowUpInput) (*runFollowUp, error) {
	if input == nil {
		return nil, nil
	}
	if !validID(input.SubmissionID, "req") || len(input.Evidence) == 0 || len(input.Evidence) > 16 {
		return nil, problem("invalid_request", "A follow-up requires a sent message and bounded Run evidence.")
	}
	origin, err := s.readLaunch(project, input.LaunchReviewID)
	if err != nil {
		return nil, err
	}
	v := origin.Value
	if v.PipelineID != pipeline || v.State != "accepted" || v.Operation == nil || v.Operation.RunRef == "" ||
		!digestPattern.MatchString(v.InputSHA256) || v.Preparation.State != "ready" {
		return nil, problem("invalid_request", "The origin must be an admitted analysis of this Pipeline.")
	}
	var name string
	s.store.mu.Lock()
	for _, r := range s.store.data.Runs {
		if r.ProjectID == project && r.RunRef == v.Operation.RunRef {
			name = r.Name
			break
		}
	}
	s.store.mu.Unlock()
	if name == "" {
		return nil, problem("not_found", "The origin analysis is no longer registered.")
	}
	seen := map[string]bool{}
	size := 0
	for _, raw := range input.Evidence {
		size += len(raw)
		var item struct {
			AttachmentID string `json:"attachmentId"`
			Evidence     struct {
				ProjectID    string `json:"projectId"`
				DataRevision string `json:"dataRevision"`
				Resource     struct {
					Kind   string `json:"kind"`
					RunRef string `json:"runRef"`
				} `json:"resource"`
			} `json:"evidence"`
			Asset struct {
				Hash string `json:"hash"`
			} `json:"asset"`
			Capture struct {
				Kind  string `json:"kind"`
				Asset struct {
					Hash string `json:"hash"`
				} `json:"asset"`
			} `json:"capture"`
			Representation struct {
				Kind string `json:"kind"`
			} `json:"representation"`
		}
		if size > 48<<10 || json.Unmarshal(raw, &item) != nil || !followUpAttachmentID.MatchString(item.AttachmentID) || seen[item.AttachmentID] ||
			item.Evidence.ProjectID != project || item.Evidence.Resource.RunRef != v.Operation.RunRef ||
			(item.Evidence.Resource.Kind != "run" && item.Evidence.Resource.Kind != "log") ||
			item.Evidence.DataRevision == "" || len(item.Evidence.DataRevision) > 256 ||
			!digestPattern.MatchString(item.Asset.Hash) || item.Capture.Kind != "observed" || item.Capture.Asset.Hash != item.Asset.Hash ||
			item.Representation.Kind != item.Evidence.Resource.Kind {
			return nil, problem("invalid_request", "Follow-up evidence must identify the same captured Run within its size limit.")
		}
		seen[item.AttachmentID] = true
	}
	return &runFollowUp{*input, 1, project, pipeline, v.Operation.RunRef, name, v.PreparationID, v.Preparation.ArtifactID, v.InputSHA256}, nil
}

// Current's proposal is the durable adoption association. Copy provenance into
// the initial launch record before the worker starts; no post-admission link write.
func (s *Service) currentFollowUpLocked(project, pipeline, artifact string) (*runFollowUp, error) {
	s.store.mu.Lock()
	revision, ok := s.store.data.Revisions[pipeline]
	s.store.mu.Unlock()
	if !ok || revision.ProposalID == "" {
		return nil, nil
	}
	if revision.Artifact.ArtifactID != artifact {
		return nil, problem("stale_revision", "Current changed before the follow-up check.")
	}
	proposal, err := s.readProposalLocked(project, pipeline, revision.ProposalID)
	if err != nil {
		return nil, err
	}
	if proposal.Proposed == nil || proposal.Proposed.ArtifactID != artifact {
		return nil, problem("internal", "The adopted proposal association is inconsistent.")
	}
	return proposal.FollowUp, nil
}
