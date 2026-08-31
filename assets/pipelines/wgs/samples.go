package wgs

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

var requiredHeaders = []string{"patient", "sample", "lane", "fastq_1", "fastq_2"}

var acceptedHeaders = map[string]bool{"patient": true, "sample": true, "lane": true, "fastq_1": true, "fastq_2": true, "sex": true}

// Parse reads a strict WGS CSV and consolidates repeated rows by sample while
// retaining every paired sequencing lane in samplesheet order.
func Parse(r io.Reader) ([]Sample, error) { return parse(r, readerPath) }

// Load opens filePath and parses a strict WGS CSV. It performs no graph work
// and does not read the process-global default samplesheet path.
func Load(filePath string) ([]Sample, error) {
	f, err := os.Open(filePath)
	if err != nil {
		code := gobble.DefectInvalidPath
		message := "WGS samplesheet is not readable"
		if os.IsNotExist(err) {
			code = gobble.DefectNotFound
			message = "WGS samplesheet not found"
		}
		return nil, sheetError(code, "samplesheet", message, filePath)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, sheetError(gobble.DefectInvalidPath, "samplesheet", "WGS samplesheet is not a regular readable file", filePath)
	}
	return parse(f, filePath)
}

func parse(r io.Reader, source string) ([]Sample, error) {
	if r == nil {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "WGS samplesheet is malformed", source)
	}
	reader := csv.NewReader(r)
	reader.ReuseRecord = false
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "WGS samplesheet is malformed", source)
	}
	records[0][0] = strings.TrimPrefix(records[0][0], "\ufeff")
	index, defects := validateHeader(records[0], source)
	if len(defects) > 0 {
		return nil, composeError(defects)
	}

	samples := make([]Sample, 0, len(records)-1)
	sampleIndex := make(map[string]int, len(records)-1)
	laneIDs := make(map[string]map[string]bool, len(records)-1)
	for rowIndex, record := range records[1:] {
		rowNumber := rowIndex + 2
		cell := func(name string) string {
			column, ok := index[name]
			if !ok || column >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[column])
		}
		patient, sample, lane := cell("patient"), cell("sample"), cell("lane")
		fastq1, fastq2, sex := cell("fastq_1"), cell("fastq_2"), cell("sex")
		var rowDefects []gobble.Defect
		for _, identity := range []struct{ name, value string }{{"patient", patient}, {"sample", sample}, {"lane", lane}} {
			name, value := identity.name, identity.value
			if value == "" {
				rowDefects = append(rowDefects, rowDefect(source, rowNumber, name, "required cell is empty"))
			} else if !identityPattern.MatchString(value) {
				rowDefects = append(rowDefects, rowDefect(source, rowNumber, name, name+" identity is invalid"))
			}
		}
		if sex != "" && !identityPattern.MatchString(sex) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "sex", "sex value is invalid"))
		}
		for _, read := range []struct{ name, value string }{{"fastq_1", fastq1}, {"fastq_2", fastq2}} {
			name, value := read.name, read.value
			if value == "" {
				rowDefects = append(rowDefects, rowDefect(source, rowNumber, name, "required paired-read cell is empty"))
			} else if !validWorkspacePath(value) {
				rowDefects = append(rowDefects, rowDefect(value, rowNumber, name, "FASTQ path must be workspace-relative and must not be a URL"))
			}
		}
		if len(rowDefects) > 0 {
			defects = append(defects, rowDefects...)
			continue
		}

		position, exists := sampleIndex[sample]
		if !exists {
			sampleIndex[sample] = len(samples)
			laneIDs[sample] = map[string]bool{lane: true}
			samples = append(samples, Sample{Patient: patient, Name: sample, Sex: sex, Lanes: []Lane{{ID: lane, Fastq1: fastq1, Fastq2: fastq2}}})
			continue
		}
		existing := &samples[position]
		if existing.Patient != patient || existing.Sex != sex {
			defects = append(defects, rowDefect(source, rowNumber, sample, "repeated sample has conflicting patient or sex identity"))
			continue
		}
		if laneIDs[sample][lane] {
			defects = append(defects, rowDefect(source, rowNumber, sample, "duplicate sample lane"))
			continue
		}
		laneIDs[sample][lane] = true
		existing.Lanes = append(existing.Lanes, Lane{ID: lane, Fastq1: fastq1, Fastq2: fastq2})
	}
	if len(samples) < 2 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "WGS joint germline requires at least two distinct samples", Paths: []string{source}})
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
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "unknown WGS samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		if _, exists := index[name]; exists {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "duplicate WGS samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		index[name] = i
	}
	for _, required := range requiredHeaders {
		if _, ok := index[required]; !ok {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "missing required WGS samplesheet column " + strconv.Quote(required), Paths: []string{source}})
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
		out[i].Lanes = append([]Lane(nil), sample.Lanes...)
	}
	return out
}
