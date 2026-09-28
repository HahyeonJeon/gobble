package appservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func draftFixture(t *testing.T) (*Service, Project, string, creationDraft) {
	t.Helper()
	s := testService(t)
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "reads.fastq.gz"), []byte("metadata only; not valid FASTQ"))
	p := testProject(t, s, root)
	file := findFile(t, s, p, "reads.fastq.gz")
	d, err := s.store.createDraft(p.ProjectID, createDraftInput{"req_draft", "Read quality"})
	if err != nil {
		t.Fatal(err)
	}
	return s, p, file, d
}
func TestCreationDraftSeparateIdentityRestoreAndTombstone(t *testing.T) {
	s, p, file, d := draftFixture(t)
	input := updateDraftInput{"req_select", 1, "Trim reads", file, "single-end"}
	selected, err := s.store.updateDraft(p.ProjectID, d.DraftID, input)
	if err != nil || selected.Generation != 2 || selected.Input == nil || selected.Input.RelativePath != "reads.fastq.gz" {
		t.Fatalf("selected %+v: %v", selected, err)
	}
	selected.Input.RelativePath = "caller mutation"
	same, err := s.store.updateDraft(p.ProjectID, d.DraftID, input)
	if err != nil || same.Input.RelativePath != "reads.fastq.gz" || same.Generation != 2 {
		t.Fatalf("retry/copy %+v: %v", same, err)
	}
	list, _ := s.store.listPipelines(p.ProjectID)
	if len(list.Pipelines) != 0 || len(s.store.data.Runs) != 0 {
		t.Fatal("draft created a Pipeline or Run")
	}
	discarded, err := s.store.discardDraft(p.ProjectID, d.DraftID, discardDraftInput{"req_discard", 2})
	if err != nil || discarded.Generation != 3 || discarded.State != "discarded" {
		t.Fatalf("discard %+v: %v", discarded, err)
	}
	_, err = s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_late", 2, "late", file, "single-end"})
	wantCode(t, err, "stale_revision")
	retry, err := s.store.createDraft(p.ProjectID, createDraftInput{"req_draft", "Read quality"})
	if err != nil || retry.State != "discarded" || retry.DraftID != d.DraftID {
		t.Fatalf("create retry recreated discarded draft: %+v %v", retry, err)
	}
	raw, err := os.ReadFile(filepath.Join(s.store.dir, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := decodeCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	reader := &store{data: restored}
	read, err := reader.creationDraft(p.ProjectID, d.DraftID)
	if err != nil || read.State != "discarded" || read.Brief != "Trim reads" || read.Input.RelativePath != "reads.fastq.gz" {
		t.Fatalf("restore %+v %v", read, err)
	}
	active, _ := reader.listCreationDrafts(p.ProjectID)
	if len(active.Drafts) != 0 {
		t.Fatal("tombstone listed as active")
	}
}
func TestCreationDraftGenerationRaceAndProjectScope(t *testing.T) {
	s, p, file, d := draftFixture(t)
	other := testProject(t, s, t.TempDir())
	_, err := s.store.creationDraft(other.ProjectID, d.DraftID)
	wantCode(t, err, "not_found")
	_, err = s.store.updateDraft(other.ProjectID, d.DraftID, updateDraftInput{"req_cross", 1, "", file, "single-end"})
	wantCode(t, err, "not_found")
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Go(func() {
			_, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{fmt.Sprintf("req_race_%d", i), 1, "same goal", file, "single-end"})
			results <- err
		})
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else {
			wantCode(t, err, "stale_revision")
			conflicts++
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("race wins %d conflicts %d", wins, conflicts)
	}
	_, err = s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_draft", 2, "reuse creation request", file, "single-end"})
	wantCode(t, err, "request_conflict")
}
func TestCreationInputObservationIsBoundedMetadataAndProjectContained(t *testing.T) {
	s, p, file, d := draftFixture(t)
	selected, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_bind", 1, "", file, "single-end"})
	if err != nil {
		t.Fatal(err)
	}
	project, _ := s.store.project(p.ProjectID)
	writeTestFile(t, filepath.Join(project.Root, "reads.fastq.gz"), []byte("changed file metadata"))
	observed, err := observeCreationInput(project, file)
	if err != nil || observed == *selected.Input {
		t.Fatalf("changed input observation: %+v %v", observed, err)
	}
	for _, tc := range []struct {
		name  string
		write func(string)
	}{
		{"missing", func(path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}},
		{"outside symlink", func(path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(t.TempDir(), "secret.fastq")
			writeTestFile(t, outside, []byte("outside"))
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss, pp, ff, dd := draftFixture(t)
			pr, _ := ss.store.project(pp.ProjectID)
			tc.write(filepath.Join(pr.Root, "reads.fastq.gz"))
			_, err := ss.store.updateDraft(pp.ProjectID, dd.DraftID, updateDraftInput{"req_bad", 1, "", ff, "single-end"})
			if err == nil {
				t.Fatal("invalid input accepted")
			}
			kept, _ := ss.store.creationDraft(pp.ProjectID, dd.DraftID)
			if kept.Generation != 1 || kept.Input != nil {
				t.Fatal("failed observation published")
			}
		})
	}
	_, err = s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_pair", 2, "", file, "paired-end"})
	wantCode(t, err, "invalid_request")
	clear, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_clear", 2, "keep brief", "", ""})
	if err != nil || clear.Input != nil || clear.Generation != 3 {
		t.Fatalf("explicit clear %+v %v", clear, err)
	}
}
func TestCreationDraftFailedSavePreservesPublishedState(t *testing.T) {
	s, p, _, d := draftFixture(t)
	before, _ := s.store.creationDraft(p.ProjectID, d.DraftID)
	backup := filepath.Join(s.store.dir, "catalog.backup.json")
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Mkdir(backup, 0700); err != nil {
		t.Fatal(err)
	}
	_, err := s.store.discardDraft(p.ProjectID, d.DraftID, discardDraftInput{"req_fail", 1})
	wantCode(t, err, "internal")
	if s.store.data.Drafts[d.DraftID].State != before.State || s.store.data.Drafts[d.DraftID].Generation != before.Generation {
		t.Fatal("failed save published discard")
	}
	_, err = s.store.createDraft(p.ProjectID, createDraftInput{"req_draft", "Read quality"})
	wantCode(t, err, "internal")
}
func TestCreationDraftHTTPStrictFieldsAndNoSourceAuthority(t *testing.T) {
	s, p, file, d := draftFixture(t)
	handler := s.Handler("127.0.0.1:7777", "test")
	route := "/v1/projects/" + p.ProjectID + "/pipeline-drafts/" + d.DraftID + "/update"
	valid := fmt.Sprintf(`{"requestId":"req_http","expectedGeneration":1,"brief":"design","resourceId":%q,"readLayout":"single-end"}`, file)
	for _, body := range []string{`null`, `{}`, `{"requestId":"req_http","expectedGeneration":1}`, `{"requestId":"req_http","expectedGeneration":1,"brief":null,"resourceId":"","readLayout":""}`, valid[:len(valid)-1] + `,"source":"hidden code"}`, valid[:len(valid)-1] + `,"brief":"duplicate"}`} {
		response := draftHTTP(t, handler, route, body)
		if response.Code != 400 {
			t.Fatalf("invalid body accepted: %s => %d %s", body, response.Code, response.Body.String())
		}
	}
	response := draftHTTP(t, handler, route, valid)
	if response.Code != 200 {
		t.Fatalf("valid update %d %s", response.Code, response.Body.String())
	}
	var result struct {
		Value creationDraft `json:"value"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Value.Generation != 2 || result.Value.Input.ResourceID != file {
		t.Fatal("wrong HTTP result")
	}
	response = draftHTTP(t, handler, "/v1/projects/"+p.ProjectID+"/pipeline-drafts/"+d.DraftID+"/adopt", `{}`)
	if response.Code != 400 {
		t.Fatal("adoption accepted missing exact identity")
	}
}
func draftHTTP(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "http://127.0.0.1:7777"+path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test")
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, req)
	return out
}
