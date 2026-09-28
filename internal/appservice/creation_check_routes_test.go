package appservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreationRoutesRequireExactScopeAndNativeAuthority(t *testing.T) {
	s, p, scope := creationFixture(t, func(context.Context, []string) ([]byte, error) { return mockCreationCheck(), nil })
	handler := s.Handler("127.0.0.1:12345", "test-secret")
	base := "/v1/projects/" + p.ProjectID + "/pipeline-drafts/" + scope.Draft.DraftID
	cases := []struct {
		method, path, body string
		status             int
	}{
		{"GET", base + "/source?generation=2", "", 200},
		{"GET", base + "/source?generation=1", "", 409},
		{"GET", base + "/source?generation=no", "", 400},
		{"GET", base + "/candidates/req_missing", "", 404},
		{"POST", base + "/candidates", `{"requestId":"req_x","expectedGeneration":2,"scopeId":"` + scope.ScopeID + `","summary":"","files":[],"extra":true}`, 400},
		{"POST", base + "/candidates", `{"requestId":"req_x","requestId":"req_y","expectedGeneration":2,"scopeId":"` + scope.ScopeID + `","summary":"","files":[]}`, 400},
		{"POST", base + "/candidates", `{"requestId":"req_x","expectedGeneration":2,"scopeId":"` + scope.ScopeID + `","summary":"","files":null}`, 400},
		{"POST", base + "/candidates/req_missing/cancel", `{"extra":true}`, 400},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(tc.method, "http://127.0.0.1:12345"+tc.path, strings.NewReader(tc.body))
		r.Header.Set("Authorization", "Bearer test-secret")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Errorf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:12345/v1/creation-runtime", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("unauthenticated runtime access accepted")
	}
	s.mu.Lock()
	s.creations = map[string]*pipelineFlight{"other": {}, "another": {}}
	s.mu.Unlock()
	_, err := s.submitCreation(p.ProjectID, scope.Draft.DraftID, creationCandidateInput{"req_budget", scope.Draft.Generation, scope.ScopeID, "", scope.Files})
	wantCode(t, err, "runtime_unavailable")
}
