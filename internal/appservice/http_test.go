package appservice

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPRejectsUntrustedRequestsBeforeSideEffects(t *testing.T) {
	s := testService(t)
	token := strings.Repeat("a", 64)
	handler := s.Handler("127.0.0.1:7777", token)
	for _, tc := range []struct {
		name, host, auth, origin string
		hasOrigin                bool
	}{
		{"missing token", "127.0.0.1:7777", "", "", false},
		{"wrong token", "127.0.0.1:7777", "Bearer wrong", "", false},
		{"browser origin", "127.0.0.1:7777", "Bearer " + token, "https://attacker.test", true},
		{"empty origin", "127.0.0.1:7777", "Bearer " + token, "", true},
		{"wrong host", "attacker.test:7777", "Bearer " + token, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "http://127.0.0.1:7777/v1/projects", strings.NewReader(`{"requestId":"req_bad","root":"/private"}`))
			r.Host = tc.host
			r.Header.Set("Authorization", tc.auth)
			r.Header.Set("Content-Type", "application/json")
			if tc.hasOrigin {
				r.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != 403 || !strings.Contains(w.Body.String(), `"code":"forbidden"`) {
				t.Fatalf("got %d %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Access-Control-Allow-Origin") != "" || strings.Contains(w.Body.String(), token) {
				t.Fatal("CORS or secret exposure")
			}
		})
	}
	if len(s.store.projects()) != 0 {
		t.Fatal("unauthorized request changed the catalog")
	}
}

func TestHTTPRejectsAmbiguousAndOversizedBodies(t *testing.T) {
	s := testService(t)
	handler := s.Handler("127.0.0.1:7777", "test")
	for _, body := range []string{
		`null`, `[]`, `{"requestId":"req_a","requestId":"req_b","root":"/tmp"}`,
		`{"RequestID":"req_a","root":"/tmp"}`, `{"requestId":"req_a","root":"/tmp","shell":"ls"}`,
		`{"requestId":"req_a","root":42}`, `{"requestId":"req_a","root":"/tmp"} {}`,
		`{"requestId":"req_a","root":"/tmp","name":null}`,
		`{"name":"` + strings.Repeat("a", 17<<10) + `"}`,
	} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:7777/v1/projects", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer test")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 400 || !strings.Contains(w.Body.String(), `"code":"invalid_request"`) {
			t.Fatalf("got %d %s", w.Code, w.Body.String())
		}
	}
}

func TestHTTPProjectRoundTripAndUnsupportedMutations(t *testing.T) {
	s := testService(t)
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = s.Handler(server.Listener.Addr().String(), "test")
	server.Start()
	defer server.Close()
	body, _ := json.Marshal(map[string]string{"requestId": "req_http", "root": t.TempDir(), "name": "Atlas"})
	request, _ := http.NewRequest("POST", server.URL+"/v1/projects", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer test")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result struct {
		OK    bool    `json:"ok"`
		Value Project `json:"value"`
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil || !result.OK || result.Value.Name != "Atlas" {
		t.Fatalf("registration: %+v %v", result, err)
	}
	for _, path := range []string{"/v1/projects/" + result.Value.ProjectID + "/runs/start", "/v1/stop", "/v2/projects"} {
		request, _ = http.NewRequest("POST", server.URL+path, nil)
		request.Header.Set("Authorization", "Bearer test")
		response, err = http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != 422 || !bytes.Contains(raw, []byte(`"code":"unsupported"`)) {
			t.Fatalf("unexpected mutation route: %d %s", response.StatusCode, raw)
		}
	}
}
