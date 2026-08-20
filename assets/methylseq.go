package assets

import "github.com/HahyeonJeon/gobble"

// MethylSeq returns a Methyl-seq proof pipeline. It loads SampleSheetPath,
// expands one module per sample, and records compose errors for sheet
// failures. Empty optional reference cells bind the official Methyl FASTA.
// Group and gtf are not required. There is no DMR task.
func MethylSeq() *gobble.Pipeline {
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

	fastaSpec := pinnedMethylFASTA()
	if ref, ok := firstNonEmpty(sheet.Rows, func(r gobble.SampleRow) string { return r.Reference }); ok {
		fastaSpec = gobble.Literal(ref)
	}
	fasta := p.AddInput("fasta", fastaSpec)
	index := AddBismarkGenome(p, fasta, BismarkGenomeOptions{})

	var reports []gobble.Handle
	for _, row := range sheet.Rows {
		r1 := p.AddInput(row.Sample+"_r1", sheetFileSpec(row.Read1))
		r2 := p.AddInput(row.Sample+"_r2", sheetFileSpec(row.Read2))
		mod := AddModule(p, row.Sample)
		work := "work/" + row.Sample
		rawQC := AddFastQC(AddModule(mod, "raw"), r1, FastQCOptions{OutDir: gobble.Dir(work + "/raw/fastqc")})
		fastp := AddFastp(mod, r1, r2, FastpOptions{OutDir: gobble.Dir(work + "/fastp")})
		cleanQC := AddFastQC(AddModule(mod, "clean"), fastp.CleanR1, FastQCOptions{OutDir: gobble.Dir(work + "/clean/fastqc")})
		align := AddBismarkAlign(mod, fasta, index.Index, fastp.CleanR1, fastp.CleanR2, BismarkAlignOptions{
			OutDir: gobble.Dir(work + "/bismark-align"),
		})
		extractor := AddBismarkMethylationExtractor(mod, align.BAM, BismarkMethylationExtractorOptions{
			OutDir: gobble.Dir(work + "/bismark-extract"),
		})
		reports = append(reports,
			rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip,
			fastp.JSON, fastp.HTML,
			align.Report, extractor.Report, extractor.Mbias, extractor.Coverage,
		)
	}
	AddMultiQC(p, reports, MultiQCOptions{})
	return p
}
