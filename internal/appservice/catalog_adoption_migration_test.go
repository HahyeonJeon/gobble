package appservice

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogV4MigrationArchivesDraftsAndFreezesBirthFields(t *testing.T) {
	s, p, scope, _ := firstAdoptionFixture(t)
	raw, err := json.Marshal(s.store.data)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(`"schemaVersion":5`), []byte(`"schemaVersion":4`), 1)
	migrated, original, err := decodeCatalog(raw)
	if err != nil || !bytes.Equal(original, raw) || migrated.Drafts[scope.Draft.DraftID].Input.Identity != scope.Draft.Input.Identity {
		t.Fatal("v4 draft migration", err)
	}
	for _, bad := range [][]byte{
		bytes.Replace(raw, []byte(`"state":"draft"`), []byte(`"state":"adopted"`), 1),
		bytes.Replace(raw, []byte(`"state":"draft"`), []byte(`"state":"draft","adoption":{}`), 1),
	} {
		if _, _, err = decodeCatalog(bad); err == nil {
			t.Fatal("v5 birth accepted as v4")
		}
	}
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "catalog.json"), raw)
	st, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.createDraft(p.ProjectID, createDraftInput{"req_next", "Next"}); err != nil {
		t.Fatal(err)
	}
	archived, err := os.ReadFile(filepath.Join(dir, "catalog.v4.backup.json"))
	if err != nil || !bytes.Equal(archived, raw) {
		t.Fatal("archive changed", err)
	}
	if err = st.close(); err != nil {
		t.Fatal(err)
	}
	st, err = openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.close()
	if st.data.SchemaVersion != 5 || len(st.data.Drafts) != 2 {
		t.Fatal("migration not retained")
	}
	missing := t.TempDir()
	writeTestFile(t, filepath.Join(missing, "catalog.v4.backup.json"), raw)
	if opened, err := openStore(missing); err == nil {
		opened.close()
		t.Fatal("lost primary reset")
	}
}
