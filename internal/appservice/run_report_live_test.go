package appservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in extension of TestLivePreparedLaunch: actual tools, fresh engine/Run and
// authenticated native transport. No report UI or real Agent claim is made here.
func verifyLiveRunReport(t *testing.T, s *Service, project, runRef string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	snapshot, err := s.queryRun(ctx, project, runRef, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	header, err := readMonitor(snapshot.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var instance string
	var attempt int
	// Fixture task IDs are taken from the checked creation, not a report filename.
	original, err := s.reportLaunch(project, runRef)
	if err != nil {
		t.Fatal(err)
	}
	if original.Value.Preparation.Prepared == nil || len(original.Value.Preparation.Prepared.Steps) != 2 {
		t.Fatal("unexpected qualification design")
	}
	qcID := original.Value.Preparation.Prepared.Steps[1].ID
	for _, task := range header.Tasks {
		if task.Identity == qcID {
			instance = task.Identity
			attempt = task.Attempt
		}
	}
	if instance == "" {
		t.Fatal("qualification quality task not found", header.Tasks)
	}
	handler := s.Handler("127.0.0.1:7777", "report-test")
	req := httptest.NewRequest("GET", "http://127.0.0.1:7777/v1/projects/"+project+"/runs/"+runRef+"/report?instance="+instance+"&attempt="+strconv.Itoa(attempt), nil)
	req.Header.Set("Authorization", "Bearer report-test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != 200 {
		t.Fatal("actual report HTTP", response.Code, response.Body.String())
	}
	var envelope struct {
		Value runReport `json:"value"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	report := envelope.Value
	bytes, err := base64.StdEncoding.DecodeString(report.Base64)
	if err != nil {
		t.Fatal(err)
	}
	if report.Evidence.Attempt != attempt || report.Evidence.Recipe != "fastqc-v1" || digest(bytes) != report.Evidence.SHA256 || int64(len(bytes)) != report.Evidence.Size || !strings.Contains(string(bytes), "FastQC") {
		t.Fatal("actual content/provenance mismatch")
	}
	p, _ := s.store.project(project)
	path := filepath.Join(p.Root, original.Value.OutputPath, report.Evidence.Path)
	source, err := os.ReadFile(path)
	if err != nil || string(source) != string(bytes) {
		t.Fatal("returned bytes differ", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.WriteFile(path, source, info.Mode()) })
	if err = os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), source...)
	changed[len(changed)-1] ^= 1
	if err = os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = s.readRunReport(ctx, project, runRef, instance, attempt)
	wantCode(t, err, "stale_revision")
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	_, err = s.readRunReport(ctx, project, runRef, instance, attempt)
	wantCode(t, err, "not_found")
	if err = os.WriteFile(path, source, info.Mode()); err != nil {
		t.Fatal(err)
	}
	restored, err := s.readRunReport(ctx, project, runRef, instance, attempt)
	if err != nil || restored.Base64 != report.Base64 || restored.Evidence != report.Evidence {
		t.Fatal("restored report differs", err)
	}
	if output := os.Getenv("GOBBLE_REPORT_EVIDENCE_DIR"); output != "" {
		if err = os.MkdirAll(output, 0700); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.MarshalIndent(struct {
			Report     runReport      `json:"report"`
			Runtime    RuntimeBinding `json:"runtime"`
			SourcePath string         `json:"sourcePath"`
		}{report, original.Runtime, path}, "", "  ")
		if err = os.WriteFile(filepath.Join(output, "actual-report.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("actual report verified: attempt=%d bytes=%d digest=%s; changed/missing output refused; restored bytes match", attempt, len(bytes), report.Evidence.SHA256)
}
