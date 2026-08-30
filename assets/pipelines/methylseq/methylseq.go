// Package methylseq owns the graph-stable Methyl-seq migration checkpoint.
package methylseq

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	bismarkmethylationextractor "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
)

const (
	methylTwoRowRuleMessage = "Methyl samplesheet requires at least two samples"
	methylMateRuleMessage   = "Methyl samplesheet requires read2 on every row"
)

// Pipeline returns the graph-stable Methyl-seq checkpoint. It loads SampleSheetPath,
// expands one module per sample, and records compose errors for sheet
// failures. Empty optional reference cells bind the official Methyl FASTA.
// Group and gtf are not required. There is no DMR task.
func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("methylseq")
	sheet, err := gobble.LoadSampleSheet()
	if err != nil {
		p.RecordComposeError(err)
		return p
	}
	if len(sheet.Rows) < 2 {
		p.RecordComposeError(sheetRuleError(sheet.Path, methylTwoRowRuleMessage))
		return p
	}
	if err := requireMateRows(sheet, methylMateRuleMessage); err != nil {
		p.RecordComposeError(err)
		return p
	}

	fastaSpec := pinnedMethylFASTA()
	if ref, ok := firstNonEmpty(sheet.Rows, func(r gobble.SampleRow) string { return r.Reference }); ok {
		fastaSpec = gobble.Literal(ref)
	}
	fasta := p.AddInput("fasta", fastaSpec)
	index := bismarkgenome.AddBismarkGenome(p, fasta, bismarkgenome.BismarkGenomeOptions{})

	var reports []gobble.Handle
	for _, row := range sheet.Rows {
		r1 := p.AddInput(row.Sample+"_r1", sheetFileSpec(row.Read1))
		r2 := p.AddInput(row.Sample+"_r2", sheetFileSpec(row.Read2))
		mod := p.AddModule(row.Sample)
		work := "work/" + row.Sample
		rawQC := fastqc.AddFastQC(mod.AddModule("raw"), r1, fastqc.FastQCOptions{OutDir: gobble.Dir(work + "/raw/fastqc")})
		trimmed := fastp.AddFastp(mod, r1, r2, fastp.FastpOptions{OutDir: gobble.Dir(work + "/fastp")})
		cleanQC := fastqc.AddFastQC(mod.AddModule("clean"), trimmed.CleanR1, fastqc.FastQCOptions{OutDir: gobble.Dir(work + "/clean/fastqc")})
		align := bismarkalign.AddBismarkAlign(mod, fasta, index.Index, trimmed.CleanR1, trimmed.CleanR2, bismarkalign.BismarkAlignOptions{
			OutDir: gobble.Dir(work + "/bismark-align"),
		})
		extractor := bismarkmethylationextractor.AddBismarkMethylationExtractor(mod, align.BAM, bismarkmethylationextractor.BismarkMethylationExtractorOptions{
			OutDir: gobble.Dir(work + "/bismark-extract"),
		})
		reports = append(reports,
			rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip,
			trimmed.JSON, trimmed.HTML,
			align.Report, extractor.Report, extractor.Mbias, extractor.Coverage,
		)
	}
	multiqc.AddMultiQC(p, reports, multiqc.MultiQCOptions{})
	return p
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
		if value := cell(row); value != "" {
			return value, true
		}
	}
	return "", false
}

func sheetFileSpec(path string) gobble.PathSpec {
	dir, file := "", path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		dir, file = path[:i], path[i+1:]
	}
	base, ext := file, ""
	lower := strings.ToLower(file)
	for _, candidate := range []string{".fastq.gz", ".fq.gz"} {
		if strings.HasSuffix(lower, candidate) {
			base = file[:len(file)-len(candidate)]
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

func pinnedMethylFASTA() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"}
}
