package assets

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
)

const (
	rnaGroupRuleMessage        = "RNA samplesheet requires group on every row and exactly two groups"
	rnaStrandednessRuleMessage = "RNA samplesheet strandedness must be unstranded, forward, or reverse"
	rnaMateRuleMessage         = "RNA samplesheet requires read2 on every row"
	methylTwoRowRuleMessage    = "Methyl samplesheet requires at least two samples"
	methylMateRuleMessage      = "Methyl samplesheet requires read2 on every row"
)

// RNASeq returns an RNA-seq proof pipeline. It loads SampleSheetPath,
// expands one module per sample, and records compose errors for sheet
// failures. Empty optional reference and gtf cells bind the official
// RNA pins.
func RNASeq() *gobble.Pipeline {
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
	index := AddSTARGenomeGenerate(p, fasta, gtf, STARGenomeGenerateOptions{
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
		mod := AddModule(p, row.Sample)
		work := "work/" + row.Sample
		rawQC := AddFastQC(AddModule(mod, "raw"), r1, FastQCOptions{OutDir: gobble.Dir(work + "/raw/fastqc")})
		fastp := AddFastp(mod, r1, r2, FastpOptions{OutDir: gobble.Dir(work + "/fastp")})
		cleanQC := AddFastQC(AddModule(mod, "clean"), fastp.CleanR1, FastQCOptions{OutDir: gobble.Dir(work + "/clean/fastqc")})
		align := AddSTARAlign(mod, index.Index, fastp.CleanR1, fastp.CleanR2, STARAlignOptions{
			OutDir:    gobble.Dir(work + "/star-align"),
			Resources: gobble.Resources{CPU: 2},
		})
		sorted := AddSamtoolsSort(mod, align.BAM, SamtoolsSortOptions{OutDir: gobble.Dir(work + "/samtools-sort")})
		AddSamtoolsIndex(mod, sorted.BAM, SamtoolsIndexOptions{})
		strand := row.Strandedness
		if strand == "" {
			strand = gobble.DefaultRNAStrandedness
		}
		fc := AddFeatureCounts(mod, sorted.BAM, gtf, FeatureCountsOptions{
			OutDir:       gobble.Dir(work + "/featurecounts"),
			Strandedness: strand,
		})
		reports = append(reports, rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip, fastp.JSON, fastp.HTML, align.LogFinalOut)
		counts = append(counts, fc.Counts)
		names = append(names, row.Sample)
		groups = append(groups, row.Group)
	}
	merged := AddMergeCounts(p, counts, MergeCountsOptions{SampleNames: names})
	AddDESeq2(p, merged.Counts, groups, DESeq2Options{})
	AddMultiQC(p, reports, MultiQCOptions{})
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
