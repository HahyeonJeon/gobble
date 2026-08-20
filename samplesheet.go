package gobble

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	intpath "github.com/HahyeonJeon/gobble/internal/path"
)

// DefaultSampleSheetPath is the process-cwd relative sheet used when
// [SetSampleSheetPath] has not been called or was restored.
const DefaultSampleSheetPath = "samplesheet.csv"

const (
	// StrandednessUnstranded is the RNA strandedness token unstranded.
	StrandednessUnstranded = "unstranded"
	// StrandednessForward is the RNA strandedness token forward.
	StrandednessForward = "forward"
	// StrandednessReverse is the RNA strandedness token reverse.
	StrandednessReverse = "reverse"
	// DefaultRNAStrandedness is the RNA strandedness used when a row
	// leaves strandedness empty.
	DefaultRNAStrandedness = StrandednessReverse
)

const (
	colSample       = "sample"
	colRead1        = "read1"
	colRead2        = "read2"
	colReference    = "reference"
	colGTF          = "gtf"
	colGroup        = "group"
	colStrandedness = "strandedness"
	utf8BOM         = "\ufeff"
	readerPath      = "<reader>"
)

var optionalCols = map[string]bool{
	colReference:    true,
	colGTF:          true,
	colGroup:        true,
	colStrandedness: true,
}

var (
	sampleSheetMu   sync.Mutex
	sampleSheetPath string
)

// SampleRow is one samplesheet data row. Fields are the locked column
// names. Empty optional cells are empty strings. The zero SampleRow is
// not a valid row.
type SampleRow struct {
	Sample       string
	Read1        string
	Read2        string
	Reference    string
	GTF          string
	Group        string
	Strandedness string
}

// SampleSheet is a parsed samplesheet. Path is the path passed to load,
// or "<reader>" when parsed from an [io.Reader]. Rows is a copied slice.
type SampleSheet struct {
	Path string
	Rows []SampleRow
}

// SetSampleSheetPath stores path for this process. Empty or
// whitespace-only restores [DefaultSampleSheetPath]. The stored string
// is copied.
func SetSampleSheetPath(path string) {
	sampleSheetMu.Lock()
	defer sampleSheetMu.Unlock()
	if strings.TrimSpace(path) == "" {
		sampleSheetPath = ""
		return
	}
	sampleSheetPath = strings.Clone(path)
}

// SampleSheetPath returns the path stored by [SetSampleSheetPath], or
// [DefaultSampleSheetPath] if never set or restored.
func SampleSheetPath() string {
	sampleSheetMu.Lock()
	defer sampleSheetMu.Unlock()
	if sampleSheetPath == "" {
		return DefaultSampleSheetPath
	}
	return sampleSheetPath
}

// LoadSampleSheet loads the sheet at [SampleSheetPath].
func LoadSampleSheet() (*SampleSheet, error) {
	return LoadSampleSheetFile(SampleSheetPath())
}

// LoadSampleSheetFile opens path relative to process cwd unless path is
// absolute, then parses it as a samplesheet. It does not search a
// workspace directory.
func LoadSampleSheetFile(path string) (*SampleSheet, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, sheetError(Defect{
				Code:    DefectNotFound,
				Unit:    "samplesheet",
				Message: "samplesheet not found",
				Paths:   []string{path},
			})
		}
		return nil, unreadableSheet(path)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return nil, unreadableSheet(path)
	}
	return parseSampleSheet(f, path)
}

// ParseSampleSheet reads a CSV samplesheet from r. The returned
// SampleSheet.Path is "<reader>".
func ParseSampleSheet(r io.Reader) (*SampleSheet, error) {
	return parseSampleSheet(r, readerPath)
}

// IsSampleSheetError reports whether err is an [*Error] whose defects
// are only samplesheet parse, load, or constructor-rule defects.
func IsSampleSheetError(err error) bool {
	var ge *Error
	if !errors.As(err, &ge) || ge == nil || len(ge.Defects) == 0 {
		return false
	}
	for _, d := range ge.Defects {
		if !isSampleSheetDefect(d) {
			return false
		}
	}
	return true
}

func isSampleSheetDefect(d Defect) bool {
	switch d.Code {
	case DefectInvalidSampleSheet:
		return true
	case DefectInvalidName:
		return len(d.Paths) > 0
	case DefectInvalidPath, DefectNotFound:
		return d.Unit == "samplesheet"
	default:
		return false
	}
}

func parseSampleSheet(r io.Reader, path string) (*SampleSheet, error) {
	if r == nil {
		return nil, malformedSheet(path)
	}
	cr := csv.NewReader(r)
	cr.ReuseRecord = false
	records, err := cr.ReadAll()
	if err != nil || len(records) == 0 {
		return nil, malformedSheet(path)
	}
	header := records[0]
	if len(header) == 0 {
		return nil, malformedSheet(path)
	}
	header[0] = strings.TrimPrefix(header[0], utf8BOM)
	idx, ok := headerIndex(header)
	if !ok || len(records) < 2 {
		return nil, malformedSheet(path)
	}

	var defects []Defect
	rows := make([]SampleRow, 0, len(records)-1)
	seen := make(map[string]bool, len(records)-1)
	var ref string
	var refSet bool
	refDisagree := false
	var gtf string
	var gtfSet bool
	gtfDisagree := false

	for i, rec := range records[1:] {
		rowNum := i + 2
		cell := func(col string) string {
			j, exists := idx[col]
			if !exists || j >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[j])
		}
		row := SampleRow{
			Sample:       cell(colSample),
			Read1:        cell(colRead1),
			Read2:        cell(colRead2),
			Reference:    cell(colReference),
			GTF:          cell(colGTF),
			Group:        cell(colGroup),
			Strandedness: cell(colStrandedness),
		}
		if row.Sample == "" {
			defects = append(defects, emptyRequiredCell(path, colSample, rowNum))
		} else {
			if !namePat.MatchString(row.Sample) {
				defects = append(defects, Defect{
					Code:    DefectInvalidName,
					Unit:    row.Sample,
					Message: "invalid sample name",
					Paths:   []string{path},
				})
			}
			if seen[row.Sample] {
				defects = append(defects, Defect{
					Code:    DefectInvalidName,
					Unit:    row.Sample,
					Message: "duplicate sample name",
					Paths:   []string{path},
				})
			} else {
				seen[row.Sample] = true
			}
		}
		if row.Read1 == "" {
			defects = append(defects, emptyRequiredCell(path, colRead1, rowNum))
		} else if !validSheetPath(row.Read1) {
			defects = append(defects, illegalSheetPath(row.Read1))
		}
		if row.Read2 == "" {
			defects = append(defects, emptyRequiredCell(path, colRead2, rowNum))
		} else if !validSheetPath(row.Read2) {
			defects = append(defects, illegalSheetPath(row.Read2))
		}
		if row.Reference != "" {
			if !validSheetPath(row.Reference) {
				defects = append(defects, illegalSheetPath(row.Reference))
			}
			if refSet && row.Reference != ref {
				refDisagree = true
			} else if !refSet {
				ref = row.Reference
				refSet = true
			}
		}
		if row.GTF != "" {
			if !validSheetPath(row.GTF) {
				defects = append(defects, illegalSheetPath(row.GTF))
			}
			if gtfSet && row.GTF != gtf {
				gtfDisagree = true
			} else if !gtfSet {
				gtf = row.GTF
				gtfSet = true
			}
		}
		rows = append(rows, row)
	}
	if refDisagree || gtfDisagree {
		defects = append(defects, Defect{
			Code:    DefectInvalidSampleSheet,
			Unit:    "samplesheet",
			Message: "shared reference or gtf cells disagree",
			Paths:   []string{path},
		})
	}
	if len(defects) > 0 {
		return nil, sheetError(defects...)
	}
	out := make([]SampleRow, len(rows))
	copy(out, rows)
	return &SampleSheet{Path: path, Rows: out}, nil
}

func headerIndex(header []string) (map[string]int, bool) {
	idx := make(map[string]int, len(header))
	seen := make(map[string]bool, len(header))
	ok := true
	for i, h := range header {
		if seen[h] {
			ok = false
			continue
		}
		seen[h] = true
		switch h {
		case colSample, colRead1, colRead2:
			idx[h] = i
		default:
			if optionalCols[h] {
				idx[h] = i
			} else {
				ok = false
			}
		}
	}
	for _, req := range []string{colSample, colRead1, colRead2} {
		if _, exists := idx[req]; !exists {
			ok = false
		}
	}
	return idx, ok
}

func validSheetPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.Contains(p, "://") {
		return false
	}
	if isAbsSheetPath(p) {
		return false
	}
	_, escaped := intpath.Clean(p)
	return !escaped
}

func isAbsSheetPath(p string) bool {
	if strings.HasPrefix(p, "/") || filepath.IsAbs(p) {
		return true
	}
	if len(p) >= 2 && p[1] == ':' {
		c := p[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	return false
}

func emptyRequiredCell(path, col string, row int) Defect {
	return Defect{
		Code:    DefectInvalidSampleSheet,
		Unit:    "samplesheet",
		Message: "required cell is empty: " + col + " row " + strconv.Itoa(row),
		Paths:   []string{path},
	}
}

func illegalSheetPath(cell string) Defect {
	return Defect{
		Code:    DefectInvalidPath,
		Unit:    "samplesheet",
		Message: "samplesheet path is not a workspace-relative path",
		Paths:   []string{cell},
	}
}

func malformedSheet(path string) *Error {
	return sheetError(Defect{
		Code:    DefectInvalidSampleSheet,
		Unit:    "samplesheet",
		Message: "samplesheet is malformed",
		Paths:   []string{path},
	})
}

func unreadableSheet(path string) *Error {
	return sheetError(Defect{
		Code:    DefectInvalidPath,
		Unit:    "samplesheet",
		Message: "samplesheet path is not readable",
		Paths:   []string{path},
	})
}

func sheetError(defects ...Defect) *Error {
	out := make([]Defect, len(defects))
	copy(out, defects)
	return &Error{Op: "compose", Defects: out}
}
