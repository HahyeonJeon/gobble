package appservice

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
)

func (s *Service) creationRoutes(mux *http.ServeMux) {
	prefix := "/v1/projects/{project}/pipeline-drafts"
	mux.HandleFunc("GET "+prefix, func(w http.ResponseWriter, r *http.Request) {
		value, err := s.store.listCreationDrafts(r.PathValue("project"))
		reply(w, value, err)
	})
	mux.HandleFunc("GET "+prefix+"/{draft}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.store.creationDraft(r.PathValue("project"), r.PathValue("draft"))
		reply(w, value, err)
	})
	mux.HandleFunc("POST "+prefix, func(w http.ResponseWriter, r *http.Request) {
		var input createDraftInput
		if err := decodeDraftRequest(w, r, &input, "requestId", "brief"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.store.createDraft(r.PathValue("project"), input)
		reply(w, value, err)
	})
	mux.HandleFunc("POST "+prefix+"/{draft}/update", func(w http.ResponseWriter, r *http.Request) {
		var input updateDraftInput
		if err := decodeDraftRequest(w, r, &input, "requestId", "expectedGeneration", "brief", "resourceId", "readLayout"); err != nil {
			writeProblem(w, err)
			return
		}
		s.mu.Lock()
		value, err := s.store.updateDraft(r.PathValue("project"), r.PathValue("draft"), input)
		if err == nil {
			s.cancelDraftFlightLocked(value)
		}
		s.mu.Unlock()
		reply(w, value, err)
	})
	mux.HandleFunc("POST "+prefix+"/{draft}/discard", func(w http.ResponseWriter, r *http.Request) {
		var input discardDraftInput
		if err := decodeDraftRequest(w, r, &input, "requestId", "expectedGeneration"); err != nil {
			writeProblem(w, err)
			return
		}
		s.mu.Lock()
		value, err := s.store.discardDraft(r.PathValue("project"), r.PathValue("draft"), input)
		if err == nil {
			s.cancelDraftFlightLocked(value)
		}
		s.mu.Unlock()
		reply(w, value, err)
	})
}
func decodeDraftRequest(w http.ResponseWriter, r *http.Request, target any, required ...string) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return problem("invalid_request", "Use application/json for this request.")
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil || validateJSON(raw) != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return problem("invalid_request", "The draft request is invalid or exceeds 64 KiB.")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return problem("invalid_request", "A draft object is required.")
	}
	for _, name := range required {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return problem("invalid_request", "Required draft fields cannot be missing or null.")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil || decoder.Decode(new(any)) != io.EOF {
		return problem("invalid_request", "The draft request contains unsupported content.")
	}
	return nil
}

func (s *Service) cancelDraftFlightLocked(d creationDraft) {
	if f := s.creations[d.DraftID]; f != nil {
		var value creationCandidate
		if readProposalFile(filepath.Join(s.creationDirectory(d.DraftID, f.id), "candidate.json"), &value) == nil && (d.Generation != value.Generation || d.State != "draft") {
			f.cancel()
		}
	}
}
