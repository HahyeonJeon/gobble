package appservice

import (
	"context"
	"net/http"
	"time"
)

func (s *Service) preparationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/preparation-engines", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		bindings, err := s.analysisEngineBindings(ctx, "io.gobble.preparation.scope=single-end-trim-fastqc-v1")
		out := []creationEngine{}
		for _, b := range bindings {
			out = append(out, creationEngine{b.ImageID, "Local engine · " + b.ImageID[7:19]})
		}
		reply(w, out, err)
	})
	const base = "/v1/projects/{project}/pipelines/{pipeline}/preparations"
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		v, e := s.preparationsList(r.PathValue("project"), r.PathValue("pipeline"))
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base, func(w http.ResponseWriter, r *http.Request) {
		var in prepareInput
		if e := decodeRequest(w, r, &in, "requestId", "artifactId", "engineId"); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.preparePipeline(r.PathValue("project"), r.PathValue("pipeline"), in)
		reply(w, v, e)
	})
	mux.HandleFunc("GET "+base+"/{request}", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.preparation(r.PathValue("project"), r.PathValue("pipeline"), r.PathValue("request"))
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{request}/cancel", func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if e := decodeRequest(w, r, &in); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.cancelPreparation(r.PathValue("project"), r.PathValue("pipeline"), r.PathValue("request"))
		reply(w, v, e)
	})
}
