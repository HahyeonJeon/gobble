package appservice

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContinuationHTTPRejectsUntrustedAndExecutableInput(t *testing.T) {
	s, p, _, _, _ := checkedContinuationFixture(t)
	handler := s.Handler("127.0.0.1:7777", "test")
	for _, tc := range []struct{ auth, path, body string }{
		{"", "/v1/projects/" + p.ProjectID + "/continuations", `{"requestId":"req_next","launchReviewId":"req_test","runRef":"run_saved"}`},
		{"Bearer test", "/v1/projects/" + p.ProjectID + "/continuations/req_review/confirm", `{"requestId":"req_continue","reviewDigest":"invalid","command":"resume"}`},
		{"Bearer test", "/v1/projects/" + p.ProjectID + "/continuations/req_review/refresh", `{"requestId":"req_replacement"}`},
	} {
		req := httptest.NewRequest("POST", "http://127.0.0.1:7777"+tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", tc.auth)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code < 400 {
			t.Fatal("untrusted request accepted", response.Body.String())
		}
	}
	v, e := s.continuation(p.ProjectID, "req_review")
	if e != nil || v.Operation != nil {
		t.Fatal("HTTP rejection submitted work", e)
	}
}
