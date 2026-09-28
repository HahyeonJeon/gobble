package appservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const entrySource = "package analysis\nfunc init() { panic(\"registration must not execute Go\") }\nfunc Pipeline() *Example { return nil }\n"

func writeSource(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPipelineRegistrationReadsSourceAndRestoresIdentity(t *testing.T) {
	profile, root := t.TempDir(), t.TempDir()
	writeSource(t, root, "pipeline.go", entrySource)
	writeSource(t, root, "invalid_test.go", "not Go; excluded")
	s, err := openStore(profile)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.registerProject("req_project", root, "Study")
	if err != nil {
		t.Fatal(err)
	}
	registered, err := s.registerPipeline(p.ProjectID, p.RootResourceID, "req_pipeline", "RNA analysis")
	if err != nil {
		t.Fatal(err)
	}
	if !validID(registered.PipelineID, "pip") || registered.Origin.PackageResourceID != p.RootResourceID ||
		registered.Origin.SourceResourceID != resourceID(p.ProjectID, "pipeline.go") {
		t.Fatalf("wrong binding: %+v", registered)
	}
	file, err := s.readFile(p.ProjectID, registered.Origin.SourceResourceID, "")
	if err != nil || file.Content != (textContent{"text", entrySource}) {
		t.Fatalf("source: %+v %v", file, err)
	}
	before, err := os.ReadFile(filepath.Join(root, "pipeline.go"))
	if err != nil || string(before) != entrySource {
		t.Fatal("registration changed source", err)
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	s, err = openStore(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	for _, id := range []string{"req_pipeline", "req_again"} {
		same, err := s.registerPipeline(p.ProjectID, p.RootResourceID, id, "RNA analysis")
		if err != nil || same != registered {
			t.Fatalf("restored repeat: %+v %v", same, err)
		}
	}
	list, err := s.listPipelines(p.ProjectID)
	if err != nil || len(list.Pipelines) != 1 || list.Pipelines[0] != registered {
		t.Fatalf("list: %+v %v", list, err)
	}
	list.Pipelines[0].Name = "caller mutation"
	again, _ := s.listPipelines(p.ProjectID)
	if again.Pipelines[0].Name != registered.Name {
		t.Fatal("caller owns catalog data")
	}
	_, err = s.registerPipeline(p.ProjectID, p.RootResourceID, "req_pipeline", "changed payload")
	wantCode(t, err, "request_conflict")
}

func TestPipelineRegistrationRejectsSourceWithoutPublishingMetadata(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"missing", "package analysis\n", "invalid_request"},
		{"syntax", "package broken\nfunc Pipeline(", "invalid_request"},
		{"parameters", "package a\nfunc Pipeline(n int) int { return n }", "invalid_request"},
		{"generic", "package a\nfunc Pipeline[T any]() T { var t T; return t }", "invalid_request"},
		{"results", "package a\nfunc Pipeline() (int,error) { return 0,nil }", "invalid_request"},
		{"method", "package a\nfunc (x T) Pipeline() int { return 0 }", "invalid_request"},
		{"ambiguous", "package a\nfunc Pipeline() int { return 0 }; func Pipeline() int { return 1 }", "unsupported"},
		{"large", entrySource + "//" + strings.Repeat("a", maxPipelineSourceBytes), "unsupported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testService(t)
			root := t.TempDir()
			writeSource(t, root, "pipeline.go", tc.source)
			p := testProject(t, s, root)
			path := filepath.Join(s.store.dir, "catalog.json")
			before, _ := os.ReadFile(path)
			_, err := s.store.registerPipeline(p.ProjectID, p.RootResourceID, "req_bad", "Analysis")
			wantCode(t, err, tc.code)
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("rejected registration changed catalog")
			}
		})
	}
}

func TestPipelineRegistrationScopeBoundsAndConcurrentDuplicates(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	writeSource(t, root, "pipeline.go", entrySource)
	p := testProject(t, s, root)
	other := testProject(t, s, t.TempDir())
	_, err := s.store.registerPipeline(other.ProjectID, p.RootResourceID, "req_cross", "No")
	wantCode(t, err, "not_found")
	_, err = s.store.listPipelines("prj_unknown")
	wantCode(t, err, "not_found")
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			if _, err := s.store.registerPipeline(p.ProjectID, p.RootResourceID, fmt.Sprintf("req_%d", i), "Analysis"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	list, err := s.store.listPipelines(p.ProjectID)
	if err != nil || len(list.Pipelines) != 1 {
		t.Fatalf("concurrent duplicates: %+v %v", list, err)
	}
	tooMany := t.TempDir()
	p2 := testProject(t, s, tooMany)
	for i := range maxDirectoryEntries + 1 {
		writeSource(t, tooMany, fmt.Sprintf("%04d.txt", i), "")
	}
	_, err = s.store.registerPipeline(p2.ProjectID, p2.RootResourceID, "req_many", "Too many")
	wantCode(t, err, "unsupported")
	linked := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "pipeline.go"), filepath.Join(linked, "pipeline.go")); err != nil {
		t.Fatal(err)
	}
	p3 := testProject(t, s, linked)
	_, err = s.store.registerPipeline(p3.ProjectID, p3.RootResourceID, "req_link", "Outside")
	if err == nil {
		t.Fatal("external source symlink accepted")
	}
}

func TestPipelineHTTPRejectsUnknownFieldsAndKeepsEffectsUnsupported(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	writeSource(t, root, "pipeline.go", entrySource)
	p := testProject(t, s, root)
	handler := s.Handler("127.0.0.1:7777", "test")
	post := func(route, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://127.0.0.1:7777"+route, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer test")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	route := "/v1/projects/" + p.ProjectID + "/pipelines"
	input := fmt.Sprintf(`{"requestId":"req_http","packageResourceId":%q,"name":"Analysis"}`, p.RootResourceID)
	w := post(route, strings.TrimSuffix(input, "}")+`,"command":"go run ."}`)
	if w.Code != 400 {
		t.Fatalf("effect argument accepted: %d %s", w.Code, w.Body.String())
	}
	w = post(route, input)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"pipelineId":"pip_`) || strings.Contains(w.Body.String(), root) {
		t.Fatalf("registration: %d %s", w.Code, w.Body.String())
	}
	for _, action := range []string{"/start", "/validate", "/resume", "/stop"} {
		w = post(route+action, input)
		if w.Code != 422 {
			t.Fatalf("unimplemented effect accepted: %s %d", action, w.Code)
		}
	}
}

func TestCatalogV1MigrationPreservesReceiptsRunsAndOriginalBytes(t *testing.T) {
	dir, root := t.TempDir(), t.TempDir()
	writeSource(t, root, "pipeline.go", entrySource)
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.registerProject("req_project", root, "Study")
	if err != nil {
		t.Fatal(err)
	}
	old := catalogV1{SchemaVersion: 1, Projects: s.data.Projects, Runs: []runRecord{}, Requests: map[string]catalogReceiptV1{}}
	for id, receipt := range s.data.Requests {
		old.Requests[id] = catalogReceiptV1{receipt.Digest, receipt.ProjectID, receipt.RunRef}
	}
	workspaceID := resourceID(p.ProjectID, "runs/prior")
	old.Projects[0].Resources[workspaceID] = "runs/prior"
	old.Runs = append(old.Runs, runRecord{
		RunRegistration: RunRegistration{p.ProjectID, "run_prior", "Prior", workspaceID, "engine-prior"},
		Binding:         RuntimeBinding{"unix:///tmp/docker.sock", "daemon", "sha256:" + strings.Repeat("a", 64), "linux/amd64", "/gobble/project", "/gobble/project/runs/prior"},
		Identity:        json.RawMessage(`{"retained":"engine-owned"}`),
	})
	old.Requests["req_prior"] = catalogReceiptV1{digest([]byte("prior")), p.ProjectID, "run_prior"}
	original, err := json.MarshalIndent(old, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	original = append(original, '\n')
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "catalog.json")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err = openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	current, _ := os.ReadFile(path)
	if !bytes.Equal(current, original) || s.data.SchemaVersion != catalogVersion {
		t.Fatal("read rewrote v1 or failed to migrate in memory")
	}
	if _, err := s.listPipelines(p.ProjectID); err != nil {
		t.Fatal(err)
	}
	registered, err := s.registerPipeline(p.ProjectID, p.RootResourceID, "req_pipe", "Analysis")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(dir, "catalog.v1.backup.json")
	archived, err := os.ReadFile(archivePath)
	if err != nil || !bytes.Equal(archived, original) {
		t.Fatal("original v1 bytes not archived", err)
	}
	if s.data.Runs[0].EngineRunID != "engine-prior" || s.data.Requests["req_prior"].RunRef != "run_prior" ||
		s.data.Requests["req_project"].Digest != old.Requests["req_project"].Digest {
		t.Fatal("migration lost registrations or receipts")
	}
	if _, err := s.registerPipeline(p.ProjectID, p.RootResourceID, "req_later", "Analysis"); err != nil {
		t.Fatal(err)
	}
	archived, _ = os.ReadFile(archivePath)
	if !bytes.Equal(archived, original) {
		t.Fatal("later save replaced v1 archive")
	}
	_, err = s.registerPipeline(p.ProjectID, p.RootResourceID, "req_project", "Analysis")
	wantCode(t, err, "request_conflict")
	current, _ = os.ReadFile(path)
	decoded, migration, err := decodeCatalog(current)
	if err != nil || migration != nil || decoded.Pipelines[0] != registered {
		t.Fatalf("v2 round-trip: %v", err)
	}
}

func TestCatalogMigrationRefusesArchiveConflictAndInvalidPipelineBindings(t *testing.T) {
	dir := t.TempDir()
	legacy := []byte(`{"schemaVersion":1,"projects":[],"runs":[],"requests":{}}`)
	if err := os.WriteFile(filepath.Join(dir, "catalog.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	writeSource(t, dir, "catalog.v1.backup.json", "preserve different archive")
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	_, err = s.registerProject("req_new", t.TempDir(), "New")
	wantCode(t, err, "internal")
	current, _ := os.ReadFile(filepath.Join(dir, "catalog.json"))
	if !bytes.Equal(current, legacy) || len(s.data.Projects) != 0 {
		t.Fatal("failed archive published state")
	}
	for _, body := range []string{
		`{"schemaVersion":1,"projects":[],"runs":[],"pipelines":[],"requests":{}}`,
		`{"schemaVersion":2,"projects":[],"runs":[],"pipelines":null,"requests":{}}`,
		`{"schemaVersion":3,"projects":[],"runs":[],"pipelines":[],"requests":{}}`,
	} {
		if _, _, err := decodeCatalog([]byte(body)); err == nil {
			t.Fatalf("unsupported shape accepted: %s", body)
		}
	}
}
