package scrnaseq

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

var identityPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

var requiredHeaders = []string{"sample", "fastq_1", "fastq_2"}

var acceptedHeaders = map[string]bool{
	"sample": true, "fastq_1": true, "fastq_2": true,
	"expected_cells": true, "seq_center": true,
}

// Parse reads a strict scRNA CSV. Repeated sample rows become ordered paired
// technical runs with deterministic run_001 identities.
func Parse(r io.Reader) ([]Sample, error) { return parse(r, readerPath) }

// Load opens filePath and parses a strict scRNA CSV. It performs no graph work
// and does not read the process-global default samplesheet path.
func Load(filePath string) ([]Sample, error) {
	f, err := os.Open(filePath)
	if err != nil {
		code := gobble.DefectInvalidPath
		message := "scRNA samplesheet is not readable"
		if os.IsNotExist(err) {
			code = gobble.DefectNotFound
			message = "scRNA samplesheet not found"
		}
		return nil, sheetError(code, "samplesheet", message, filePath)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, sheetError(gobble.DefectInvalidPath, "samplesheet", "scRNA samplesheet is not a regular readable file", filePath)
	}
	return parse(f, filePath)
}

func parse(r io.Reader, source string) ([]Sample, error) {
	if r == nil {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "scRNA samplesheet is malformed", source)
	}
	reader := csv.NewReader(r)
	reader.ReuseRecord = false
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "scRNA samplesheet is malformed", source)
	}
	records[0][0] = strings.TrimPrefix(records[0][0], "\ufeff")
	index, defects := validateHeader(records[0], source)
	if len(defects) > 0 {
		return nil, composeError(defects)
	}

	samples := make([]Sample, 0, len(records)-1)
	sampleIndex := make(map[string]int, len(records)-1)
	runKeys := make(map[string]map[string]bool, len(records)-1)
	readPaths := make(map[string]bool, (len(records)-1)*2)
	for rowIndex, record := range records[1:] {
		rowNumber := rowIndex + 2
		cell := func(name string) string {
			column, ok := index[name]
			if !ok || column >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[column])
		}
		name, fastq1, fastq2 := cell("sample"), cell("fastq_1"), cell("fastq_2")
		expectedCell, seqCenter := cell("expected_cells"), cell("seq_center")
		var rowDefects []gobble.Defect
		if !identityPattern.MatchString(name) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "sample", "sample identity is empty or invalid"))
		}
		if !validWorkspacePath(fastq1) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "fastq_1", "mate 1 path must be workspace-relative and must not be a URL"))
		}
		if !validWorkspacePath(fastq2) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "fastq_2", "mate 2 path must be workspace-relative and must not be a URL"))
		}
		expected := 0
		if expectedCell != "" {
			expected, err = strconv.Atoi(expectedCell)
			if err != nil || expected < 1 {
				rowDefects = append(rowDefects, rowDefect(source, rowNumber, "expected_cells", "expected_cells must be empty or a positive integer"))
			}
		}
		if len(rowDefects) > 0 {
			defects = append(defects, rowDefects...)
			continue
		}

		runKey := fastq1 + "\x00" + fastq2
		position, exists := sampleIndex[name]
		if exists {
			existing := &samples[position]
			if existing.ExpectedCells != expected || existing.SeqCenter != seqCenter {
				defects = append(defects, rowDefect(source, rowNumber, name, "repeated runs have conflicting expected_cells or seq_center metadata"))
				continue
			}
			if runKeys[name][runKey] {
				defects = append(defects, rowDefect(source, rowNumber, name, "duplicate technical-run read pair"))
				continue
			}
		}
		if fastq1 == fastq2 {
			defects = append(defects, rowDefect(source, rowNumber, "fastq_2", "mate paths must be distinct"))
			continue
		}
		if readPaths[fastq1] {
			defects = append(defects, rowDefect(source, rowNumber, "fastq_1", "read path is already assigned to another mate or technical run"))
			continue
		}
		if readPaths[fastq2] {
			defects = append(defects, rowDefect(source, rowNumber, "fastq_2", "read path is already assigned to another mate or technical run"))
			continue
		}
		readPaths[fastq1] = true
		readPaths[fastq2] = true
		if !exists {
			position = len(samples)
			sampleIndex[name] = position
			runKeys[name] = map[string]bool{runKey: true}
			samples = append(samples, Sample{Name: name, ExpectedCells: expected, SeqCenter: seqCenter, Runs: []Run{{ID: "run_001", Fastq1: fastq1, Fastq2: fastq2}}})
			continue
		}
		runKeys[name][runKey] = true
		existing := &samples[position]
		existing.Runs = append(existing.Runs, Run{ID: "run_" + leftPad3(len(existing.Runs)+1), Fastq1: fastq1, Fastq2: fastq2})
	}
	if len(samples) == 0 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "scRNA samplesheet has no valid samples", Paths: []string{source}})
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
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "unknown scRNA samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		if _, exists := index[name]; exists {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "duplicate scRNA samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		index[name] = i
	}
	for _, required := range requiredHeaders {
		if _, ok := index[required]; !ok {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "missing required scRNA samplesheet column " + strconv.Quote(required), Paths: []string{source}})
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

func leftPad3(value int) string {
	if value < 10 {
		return "00" + strconv.Itoa(value)
	}
	if value < 100 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
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
