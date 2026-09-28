package appservice

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// These authenticated native routes are not renderer IPC or Agent tools. Their
// message/authoring policy will be connected by Main in the next part.
func (s *Service) creationCheckRoutes(mux *http.ServeMux) {
	s.registerCreationEngines(mux)
	s.creationAdoptionRoutes(mux)
	mux.HandleFunc("POST /v1/creation-runtime", func(w http.ResponseWriter, r *http.Request) {
		var input RuntimeBinding
		if err := decodeDraftRequest(w, r, &input, "endpoint", "daemonId", "imageId", "platform", "projectPath", "workspacePath"); err != nil {
			writeProblem(w, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		value, err := s.bindCreationRuntime(ctx, input)
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/creation-runtime", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		value, err := s.readCreationRuntime()
		s.mu.Unlock()
		reply(w, value, err)
	})
	prefix := "/v1/projects/{project}/pipeline-drafts/{draft}"
	mux.HandleFunc("GET "+prefix+"/source", func(w http.ResponseWriter, r *http.Request) {
		generation, err := strconv.ParseInt(r.URL.Query().Get("generation"), 10, 64)
		if err != nil {
			writeProblem(w, problem("invalid_request", "A draft generation is required."))
			return
		}
		value, err := s.creationAuthoring(r.PathValue("project"), r.PathValue("draft"), generation)
		reply(w, value, err)
	})
	mux.HandleFunc("GET "+prefix+"/candidates", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.listCreationCandidates(r.PathValue("project"), r.PathValue("draft"))
		reply(w, value, err)
	})
	mux.HandleFunc("POST "+prefix+"/candidates", func(w http.ResponseWriter, r *http.Request) {
		var input creationCandidateInput
		if err := decodeDraftRequest(w, r, &input, "requestId", "expectedGeneration", "scopeId", "summary", "files"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.submitCreation(r.PathValue("project"), r.PathValue("draft"), input)
		reply(w, value, err)
	})
	mux.HandleFunc("GET "+prefix+"/candidates/{candidate}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.creationCandidate(r.PathValue("project"), r.PathValue("draft"), r.PathValue("candidate"))
		reply(w, value, err)
	})
	mux.HandleFunc("POST "+prefix+"/candidates/{candidate}/cancel", func(w http.ResponseWriter, r *http.Request) {
		var input struct{}
		if err := decodeDraftRequest(w, r, &input); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.cancelCreation(r.PathValue("project"), r.PathValue("draft"), r.PathValue("candidate"))
		reply(w, value, err)
	})
}
