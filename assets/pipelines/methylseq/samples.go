package methylseq

import (
	"encoding/csv"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
)

const readerPath = "<reader>"

var sampleNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

var requiredHeaders = []string{"sample", "fastq_1", "fastq_2"}

var acceptedHeaders = map[string]bool{"sample": true, "fastq_1": true, "fastq_2": true}

// Parse reads a strict Methyl CSV and consolidates repeated rows by sample
// while retaining each sequencing run as a typed Run.
func Parse(r io.Reader) ([]Sample, error) { return parse(r, readerPath) }

// Load opens path and parses a strict Methyl CSV. It performs no graph work and
// does not read the process-global default samplesheet path.
func Load(filePath string) ([]Sample, error) {
	f, err := os.Open(filePath)
	if err != nil {
		code := gobble.DefectInvalidPath
		message := "Methyl samplesheet is not readable"
		if os.IsNotExist(err) {
			code = gobble.DefectNotFound
			message = "Methyl samplesheet not found"
		}
		return nil, sheetError(code, "samplesheet", message, filePath)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, sheetError(gobble.DefectInvalidPath, "samplesheet", "Methyl samplesheet is not a regular readable file", filePath)
	}
	return parse(f, filePath)
}

func parse(r io.Reader, source string) ([]Sample, error) {
	if r == nil {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "Methyl samplesheet is malformed", source)
	}
	reader := csv.NewReader(r)
	reader.ReuseRecord = false
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "Methyl samplesheet is malformed", source)
	}
	records[0][0] = strings.TrimPrefix(records[0][0], "\ufeff")
	index, defects := validateHeader(records[0], source)
	if len(defects) > 0 {
		return nil, composeError(defects)
	}

	samples := make([]Sample, 0, len(records)-1)
	sampleIndex := make(map[string]int, len(records)-1)
	runKeys := make(map[string]map[string]bool, len(records)-1)
	for rowIndex, record := range records[1:] {
		rowNumber := rowIndex + 2
		cell := func(name string) string {
			column := index[name]
			if column >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[column])
		}
		name, fastq1, fastq2 := cell("sample"), cell("fastq_1"), cell("fastq_2")
		var rowDefects []gobble.Defect
		if name == "" {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "sample", "required cell is empty"))
		} else if !sampleNamePattern.MatchString(name) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "sample", "sample name is invalid"))
		}
		if fastq1 == "" {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "fastq_1", "required cell is empty"))
		} else if !validWorkspacePath(fastq1) {
			rowDefects = append(rowDefects, rowDefect(fastq1, rowNumber, "fastq_1", "FASTQ path must be workspace-relative and must not be a URL"))
		}
		if fastq2 != "" && !validWorkspacePath(fastq2) {
			rowDefects = append(rowDefects, rowDefect(fastq2, rowNumber, "fastq_2", "FASTQ path must be workspace-relative and must not be a URL"))
		}
		if len(rowDefects) > 0 {
			defects = append(defects, rowDefects...)
			continue
		}

		position, exists := sampleIndex[name]
		if !exists {
			sampleIndex[name] = len(samples)
			runKeys[name] = map[string]bool{fastq1 + "\x00" + fastq2: true}
			samples = append(samples, Sample{Name: name, Runs: []Run{{ID: "run_1", Fastq1: fastq1, Fastq2: fastq2}}})
			continue
		}
		existing := &samples[position]
		if (existing.Runs[0].Fastq2 == "") != (fastq2 == "") {
			defects = append(defects, rowDefect(source, rowNumber, name, "repeated sample mixes single-end and paired-end runs"))
			continue
		}
		key := fastq1 + "\x00" + fastq2
		if runKeys[name][key] {
			defects = append(defects, rowDefect(source, rowNumber, name, "duplicate sequencing run"))
			continue
		}
		runKeys[name][key] = true
		existing.Runs = append(existing.Runs, Run{ID: "run_" + strconv.Itoa(len(existing.Runs)+1), Fastq1: fastq1, Fastq2: fastq2})
	}
	if len(defects) > 0 {
		return nil, composeError(defects)
	}
	return cloneSamples(samples), nil
}

func validateHeader(header []string, source string) (map[string]int, []gobble.Defect) {
	index := make(map[string]int, len(header))
	var defects []gobble.Defect
	for i, name := range header {
		if !acceptedHeaders[name] {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "unknown Methyl samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		if _, exists := index[name]; exists {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "duplicate Methyl samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		index[name] = i
	}
	for _, required := range requiredHeaders {
		if _, ok := index[required]; !ok {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "missing required Methyl samplesheet column " + strconv.Quote(required), Paths: []string{source}})
		}
	}
	return index, defects
}

func validWorkspacePath(value string) bool {
	if value == "" || strings.Contains(value, "\\") || strings.Contains(value, "://") || strings.HasPrefix(value, "/") || len(value) > 1 && value[1] == ':' {
		return false
	}
	cleaned := path.Clean(value)
	return cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

func rowDefect(source string, row int, unit, message string) gobble.Defect {
	return gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: unit, Message: message + " at row " + strconv.Itoa(row), Paths: []string{source}}
}

func sheetError(code gobble.DefectCode, unit, message string, paths ...string) *gobble.Error {
	return composeError([]gobble.Defect{{Code: code, Unit: unit, Message: message, Paths: append([]string(nil), paths...)}})
}

func composeError(defects []gobble.Defect) *gobble.Error {
	copied := make([]gobble.Defect, len(defects))
	for i, defect := range defects {
		copied[i] = defect
		copied[i].Paths = append([]string(nil), defect.Paths...)
	}
	return &gobble.Error{Op: "compose", Defects: copied}
}

func cloneSamples(samples []Sample) []Sample {
	out := make([]Sample, len(samples))
	for i, sample := range samples {
		out[i] = sample
		out[i].Runs = append([]Run(nil), sample.Runs...)
	}
	return out
}
