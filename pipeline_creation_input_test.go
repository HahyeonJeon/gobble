package gobble_test

import (
	"github.com/HahyeonJeon/gobble"
	"os"
	"testing"
)

func TestCreationInputDescriptor(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := gobble.ReadCreationInput(); err == nil {
		t.Fatal("missing descriptor accepted")
	}
	for _, raw := range []string{`{"schemaVersion":1,"path":"inputs/reads.fastq","readLayout":"single-end"}`, `{"schemaVersion":1,"path":"inputs/reads.fq.gz","readLayout":"single-end"}`} {
		if err := os.WriteFile("creation-input.json", []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := gobble.ReadCreationInput(); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`{"schemaVersion":2,"path":"inputs/reads.fastq","readLayout":"single-end"}`, `{"schemaVersion":1,"path":"../private.fastq","readLayout":"single-end"}`, `{"schemaVersion":1,"path":"inputs/reads.fastq","readLayout":"paired-end"}`, `{"schemaVersion":1,"path":"inputs/reads.fastq","readLayout":"single-end","extra":1}`} {
		if err := os.WriteFile("creation-input.json", []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := gobble.ReadCreationInput(); err == nil {
			t.Fatal("invalid descriptor accepted")
		}
	}
}
