package rnaseq

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bedtoolsgenomecov "github.com/HahyeonJeon/gobble/assets/modules/bedtools-genomecov"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	cutchromsizes "github.com/HahyeonJeon/gobble/assets/modules/cut-chrom-sizes"
	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
	"github.com/HahyeonJeon/gobble/assets/modules/dupradar"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	fqlint "github.com/HahyeonJeon/gobble/assets/modules/fq-lint"
	"github.com/HahyeonJeon/gobble/assets/modules/gffread"
	"github.com/HahyeonJeon/gobble/assets/modules/gunzip"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	picardmarkduplicates "github.com/HahyeonJeon/gobble/assets/modules/picard-markduplicates"
	qualimapbamqc "github.com/HahyeonJeon/gobble/assets/modules/qualimap-bamqc"
	rseqcinferexperiment "github.com/HahyeonJeon/gobble/assets/modules/rseqc-inferexperiment"
	salmonindex "github.com/HahyeonJeon/gobble/assets/modules/salmon-index"
	salmonquant "github.com/HahyeonJeon/gobble/assets/modules/salmon-quant"
	samtoolsfaidx "github.com/HahyeonJeon/gobble/assets/modules/samtools-faidx"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
	staralign "github.com/HahyeonJeon/gobble/assets/modules/star-align"
	stargenomegenerate "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
	"github.com/HahyeonJeon/gobble/assets/modules/stringtie"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	"github.com/HahyeonJeon/gobble/assets/modules/tximport"
	ucscbedclip "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedclip"
	ucscbedgraphtobigwig "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedgraphtobigwig"
)

// Build constructs the selected STAR-Salmon graph only from supplied values.
// It copies samples, runs, config slices, and module ExtraArgs before use.
func Build(inputSamples []Sample, inputConfig Config) *gobble.Pipeline {
	pipeline := gobble.NewPipeline("rnaseq")
	samples := cloneSamples(inputSamples)
	config := cloneConfig(inputConfig)
	if defects := validateBuild(samples, config); len(defects) > 0 {
		pipeline.RecordComposeError(composeError(defects))
		return pipeline
	}

	fasta := pipeline.AddInput("fasta", config.Reference.FASTA)
	gtfCompressed := pipeline.AddInput("gtf", config.Reference.GTF)
	reference := pipeline.AddModule("reference")
	referenceDir := gobble.Dir("work/reference")
	if config.Publication.GeneratedReference {
		referenceDir = gobble.Dir(config.Results.String() + "/reference")
	}
	gtf := gtfCompressed
	if config.Reference.GTFCompressed {
		gunzipOptions := config.Gunzip
		gunzipOptions.OutDir = referenceDir
		gunzipOptions.Prefix = "genes"
		annotation, err := gunzip.Add(reference, gtfCompressed, gunzipOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		gtf = annotation.File
	}

	transcriptOptions := config.GFFRead
	transcriptOptions.OutDir = referenceDir
	transcriptOptions.Prefix = "transcriptome"
	transcriptome, err := gffread.AddTranscriptome(reference.AddModule("transcriptome"), gtf, fasta, transcriptOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	bedOptions := config.GFFRead
	bedOptions.OutDir = referenceDir
	bedOptions.Prefix = "genes"
	geneBED, err := gffread.AddBED(reference.AddModule("gene_intervals"), gtf, fasta, bedOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	fai, err := samtoolsfaidx.Add(reference, fasta, config.FAIDX)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	chromSizes, err := cutchromsizes.Add(reference, fai.FAI, config.ChromSizes)
	if recordModuleError(pipeline, err) {
		return pipeline
	}

	var starIndexHandle gobble.Handle
	if config.Reference.STARIndex.IsZero() {
		starOptions := config.STARGenome
		starOptions.OutDir = referenceDir.Join("star-index")
		starIndex, addErr := stargenomegenerate.Add(reference, fasta, gtf, starOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		starIndexHandle = starIndex.Index
	} else {
		starIndexHandle = pipeline.AddInputTree("star_index", config.Reference.STARIndex)
	}

	needsInference := false
	for _, sample := range samples {
		needsInference = needsInference || sample.Strandedness == StrandednessAuto
	}
	var salmonIndexHandle gobble.Handle
	if needsInference {
		if config.Reference.SalmonIndex.IsZero() {
			salmonOptions := config.SalmonIndex
			salmonOptions.OutDir = referenceDir.Join("salmon-index")
			salmonIndexPorts, addErr := salmonindex.Add(reference, transcriptome.Output, salmonOptions)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			salmonIndexHandle = salmonIndexPorts.Index
		} else {
			salmonIndexHandle = pipeline.AddInputTree("salmon_index", config.Reference.SalmonIndex)
		}
	}

	var reports []gobble.Handle
	var quants []gobble.Handle
	sampleNames := make([]string, 0, len(samples))
	for _, sample := range samples {
		module := pipeline.AddModule(sample.Name)
		read1s := make([]gobble.Handle, 0, len(sample.Runs))
		read2s := make([]gobble.Handle, 0, len(sample.Runs))
		for _, run := range sample.Runs {
			runName := run.ID
			read1 := pipeline.AddInput(sample.Name+"_"+runName+"_r1", sheetFileSpec(run.Fastq1))
			read1s = append(read1s, read1)
			raw1 := module.AddModule(runName + "_raw_r1")
			lintOptions := config.FQLint
			lintOptions.OutDir = gobble.Dir("work/" + sample.Name + "/raw")
			lintOptions.Prefix = runName + "_r1"
			lint, addErr := fqlint.Add(raw1, read1, lintOptions)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			qcOptions := config.FastQC
			qcOptions.OutDir = gobble.Dir("work/" + sample.Name + "/raw/fastqc")
			qc, addErr := fastqc.Add(raw1, read1, qcOptions)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			reports = append(reports, lint.Report, qc.HTML, qc.Zip)
			if run.Fastq2 == "" {
				continue
			}
			read2 := pipeline.AddInput(sample.Name+"_"+runName+"_r2", sheetFileSpec(run.Fastq2))
			read2s = append(read2s, read2)
			raw2 := module.AddModule(runName + "_raw_r2")
			lintOptions.Prefix = runName + "_r2"
			lint, addErr = fqlint.Add(raw2, read2, lintOptions)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			qc, addErr = fastqc.Add(raw2, read2, qcOptions)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			reports = append(reports, lint.Report, qc.HTML, qc.Zip)
		}

		read1 := read1s[0]
		var read2 gobble.Handle
		if len(read2s) > 0 {
			read2 = read2s[0]
		}
		if len(sample.Runs) > 1 {
			catOptions := config.CatFASTQ
			catOptions.OutDir = gobble.Dir("work/" + sample.Name + "/consolidated")
			catOptions.Prefix = sample.Name + "_1"
			consolidated, addErr := catfastq.Add(module.AddModule("consolidate_r1"), read1s, catOptions)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			read1 = consolidated.FASTQ
			if len(read2s) > 0 {
				catOptions.Prefix = sample.Name + "_2"
				consolidated, addErr = catfastq.Add(module.AddModule("consolidate_r2"), read2s, catOptions)
				if recordModuleError(pipeline, addErr) {
					return pipeline
				}
				read2 = consolidated.FASTQ
			}
		}

		trimOptions := config.TrimGalore
		trimOptions.OutDir = gobble.Dir("work/" + sample.Name + "/trim-galore")
		if config.Publication.TrimmedReads {
			trimOptions.OutDir = gobble.Dir(config.Results.String() + "/intermediates/trimmed/" + sample.Name)
		}
		trimOptions.Prefix = sample.Name
		trimmed, addErr := trimgalore.Add(module, read1, read2, trimOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, trimmed.Report1)
		if !trimmed.Report2.IsZero() {
			reports = append(reports, trimmed.Report2)
		}
		cleanReads := []struct {
			name   string
			handle gobble.Handle
		}{{name: "r1", handle: trimmed.Read1}}
		if !trimmed.Read2.IsZero() {
			cleanReads = append(cleanReads, struct {
				name   string
				handle gobble.Handle
			}{name: "r2", handle: trimmed.Read2})
		}
		for _, clean := range cleanReads {
			cleanModule := module.AddModule("post_trim_" + clean.name)
			lintOptions := config.FQLint
			lintOptions.OutDir = gobble.Dir("work/" + sample.Name + "/post-trim")
			lintOptions.Prefix = sample.Name + "_" + clean.name
			lint, lintErr := fqlint.Add(cleanModule, clean.handle, lintOptions)
			if recordModuleError(pipeline, lintErr) {
				return pipeline
			}
			qcOptions := config.FastQC
			qcOptions.OutDir = gobble.Dir("work/" + sample.Name + "/post-trim/fastqc")
			qc, qcErr := fastqc.Add(cleanModule, clean.handle, qcOptions)
			if recordModuleError(pipeline, qcErr) {
				return pipeline
			}
			reports = append(reports, lint.Report, qc.HTML, qc.Zip)
		}

		retained, retentionErr := addTrimmedReadGate(module, trimmed.Read1, trimmed.Read2, sample.Name, config.SampleRemoval.MinTrimmedReads)
		if recordModuleError(pipeline, retentionErr) {
			return pipeline
		}

		var inferred gobble.Handle
		if sample.Strandedness == StrandednessAuto {
			inferenceOptions := config.Salmon
			inferenceOptions.OutDir = gobble.Dir("work/" + sample.Name + "/strandedness")
			inferenceOptions.Prefix = sample.Name
			inferenceThresholds := salmonquant.InferenceThresholds{
				StrandedFraction:     config.StrandednessInference.StrandedFraction,
				UnstrandedDifference: config.StrandednessInference.UnstrandedDifference,
			}
			inference, inferenceErr := salmonquant.AddInference(module.AddModule("strandedness"), salmonIndexHandle, trimmed.Read1, trimmed.Read2, retained, inferenceThresholds, inferenceOptions)
			if recordModuleError(pipeline, inferenceErr) {
				return pipeline
			}
			inferred = inference.Strandedness
			reports = append(reports, inference.MetaInfo, inference.LibFormatCounts, inference.Log)
		}

		starOptions := config.STAR
		starOptions.OutDir = gobble.Dir("work/" + sample.Name + "/star")
		if config.Publication.STARAlignments {
			starOptions.OutDir = gobble.Dir(config.Results.String() + "/intermediates/star/" + sample.Name)
		}
		starOptions.Sample = sample.Name
		starOptions.ReadGroup = sample.Name
		starOptions.Platform = sample.SeqPlatform
		starOptions.Center = sample.SeqCenter
		aligned, addErr := staralign.AddAfter(module, starIndexHandle, gtf, trimmed.Read1, trimmed.Read2, []gobble.Handle{retained, inferred}, starOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, aligned.LogFinal, aligned.Junctions)
		accepted, policyErr := addMappedReadGate(module, aligned.LogFinal, retained, sample.Name, config.SampleRemoval.MinMappedPercent)
		if recordModuleError(pipeline, policyErr) {
			return pipeline
		}

		sortOptions := config.Sort
		sortOptions.OutDir = gobble.Dir("work/" + sample.Name + "/sorted")
		sortOptions.Prefix = sample.Name
		sorted, addErr := samtoolssort.AddAfter(module, aligned.GenomeBAM, accepted, sortOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		markOptions := config.MarkDuplicates
		markOptions.OutDir = gobble.Dir(config.Results.String() + "/bam/" + sample.Name)
		markOptions.Prefix = sample.Name + ".marked"
		marked, addErr := picardmarkduplicates.Add(module, sorted.BAM, markOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		indexed, addErr := samtoolsindex.Add(module, marked.BAM, config.Index)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		statsOptions := config.Stats
		statsOptions.OutDir = gobble.Dir(config.Results.String() + "/qc/" + sample.Name)
		statsOptions.Prefix = sample.Name
		stats, addErr := samtoolsstats.Add(module, marked.BAM, statsOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, markOptionsResult(marked), stats.Stats)

		salmonOptions := config.Salmon
		salmonOptions.OutDir = gobble.Dir(config.Results.String() + "/salmon")
		salmonOptions.Prefix = sample.Name
		paired := sample.Runs[0].Fastq2 != ""
		var quant salmonquant.Ports
		if inferred.IsZero() {
			salmonOptions.LibType = salmonLibType(sample.Strandedness, paired)
			quant, addErr = salmonquant.AddAlignmentAfter(module, aligned.TranscriptBAM, transcriptome.Output, gtf, accepted, salmonOptions)
		} else {
			salmonOptions.LibType = ""
			quant, addErr = salmonquant.AddAlignmentInferred(module, aligned.TranscriptBAM, transcriptome.Output, gtf, inferred, accepted, paired, salmonOptions)
		}
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		quants = append(quants, quant.Quant)
		sampleNames = append(sampleNames, sample.Name)
		reports = append(reports, quant.MetaInfo, quant.LibFormatCounts, quant.Log)

		stringTieOptions := config.StringTie
		stringTieOptions.OutDir = gobble.Dir(config.Results.String() + "/stringtie/" + sample.Name)
		stringTieOptions.Prefix = sample.Name
		var stringTiePorts stringtie.Ports
		if inferred.IsZero() {
			stringTieOptions.Strandedness = string(sample.Strandedness)
			stringTiePorts, addErr = stringtie.Add(module, marked.BAM, gtf, stringTieOptions)
		} else {
			stringTieOptions.Strandedness = ""
			stringTiePorts, addErr = stringtie.AddInferred(module, marked.BAM, gtf, inferred, stringTieOptions)
		}
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, stringTiePorts.Abundance, stringTiePorts.Coverage)

		coverageErr := addCoverage(module, marked.BAM, chromSizes.Sizes, sample, inferred, config)
		if recordModuleError(pipeline, coverageErr) {
			return pipeline
		}

		rseqcOptions := config.RSeQC
		rseqcOptions.OutDir = gobble.Dir(config.Results.String() + "/qc/" + sample.Name)
		rseqcOptions.Prefix = sample.Name
		rseqc, addErr := rseqcinferexperiment.Add(module, marked.BAM, indexed.BAI, geneBED.Output, rseqcOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		qualimapOptions := config.Qualimap
		qualimapOptions.OutDir = gobble.Dir(config.Results.String() + "/qc/qualimap")
		qualimapOptions.Prefix = sample.Name
		qualimapOptions.Paired = paired
		var qualimapPorts qualimapbamqc.Ports
		if inferred.IsZero() {
			qualimapOptions.Strandedness = string(sample.Strandedness)
			qualimapPorts, addErr = qualimapbamqc.Add(module, marked.BAM, gtf, qualimapOptions)
		} else {
			qualimapOptions.Strandedness = ""
			qualimapPorts, addErr = qualimapbamqc.AddInferred(module, marked.BAM, gtf, inferred, qualimapOptions)
		}
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		dupOptions := config.DupRadar
		dupOptions.OutDir = gobble.Dir(config.Results.String() + "/qc/" + sample.Name)
		dupOptions.Prefix = sample.Name
		dupOptions.Paired = paired
		var dup dupradar.Ports
		if inferred.IsZero() {
			dupOptions.Strandedness = string(sample.Strandedness)
			dup, addErr = dupradar.Add(module, marked.BAM, gtf, dupOptions)
		} else {
			dupOptions.Strandedness = ""
			dup, addErr = dupradar.AddInferred(module, marked.BAM, gtf, inferred, dupOptions)
		}
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		biotypeOptions := config.BiotypeQC
		biotypeOptions.OutDir = gobble.Dir(config.Results.String() + "/qc/" + sample.Name)
		biotypeOptions.Paired = paired
		var biotype featurecounts.BiotypePorts
		if inferred.IsZero() {
			biotypeOptions.Strandedness = string(sample.Strandedness)
			biotype, addErr = featurecounts.AddBiotype(module, marked.BAM, gtf, biotypeOptions)
		} else {
			biotypeOptions.Strandedness = ""
			biotype, addErr = featurecounts.AddBiotypeInferred(module, marked.BAM, gtf, inferred, biotypeOptions)
		}
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		// Tree outputs remain declared final artifacts. MultiQC consumes regular
		// reports only and never treats a directory as one file.
		reports = append(reports, rseqc.Report, qualimapPorts.Report, dup.MultiQC, biotype.Counts, biotype.Summary)
	}

	txOptions := config.TxImport
	txOptions.OutDir = gobble.Dir(config.Results.String() + "/matrices")
	matrices, err := tximport.Add(pipeline.AddModule("cohort"), quants, sampleNames, gtf, txOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	deseqOptions := config.DESeq2QC
	deseqOptions.OutDir = gobble.Dir(config.Results.String() + "/deseq2-qc")
	cohortQC, err := deseq2qc.Add(pipeline.AddModule("cohort_qc"), matrices.GeneLengthScaled, deseqOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	reports = append(reports, matrices.GeneCounts, matrices.GeneTPM, matrices.TranscriptCounts, matrices.TranscriptTPM, cohortQC.PCA, cohortQC.Distance)
	multiQCOptions := config.MultiQC
	multiQCOptions.OutDir = gobble.Dir(config.Results.String() + "/multiqc")
	if _, err = multiqc.Add(pipeline, reports, multiQCOptions); recordModuleError(pipeline, err) {
		return pipeline
	}
	return pipeline
}

func addCoverage(parent *gobble.Module, bam, sizes gobble.Handle, sample Sample, inferred gobble.Handle, config Config) error {
	strands := []struct {
		name   string
		strand string
	}{{name: "combined"}}
	if sample.Strandedness != StrandednessUnstranded {
		strands = append(strands, struct{ name, strand string }{name: "forward", strand: "+"}, struct{ name, strand string }{name: "reverse", strand: "-"})
	}
	for _, strand := range strands {
		stage := parent.AddModule("coverage_" + strand.name)
		coverageOptions := config.GenomeCov
		coverageOptions.OutDir = gobble.Dir("work/" + sample.Name + "/coverage")
		coverageOptions.Prefix = sample.Name + "." + strand.name
		coverageOptions.Strand = strand.strand
		if !inferred.IsZero() && strand.strand != "" {
			coverage, err := bedtoolsgenomecov.AddInferred(stage, bam, inferred, strand.strand, coverageOptions)
			if err != nil {
				return err
			}
			clipOptions := config.BedClip
			clipOptions.OutDir = gobble.Dir("work/" + sample.Name + "/coverage")
			clipOptions.Prefix = sample.Name + "." + strand.name + ".clipped"
			clipped, err := ucscbedclip.AddOptional(stage, coverage.Artifacts, sizes, clipOptions)
			if err != nil {
				return err
			}
			bigWigOptions := config.BedGraphToBigWig
			bigWigOptions.OutDir = gobble.Dir(config.Results.String() + "/coverage/" + sample.Name)
			bigWigOptions.Prefix = strand.name
			if _, err = ucscbedgraphtobigwig.AddOptional(stage, clipped.Artifacts, sizes, bigWigOptions); err != nil {
				return err
			}
			continue
		}
		coverage, err := bedtoolsgenomecov.Add(stage, bam, coverageOptions)
		if err != nil {
			return err
		}
		clipOptions := config.BedClip
		clipOptions.OutDir = gobble.Dir("work/" + sample.Name + "/coverage")
		clipOptions.Prefix = sample.Name + "." + strand.name + ".clipped"
		clipped, err := ucscbedclip.Add(stage, coverage.BedGraph, sizes, clipOptions)
		if err != nil {
			return err
		}
		bigWigOptions := config.BedGraphToBigWig
		bigWigOptions.OutDir = gobble.Dir(config.Results.String() + "/coverage/" + sample.Name)
		bigWigOptions.Prefix = sample.Name + "." + strand.name
		_, err = ucscbedgraphtobigwig.Add(stage, clipped.BedGraph, sizes, bigWigOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

func addTrimmedReadGate(parent *gobble.Module, read1, read2 gobble.Handle, sample string, minimum int64) (gobble.Handle, error) {
	const unit = "sample_retention_trimmed"
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return gobble.Handle{}, err
	}
	out := gobble.PathSpec{Dir: gobble.Dir("work/" + sample + "/policy"), Base: "trimmed_reads", Ext: ".accepted.txt"}
	outPath, _ := out.Render()
	script := fmt.Sprintf(`count=$(gzip -cd %s | awk 'END {if (NR %% 4 != 0) exit 2; print NR / 4}')
awk -v count="$count" -v minimum=%d 'BEGIN {exit !(count >= minimum)}'
printf '%%s\n' "$count" > %s`, modules.ShellQuote(read1Path), minimum, modules.ShellQuote(outPath))
	inputs := []gobble.Bind{{Name: "read1", From: read1}}
	if !read2.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "read2", From: read2})
	}
	image, err := staralign.DefaultImage.TaskReference(unit)
	if err != nil {
		return gobble.Handle{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: script, Image: image, Resources: gobble.Resources{CPU: 1, Memory: "256m"}, Inputs: inputs, Outputs: []gobble.Bind{{Name: "accepted", Spec: out}}})
	return task.Out("accepted"), nil
}

func addMappedReadGate(parent *gobble.Module, starLog, trimmedAccepted gobble.Handle, sample string, minimum float64) (gobble.Handle, error) {
	const unit = "sample_retention_mapped"
	logPath, err := modules.HandlePath(unit, starLog)
	if err != nil {
		return gobble.Handle{}, err
	}
	out := gobble.PathSpec{Dir: gobble.Dir("work/" + sample + "/policy"), Base: "mapped_reads", Ext: ".accepted.txt"}
	outPath, _ := out.Render()
	script := fmt.Sprintf(`mapped=$(awk -F'|' '/Uniquely mapped reads %%/ {v=$2; gsub(/[ %%]/, "", v); print v; exit}' %s)
test -n "$mapped"
awk -v mapped="$mapped" -v minimum=%s 'BEGIN {exit !(mapped >= minimum)}'
printf '%%s\n' "$mapped" > %s`, modules.ShellQuote(logPath), strconv.FormatFloat(minimum, 'g', -1, 64), modules.ShellQuote(outPath))
	image, err := staralign.DefaultImage.TaskReference(unit)
	if err != nil {
		return gobble.Handle{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: gobble.Resources{CPU: 1, Memory: "256m"},
		Inputs: []gobble.Bind{{Name: "star_log", From: starLog}, {Name: "trimmed_accepted", From: trimmedAccepted}}, Outputs: []gobble.Bind{{Name: "accepted", Spec: out}},
	})
	return task.Out("accepted"), nil
}

func recordModuleError(pipeline *gobble.Pipeline, err error) bool {
	if err == nil {
		return false
	}
	pipeline.RecordComposeError(err)
	return true
}

func markOptionsResult(ports picardmarkduplicates.Ports) gobble.Handle { return ports.Metrics }

func salmonLibType(strandedness Strandedness, paired bool) string {
	if strandedness == StrandednessAuto {
		return "A"
	}
	if paired {
		switch strandedness {
		case StrandednessForward:
			return "ISF"
		case StrandednessReverse:
			return "ISR"
		default:
			return "IU"
		}
	}
	switch strandedness {
	case StrandednessForward:
		return "SF"
	case StrandednessReverse:
		return "SR"
	default:
		return "U"
	}
}

func sheetFileSpec(value string) gobble.PathSpec {
	dir, file := "", value
	if split := strings.LastIndex(value, "/"); split >= 0 {
		dir, file = value[:split], value[split+1:]
	}
	base, ext := file, ""
	for _, suffix := range []string{".fastq.gz", ".fq.gz", ".fastq", ".fq"} {
		if strings.HasSuffix(strings.ToLower(file), suffix) {
			base, ext = file[:len(file)-len(suffix)], file[len(file)-len(suffix):]
			break
		}
	}
	spec := gobble.PathSpec{Base: base, Ext: ext}
	if dir != "" {
		spec.Dir = gobble.Dir(dir)
	}
	return spec
}

func validateBuild(samples []Sample, config Config) []gobble.Defect {
	var defects []gobble.Defect
	if len(samples) == 0 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "RNA samples must not be empty"})
	}
	seen := make(map[string]bool, len(samples))
	for _, sample := range samples {
		if !sampleNamePattern.MatchString(sample.Name) || seen[sample.Name] {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "RNA sample name is invalid or duplicated"})
		}
		seen[sample.Name] = true
		if len(sample.Runs) == 0 || !validStrandedness(sample.Strandedness) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "RNA sample requires runs and supported strandedness"})
			continue
		}
		paired := sample.Runs[0].Fastq2 != ""
		runs := make(map[string]bool, len(sample.Runs))
		runIDs := make(map[string]bool, len(sample.Runs))
		for _, run := range sample.Runs {
			key := run.Fastq1 + "\x00" + run.Fastq2
			if !sampleNamePattern.MatchString(run.ID) || runIDs[run.ID] || !validWorkspacePath(run.Fastq1) || run.Fastq2 != "" && !validWorkspacePath(run.Fastq2) || (run.Fastq2 != "") != paired || runs[key] {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "RNA run paths, read mode, or identity are invalid", Paths: []string{run.Fastq1, run.Fastq2}})
			}
			runs[key] = true
			runIDs[run.ID] = true
		}
	}
	for name, spec := range map[string]gobble.PathSpec{"reference.fasta": config.Reference.FASTA, "reference.gtf": config.Reference.GTF} {
		rendered, err := spec.Render()
		if err != nil || !validWorkspacePath(rendered) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: name, Message: "RNA reference path must be workspace-relative", Paths: []string{rendered}})
		}
	}
	if config.Results.IsZero() {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "results", Message: "RNA results directory is required"})
	}
	if config.SampleRemoval.MinTrimmedReads < 0 || config.SampleRemoval.MinMappedPercent < 0 || config.SampleRemoval.MinMappedPercent > 100 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "sample_removal", Message: "RNA sample-removal thresholds must be non-negative and mapped percent must not exceed 100"})
	}
	if config.StrandednessInference.StrandedFraction < 0.5 || config.StrandednessInference.StrandedFraction > 1 || config.StrandednessInference.UnstrandedDifference < 0 || config.StrandednessInference.UnstrandedDifference > 1 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "strandedness_inference", Message: "RNA inference thresholds must satisfy stranded fraction 0.5..1 and unstranded difference 0..1"})
	}
	if !config.Publication.FinalBAMs || !config.Publication.Quantification || !config.Publication.Matrices || !config.Publication.CoverageTracks || !config.Publication.Reports {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "publication", Message: "RNA required final BAM, quantification, matrix, coverage, and report publication cannot be disabled"})
	}
	if !config.Reference.STARIndex.IsZero() && config.Reference.STARIndex.Dir.IsZero() {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.star_index", Message: "ready STAR index directory is required"})
	}
	if !config.Reference.SalmonIndex.IsZero() && config.Reference.SalmonIndex.Dir.IsZero() {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.salmon_index", Message: "ready Salmon index directory is required"})
	}
	return defects
}

func cloneConfig(config Config) Config {
	clone := func(options *modules.Options) { *options = options.Clone() }
	clone(&config.GFFRead.Options)
	clone(&config.Gunzip.Options)
	clone(&config.STARGenome.Options)
	clone(&config.SalmonIndex.Options)
	clone(&config.FAIDX.Options)
	clone(&config.ChromSizes.Options)
	clone(&config.CatFASTQ.Options)
	clone(&config.FQLint.Options)
	clone(&config.FastQC.Options)
	clone(&config.TrimGalore.Options)
	clone(&config.STAR.Options)
	clone(&config.Salmon.Options)
	clone(&config.Sort.Options)
	clone(&config.MarkDuplicates.Options)
	clone(&config.Index.Options)
	clone(&config.Stats.Options)
	clone(&config.StringTie.Options)
	clone(&config.GenomeCov.Options)
	clone(&config.BedClip.Options)
	clone(&config.BedGraphToBigWig.Options)
	clone(&config.RSeQC.Options)
	clone(&config.Qualimap.Options)
	clone(&config.DupRadar.Options)
	clone(&config.BiotypeQC.Options)
	clone(&config.TxImport.Options)
	clone(&config.DESeq2QC.Options)
	clone(&config.MultiQC.Options)
	return config
}
