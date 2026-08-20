package local_e2e_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRNASeqCLIOmittedSampleReadsCwdSheet(t *testing.T) {
	bin := buildGobble(t)
	cwd, err := os.MkdirTemp(moduleRoot(t), "local-e2e-cwd-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cwd) })
	src := packSheet(t, rnaSheetRel)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", src, err)
	}
	dst := filepath.Join(cwd, "samplesheet.csv")
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", dst, err)
	}
	pkg := filepath.Join(moduleRoot(t), "tests", "local-e2e", "rnaseq")
	res := runGobbleDir(t, bin, cwd, "compose", pkg)
	requireCLIOp(t, res, "{\"op\":\"compose\",\"pipeline\":\"rnaseq\"}\n")
}
