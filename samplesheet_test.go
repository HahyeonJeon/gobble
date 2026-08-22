package gobble_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

const validSheetCSV = "sample,read1,read2\ns1,reads/r1.fq,reads/r2.fq\n"

func TestParseSampleSheetValid(t *testing.T) {
	csv := strings.Join([]string{
		"sample,read1,read2,reference,gtf,group,strandedness",
		"s1, reads/s1_r1.fq ,reads/s1_r2.fq,ref/genome.fa,ref/genes.gtf,ctrl,reverse",
		"s2,reads/s2_r1.fq,reads/s2_r2.fq,ref/genome.fa,ref/genes.gtf,treat,",
	}, "\n") + "\n"
	sheet, err := gobble.ParseSampleSheet(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseSampleSheet() error = %v, want nil", err)
	}
	if sheet.Path != "<reader>" {
		t.Fatalf("ParseSampleSheet() Path got %q, want %q", sheet.Path, "<reader>")
	}
	want := []gobble.SampleRow{
		{
			Sample:       "s1",
			Read1:        "reads/s1_r1.fq",
			Read2:        "reads/s1_r2.fq",
			Reference:    "ref/genome.fa",
			GTF:          "ref/genes.gtf",
			Group:        "ctrl",
			Strandedness: gobble.StrandednessReverse,
		},
		{
			Sample:    "s2",
			Read1:     "reads/s2_r1.fq",
			Read2:     "reads/s2_r2.fq",
			Reference: "ref/genome.fa",
			GTF:       "ref/genes.gtf",
			Group:     "treat",
		},
	}
	if len(sheet.Rows) != len(want) {
		t.Fatalf("ParseSampleSheet() rows got %d, want %d", len(sheet.Rows), len(want))
	}
	for i := range want {
		if sheet.Rows[i] != want[i] {
			t.Fatalf("ParseSampleSheet() row %d got %+v, want %+v", i, sheet.Rows[i], want[i])
		}
	}
	sheet.Rows[0].Sample = "mutated"
	again, err := gobble.ParseSampleSheet(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseSampleSheet() second error = %v, want nil", err)
	}
	if again.Rows[0].Sample != "s1" {
		t.Fatalf("ParseSampleSheet() retained caller mutation, got %q", again.Rows[0].Sample)
	}
}

func TestParseSampleSheetOmitsOptionalHeaders(t *testing.T) {
	sheet, err := gobble.ParseSampleSheet(strings.NewReader(validSheetCSV))
	if err != nil {
		t.Fatalf("ParseSampleSheet() error = %v, want nil", err)
	}
	if len(sheet.Rows) != 1 {
		t.Fatalf("ParseSampleSheet() rows got %d, want 1", len(sheet.Rows))
	}
	got := sheet.Rows[0]
	want := gobble.SampleRow{Sample: "s1", Read1: "reads/r1.fq", Read2: "reads/r2.fq"}
	if got != want {
		t.Fatalf("ParseSampleSheet() row got %+v, want %+v", got, want)
	}
}

func TestParseSampleSheetAllowsConstructorDeferredRules(t *testing.T) {
	csv := "sample,read1,read2,group,strandedness\ns1,reads/r1.fq,reads/r2.fq,,bogus\n"
	sheet, err := gobble.ParseSampleSheet(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseSampleSheet() error = %v, want nil (RNA group and strandedness are constructor rules)", err)
	}
	if sheet.Rows[0].Group != "" || sheet.Rows[0].Strandedness != "bogus" {
		t.Fatalf("ParseSampleSheet() row got %+v, want empty group and strandedness bogus", sheet.Rows[0])
	}
}

func TestParseSampleSheetEmptyRead2(t *testing.T) {
	sheet, err := gobble.ParseSampleSheet(strings.NewReader("sample,read1,read2\ns1,reads/r1.fq,\n"))
	if err != nil {
		t.Fatalf("ParseSampleSheet() empty read2 error = %v, want nil", err)
	}
	got := sheet.Rows[0]
	want := gobble.SampleRow{Sample: "s1", Read1: "reads/r1.fq"}
	if got != want {
		t.Fatalf("ParseSampleSheet() empty read2 row got %+v, want %+v", got, want)
	}
	sheet, err = gobble.ParseSampleSheet(strings.NewReader("sample,read1\ns1,reads/r1.fq\n"))
	if err != nil {
		t.Fatalf("ParseSampleSheet() omitted read2 header error = %v, want nil", err)
	}
	got = sheet.Rows[0]
	if got != want {
		t.Fatalf("ParseSampleSheet() omitted read2 row got %+v, want %+v", got, want)
	}
}

func TestParseSampleSheetEmptyOptionalCells(t *testing.T) {
	csv := "sample,read1,read2,reference,gtf,group,strandedness\ns1,reads/r1.fq,reads/r2.fq, ,\t,,\n"
	sheet, err := gobble.ParseSampleSheet(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseSampleSheet() error = %v, want nil", err)
	}
	got := sheet.Rows[0]
	if got.Reference != "" || got.GTF != "" || got.Group != "" || got.Strandedness != "" {
		t.Fatalf("ParseSampleSheet() optional cells got %+v, want empty strings", got)
	}
}

func TestParseSampleSheetBOM(t *testing.T) {
	csv := "\ufeffsample,read1,read2\ns1,reads/r1.fq,reads/r2.fq\n"
	sheet, err := gobble.ParseSampleSheet(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseSampleSheet() BOM error = %v, want nil", err)
	}
	if sheet.Rows[0].Sample != "s1" {
		t.Fatalf("ParseSampleSheet() BOM sample got %q, want %q", sheet.Rows[0].Sample, "s1")
	}
}

func TestParseSampleSheetReject(t *testing.T) {
	tests := []struct {
		name    string
		csv     string
		code    gobble.DefectCode
		unit    string
		message string
		path    string
	}{
		{
			name:    "malformed CSV",
			csv:     "sample,read1,read2\n\"unterminated\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "samplesheet is malformed",
			path:    "<reader>",
		},
		{
			name:    "tab separated",
			csv:     "sample\tread1\tread2\ns1\treads/r1.fq\treads/r2.fq\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "samplesheet is malformed",
			path:    "<reader>",
		},
		{
			name:    "missing header",
			csv:     "",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "samplesheet is malformed",
			path:    "<reader>",
		},
		{
			name:    "no data rows",
			csv:     "sample,read1,read2\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "samplesheet is malformed",
			path:    "<reader>",
		},
		{
			name:    "missing required header",
			csv:     "sample,read2\ns1,reads/r2.fq\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "samplesheet is malformed",
			path:    "<reader>",
		},
		{
			name:    "unknown header",
			csv:     "sample,read1,read2,extra\ns1,reads/r1.fq,reads/r2.fq,x\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "samplesheet is malformed",
			path:    "<reader>",
		},
		{
			name:    "duplicate header",
			csv:     "sample,read1,read2,sample\ns1,reads/r1.fq,reads/r2.fq,s1\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "samplesheet is malformed",
			path:    "<reader>",
		},
		{
			name:    "empty sample",
			csv:     "sample,read1,read2\n,reads/r1.fq,reads/r2.fq\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "required cell is empty: sample row 2",
			path:    "<reader>",
		},
		{
			name:    "empty read1",
			csv:     "sample,read1,read2\ns1,,reads/r2.fq\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "required cell is empty: read1 row 2",
			path:    "<reader>",
		},
		{
			name:    "illegal sample name",
			csv:     "sample,read1,read2\n1bad,reads/r1.fq,reads/r2.fq\n",
			code:    gobble.DefectInvalidName,
			unit:    "1bad",
			message: "invalid sample name",
			path:    "<reader>",
		},
		{
			name:    "duplicate sample name",
			csv:     "sample,read1,read2\ns1,reads/a_r1.fq,reads/a_r2.fq\ns1,reads/b_r1.fq,reads/b_r2.fq\n",
			code:    gobble.DefectInvalidName,
			unit:    "s1",
			message: "duplicate sample name",
			path:    "<reader>",
		},
		{
			name:    "backslash path",
			csv:     "sample,read1,read2\ns1,reads\\r1.fq,reads/r2.fq\n",
			code:    gobble.DefectInvalidPath,
			unit:    "samplesheet",
			message: "samplesheet path is not a workspace-relative path",
			path:    `reads\r1.fq`,
		},
		{
			name:    "absolute path",
			csv:     "sample,read1,read2\ns1,/abs/r1.fq,reads/r2.fq\n",
			code:    gobble.DefectInvalidPath,
			unit:    "samplesheet",
			message: "samplesheet path is not a workspace-relative path",
			path:    "/abs/r1.fq",
		},
		{
			name:    "windows volume path",
			csv:     "sample,read1,read2\ns1,C:/reads/r1.fq,reads/r2.fq\n",
			code:    gobble.DefectInvalidPath,
			unit:    "samplesheet",
			message: "samplesheet path is not a workspace-relative path",
			path:    "C:/reads/r1.fq",
		},
		{
			name:    "url path",
			csv:     "sample,read1,read2\ns1,s3://bucket/r1.fq,reads/r2.fq\n",
			code:    gobble.DefectInvalidPath,
			unit:    "samplesheet",
			message: "samplesheet path is not a workspace-relative path",
			path:    "s3://bucket/r1.fq",
		},
		{
			name:    "dotdot escape",
			csv:     "sample,read1,read2\ns1,../r1.fq,reads/r2.fq\n",
			code:    gobble.DefectInvalidPath,
			unit:    "samplesheet",
			message: "samplesheet path is not a workspace-relative path",
			path:    "../r1.fq",
		},
		{
			name:    "inconsistent reference",
			csv:     "sample,read1,read2,reference\ns1,reads/s1_r1.fq,reads/s1_r2.fq,ref/a.fa\ns2,reads/s2_r1.fq,reads/s2_r2.fq,ref/b.fa\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "shared reference or gtf cells disagree",
			path:    "<reader>",
		},
		{
			name:    "inconsistent gtf",
			csv:     "sample,read1,read2,gtf\ns1,reads/s1_r1.fq,reads/s1_r2.fq,ref/a.gtf\ns2,reads/s2_r1.fq,reads/s2_r2.fq,ref/b.gtf\n",
			code:    gobble.DefectInvalidSampleSheet,
			unit:    "samplesheet",
			message: "shared reference or gtf cells disagree",
			path:    "<reader>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet, err := gobble.ParseSampleSheet(strings.NewReader(tt.csv))
			if sheet != nil {
				t.Fatalf("ParseSampleSheet() sheet != nil, want nil")
			}
			ge := mustSheetError(t, err)
			if ge.Op != "compose" {
				t.Fatalf("Error.Op got %q, want %q", ge.Op, "compose")
			}
			if !gobble.IsSampleSheetError(err) {
				t.Fatalf("IsSampleSheetError() got false, want true")
			}
			if !hasSheetDefect(ge, tt.code, tt.unit, tt.message, tt.path) {
				t.Fatalf("ParseSampleSheet() defects got %+v, want code %s unit %q message %q paths %q", ge.Defects, tt.code, tt.unit, tt.message, tt.path)
			}
		})
	}
}

func TestLoadSampleSheetFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.csv")
	if err := os.WriteFile(path, []byte(validSheetCSV), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	sheet, err := gobble.LoadSampleSheetFile(path)
	if err != nil {
		t.Fatalf("LoadSampleSheetFile() error = %v, want nil", err)
	}
	if sheet.Path != path {
		t.Fatalf("LoadSampleSheetFile() Path got %q, want %q", sheet.Path, path)
	}
	if len(sheet.Rows) != 1 || sheet.Rows[0].Sample != "s1" {
		t.Fatalf("LoadSampleSheetFile() rows got %+v", sheet.Rows)
	}
}

func TestLoadSampleSheetFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.csv")
	sheet, err := gobble.LoadSampleSheetFile(path)
	if sheet != nil {
		t.Fatalf("LoadSampleSheetFile() sheet != nil, want nil")
	}
	ge := mustSheetError(t, err)
	if !gobble.IsSampleSheetError(err) {
		t.Fatalf("IsSampleSheetError() got false, want true")
	}
	if !hasSheetDefect(ge, gobble.DefectNotFound, "samplesheet", "samplesheet not found", path) {
		t.Fatalf("LoadSampleSheetFile() defects got %+v, want not-found", ge.Defects)
	}
}

func TestLoadSampleSheetFileUnreadable(t *testing.T) {
	dir := t.TempDir()
	sheet, err := gobble.LoadSampleSheetFile(dir)
	if sheet != nil {
		t.Fatalf("LoadSampleSheetFile() sheet != nil, want nil")
	}
	ge := mustSheetError(t, err)
	if !gobble.IsSampleSheetError(err) {
		t.Fatalf("IsSampleSheetError() got false, want true")
	}
	if !hasSheetDefect(ge, gobble.DefectInvalidPath, "samplesheet", "samplesheet path is not readable", dir) {
		t.Fatalf("LoadSampleSheetFile() defects got %+v, want invalid-path", ge.Defects)
	}

	path := filepath.Join(dir, "noperm.csv")
	if err := os.WriteFile(path, []byte(validSheetCSV), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if f, openErr := os.Open(path); openErr == nil {
		f.Close()
		t.Skip("process can read mode 0 file")
	}
	sheet, err = gobble.LoadSampleSheetFile(path)
	if sheet != nil {
		t.Fatalf("LoadSampleSheetFile() chmod 0 sheet != nil, want nil")
	}
	ge = mustSheetError(t, err)
	if !hasSheetDefect(ge, gobble.DefectInvalidPath, "samplesheet", "samplesheet path is not readable", path) {
		t.Fatalf("LoadSampleSheetFile() chmod 0 defects got %+v, want invalid-path", ge.Defects)
	}
}

func TestSampleSheetPathSetter(t *testing.T) {
	t.Cleanup(func() { gobble.SetSampleSheetPath("") })
	gobble.SetSampleSheetPath("")
	if gobble.DefaultSampleSheetPath != "samplesheet.csv" {
		t.Fatalf("DefaultSampleSheetPath got %q, want %q", gobble.DefaultSampleSheetPath, "samplesheet.csv")
	}
	if got := gobble.SampleSheetPath(); got != gobble.DefaultSampleSheetPath {
		t.Fatalf("SampleSheetPath() got %q, want %q", got, gobble.DefaultSampleSheetPath)
	}
	gobble.SetSampleSheetPath("other.csv")
	if got := gobble.SampleSheetPath(); got != "other.csv" {
		t.Fatalf("SampleSheetPath() after set got %q, want %q", got, "other.csv")
	}
	gobble.SetSampleSheetPath("   ")
	if got := gobble.SampleSheetPath(); got != gobble.DefaultSampleSheetPath {
		t.Fatalf("SampleSheetPath() after whitespace restore got %q, want %q", got, gobble.DefaultSampleSheetPath)
	}
}

func TestLoadSampleSheetUsesSetterPath(t *testing.T) {
	t.Cleanup(func() { gobble.SetSampleSheetPath("") })
	path := filepath.Join(t.TempDir(), "custom.csv")
	if err := os.WriteFile(path, []byte(validSheetCSV), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	gobble.SetSampleSheetPath(path)
	sheet, err := gobble.LoadSampleSheet()
	if err != nil {
		t.Fatalf("LoadSampleSheet() error = %v, want nil", err)
	}
	if sheet.Path != path {
		t.Fatalf("LoadSampleSheet() Path got %q, want %q", sheet.Path, path)
	}
}

func TestLoadSampleSheetDefaultPath(t *testing.T) {
	t.Cleanup(func() { gobble.SetSampleSheetPath("") })
	gobble.SetSampleSheetPath("")
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Errorf("Chdir(%s) error = %v", cwd, err)
		}
	})
	if err := os.WriteFile(gobble.DefaultSampleSheetPath, []byte(validSheetCSV), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	sheet, err := gobble.LoadSampleSheet()
	if err != nil {
		t.Fatalf("LoadSampleSheet() error = %v, want nil", err)
	}
	if sheet.Path != gobble.DefaultSampleSheetPath {
		t.Fatalf("LoadSampleSheet() Path got %q, want %q", sheet.Path, gobble.DefaultSampleSheetPath)
	}
}

func TestIsSampleSheetError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("nope"),
			want: false,
		},
		{
			name: "missing-input",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectMissingInput,
				Unit:    "copy.reads",
				Message: "missing input",
			}}},
			want: false,
		},
		{
			name: "ordinary invalid-name",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectInvalidName,
				Unit:    "1copy",
				Message: "invalid name",
			}}},
			want: false,
		},
		{
			name: "ordinary invalid-path",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectInvalidPath,
				Unit:    "copy.out",
				Message: "path escapes directory",
				Paths:   []string{"../out"},
			}}},
			want: false,
		},
		{
			name: "ordinary not-found",
			err: &gobble.Error{Op: "run", Defects: []gobble.Defect{{
				Code:    gobble.DefectNotFound,
				Unit:    "workspace",
				Message: "workspace not found",
			}}},
			want: false,
		},
		{
			name: "mix with missing-input",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{
				{
					Code:    gobble.DefectInvalidSampleSheet,
					Unit:    "samplesheet",
					Message: "samplesheet is malformed",
					Paths:   []string{"sheet.csv"},
				},
				{
					Code:    gobble.DefectMissingInput,
					Unit:    "copy.reads",
					Message: "missing input",
				},
			}},
			want: false,
		},
		{
			name: "malformed",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectInvalidSampleSheet,
				Unit:    "samplesheet",
				Message: "samplesheet is malformed",
				Paths:   []string{"sheet.csv"},
			}}},
			want: true,
		},
		{
			name: "duplicate sample",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectInvalidName,
				Unit:    "s1",
				Message: "duplicate sample name",
				Paths:   []string{"sheet.csv"},
			}}},
			want: true,
		},
		{
			name: "illegal path cell",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectInvalidPath,
				Unit:    "samplesheet",
				Message: "samplesheet path is not a workspace-relative path",
				Paths:   []string{"../r1.fq"},
			}}},
			want: true,
		},
		{
			name: "not-found sheet",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectNotFound,
				Unit:    "samplesheet",
				Message: "samplesheet not found",
				Paths:   []string{"sheet.csv"},
			}}},
			want: true,
		},
		{
			name: "RNA constructor rule",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectInvalidSampleSheet,
				Unit:    "samplesheet",
				Message: "RNA samplesheet requires group on every row and exactly two groups",
				Paths:   []string{"sheet.csv"},
			}}},
			want: true,
		},
		{
			name: "Methyl constructor rule",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
				Code:    gobble.DefectInvalidSampleSheet,
				Unit:    "samplesheet",
				Message: "Methyl samplesheet requires at least two samples",
				Paths:   []string{"sheet.csv"},
			}}},
			want: true,
		},
		{
			name: "locked mix",
			err: &gobble.Error{Op: "compose", Defects: []gobble.Defect{
				{
					Code:    gobble.DefectInvalidSampleSheet,
					Unit:    "samplesheet",
					Message: "required cell is empty: sample row 2",
					Paths:   []string{"sheet.csv"},
				},
				{
					Code:    gobble.DefectInvalidName,
					Unit:    "s1",
					Message: "duplicate sample name",
					Paths:   []string{"sheet.csv"},
				},
				{
					Code:    gobble.DefectInvalidPath,
					Unit:    "samplesheet",
					Message: "samplesheet path is not a workspace-relative path",
					Paths:   []string{"../r1.fq"},
				},
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gobble.IsSampleSheetError(tt.err)
			if got != tt.want {
				t.Fatalf("IsSampleSheetError() got %v, want %v", got, tt.want)
			}
		})
	}
	wrapped := fmt.Errorf("load: %w", &gobble.Error{Op: "compose", Defects: []gobble.Defect{{
		Code:    gobble.DefectInvalidSampleSheet,
		Unit:    "samplesheet",
		Message: "samplesheet is malformed",
		Paths:   []string{"sheet.csv"},
	}}})
	if !gobble.IsSampleSheetError(wrapped) {
		t.Fatalf("IsSampleSheetError() wrapped got false, want true")
	}
}

func TestRecordComposeError(t *testing.T) {
	var nilPipe *gobble.Pipeline
	nilPipe.RecordComposeError(errors.New("ignored"))

	ok := validComposePipeline()
	ok.RecordComposeError(nil)
	g, err := gobble.Compose(ok)
	if err != nil || g == nil {
		t.Fatalf("Compose() after nil RecordComposeError error = %v graph = %v", err, g)
	}

	sheetErr := &gobble.Error{Op: "validate", Defects: []gobble.Defect{{
		Code:    gobble.DefectInvalidSampleSheet,
		Unit:    "samplesheet",
		Message: "samplesheet is malformed",
		Paths:   []string{"sheet.csv"},
	}}}
	p := validComposePipeline()
	p.AddTask(gobble.TaskSpec{
		Name:    "later",
		Command: []string{"true"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "later", Ext: ".txt"}}},
	})
	p.RecordComposeError(sheetErr)
	p.RecordComposeError(errors.New("second"))
	g, err = gobble.Compose(p)
	if g != nil {
		t.Fatalf("Compose() graph != nil, want nil (no tasks)")
	}
	ge := mustSheetError(t, err)
	if ge.Op != "compose" {
		t.Fatalf("Error.Op got %q, want %q", ge.Op, "compose")
	}
	if !gobble.IsSampleSheetError(err) {
		t.Fatalf("IsSampleSheetError() got false, want true")
	}
	if !hasSheetDefect(ge, gobble.DefectInvalidSampleSheet, "samplesheet", "samplesheet is malformed", "sheet.csv") {
		t.Fatalf("Compose() defects got %+v, want recorded samplesheet error", ge.Defects)
	}

	plain := gobble.NewPipeline("plain")
	plain.RecordComposeError(errors.New("load failed"))
	g, err = gobble.Compose(plain)
	if g != nil {
		t.Fatalf("Compose() plain graph != nil, want nil")
	}
	var pe *gobble.Error
	if !errors.As(err, &pe) {
		t.Fatalf("Compose() error = %v, want *Error", err)
	}
	if pe.Op != "compose" {
		t.Fatalf("Error.Op got %q, want %q", pe.Op, "compose")
	}
}

func validComposePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("sheet")
	in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{{Name: "dst", Spec: gobble.PathSpec{Base: "out", Ext: ".txt"}}},
	})
	return p
}

func mustSheetError(t *testing.T, err error) *gobble.Error {
	t.Helper()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("error = %v, want *Error", err)
	}
	return ge
}

func hasSheetDefect(ge *gobble.Error, code gobble.DefectCode, unit, message, path string) bool {
	for _, d := range ge.Defects {
		if d.Code != code || d.Unit != unit || d.Message != message {
			continue
		}
		if len(d.Paths) == 1 && d.Paths[0] == path {
			return true
		}
	}
	return false
}
