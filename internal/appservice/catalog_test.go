package appservice

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testService(t *testing.T) *Service {
	t.Helper()
	s, err := Open(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}
func testProject(t *testing.T, s *Service, path string) Project {
	t.Helper()
	p, err := s.store.registerProject(newID("req"), path, "Test project")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	var value *apiError
	if !errors.As(err, &value) || value.Code != code {
		t.Fatalf("got error %v, want code %q", err, code)
	}
}

func TestProjectRegistrationRestoreAndRequests(t *testing.T) {
	dir, root := t.TempDir(), t.TempDir()
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.registerProject("req_first", root, "Atlas")
	if err != nil {
		t.Fatal(err)
	}
	same, err := s.registerProject("req_first", root, "Atlas")
	if err != nil || same != p {
		t.Fatalf("retry: got %+v, %v; want %+v", same, err, p)
	}
	_, err = s.registerProject("req_first", root, "Other")
	wantCode(t, err, "request_conflict")
	same, err = s.registerProject("req_second", root, "Rename is not registration")
	if err != nil || same != p {
		t.Fatalf("same root: got %+v, %v", same, err)
	}
	if _, err := openStore(dir); err == nil {
		t.Fatal("second catalog writer was allowed")
	}
	if err = s.close(); err != nil {
		t.Fatal(err)
	}
	s, err = openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	same, err = s.registerProject("req_first", root, "Atlas")
	if err != nil || same != p {
		t.Fatalf("restored retry: got %+v, %v", same, err)
	}
	if _, err = os.Stat(filepath.Join(dir, "catalog.backup.json")); err != nil {
		t.Fatal("normal backup missing:", err)
	}
}

func TestProjectsRejectOverlappingRootsAndReplacement(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := s.store.registerProject("req_nested", nested, "")
	wantCode(t, err, "request_conflict")
	_, err = s.store.registerProject("req_parent", filepath.Dir(root), "")
	wantCode(t, err, "request_conflict")
	link := filepath.Join(t.TempDir(), "alias")
	if err = os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	same, err := s.store.registerProject("req_alias", link, "")
	if err != nil || same != p {
		t.Fatalf("canonical alias: %+v %v", same, err)
	}
	if err = os.Rename(root, root+"-old"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root + "-old") })
	if err = os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = s.store.listFiles(p.ProjectID, "")
	wantCode(t, err, "stale_revision")
}

func TestCatalogRefusesCorruptionAndUnknownVersion(t *testing.T) {
	for _, body := range []string{"{", `{"schemaVersion":2,"projects":[],"runs":[],"requests":{}}`, `{"schemaVersion":1,"projects":null,"runs":[],"requests":{}}`} {
		t.Run(body, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "catalog.json")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := openStore(dir); err == nil {
				t.Fatal("invalid catalog accepted")
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != body {
				t.Fatal("invalid catalog was changed")
			}
		})
	}
}

func TestExplicitFolderMovePreservesProjectAndResources(t *testing.T) {
	s := testService(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "before")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "notes.txt"), []byte("same folder"))
	p := testProject(t, s, root)
	id := findFile(t, s, p, "notes.txt")
	moved := filepath.Join(parent, "after")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	same, err := s.store.registerProject("req_move", moved, "")
	if err != nil || same != p {
		t.Fatalf("folder move: %+v %v", same, err)
	}
	value, err := s.store.readFile(p.ProjectID, id, "")
	if err != nil || value.Content.(textContent).Text != "same folder" {
		t.Fatalf("moved resource: %+v %v", value, err)
	}
}

func TestMovedFolderCannotOverlapAnotherRegisteredProject(t *testing.T) {
	s := testService(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "before")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	testProject(t, s, root)
	other := t.TempDir()
	testProject(t, s, other)
	moved := filepath.Join(other, "moved")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	_, err := s.store.registerProject("req_move", moved, "")
	wantCode(t, err, "request_conflict")
}

func TestConcurrentRegistrationHasOneDurableProject(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Go(func() {
			_, err := s.store.registerProject("req_shared", root, "Atlas")
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if len(s.store.projects()) != 1 {
		t.Fatal("concurrent retries created multiple projects")
	}
}

func TestCatalogFailedWriteDoesNotPublishNewProject(t *testing.T) {
	s := testService(t)
	if err := os.Mkdir(filepath.Join(s.store.dir, "catalog.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := s.store.registerProject("req_write", t.TempDir(), "Atlas")
	wantCode(t, err, "internal")
	if len(s.store.projects()) != 0 {
		t.Fatal("failed registration was published in memory")
	}
}

func TestMissingCatalogDoesNotDiscardBackup(t *testing.T) {
	for _, name := range []string{"catalog.backup.json", "catalog.v1.backup.json"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			backup := filepath.Join(dir, name)
			if err := os.WriteFile(backup, []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}
			if s, err := openStore(dir); err == nil {
				s.close()
				t.Fatal("missing primary silently created a fresh catalog")
			}
			data, err := os.ReadFile(backup)
			if err != nil || string(data) != "keep" {
				t.Fatal("backup changed:", err)
			}
		})
	}
}
