package appservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func followUpFixture(t *testing.T) (*Service, Project, launchRecord, runFollowUpInput) {
	t.Helper()
	s, p, rec := readyLaunchFixture(t)
	rec.Value.State = "accepted"
	rec.Value.InputSHA256 = digest([]byte("original input"))
	rec.Value.Operation = &launchOperation{RequestID: "req_start", State: "admitted", RunRef: "run_origin", RunStatus: "succeeded"}
	if err := s.saveLaunch(rec); err != nil {
		t.Fatal(err)
	}
	s.store.mu.Lock()
	s.store.data.Runs = append(s.store.data.Runs, runRecord{RunRegistration: RunRegistration{ProjectID: p.ProjectID, RunRef: "run_origin", Name: "Analysis 12"}})
	s.store.mu.Unlock()
	hash := digest([]byte("captured evidence"))
	raw, _ := json.Marshal(map[string]any{"attachmentId": "att_origin", "evidence": map[string]any{"projectId": p.ProjectID, "dataRevision": "observed-original", "resource": map[string]any{"kind": "run", "runRef": "run_origin"}}, "asset": map[string]any{"hash": hash}, "capture": map[string]any{"kind": "observed", "asset": map[string]any{"hash": hash}}, "representation": map[string]any{"kind": "run"}})
	return s, p, rec, runFollowUpInput{rec.Value.RequestID, "req_sent", []json.RawMessage{raw}}
}
func TestFollowUpRejectsWrongAssociationAndForgedEvidence(t *testing.T) {
	s, p, rec, input := followUpFixture(t)
	v, err := s.resolveRunFollowUpLocked(p.ProjectID, rec.Value.PipelineID, &input)
	if err != nil || v.RunRef != "run_origin" || v.InputSHA256 != rec.Value.InputSHA256 {
		t.Fatal(v, err)
	}
	_, err = s.resolveRunFollowUpLocked(p.ProjectID, "pip_other", &input)
	wantCode(t, err, "invalid_request")
	for _, field := range []string{"project", "run", "hash", "capture", "duplicate", "missing", "oversize", "representation"} {
		t.Run(field, func(t *testing.T) {
			bad := input
			bad.Evidence = append([]json.RawMessage{}, input.Evidence...)
			var item map[string]any
			_ = json.Unmarshal(bad.Evidence[0], &item)
			target := item["evidence"].(map[string]any)
			switch field {
			case "project":
				target["projectId"] = "prj_elsewhere"
			case "run":
				target["resource"].(map[string]any)["runRef"] = "run_elsewhere"
			case "hash":
				item["asset"].(map[string]any)["hash"] = "sha256:bad"
			case "capture":
				delete(item, "capture")
			case "duplicate":
				bad.Evidence = append(bad.Evidence, bad.Evidence[0])
			case "representation":
				item["representation"].(map[string]any)["kind"] = "log"
			case "missing":
				bad.SubmissionID = ""
			case "oversize":
				item["extra"] = make([]byte, 50<<10)
			}
			bad.Evidence[0], _ = json.Marshal(item)
			_, err := s.resolveRunFollowUpLocked(p.ProjectID, rec.Value.PipelineID, &bad)
			wantCode(t, err, "invalid_request")
		})
	}
	rec.Value.Operation.RunRef = ""
	_ = s.saveLaunch(rec)
	_, err = s.resolveRunFollowUpLocked(p.ProjectID, rec.Value.PipelineID, &input)
	wantCode(t, err, "invalid_request")
}
func TestFollowUpStoredWithProposalAndInheritedOnlyFromExactCurrent(t *testing.T) {
	s, p, origin, input := followUpFixture(t)
	v, err := s.resolveRunFollowUpLocked(p.ProjectID, origin.Value.PipelineID, &input)
	if err != nil {
		t.Fatal(err)
	}
	pipeline := origin.Value.PipelineID
	artifact := pipelineArtifact{ArtifactID: digest([]byte("adopted"))}
	proposal := pipelineProposal{ProjectID: p.ProjectID, PipelineID: pipeline, ProposalID: "req_proposal", State: "ready", Proposed: &artifact, FollowUp: v}
	dir := s.proposalDirectory(pipeline, proposal.ProposalID)
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err = writeInspectionJSON(filepath.Join(dir, "review.json"), proposal); err != nil {
		t.Fatal(err)
	}
	s.store.mu.Lock()
	s.store.data.Revisions[pipeline] = managedRevision{PipelineID: pipeline, ProposalID: proposal.ProposalID, Artifact: artifact}
	s.store.mu.Unlock()
	// A new service object reads the saved proposal; no transient link map exists.
	reader := &Service{store: s.store}
	got, err := reader.currentFollowUpLocked(p.ProjectID, pipeline, artifact.ArtifactID)
	if err != nil || got == nil || got.SubmissionID != "req_sent" {
		t.Fatal(got, err)
	}
	_, err = reader.currentFollowUpLocked(p.ProjectID, pipeline, origin.Value.Preparation.ArtifactID)
	wantCode(t, err, "stale_revision")
	origin.Value.FollowUp = got
	if err = s.saveLaunch(origin); err != nil {
		t.Fatal(err)
	}
	restored, err := reader.readLaunch(p.ProjectID, origin.Value.RequestID)
	if err != nil || restored.Value.FollowUp.RunRef != "run_origin" {
		t.Fatal(err)
	}
	// Checksum covers provenance as well as execution metadata.
	restored.Value.FollowUp.SubmissionID = "req_forged"
	_ = writeInspectionJSON(filepath.Join(s.launchDirectory(p.ProjectID, origin.Value.RequestID), "record.json"), restored)
	_, err = reader.readLaunch(p.ProjectID, origin.Value.RequestID)
	wantCode(t, err, "internal")
}

// A replay keeps the original sent-message origin even after Current advances.
func TestFollowUpProposalReplayRejectsChangedOrigin(t *testing.T) {
	s, p, origin, input := followUpFixture(t)
	value, err := s.resolveRunFollowUpLocked(p.ProjectID, origin.Value.PipelineID, &input)
	if err != nil {
		t.Fatal(err)
	}
	request := proposalInput{FollowUp: &input, RequestID: "req_replay", BaseArtifactID: origin.Value.Preparation.ArtifactID, Summary: "Supported refinement"}
	proposal := pipelineProposal{ProjectID: p.ProjectID, PipelineID: origin.Value.PipelineID, ProposalID: request.RequestID, State: "ready", Digest: requestDigest(request), FollowUp: value}
	dir := s.proposalDirectory(proposal.PipelineID, proposal.ProposalID)
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err = writeInspectionJSON(filepath.Join(dir, "review.json"), proposal); err != nil {
		t.Fatal(err)
	}
	got, err := s.proposePipeline(p.ProjectID, proposal.PipelineID, request)
	if err != nil || got.FollowUp.SubmissionID != input.SubmissionID {
		t.Fatal(got, err)
	}
	changed := input
	changed.SubmissionID = "req_other"
	request.FollowUp = &changed
	_, err = s.proposePipeline(p.ProjectID, proposal.PipelineID, request)
	wantCode(t, err, "request_conflict")
}
