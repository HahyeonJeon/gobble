package gobble_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func mustRender(t *testing.T, p gobble.PathSpec) string {
	t.Helper()
	got, err := p.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return got
}

func defectCodes(e *gobble.Error) []gobble.DefectCode {
	codes := make([]gobble.DefectCode, len(e.Defects))
	for i, d := range e.Defects {
		codes[i] = d.Code
	}
	return codes
}

func TestPathSpecRender(t *testing.T) {
	bam := gobble.PathSpec{Base: "aln", Ext: ".bam"}
	cram := gobble.PathSpec{Base: "aln", Ext: ".cram"}
	vcf := gobble.PathSpec{Base: "calls", Ext: ".vcf.gz"}
	fasta := gobble.PathSpec{Base: "reference", Ext: ".fasta"}
	fa := gobble.PathSpec{Base: "ref", Ext: ".fa"}
	fagz := gobble.PathSpec{Base: "ref", Ext: ".fa.gz"}
	r1 := gobble.PathSpec{Prefix: "samplename_S1_L001_R1_", Base: "001", Ext: ".fastq.gz"}
	chain := gobble.PathSpec{Base: "sample", Suffixes: []string{"sorted", "markdup"}, Ext: ".bam"}

	tests := []struct {
		name string
		spec gobble.PathSpec
		want string
	}{
		{
			name: "gzipped FASTQ",
			spec: gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"},
			want: "sample.fastq.gz",
		},
		{
			name: "Illumina paired FASTQ R1",
			spec: r1,
			want: "samplename_S1_L001_R1_001.fastq.gz",
		},
		{
			name: "Illumina paired FASTQ R2",
			spec: r1.WithPrefix("samplename_S1_L001_R2_"),
			want: "samplename_S1_L001_R2_001.fastq.gz",
		},
		{
			name: "interleaved FASTQ",
			spec: gobble.PathSpec{Base: "sample", Ext: ".interleaved.fastq.gz"},
			want: "sample.interleaved.fastq.gz",
		},
		{
			name: "htslib BAM",
			spec: bam,
			want: "aln.bam",
		},
		{
			name: "htslib BAI",
			spec: bam.AppendExt(".bai"),
			want: "aln.bam.bai",
		},
		{
			name: "htslib CSI",
			spec: bam.AppendExt(".csi"),
			want: "aln.bam.csi",
		},
		{
			name: "Picard BAI",
			spec: bam.WithExt(".bai"),
			want: "aln.bai",
		},
		{
			name: "CRAM CRAI",
			spec: cram.AppendExt(".crai"),
			want: "aln.cram.crai",
		},
		{
			name: "VCF TBI",
			spec: vcf.AppendExt(".tbi"),
			want: "calls.vcf.gz.tbi",
		},
		{
			name: "VCF CSI",
			spec: vcf.AppendExt(".csi"),
			want: "calls.vcf.gz.csi",
		},
		{
			name: "FASTA FAI",
			spec: fa.AppendExt(".fai"),
			want: "ref.fa.fai",
		},
		{
			name: "compressed FASTA FAI",
			spec: fagz.AppendExt(".fai"),
			want: "ref.fa.gz.fai",
		},
		{
			name: "compressed FASTA GZI",
			spec: fagz.AppendExt(".gzi"),
			want: "ref.fa.gz.gzi",
		},
		{
			name: "FASTA dict",
			spec: fasta.WithExt(".dict"),
			want: "reference.dict",
		},
		{
			name: "processing chain",
			spec: chain,
			want: "sample.sorted.markdup.bam",
		},
		{
			name: "processing chain Append index",
			spec: chain.AppendExt(".bai"),
			want: "sample.sorted.markdup.bam.bai",
		},
		{
			name: "tool-chosen literal",
			spec: gobble.Literal("sample.Aligned.sortedByCoord.out.bam"),
			want: "sample.Aligned.sortedByCoord.out.bam",
		},
		{
			name: "literal Append",
			spec: gobble.Literal("aln.bam").AppendExt(".bai"),
			want: "aln.bam.bai",
		},
		{
			name: "literal with directory",
			spec: gobble.Literal("work/star/sample.Aligned.sortedByCoord.out.bam"),
			want: "work/star/sample.Aligned.sortedByCoord.out.bam",
		},
		{
			name: "literal WithDir",
			spec: gobble.Literal("aln.bam").WithDir(gobble.Dir("work")),
			want: "work/aln.bam",
		},
		{
			name: "directory placement",
			spec: gobble.PathSpec{Dir: gobble.Dir("work/align"), Base: "sample", Suffixes: []string{"sorted"}, Ext: ".bam"},
			want: "work/align/sample.sorted.bam",
		},
		{
			name: "empty Steps and Ext",
			spec: gobble.PathSpec{Base: "LICENSE"},
			want: "LICENSE",
		},
		{
			name: "Ext without leading dot",
			spec: gobble.PathSpec{Base: "sample", Ext: "bam"},
			want: "sample.bam",
		},
		{
			name: "AppendSuffix strips one leading dot",
			spec: gobble.PathSpec{Base: "sample"}.AppendSuffix(".sorted").WithExt(".bam"),
			want: "sample.sorted.bam",
		},
		{
			name: "last-dot temptation",
			spec: gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"},
			want: "sample.fastq.gz",
		},
		{
			name: ".fastq stays on name",
			spec: gobble.PathSpec{Base: "sample", Ext: ".fastq"},
			want: "sample.fastq",
		},
		{
			name: "internal dotdot stays under first component",
			spec: gobble.PathSpec{Dir: gobble.Dir("work/align/../lane"), Base: "x", Ext: ".txt"},
			want: "work/lane/x.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.spec.Render()
			if err != nil {
				t.Fatalf("case %s: Render() error = %v", tt.name, err)
			}
			if got != tt.want {
				t.Fatalf("case %s: Render() got %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestPathSpecExtNotFilepathExt(t *testing.T) {
	p := gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"}
	lastDot := filepath.Ext("sample.fastq.gz")
	if p.Ext == lastDot {
		t.Fatalf("Ext got %q from last-dot filepath.Ext; want compound %q", p.Ext, ".fastq.gz")
	}
	got := mustRender(t, p)
	if got != "sample.fastq.gz" {
		t.Fatalf("Render() got %q, want %q", got, "sample.fastq.gz")
	}
}

func TestPathSpecInvalidPath(t *testing.T) {
	lit := gobble.Literal("aln.bam")
	tests := []struct {
		name string
		spec gobble.PathSpec
	}{
		{name: "lead slash", spec: gobble.PathSpec{Prefix: "a/b", Base: "x"}},
		{name: "lead backslash", spec: gobble.PathSpec{Prefix: `a\b`, Base: "x"}},
		{name: "lead NUL", spec: gobble.PathSpec{Prefix: "a\x00b", Base: "x"}},
		{name: "name slash", spec: gobble.PathSpec{Base: "a/b"}},
		{name: "name backslash", spec: gobble.PathSpec{Base: `a\b`}},
		{name: "name NUL", spec: gobble.PathSpec{Base: "a\x00b"}},
		{name: "step slash", spec: gobble.PathSpec{Base: "x", Suffixes: []string{"a/b"}}},
		{name: "step backslash", spec: gobble.PathSpec{Base: "x", Suffixes: []string{`a\b`}}},
		{name: "step NUL", spec: gobble.PathSpec{Base: "x", Suffixes: []string{"a\x00b"}}},
		{name: "ext slash", spec: gobble.PathSpec{Base: "x", Ext: ".a/b"}},
		{name: "ext backslash", spec: gobble.PathSpec{Base: "x", Ext: `.a\b`}},
		{name: "ext NUL", spec: gobble.PathSpec{Base: "x", Ext: ".\x00gz"}},
		{name: "step is dot", spec: gobble.PathSpec{Base: "x", Suffixes: []string{"."}}},
		{name: "step is dotdot", spec: gobble.PathSpec{Base: "x", Suffixes: []string{".."}}},
		{name: "step has dotdot component", spec: gobble.PathSpec{Base: "x", Suffixes: []string{"foo/.."}}},
		{name: "empty step token", spec: gobble.PathSpec{Base: "sample", Suffixes: []string{""}}},
		{name: "AppendSuffix empty", spec: gobble.PathSpec{Base: "sample"}.AppendSuffix("")},
		{name: "AppendSuffix dot", spec: gobble.PathSpec{Base: "sample", Ext: ".bam"}.AppendSuffix(".")},
		{name: "sample. hole", spec: gobble.PathSpec{Base: "sample"}.AppendSuffix("")},
		{name: ".fastq hole empty step", spec: gobble.PathSpec{Base: "sample", Suffixes: []string{""}, Ext: ".fastq"}},
		{name: "ext is dot", spec: gobble.PathSpec{Base: "x", Ext: "."}},
		{name: "ext is dotdot", spec: gobble.PathSpec{Base: "x", Ext: ".."}},
		{name: "ext has dotdot component", spec: gobble.PathSpec{Base: "x", Ext: "foo/.."}},
		{name: "empty lead and name", spec: gobble.PathSpec{Ext: ".bam"}},
		{name: "zero PathSpec", spec: gobble.PathSpec{}},
		{name: "empty literal", spec: gobble.Literal("")},
		{name: "literal AppendSuffix", spec: lit.AppendSuffix("sorted")},
		{name: "literal WithPrefix", spec: lit.WithPrefix("x")},
		{name: "literal WithBase", spec: lit.WithBase("x")},
		{name: "literal WithExt", spec: lit.WithExt(".bai")},
		{name: "dir escape parent", spec: gobble.PathSpec{Dir: gobble.Dir("work/../out"), Base: "x"}},
		{name: "dir escape above first", spec: gobble.PathSpec{Dir: gobble.Dir("../out"), Base: "x"}},
		{name: "dir escape two up", spec: gobble.PathSpec{Dir: gobble.Dir("work/align/../.."), Base: "x"}},
		{name: "name is dot", spec: gobble.PathSpec{Base: "."}},
		{name: "name is dotdot", spec: gobble.PathSpec{Base: ".."}},
		{name: "Append empty extra", spec: gobble.PathSpec{Base: "sample", Ext: ".bam"}.AppendExt("")},
		{name: "Append dot extra", spec: gobble.PathSpec{Base: "sample", Ext: ".bam"}.AppendExt(".")},
		{name: "ext trailing dot", spec: gobble.PathSpec{Base: "sample", Ext: ".bam."}},
		{name: "literal Append empty extra", spec: gobble.Literal("aln.bam").AppendExt("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.spec.Render()
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: Render() got path %q, error = %v, want *Error", tt.name, got, err)
			}
			if ge.Op != "render" {
				t.Fatalf("case %s: Error.Op got %q, want %q", tt.name, ge.Op, "render")
			}
			found := false
			for _, d := range ge.Defects {
				if d.Code == gobble.DefectInvalidPath {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("case %s: Error.Defects codes got %v, want %s", tt.name, defectCodes(ge), gobble.DefectInvalidPath)
			}
			if got != "" {
				t.Fatalf("case %s: Render() got path %q, want empty", tt.name, got)
			}
		})
	}
}

func TestPathSpecEqual(t *testing.T) {
	a := gobble.PathSpec{Dir: gobble.Dir("work"), Prefix: "L", Base: "n", Suffixes: []string{"sorted"}, Ext: ".bam"}
	b := gobble.PathSpec{Dir: gobble.Dir("work"), Prefix: "L", Base: "n", Suffixes: []string{"sorted"}, Ext: ".bam"}
	if !a.Equal(b) {
		t.Fatalf("Equal() got false, want true for same fields")
	}
	if a.Equal(a.WithPrefix("other")) {
		t.Fatalf("Equal() got true, want false after WithPrefix")
	}
	lit := gobble.Literal("sample.bam")
	fields := gobble.PathSpec{Base: "sample", Ext: ".bam"}
	if lit.Equal(fields) {
		t.Fatalf("Equal() got true, want false for literal vs fields")
	}
	if !lit.Equal(gobble.Literal("sample.bam")) {
		t.Fatalf("Equal() got false, want true for same literal")
	}
}

func TestPathSpecStepsOwnership(t *testing.T) {
	orig := gobble.PathSpec{Base: "sample", Suffixes: []string{"sorted"}}
	next := orig.AppendSuffix("markdup")
	orig.Suffixes[0] = "mutated"
	if next.Suffixes[0] != "sorted" {
		t.Fatalf("AppendSuffix share: next.Suffixes[0] got %q, want %q", next.Suffixes[0], "sorted")
	}
	copied := orig.WithExt(".bam")
	orig.Suffixes[0] = "again"
	if copied.Suffixes[0] != "mutated" {
		t.Fatalf("WithExt share: copied.Suffixes[0] got %q, want %q", copied.Suffixes[0], "mutated")
	}
}

func TestDirectory(t *testing.T) {
	var z gobble.Directory
	if !z.IsZero() {
		t.Fatalf("zero Directory.IsZero() got false, want true")
	}
	if z.String() != "" {
		t.Fatalf("zero Directory.String() got %q, want empty", z.String())
	}
	d := gobble.Dir("work/align")
	if d.IsZero() {
		t.Fatalf("Dir(%q).IsZero() got true, want false", "work/align")
	}
	if d.String() != "work/align" {
		t.Fatalf("Dir.String() got %q, want %q", d.String(), "work/align")
	}
	got := d.Join("lane1")
	if got.String() != "work/align/lane1" {
		t.Fatalf("Join() got %q, want %q", got.String(), "work/align/lane1")
	}
	escaped := gobble.PathSpec{Dir: gobble.Dir("work").Join("..", "out"), Base: "x"}
	path, err := escaped.Render()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("Join escape: Render() got path %q, error = %v, want *Error", path, err)
	}
	found := false
	for _, d := range ge.Defects {
		if d.Code == gobble.DefectInvalidPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Join escape: Error.Defects codes got %v, want %s", defectCodes(ge), gobble.DefectInvalidPath)
	}
	if path != "" {
		t.Fatalf("Join escape: Render() got path %q, want empty", path)
	}
}

func TestDeriveRuleConstants(t *testing.T) {
	if gobble.DeriveAppend != 0 {
		t.Fatalf("DeriveAppend got %d, want 0", gobble.DeriveAppend)
	}
	if gobble.DeriveReplaceExt != 1 {
		t.Fatalf("DeriveReplaceExt got %d, want 1", gobble.DeriveReplaceExt)
	}
	var zero gobble.DeriveRule
	if zero != gobble.DeriveAppend {
		t.Fatalf("zero DeriveRule got %d, want DeriveAppend", zero)
	}
}

func TestErrorAs(t *testing.T) {
	_, err := gobble.PathSpec{}.Render()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("errors.As got false, want *Error")
	}
	if ge.Error() == "" {
		t.Fatalf("Error.Error() got empty, want a message")
	}
}
