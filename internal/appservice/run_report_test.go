package appservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type reportFixture struct {
	s           *Service
	project     Project
	original    launchRecord
	evidence    outputEvidence
	support     string
	reads       int
	beforeQuery func(int)
	commands    [][]string
}

func newReportFixture(t *testing.T) *reportFixture {
	t.Helper()
	s, p, rec := readyLaunchFixture(t)
	rec.Value.State = "accepted"
	rec.Value.Operation = &launchOperation{State: "admitted", RunRef: "run_report"}
	if err := s.saveLaunch(rec); err != nil {
		t.Fatal(err)
	}
	s.store.mu.Lock()
	workspaceID := resourceID(p.ProjectID, rec.Value.OutputPath)
	for i := range s.store.data.Projects {
		if s.store.data.Projects[i].ProjectID == p.ProjectID {
			s.store.data.Projects[i].Resources[workspaceID] = rec.Value.OutputPath
		}
	}
	s.store.data.Runs = append(s.store.data.Runs, runRecord{RunRegistration{p.ProjectID, "run_report", "Analysis 1", workspaceID, "engine-run"}, rec.Runtime, json.RawMessage(`{"engine":"fixture"}`)})
	s.store.mu.Unlock()
	bytes := []byte("<html>saved quality result</html>")
	f := &reportFixture{s: s, project: p, original: rec, support: "1", evidence: outputEvidence{1, "engine-run", strings.Repeat("a", 32), launchIntentDigest(rec.Intent), "qc", 1, "html", "fastqc-v1", "work/fastqc/report.html", digest(bytes), int64(len(bytes))}}
	project, _ := s.store.project(p.ProjectID)
	reportPath := filepath.Join(project.Root, rec.Value.OutputPath, f.evidence.Path)
	if err := os.MkdirAll(filepath.Dir(reportPath), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, reportPath, bytes)
	s.runtime.command = func(ctx context.Context, args, env []string) ([]byte, error) {
		f.commands = append(f.commands, append([]string(nil), args...))
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		joined := strings.Join(args, " ")
		switch {
		case args[0] == "info":
			return []byte(rec.Runtime.DaemonID + " linux"), nil
		case args[0] == "image" && strings.Contains(joined, "output-evidence.version"):
			return []byte(f.support), nil
		case args[0] == "image":
			return []byte(rec.Runtime.ImageID + " linux/amd64"), nil
		case args[0] == "rm":
			return nil, nil
		case strings.Contains(joined, "output-capabilities"):
			return []byte(`{"schemaVersion":1,"outputEvidenceVersion":1}`), nil
		case strings.Contains(joined, "output-evidence"):
			f.reads++
			if f.beforeQuery != nil {
				f.beforeQuery(f.reads)
			}
			return json.Marshal(f.evidence)
		case strings.Contains(joined, "identity"):
			return []byte(`{"schema_version":2,"view":"identity","match":true,"required":{"engine":"fixture"}}`), nil
		default:
			return nil, errors.New("unexpected query: " + joined)
		}
	}
	return f
}
func (f *reportFixture) read() (runReport, error) {
	return f.s.readRunReport(context.Background(), f.project.ProjectID, "run_report", "qc", 1)
}
func TestRunReportHTTPReturnsVerifiedBytesWithoutExecution(t *testing.T) {
	f := newReportFixture(t)
	f.beforeQuery = func(n int) {
		if n == 2 {
			f.evidence.Snapshot = strings.Repeat("b", 32)
		}
	}
	h := f.s.Handler("127.0.0.1:7777", "test")
	req := httptest.NewRequest("GET", "http://127.0.0.1:7777/v1/projects/"+f.project.ProjectID+"/runs/run_report/report?instance=qc&attempt=1", nil)
	req.Header.Set("Authorization", "Bearer test")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, req)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var envelope struct {
		Value runReport `json:"value"`
	}
	if json.Unmarshal(response.Body.Bytes(), &envelope) != nil {
		t.Fatal("invalid response")
	}
	got := envelope.Value
	data, err := base64.StdEncoding.DecodeString(got.Base64)
	if err != nil || digest(data) != got.Evidence.SHA256 || int64(len(data)) != got.Evidence.Size || got.RunRef != "run_report" || f.reads != 2 {
		t.Fatal(got, err)
	}
	for _, args := range f.commands {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "output-evidence --workspace") {
			if !strings.Contains(joined, "--read-only") || !strings.Contains(joined, "--network=none") || strings.Contains(joined, "docker.sock") || strings.Contains(joined, "prepared-run") {
				t.Fatal("unsafe output query", joined)
			}
		}
	}
}
func TestRunReportRefusals(t *testing.T) {
	cases := []struct {
		name, code string
		change     func(*reportFixture)
	}{
		{"legacy", "unsupported", func(f *reportFixture) { f.support = "<no value>" }},
		{"wrong-run", "incompatible_runtime", func(f *reportFixture) { f.evidence.RunID = "other" }},
		{"wrong-origin", "incompatible_runtime", func(f *reportFixture) { f.evidence.OriginDigest = digest([]byte("other")) }},
		{"wrong-attempt", "incompatible_runtime", func(f *reportFixture) { f.evidence.Attempt = 2 }},
		{"wrong-recipe", "incompatible_runtime", func(f *reportFixture) { f.evidence.Recipe = "" }},
		{"path-escape", "incompatible_runtime", func(f *reportFixture) { f.evidence.Path = "../other.html" }},
		{"oversize", "unsupported", func(f *reportFixture) { f.evidence.Size = maxReportBytes + 1 }},
		{"changed-content", "stale_revision", func(f *reportFixture) { f.evidence.SHA256 = digest([]byte("changed")) }},
		{"producer-drift", "incompatible_runtime", func(f *reportFixture) {
			f.beforeQuery = func(n int) {
				if n == 2 {
					f.evidence.Attempt++
				}
			}
		}},
		{"same-attempt-drift", "stale_revision", func(f *reportFixture) {
			f.beforeQuery = func(n int) {
				if n == 2 {
					f.evidence.SHA256 = digest([]byte("different"))
				}
			}
		}},
		{"changed-binding", "stale_revision", func(f *reportFixture) { f.s.store.data.Runs[0].Binding.ImageID = digest([]byte("different-engine")) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReportFixture(t)
			tc.change(f)
			_, err := f.read()
			wantCode(t, err, tc.code)
			if tc.name == "legacy" && f.reads != 0 {
				t.Fatal("queried old engine")
			}
		})
	}
}
func TestRunReportContainedFiles(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "symlink", "outside-run", "size", "content"} {
		t.Run(kind, func(t *testing.T) {
			f := newReportFixture(t)
			p, _ := f.s.store.project(f.project.ProjectID)
			path := filepath.Join(p.Root, f.original.Value.OutputPath, f.evidence.Path)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing":
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink", "outside-run":
				target := filepath.Join(p.Root, "elsewhere.html")
				if kind == "symlink" {
					target = filepath.Join(filepath.Dir(path), "other.html")
				}
				writeTestFile(t, target, []byte("<html>saved quality result</html>"))
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			case "size":
				writeTestFile(t, path, []byte("short"))
			case "content":
				writeTestFile(t, path, []byte("<html>other quality result</html>"))
			}
			if _, err := f.read(); err == nil {
				t.Fatal("invalid file accepted")
			}
		})
	}
}
func TestRunReportRejectsUntrustedAndCrossProject(t *testing.T) {
	f := newReportFixture(t)
	h := f.s.Handler("127.0.0.1:7777", "test")
	for _, auth := range []string{"", "Bearer wrong"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1:7777/v1/projects/"+f.project.ProjectID+"/runs/run_report/report?instance=qc&attempt=1", nil)
		req.Header.Set("Authorization", auth)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
	if len(f.commands) != 0 {
		t.Fatal("unauthorized runtime query")
	}
	_, err := f.s.readRunReport(context.Background(), f.project.ProjectID, "run_other", "qc", 1)
	wantCode(t, err, "not_found")
	_, err = f.s.readRunReport(context.Background(), "prj_other", "run_report", "qc", 1)
	wantCode(t, err, "not_found")
}

func TestRunReportReplacementDuringAcquisition(t *testing.T) {
	for _, kind := range []string{"ancestor-symlink", "leaf-replacement", "in-place-write"} {
		t.Run(kind, func(t *testing.T) {
			f := newReportFixture(t)
			p, _ := f.s.store.project(f.project.ProjectID)
			dir := filepath.Join(p.Root, f.original.Value.OutputPath)
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			file, err := openReportSource(root, f.evidence.Path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			before, err := file.Stat()
			if err != nil {
				t.Fatal(err)
			}
			bytes := []byte("<html>saved quality result</html>")
			path := filepath.Join(dir, f.evidence.Path)
			switch kind {
			case "ancestor-symlink":
				parent := filepath.Dir(path)
				if err = os.Rename(parent, parent+"-moved"); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(parent+"-moved", parent); err != nil {
					t.Fatal(err)
				}
			case "leaf-replacement":
				if err = os.Rename(path, path+"-moved"); err != nil {
					t.Fatal(err)
				}
				writeTestFile(t, path, bytes)
			case "in-place-write":
				writeTestFile(t, path, []byte("changed after acquisition"))
			}
			_, err = finishReportRead(root, file, before, f.evidence, bytes)
			wantCode(t, err, "stale_revision")
		})
	}
}

func TestReportQueryErrorsAreBounded(t *testing.T) {
	for _, tc := range []struct{ message, code string }{
		{"output attempt is no longer current", "stale_revision"},
		{"output attempt has not succeeded", "invalid_request"},
		{"output checksum is unavailable", "not_found"},
		{"output checksums are ambiguous", "incompatible_runtime"},
		{"private /host/file diagnostic", "runtime_unavailable"},
	} {
		raw, _ := json.Marshal(map[string]any{"op": "cli", "defects": []any{map[string]any{"code": "invalid-request", "message": tc.message}}})
		err := reportQueryError(&runtimeCommandError{cause: problem("runtime_unavailable", "Safe transport message."), diagnostic: raw})
		wantCode(t, err, tc.code)
		if strings.Contains(err.Error(), "/host/") {
			t.Fatal("diagnostic leaked")
		}
	}
}
