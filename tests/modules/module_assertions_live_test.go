//go:build live

package moduleevidence

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

func starGenomePublishedPaths(dir string, sjdb bool) []string {
	files := []string{"Genome", "SA", "SAindex", "chrLength.txt", "chrName.txt", "chrNameLength.txt", "chrStart.txt", "genomeParameters.txt"}
	if sjdb {
		files = append(files, "Log.out", "exonGeTrInfo.tab", "exonInfo.tab", "geneInfo.tab", "sjdbInfo.txt", "sjdbList.fromGTF.out.tab", "sjdbList.out.tab", "transcriptInfo.tab")
	}
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = dir + "/" + file
	}
	return paths
}

func listRegularRel(t *testing.T, absolute, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(absolute)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", absolute, err)
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.Mode().IsRegular() {
			paths = append(paths, prefix+"/"+entry.Name())
		}
	}
	return paths
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(want))
	for _, value := range want {
		seen[value]++
	}
	for _, value := range got {
		if seen[value] == 0 {
			return false
		}
		seen[value]--
	}
	return true
}

func assertUniquelyMappedAbove(t *testing.T, path string, floor int) {
	t.Helper()
	if count := starLogInt(t, path, "Uniquely mapped reads number"); count < floor {
		t.Fatalf("uniquely mapped reads = %d, want >= %d in %s", count, floor, path)
	}
}

func assertSplicesRecorded(t *testing.T, path string) {
	t.Helper()
	if count := starLogInt(t, path, "Number of splices: Total"); count < 1 {
		t.Fatalf("splices = %d, want >= 1 in %s", count, path)
	}
}

func starLogInt(t *testing.T, path, field string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, field) {
			continue
		}
		separator := strings.LastIndex(line, "|")
		if separator < 0 {
			t.Fatalf("%s line %q missing |", field, line)
		}
		count, err := strconv.Atoi(strings.TrimSpace(line[separator+1:]))
		if err != nil {
			t.Fatal(err)
		}
		return count
	}
	t.Fatalf("%s missing %s", path, field)
	return 0
}

func uniquePEAlignments(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const field = "Number of paired-end alignments with a unique best hit:"
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, field) {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("unique-alignment line %q missing value", line)
		}
		count, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			t.Fatal(err)
		}
		return count
	}
	t.Fatalf("%s missing unique-alignment field", path)
	return 0
}

func assertUniqueAlignmentFloor(t *testing.T, unique int) {
	t.Helper()
	if unique < 1 {
		t.Fatalf("unique paired-end alignments = %d, want floor > 0", unique)
	}
}

func assertMethylationCallRows(t *testing.T, unique int, paths ...string) {
	t.Helper()
	rows := 0
	for _, path := range paths {
		rows += methylationCallRows(t, path)
	}
	if unique > 0 && rows == 0 {
		t.Fatalf("no methylation call row in %v", paths)
	}
}

func methylationCallRows(t *testing.T, path string) int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var reader io.Reader = file
	if strings.HasSuffix(path, ".gz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			t.Fatal(err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	scanner := bufio.NewScanner(reader)
	rows := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "Bismark") && !strings.HasPrefix(line, "#") && len(strings.Fields(line)) >= 4 {
			rows++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}
