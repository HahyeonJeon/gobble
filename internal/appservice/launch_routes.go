package appservice

import "net/http"

func (s *Service) launchRoutes(mux *http.ServeMux) {
	const base = "/v1/projects/{project}/launch-reviews"
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		v, e := s.launchReviews(r.PathValue("project"))
		reply(w, v, e)
	})
	mux.HandleFunc("GET "+base+"/{review}", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.launchView(r.PathValue("project"), r.PathValue("review"))
		reply(w, v, e)
	})
	mux.HandleFunc("POST /v1/projects/{project}/pipelines/{pipeline}/launch-reviews", func(w http.ResponseWriter, r *http.Request) {
		var in launchReviewInput
		if e := decodeRequest(w, r, &in, "requestId", "preparationId"); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.checkLaunch(r.PathValue("project"), r.PathValue("pipeline"), in)
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{review}/cancel", func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if e := decodeRequest(w, r, &in); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.cancelLaunch(r.PathValue("project"), r.PathValue("review"))
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{review}/start", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			RequestID string `json:"requestId"`
		}
		if e := decodeRequest(w, r, &in, "requestId"); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.startLaunch(r.PathValue("project"), launchStartInput{in.RequestID, r.PathValue("review")})
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{review}/refresh", func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if e := decodeRequest(w, r, &in); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.refreshLaunch(r.PathValue("project"), r.PathValue("review"))
		reply(w, v, e)
	})
	mux.HandleFunc("POST "+base+"/{review}/stop", func(w http.ResponseWriter, r *http.Request) {
		var in launchStopInput
		if e := decodeRequest(w, r, &in, "requestId", "expectedLease"); e != nil {
			writeProblem(w, e)
			return
		}
		v, e := s.stopLaunch(r.PathValue("project"), r.PathValue("review"), in)
		reply(w, v, e)
	})
}
