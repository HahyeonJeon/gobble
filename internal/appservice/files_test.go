package appservice

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
func findFile(t *testing.T, s *Service, p Project, name string) string {
	t.Helper()
	directory, err := s.store.listFiles(p.ProjectID, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range directory.Entries {
		if entry.Name == name {
			return entry.ResourceID
		}
	}
	t.Fatalf("file %q not listed", name)
	return ""
}

func TestFilePreviewsAndVersionedSelection(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	writeTestFile(t, filepath.Join(root, "notes.txt"), []byte("Review S03"))
	id := findFile(t, s, p, "notes.txt")
	value, err := s.store.readFile(p.ProjectID, id, "")
	if err != nil {
		t.Fatal(err)
	}
	if value.Content.(textContent).Text != "Review S03" {
		t.Fatal("text changed")
	}
	writeTestFile(t, filepath.Join(root, "notes.txt"), []byte("Review S04"))
	_, err = s.store.readFile(p.ProjectID, id, value.Revision)
	wantCode(t, err, "stale_revision")
	other := testProject(t, s, t.TempDir())
	_, err = s.store.readFile(other.ProjectID, id, "")
	wantCode(t, err, "not_found")
	_, err = s.store.listFiles(p.ProjectID, "../outside")
	wantCode(t, err, "not_found")
}

func TestNotebookBytesStayBoundedAndRevisionAddressed(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	data := []byte(`{"nbformat":4,"metadata":{"padding":"` + strings.Repeat("x", 2<<20) + `"}}`)
	path := filepath.Join(root, "analysis.IPYNB")
	writeTestFile(t, path, data)
	id := findFile(t, s, p, "analysis.IPYNB")
	value, err := s.store.readFile(p.ProjectID, id, "")
	if err != nil {
		t.Fatal(err)
	}
	content, ok := value.Content.(notebookContent)
	if !ok || content.Kind != "notebook" {
		t.Fatalf("Unexpected Notebook transport: %#v", value.Content)
	}
	decoded, err := base64.StdEncoding.DecodeString(content.Base64)
	if err != nil || !bytes.Equal(decoded, data) || value.Revision != digest(data) || value.Size != len(data) {
		t.Fatal("Notebook source bytes changed")
	}
	_, err = decodeContent("same.txt", data)
	wantCode(t, err, "unsupported")
	writeTestFile(t, path, []byte("malformed Notebook JSON"))
	_, err = s.store.readFile(p.ProjectID, id, value.Revision)
	wantCode(t, err, "stale_revision")
	// The service owns byte access, while the dedicated App parser owns Notebook validity.
	if _, err = s.store.readFile(p.ProjectID, id, ""); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, bytes.Repeat([]byte("x"), maxFileBytes+1))
	_, err = s.store.readFile(p.ProjectID, id, "")
	wantCode(t, err, "unsupported")
	other := testProject(t, s, t.TempDir())
	_, err = s.store.readFile(other.ProjectID, id, "")
	wantCode(t, err, "not_found")
}

func TestCSVAndImagePreviews(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	writeTestFile(t, filepath.Join(root, "samples.csv"), []byte("sample,condition\nS03,\"control, A\"\n"))
	value, err := s.store.readFile(p.ProjectID, findFile(t, s, p, "samples.csv"), "")
	if err != nil {
		t.Fatal(err)
	}
	table := value.Content.(tableContent)
	if table.Rows[0].Key != "row_1" || table.Rows[0].Cells[1] != "control, A" {
		t.Fatalf("CSV: %+v", table)
	}
	var data bytes.Buffer
	if err = png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "plot.png"), data.Bytes())
	value, err = s.store.readFile(p.ProjectID, findFile(t, s, p, "plot.png"), "")
	if err != nil {
		t.Fatal(err)
	}
	preview := value.Content.(imageContent)
	if preview.Width != 3 || preview.Height != 2 || preview.MediaType != "image/png" {
		t.Fatalf("image: %+v", preview)
	}
}

func TestPreviewLimitsAndInvalidInputs(t *testing.T) {
	_, err := decodeContent("bad.csv", []byte("a,b\n1\n"))
	wantCode(t, err, "invalid_request")
	_, err = decodeContent("bad.png", []byte("not an image"))
	wantCode(t, err, "unsupported")
	_, err = decodeContent("binary", []byte{0, 1, 2})
	wantCode(t, err, "unsupported")
	_, err = decodeContent("long.txt", bytes.Repeat([]byte("a"), (1<<20)+1))
	wantCode(t, err, "unsupported")
	table, err := readCSV([]byte("sample\n" + strings.Repeat("S03\n", 501)))
	if err != nil || !table.Truncated || len(table.Rows) != 500 {
		t.Fatalf("CSV truncation: %d %t %v", len(table.Rows), table.Truncated, err)
	}
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	f, err := os.Create(filepath.Join(root, "large.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(maxFileBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	_, err = s.store.readFile(p.ProjectID, findFile(t, s, p, "large.txt"), "")
	wantCode(t, err, "unsupported")
}

func TestResourceSymlinkAndFIFOReplacementsAreDenied(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	path := filepath.Join(root, "notes.txt")
	writeTestFile(t, path, []byte("inside"))
	id := findFile(t, s, p, "notes.txt")
	out := filepath.Join(t.TempDir(), "private.txt")
	writeTestFile(t, out, []byte("outside"))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(out, path); err != nil {
		t.Fatal(err)
	}
	_, err := s.store.readFile(p.ProjectID, id, "")
	wantCode(t, err, "outside_project")
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = s.store.readFile(p.ProjectID, id, "")
	wantCode(t, err, "unsupported")
}

func TestResourceReferencesSurviveServiceRestart(t *testing.T) {
	profile, root := t.TempDir(), t.TempDir()
	writeTestFile(t, filepath.Join(root, "notes.txt"), []byte("saved"))
	s, err := Open(t.Context(), profile)
	if err != nil {
		t.Fatal(err)
	}
	p := testProject(t, s, root)
	id := findFile(t, s, p, "notes.txt")
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), profile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	value, err := s.store.readFile(p.ProjectID, id, "")
	if err != nil || value.Content.(textContent).Text != "saved" {
		t.Fatalf("restored read: %+v %v", value, err)
	}
}

func TestPDFSourceTransportPreservesBytesAndRevision(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	path := filepath.Join(root, "report.PDF")
	// The service checks transport/header limits; only the App sandbox parses pages.
	data := []byte("%PDF-1.7\nopaque page bytes\x00\xff")
	writeTestFile(t, path, data)
	id := findFile(t, s, p, "report.PDF")
	value, err := s.store.readFile(p.ProjectID, id, "")
	if err != nil {
		t.Fatal(err)
	}
	content, ok := value.Content.(pdfContent)
	if !ok || content.Kind != "pdf" {
		t.Fatalf("PDF transport: %#v", value.Content)
	}
	decoded, err := base64.StdEncoding.DecodeString(content.Base64)
	if err != nil || !bytes.Equal(decoded, data) || value.Revision != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) {
		t.Fatal("PDF bytes/revision changed")
	}
	writeTestFile(t, path, append(data, '\n'))
	_, err = s.store.readFile(p.ProjectID, id, value.Revision)
	wantCode(t, err, "stale_revision")
	writeTestFile(t, path, []byte("not a PDF"))
	_, err = s.store.readFile(p.ProjectID, id, "")
	wantCode(t, err, "unsupported")
	writeTestFile(t, path, append([]byte("%PDF-"), make([]byte, maxFileBytes)...))
	_, err = s.store.readFile(p.ProjectID, id, "")
	wantCode(t, err, "unsupported")
}
