package modulecommand

import (
	"path"
	"strconv"
	"strings"
)

const FastQCImage = "quay.io/biocontainers/fastqc:0.12.1--hdfd78af_0@sha256:e194048df39c3145d9b4e0a14f4da20b59d59250465b6f2a9cb698445fd45900"

func FastQC(read, outDir string, threads int) []string {
	argv := []string{"fastqc", "--outdir", outDir, "--noextract"}
	if threads > 0 {
		argv = append(argv, "--threads", strconv.Itoa(threads))
	}
	return append(argv, read)
}

func FastQCStem(read string) string {
	base := path.Base(read)
	lower := strings.ToLower(base)
	for _, suffix := range []string{".gz", ".bz2", ".xz"} {
		if strings.HasSuffix(lower, suffix) {
			base = base[:len(base)-len(suffix)]
			lower = strings.ToLower(base)
			break
		}
	}
	for _, suffix := range []string{".fastq", ".fq", ".sam", ".bam", ".txt"} {
		if strings.HasSuffix(lower, suffix) {
			base = base[:len(base)-len(suffix)]
			break
		}
	}
	return base + "_fastqc"
}
