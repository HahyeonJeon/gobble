package appservice

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func (s *Service) proposalRoutes(mux *http.ServeMux) {
	prefix := "/v1/projects/{project}/pipelines/{pipeline}"
	mux.HandleFunc("GET "+prefix+"/proposal-source", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.pipelineProposalSource(r.PathValue("project"), r.PathValue("pipeline"), r.URL.Query().Get("baseArtifactId"))
		reply(w, value, err)
	})
	mux.HandleFunc("GET "+prefix+"/proposals", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.pipelineProposals(r.PathValue("project"), r.PathValue("pipeline"))
		reply(w, value, err)
	})
	mux.HandleFunc("POST "+prefix+"/proposals", func(w http.ResponseWriter, r *http.Request) {
		var input proposalInput
		if err := decodeProposalRequest(w, r, &input); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.proposePipeline(r.PathValue("project"), r.PathValue("pipeline"), input)
		reply(w, value, err)
	})
	mux.HandleFunc("POST "+prefix+"/adopt", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RequestID  string `json:"requestId"`
			ProposalID string `json:"proposalId"`
			ArtifactID string `json:"artifactId"`
		}
		if err := decodeRequest(w, r, &input, "requestId", "proposalId", "artifactId"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.adoptPipeline(r.Context(), r.PathValue("project"), r.PathValue("pipeline"), input.ProposalID, input.RequestID, input.ArtifactID)
		reply(w, value, err)
	})
	mux.HandleFunc("GET "+prefix+"/adoptions/{request}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.adoptionOutcome(r.PathValue("project"), r.PathValue("pipeline"), r.PathValue("request"))
		reply(w, value, err)
	})
}
func decodeProposalRequest(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return problem("invalid_request", "Use application/json for this request.")
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil || validateJSON(raw) != nil {
		return problem("invalid_request", "The proposal is invalid or exceeds its 64 KiB limit.")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil || decoder.Decode(new(any)) != io.EOF {
		return problem("invalid_request", "The proposal contains unsupported content.")
	}
	return nil
}
