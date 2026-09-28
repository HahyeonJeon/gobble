package appservice

import (
	"context"
	"net/http"
	"time"
)

func (s *Service) creationAdoptionRoutes(mux *http.ServeMux) {
	prefix := "/v1/projects/{project}/pipeline-drafts/{draft}"
	mux.HandleFunc("POST "+prefix+"/adopt", func(w http.ResponseWriter, r *http.Request) {
		var input adoptCreationInput
		if err := decodeDraftRequest(w, r, &input, "requestId", "candidateId", "artifactId", "expectedGeneration", "name"); err != nil {
			writeProblem(w, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		value, err := s.adoptCreation(ctx, r.PathValue("project"), r.PathValue("draft"), input)
		reply(w, value, err)
	})
	mux.HandleFunc("GET "+prefix+"/adoptions/{request}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.creationAdoptionOutcome(r.PathValue("project"), r.PathValue("draft"), r.PathValue("request"))
		reply(w, value, err)
	})
}
