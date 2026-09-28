package pipelinereview

import (
	"strings"
	"testing"
)

func TestCreationScaffoldBoundedSetup(t *testing.T) {
	v, err := ExportCreationScaffold("module github.com/HahyeonJeon/gobble\n\ngo 1.26\n", "checksum\n")
	if err != nil || len(v.Files) != 4 || len(v.Editable) != 1 || v.Editable[0] != CreationSourcePath || !strings.Contains(v.Files[0].Content, "=> /opt/gobble") {
		t.Fatalf("%+v %v", v, err)
	}
	v.Files[2].Content = "caller edit"
	fresh, _ := ExportCreationScaffold("module github.com/HahyeonJeon/gobble\n\ngo 1.26\n", "checksum\n")
	if fresh.Files[2].Content == v.Files[2].Content {
		t.Fatal("scaffold aliased")
	}
	for _, mod := range []string{"module other\n", "module github.com/HahyeonJeon/gobble\nreplace bad => elsewhere\n"} {
		if _, err := ExportCreationScaffold(mod, ""); err == nil {
			t.Fatal("foreign setup accepted")
		}
	}
}

func TestCreationPathPreservesDeclaredFileFormat(t *testing.T) {
	for _, suffix := range []string{".fastq", ".fq", ".fastq.gz", ".fq.gz"} {
		v, err := CreationPath("sample" + suffix)
		if err != nil || v != "inputs/reads"+suffix || !ValidCreationPath(v) {
			t.Fatalf("%s %v", v, err)
		}
	}
	if ValidCreationPath("../reads.fastq") || ValidCreationPath("inputs/reads.txt") {
		t.Fatal("invalid logical path")
	}
}
