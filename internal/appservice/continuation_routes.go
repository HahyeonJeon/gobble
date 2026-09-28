package appservice

import "net/http"

func (s *Service) continuationRoutes(mux *http.ServeMux) {
	const base = "/v1/projects/{project}/continuations"
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		v, e := s.continuations(r.PathValue("project"))
		reply(w, v, e)
	})
	mux.HandleFunc("GET "+base+"/{review}", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.continuation(r.PathValue("project"), r.PathValue("review"))
		reply(w, v, e)
	})
	mux.HandleFunc("GET /v1/projects/{project}/launch-reviews/{launch}/continuation-support", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.continuationSupport(r.Context(), r.PathValue("project"), r.PathValue("launch"))
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base, func(w http.ResponseWriter, r *http.Request) {
		var in continuationCheckInput
		if e := decodeRequest(w, r, &in, "requestId", "launchReviewId", "runRef"); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.checkContinuation(r.PathValue("project"), in)
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{review}/confirm", func(w http.ResponseWriter, r *http.Request) {
		var in continuationConfirmInput
		if e := decodeRequest(w, r, &in, "requestId", "reviewDigest"); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.confirmContinuation(r.PathValue("project"), r.PathValue("review"), in)
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{review}/refresh", func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if e := decodeRequest(w, r, &in); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.refreshContinuation(r.PathValue("project"), r.PathValue("review"))
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{review}/stop", func(w http.ResponseWriter, r *http.Request) {
		var in launchStopInput
		if e := decodeRequest(w, r, &in, "requestId", "expectedLease"); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.stopContinuation(r.PathValue("project"), r.PathValue("review"), in)
		reply(w, v, e)
	})
}
