// Package rnaseq owns the graph-stable RNA-seq migration checkpoint.
package rnaseq

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules/deseq2"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	mergecounts "github.com/HahyeonJeon/gobble/assets/modules/merge-counts"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	staralign "github.com/HahyeonJeon/gobble/assets/modules/star-align"
	stargenomegenerate "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
)

const (
	rnaGroupRuleMessage        = "RNA samplesheet requires group on every row and exactly two groups"
	rnaStrandednessRuleMessage = "RNA samplesheet strandedness must be unstranded, forward, or reverse"
	rnaMateRuleMessage         = "RNA samplesheet requires read2 on every row"
)

// Pipeline returns the graph-stable RNA-seq checkpoint. It loads SampleSheetPath,
// expands one module per sample, and records compose errors for sheet
// failures. Empty optional reference and gtf cells bind the official
// RNA pins.
func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("rnaseq")
	sheet, err := gobble.LoadSampleSheet()
	if err != nil {
		p.RecordComposeError(err)
		return p
	}
	if err := rnaSheetRules(sheet); err != nil {
		p.RecordComposeError(err)
		return p
	}

	fastaSpec := pinnedRNAFASTA()
	if ref, ok := firstNonEmpty(sheet.Rows, func(r gobble.SampleRow) string { return r.Reference }); ok {
		fastaSpec = gobble.Literal(ref)
	}
	gtfSpec := pinnedRNAGTF()
	if gtfPath, ok := firstNonEmpty(sheet.Rows, func(r gobble.SampleRow) string { return r.GTF }); ok {
		gtfSpec = gobble.Literal(gtfPath)
	}
	fasta := p.AddInput("fasta", fastaSpec)
	gtf := p.AddInput("gtf", gtfSpec)
	index := stargenomegenerate.AddSTARGenomeGenerate(p, fasta, gtf, stargenomegenerate.STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 2},
	})

	var reports []gobble.Handle
	var counts []gobble.Handle
	var names []string
	var groups []string
	for _, row := range sheet.Rows {
		r1 := p.AddInput(row.Sample+"_r1", sheetFileSpec(row.Read1))
		r2 := p.AddInput(row.Sample+"_r2", sheetFileSpec(row.Read2))
		mod := p.AddModule(row.Sample)
		work := "work/" + row.Sample
		rawQC := fastqc.AddFastQC(mod.AddModule("raw"), r1, fastqc.FastQCOptions{OutDir: gobble.Dir(work + "/raw/fastqc")})
		trimmed := fastp.AddFastp(mod, r1, r2, fastp.FastpOptions{OutDir: gobble.Dir(work + "/fastp")})
		cleanQC := fastqc.AddFastQC(mod.AddModule("clean"), trimmed.CleanR1, fastqc.FastQCOptions{OutDir: gobble.Dir(work + "/clean/fastqc")})
		align := staralign.AddSTARAlign(mod, index.Index, trimmed.CleanR1, trimmed.CleanR2, staralign.STARAlignOptions{
			OutDir:    gobble.Dir(work + "/star-align"),
			Resources: gobble.Resources{CPU: 2},
		})
		sorted := samtoolssort.AddSamtoolsSort(mod, align.BAM, samtoolssort.SamtoolsSortOptions{OutDir: gobble.Dir(work + "/samtools-sort")})
		samtoolsindex.AddSamtoolsIndex(mod, sorted.BAM, samtoolsindex.SamtoolsIndexOptions{})
		strand := row.Strandedness
		if strand == "" {
			strand = gobble.DefaultRNAStrandedness
		}
		fc := featurecounts.AddFeatureCounts(mod, sorted.BAM, gtf, featurecounts.FeatureCountsOptions{
			OutDir:       gobble.Dir(work + "/featurecounts"),
			Strandedness: strand,
		})
		reports = append(reports, rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip, trimmed.JSON, trimmed.HTML, align.LogFinalOut)
		counts = append(counts, fc.Counts)
		names = append(names, row.Sample)
		groups = append(groups, row.Group)
	}
	merged := mergecounts.AddMergeCounts(p, counts, mergecounts.MergeCountsOptions{SampleNames: names})
	deseq2.AddDESeq2(p, merged.Counts, groups, deseq2.DESeq2Options{})
	multiqc.AddMultiQC(p, reports, multiqc.MultiQCOptions{})
	return p
}

func rnaSheetRules(sheet *gobble.SampleSheet) error {
	if err := requireMateRows(sheet, rnaMateRuleMessage); err != nil {
		return err
	}
	groups := make(map[string]struct{})
	for _, row := range sheet.Rows {
		if row.Group == "" {
			return sheetRuleError(sheet.Path, rnaGroupRuleMessage)
		}
		groups[row.Group] = struct{}{}
		switch row.Strandedness {
		case "", gobble.StrandednessUnstranded, gobble.StrandednessForward, gobble.StrandednessReverse:
		default:
			return sheetRuleError(sheet.Path, rnaStrandednessRuleMessage)
		}
	}
	if len(groups) != 2 {
		return sheetRuleError(sheet.Path, rnaGroupRuleMessage)
	}
	return nil
}

func requireMateRows(sheet *gobble.SampleSheet, message string) error {
	for _, row := range sheet.Rows {
		if row.Read2 == "" {
			return sheetRuleError(sheet.Path, message)
		}
	}
	return nil
}

func sheetRuleError(path, message string) error {
	return &gobble.Error{
		Op: "compose",
		Defects: []gobble.Defect{{
			Code:    gobble.DefectInvalidSampleSheet,
			Unit:    "samplesheet",
			Message: message,
			Paths:   []string{path},
		}},
	}
}

func firstNonEmpty(rows []gobble.SampleRow, cell func(gobble.SampleRow) string) (string, bool) {
	for _, row := range rows {
		if v := cell(row); v != "" {
			return v, true
		}
	}
	return "", false
}

// sheetFileSpec turns a workspace-relative sheet cell into Dir/Base/Ext.
// Adders such as fastp call AppendSuffix on the read PathSpec, which
// Literal forbids.
func sheetFileSpec(path string) gobble.PathSpec {
	dir, file := "", path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		dir, file = path[:i], path[i+1:]
	}
	base, ext := file, ""
	lower := strings.ToLower(file)
	for _, e := range []string{".fastq.gz", ".fq.gz"} {
		if strings.HasSuffix(lower, e) {
			base = file[:len(file)-len(e)]
			ext = file[len(base):]
			return fileSpec(dir, base, ext)
		}
	}
	if i := strings.LastIndex(file, "."); i > 0 {
		base, ext = file[:i], file[i:]
	}
	return fileSpec(dir, base, ext)
}

func fileSpec(dir, base, ext string) gobble.PathSpec {
	spec := gobble.PathSpec{Base: base, Ext: ext}
	if dir != "" {
		spec.Dir = gobble.Dir(dir)
	}
	return spec
}

func pinnedRNAFASTA() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
}

func pinnedRNAGTF() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
}
