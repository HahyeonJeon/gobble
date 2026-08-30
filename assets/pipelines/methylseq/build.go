package methylseq

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkdeduplicate "github.com/HahyeonJeon/gobble/assets/modules/bismark-deduplicate"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	bismarkmethylationextractor "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	bismarkreport "github.com/HahyeonJeon/gobble/assets/modules/bismark-report"
	bismarksummary "github.com/HahyeonJeon/gobble/assets/modules/bismark-summary"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
)

// Build constructs the selected directional Bismark graph only from supplied
// values. It copies samples, runs, and every module ExtraArgs slice before use.
func Build(inputSamples []Sample, inputConfig Config) *gobble.Pipeline {
	pipeline := gobble.NewPipeline("methylseq")
	samples := cloneSamples(inputSamples)
	config := cloneConfig(inputConfig)
	if defects := validateBuild(samples, config); len(defects) > 0 {
		pipeline.RecordComposeError(composeError(defects))
		return pipeline
	}

	var index gobble.Handle
	if config.Reference.BismarkIndex.IsZero() {
		fasta := pipeline.AddInput("fasta", config.Reference.FASTA)
		options := config.BismarkGenome
		options.OutDir = gobble.Dir("work/reference/bismark-index")
		if config.Publication.GeneratedIndex {
			options.OutDir = gobble.Dir(config.Results.String() + "/reference/bismark-index")
		}
		prepared, err := bismarkgenome.Add(pipeline.AddModule("reference"), fasta, options)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		index = prepared.Index
	} else {
		index = pipeline.AddInputTree("bismark_index", config.Reference.BismarkIndex)
	}

	var reports []gobble.Handle
	summaryInputs := make([]bismarksummary.SampleReports, 0, len(samples))
	for _, sample := range samples {
		module := pipeline.AddModule(sample.Name)
		read1s := make([]gobble.Handle, 0, len(sample.Runs))
		read2s := make([]gobble.Handle, 0, len(sample.Runs))
		for _, run := range sample.Runs {
			read1 := pipeline.AddInput(sample.Name+"_"+run.ID+"_r1", sheetFileSpec(run.Fastq1))
			read1s = append(read1s, read1)
			raw1Options := config.FastQC
			raw1Options.OutDir = gobble.Dir("work/" + sample.Name + "/raw/fastqc/" + run.ID + "/r1")
			raw1, err := fastqc.Add(module.AddModule(run.ID+"_raw_r1"), read1, raw1Options)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			reports = append(reports, raw1.HTML, raw1.Zip)
			if run.Fastq2 == "" {
				continue
			}
			read2 := pipeline.AddInput(sample.Name+"_"+run.ID+"_r2", sheetFileSpec(run.Fastq2))
			read2s = append(read2s, read2)
			raw2Options := config.FastQC
			raw2Options.OutDir = gobble.Dir("work/" + sample.Name + "/raw/fastqc/" + run.ID + "/r2")
			raw2, err := fastqc.Add(module.AddModule(run.ID+"_raw_r2"), read2, raw2Options)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			reports = append(reports, raw2.HTML, raw2.Zip)
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
			consolidated, err := catfastq.Add(module.AddModule("consolidate_r1"), read1s, catOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			read1 = consolidated.FASTQ
			if len(read2s) > 0 {
				catOptions.Prefix = sample.Name + "_2"
				consolidated, err = catfastq.Add(module.AddModule("consolidate_r2"), read2s, catOptions)
				if recordModuleError(pipeline, err) {
					return pipeline
				}
				read2 = consolidated.FASTQ
			}
		}

		trimOptions := config.TrimGalore
		if read2.IsZero() {
			trimOptions.ClipR2 = 0
			trimOptions.ThreePrimeClipR2 = 0
			trimOptions.Adapter2 = ""
		}
		trimOptions.OutDir = gobble.Dir("work/" + sample.Name + "/trim-galore")
		if config.Publication.TrimmedReads {
			trimOptions.OutDir = gobble.Dir(config.Results.String() + "/intermediates/trimmed/" + sample.Name)
		}
		trimOptions.Prefix = sample.Name
		trimmed, err := trimgalore.Add(module, read1, read2, trimOptions)
		if recordModuleError(pipeline, err) {
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
			qcOptions := config.FastQC
			qcOptions.OutDir = gobble.Dir("work/" + sample.Name + "/post-trim/fastqc/" + clean.name)
			qc, qcErr := fastqc.Add(module.AddModule("post_trim_"+clean.name), clean.handle, qcOptions)
			if recordModuleError(pipeline, qcErr) {
				return pipeline
			}
			reports = append(reports, qc.HTML, qc.Zip)
		}

		paired := !trimmed.Read2.IsZero()
		alignOptions := config.BismarkAlign
		alignOptions.OutDir = gobble.Dir("work/" + sample.Name + "/bismark-align")
		alignOptions.Prefix = sample.Name
		aligned, err := bismarkalign.Add(module, index, trimmed.Read1, trimmed.Read2, alignOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		prefix := sample.Name
		if paired {
			prefix += "_pe"
		}
		dedupOptions := config.Deduplicate
		dedupOptions.OutDir = gobble.Dir(config.Results.String() + "/bismark/" + sample.Name)
		dedupOptions.Prefix = prefix
		deduplicated, err := bismarkdeduplicate.Add(module, aligned.BAM, paired, dedupOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		extractorOptions := config.Extractor
		if !paired {
			extractorOptions.IgnoreR2 = 0
			extractorOptions.Ignore3PrimeR2 = 0
		}
		extractorOptions.OutDir = gobble.Dir(config.Results.String() + "/methylation-calls/" + sample.Name)
		extracted, err := bismarkmethylationextractor.Add(module, deduplicated.BAM, paired, extractorOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		reportOptions := config.Report
		reportOptions.OutDir = gobble.Dir(config.Results.String() + "/reports/" + sample.Name)
		reportOptions.Prefix = sample.Name
		sampleReport, err := bismarkreport.Add(module, aligned.Report, deduplicated.Report, extracted.Report, extracted.MBias, reportOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		reports = append(reports, aligned.Report, deduplicated.Report, extracted.Report, extracted.MBias, sampleReport.HTML)
		summaryInputs = append(summaryInputs, bismarksummary.SampleReports{BAM: aligned.BAM, AlignmentReport: aligned.Report, DeduplicationReport: deduplicated.Report, SplittingReport: extracted.Report, MBiasReport: extracted.MBias})
	}

	summaryOptions := config.Summary
	summaryOptions.OutDir = gobble.Dir(config.Results.String() + "/summary")
	summary, err := bismarksummary.Add(pipeline, summaryInputs, summaryOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	reports = append(reports, summary.HTML, summary.Text)
	multiQCOptions := config.MultiQC
	multiQCOptions.OutDir = gobble.Dir(config.Results.String() + "/multiqc")
	if _, err = multiqc.Add(pipeline, reports, multiQCOptions); recordModuleError(pipeline, err) {
		return pipeline
	}
	return pipeline
}

func recordModuleError(pipeline *gobble.Pipeline, err error) bool {
	if err == nil {
		return false
	}
	pipeline.RecordComposeError(err)
	return true
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
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "Methyl samples must not be empty"})
	}
	seen := make(map[string]bool, len(samples))
	for _, sample := range samples {
		if !sampleNamePattern.MatchString(sample.Name) || seen[sample.Name] {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "Methyl sample name is invalid or duplicated"})
		}
		seen[sample.Name] = true
		if len(sample.Runs) == 0 {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "Methyl sample requires at least one run"})
			continue
		}
		paired := sample.Runs[0].Fastq2 != ""
		runIDs, runPaths := map[string]bool{}, map[string]bool{}
		for _, run := range sample.Runs {
			key := run.Fastq1 + "\x00" + run.Fastq2
			if !sampleNamePattern.MatchString(run.ID) || runIDs[run.ID] || !validWorkspacePath(run.Fastq1) || run.Fastq2 != "" && !validWorkspacePath(run.Fastq2) || (run.Fastq2 != "") != paired || runPaths[key] {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "Methyl run paths, read mode, or identity are invalid", Paths: []string{run.Fastq1, run.Fastq2}})
			}
			runIDs[run.ID], runPaths[key] = true, true
		}
	}
	if config.Reference.BismarkIndex.IsZero() {
		rendered, err := config.Reference.FASTA.Render()
		if err != nil || !validWorkspacePath(rendered) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.fasta", Message: "Methyl reference FASTA must be workspace-relative", Paths: []string{rendered}})
		}
	} else if config.Reference.BismarkIndex.Dir.IsZero() {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.bismark_index", Message: "ready Bismark index Tree directory is required"})
	}
	if config.Results.IsZero() || !validWorkspacePath(config.Results.String()+"/result") {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "results", Message: "Methyl results directory must be workspace-relative"})
	}
	if config.LibraryMode != LibraryModeDirectional {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "library_mode", Message: "Methyl library mode must be directional"})
	}
	if !config.Publication.DeduplicatedBAMs || !config.Publication.MethylationCalls || !config.Publication.Reports {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "publication", Message: "Methyl deduplicated BAM, methylation-call, and report publication cannot be disabled"})
	}
	if route, flag := unsupportedRouteExtra(config); flag != "" {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: route, Message: "Methyl ExtraArgs contains unsupported route option " + flag})
	}
	return defects
}

func unsupportedRouteExtra(config Config) (string, string) {
	sets := []struct {
		unit  string
		args  []string
		flags []string
	}{
		{unit: "trim_galore", args: config.TrimGalore.ExtraArgs, flags: []string{"--rrbs", "--fastqc", "--dont_gzip", "--retain_unpaired"}},
		{unit: "bismark_genome_preparation", args: config.BismarkGenome.ExtraArgs, flags: []string{"--hisat2", "--slam"}},
		{unit: "bismark_align", args: config.BismarkAlign.ExtraArgs, flags: []string{"--hisat2", "--minimap2", "--mm2", "--non_directional", "--pbat", "--se", "--single_end", "-f", "--fasta", "-un", "--unmapped", "--ambiguous", "--ambig_bam", "--sam", "--cram", "--nucleotide_coverage", "--genome", "--genome_folder", "-o", "--output_dir", "-B", "--basename", "--prefix"}},
		{unit: "bismark_deduplicate", args: config.Deduplicate.ExtraArgs, flags: []string{"--sam", "--multiple"}},
		{unit: "bismark_methylation_extractor", args: config.Extractor.ExtraArgs, flags: []string{"--CX", "--CX_context", "--cytosine_report", "--yacht", "--merge_non_CpG", "--zero_based", "--ucsc", "--mbias_only", "--mbias_off", "--sam"}},
	}
	for _, set := range sets {
		for _, arg := range set.args {
			for _, flag := range set.flags {
				if arg == flag || strings.HasPrefix(arg, flag+"=") {
					return set.unit, flag
				}
			}
		}
	}
	return "", ""
}

func cloneConfig(config Config) Config {
	clone := func(options *modules.Options) { *options = options.Clone() }
	clone(&config.CatFASTQ.Options)
	clone(&config.FastQC.Options)
	clone(&config.TrimGalore.Options)
	clone(&config.BismarkGenome.Options)
	clone(&config.BismarkAlign.Options)
	clone(&config.Deduplicate.Options)
	clone(&config.Extractor.Options)
	clone(&config.Report.Options)
	clone(&config.Summary.Options)
	clone(&config.MultiQC.Options)
	return config
}
