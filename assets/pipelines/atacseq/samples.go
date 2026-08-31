package atacseq

import (
	"encoding/csv"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
)

const readerPath = "<reader>"

var identityPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

var requiredHeaders = []string{"sample", "fastq_1", "fastq_2", "replicate"}

var acceptedHeaders = map[string]bool{
	"sample": true, "fastq_1": true, "fastq_2": true, "replicate": true,
	"control": true, "control_replicate": true,
}

// Parse reads a strict ATAC CSV. Repeated sample/replicate rows become ordered
// technical runs with deterministic run_001 identities.
func Parse(r io.Reader) ([]Sample, error) { return parse(r, readerPath) }

// Load opens filePath and parses a strict ATAC CSV. It performs no graph work
// and does not read the process-global default samplesheet path.
func Load(filePath string) ([]Sample, error) {
	f, err := os.Open(filePath)
	if err != nil {
		code := gobble.DefectInvalidPath
		message := "ATAC samplesheet is not readable"
		if os.IsNotExist(err) {
			code = gobble.DefectNotFound
			message = "ATAC samplesheet not found"
		}
		return nil, sheetError(code, "samplesheet", message, filePath)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, sheetError(gobble.DefectInvalidPath, "samplesheet", "ATAC samplesheet is not a regular readable file", filePath)
	}
	return parse(f, filePath)
}

func parse(r io.Reader, source string) ([]Sample, error) {
	if r == nil {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "ATAC samplesheet is malformed", source)
	}
	reader := csv.NewReader(r)
	reader.ReuseRecord = false
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, sheetError(gobble.DefectInvalidSampleSheet, "samplesheet", "ATAC samplesheet is malformed", source)
	}
	records[0][0] = strings.TrimPrefix(records[0][0], "\ufeff")
	index, defects := validateHeader(records[0], source)
	if len(defects) > 0 {
		return nil, composeError(defects)
	}

	samples := make([]Sample, 0, len(records)-1)
	sampleIndex := make(map[string]int, len(records)-1)
	replicateIndex := make(map[string]map[int]int, len(records)-1)
	runKeys := make(map[string]map[string]bool, len(records)-1)
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
		replicateCell, controlName, controlReplicateCell := cell("replicate"), cell("control"), cell("control_replicate")
		var rowDefects []gobble.Defect
		if !identityPattern.MatchString(name) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "sample", "sample identity is empty or invalid"))
		}
		replicate, replicateErr := strconv.Atoi(replicateCell)
		if replicateErr != nil || replicate < 1 {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "replicate", "replicate must be a positive integer"))
		}
		if !validWorkspacePath(fastq1) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "fastq_1", "FASTQ path must be workspace-relative and must not be a URL"))
		}
		if fastq2 != "" && !validWorkspacePath(fastq2) {
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "fastq_2", "FASTQ path must be workspace-relative and must not be a URL"))
		}
		var control *ControlRef
		switch {
		case controlName == "" && controlReplicateCell == "":
		case controlName == "" || controlReplicateCell == "" || !identityPattern.MatchString(controlName):
			rowDefects = append(rowDefects, rowDefect(source, rowNumber, "control", "control and control_replicate must be supplied together with a valid control identity"))
		default:
			controlReplicate, controlErr := strconv.Atoi(controlReplicateCell)
			if controlErr != nil || controlReplicate < 1 {
				rowDefects = append(rowDefects, rowDefect(source, rowNumber, "control_replicate", "control replicate must be a positive integer"))
			} else {
				control = &ControlRef{Sample: controlName, Replicate: controlReplicate}
			}
		}
		if len(rowDefects) > 0 {
			defects = append(defects, rowDefects...)
			continue
		}

		samplePosition, exists := sampleIndex[name]
		if !exists {
			samplePosition = len(samples)
			sampleIndex[name] = samplePosition
			replicateIndex[name] = make(map[int]int)
			samples = append(samples, Sample{Name: name})
		}
		replicatePosition, exists := replicateIndex[name][replicate]
		key := name + "#" + strconv.Itoa(replicate)
		runKey := fastq1 + "\x00" + fastq2
		if !exists {
			replicatePosition = len(samples[samplePosition].Replicates)
			replicateIndex[name][replicate] = replicatePosition
			runKeys[key] = map[string]bool{runKey: true}
			samples[samplePosition].Replicates = append(samples[samplePosition].Replicates, Replicate{Number: replicate, Control: control, Runs: []Run{{ID: "run_001", Fastq1: fastq1, Fastq2: fastq2}}})
			continue
		}
		existing := &samples[samplePosition].Replicates[replicatePosition]
		if (existing.Runs[0].Fastq2 == "") != (fastq2 == "") {
			defects = append(defects, rowDefect(source, rowNumber, name, "technical runs within one replicate have conflicting read modes"))
			continue
		}
		if !equalControl(existing.Control, control) {
			defects = append(defects, rowDefect(source, rowNumber, name, "technical runs within one replicate have conflicting control links"))
			continue
		}
		if runKeys[key][runKey] {
			defects = append(defects, rowDefect(source, rowNumber, name, "duplicate technical run read pair"))
			continue
		}
		runKeys[key][runKey] = true
		existing.Runs = append(existing.Runs, Run{ID: "run_" + leftPad3(len(existing.Runs)+1), Fastq1: fastq1, Fastq2: fastq2})
	}

	for i := range samples {
		sort.Slice(samples[i].Replicates, func(a, b int) bool { return samples[i].Replicates[a].Number < samples[i].Replicates[b].Number })
		for j, replicate := range samples[i].Replicates {
			if replicate.Number != j+1 {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: samples[i].Name, Message: "ATAC biological replicates must start at 1 without gaps", Paths: []string{source}})
				break
			}
		}
	}
	if countReplicates(samples) < 2 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "ATAC consensus requires at least two biological replicate members", Paths: []string{source}})
	}
	for _, sample := range samples {
		for _, replicate := range sample.Replicates {
			if replicate.Control == nil {
				continue
			}
			position, ok := sampleIndex[replicate.Control.Sample]
			if !ok || !hasReplicate(samples[position], replicate.Control.Replicate) {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "ATAC control link does not resolve to an existing sample replicate", Paths: []string{source}})
			}
		}
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
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "unknown ATAC samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		if _, exists := index[name]; exists {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "duplicate ATAC samplesheet column " + strconv.Quote(name), Paths: []string{source}})
			continue
		}
		index[name] = i
	}
	for _, required := range requiredHeaders {
		if _, ok := index[required]; !ok {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "missing required ATAC samplesheet column " + strconv.Quote(required), Paths: []string{source}})
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

func equalControl(left, right *ControlRef) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func hasReplicate(sample Sample, number int) bool {
	return number >= 1 && number <= len(sample.Replicates) && sample.Replicates[number-1].Number == number
}

func countReplicates(samples []Sample) int {
	total := 0
	for _, sample := range samples {
		total += len(sample.Replicates)
	}
	return total
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
		out[i].Name = sample.Name
		out[i].Replicates = make([]Replicate, len(sample.Replicates))
		for j, replicate := range sample.Replicates {
			out[i].Replicates[j] = replicate
			out[i].Replicates[j].Runs = append([]Run(nil), replicate.Runs...)
			if replicate.Control != nil {
				control := *replicate.Control
				out[i].Replicates[j].Control = &control
			}
		}
	}
	return out
}
