package appservice

import "path/filepath"

// Verify the exact retained artifact; adopting cannot turn incomplete evidence into Current.
func (s *Service) checkedCreationSource(v creationCandidate) (creationManifest, error) {
	var m creationManifest
	if v.State != "ready" || v.Artifact == nil || len(v.Artifact.Check.Gaps) != 0 || validateCreationCheck(v.Artifact.Check) != nil {
		return m, problem("unsupported", "Only a completely checked creation proposal can be adopted.")
	}
	dir := s.creationDirectory(v.DraftID, v.CandidateID)
	if readProposalFile(filepath.Join(dir, "manifest.json"), &m) != nil || m.SchemaVersion != 1 || m.Draft.ProjectID != v.ProjectID || m.Draft.DraftID != v.DraftID || m.Draft.Generation != v.Generation || m.Draft.Input == nil || requestDigest(m) != v.Artifact.SourceRevision || v.Artifact.ArtifactID != requestDigest([]any{v.Artifact.SourceRevision, v.Artifact.Check}) || *m.Draft.Input != v.Artifact.Input || m.Runtime.ID != v.Artifact.RuntimeID || m.InputPath != v.Artifact.Check.InputPath {
		return m, problem("internal", "The retained creation evidence is inconsistent. Preserve the profile.")
	}
	return m, verifyCreationSource(filepath.Join(dir, "project"), m)
}

// Adapt the retained birth source to the existing refinement reader without copying
// bytes, fabricating a before-version, or rewriting the birth artifact identity.
func (s *Service) creationProposalBaseLocked(projectID, pipelineID string, revision managedRevision) (proposalBase, error) {
	origin := revision.Creation
	if origin == nil {
		return proposalBase{}, problem("internal", "Creation source is unavailable.")
	}
	candidate, err := s.readCreationCandidateLocked(projectID, origin.DraftID, origin.CandidateID)
	if err != nil {
		return proposalBase{}, err
	}
	m, err := s.checkedCreationSource(candidate)
	if err != nil {
		return proposalBase{}, err
	}
	if revision.Artifact.ArtifactID != candidate.Artifact.ArtifactID || revision.Artifact.SourceRevision != candidate.Artifact.SourceRevision || !sameReviewJSON(revision.Artifact.Flow, candidate.Artifact.Check.Review.Flow) {
		return proposalBase{}, problem("internal", "Current no longer matches its creation source.")
	}
	manifest := inspectionSource{SchemaVersion: 1, ProjectID: projectID, PipelineID: pipelineID, Package: m.Runtime.Scaffold.Package, Files: append([]inspectionFile{}, m.Files...), Runtime: m.Runtime.Binding}
	return proposalBase{Artifact: revision.Artifact, Manifest: manifest, Directory: filepath.Join(s.creationDirectory(origin.DraftID, origin.CandidateID), "project"), Managed: true}, nil
}
