package scrnaseq

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	anndatarconvert "github.com/HahyeonJeon/gobble/assets/modules/anndatar-convert"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	gffreadtranscriptome "github.com/HahyeonJeon/gobble/assets/modules/gffread-transcriptome"
	gtfgenefilter "github.com/HahyeonJeon/gobble/assets/modules/gtf-gene-filter"
	gtftot2g "github.com/HahyeonJeon/gobble/assets/modules/gtf-to-t2g"
	h5adconcat "github.com/HahyeonJeon/gobble/assets/modules/h5ad-concat"
	matrixtoh5ad "github.com/HahyeonJeon/gobble/assets/modules/matrix-to-h5ad"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	"github.com/HahyeonJeon/gobble/assets/modules/qcatch"
	simpleafindex "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-index"
	simpleafquant "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-quant"
)

type referenceHandles struct {
	index     gobble.Handle
	t2g       gobble.Handle
	whitelist gobble.Handle
}

// Build constructs the selected Simpleaf scRNA-seq graph only from supplied
// values. It copies every caller-owned run, path suffix, label, and ExtraArgs
// slice. Build performs no filesystem or network access.
func Build(inputSamples []Sample, inputConfig Config) *gobble.Pipeline {
	pipeline := gobble.NewPipeline("scrnaseq")
	samples := cloneSamples(inputSamples)
	config := cloneConfig(inputConfig)
	if defects := validateBuild(samples, config); len(defects) > 0 {
		pipeline.RecordComposeError(composeError(defects))
		return pipeline
	}

	reference, err := addReference(pipeline, config)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	chemistry, _ := simpleafChemistry(config.Protocol)
	reports := make([]gobble.Handle, 0, len(samples)*8)
	rawH5ADs := make([]gobble.Handle, 0, len(samples))
	labels := make([]string, 0, len(samples))
	for _, sample := range samples {
		sampleModule := pipeline.AddModule(sample.Name).WithDisplay(gobble.TaskDisplay{Samples: []string{sample.Name}, Scope: gobble.DisplaySample})
		read1s := make([]gobble.Handle, 0, len(sample.Runs))
		read2s := make([]gobble.Handle, 0, len(sample.Runs))
		for _, run := range sample.Runs {
			runModule := sampleModule.AddModule(run.ID)
			read1 := pipeline.AddInput(inputName(sample.Name, run.ID, "r1"), sheetFileSpec(run.Fastq1))
			read2 := pipeline.AddInput(inputName(sample.Name, run.ID, "r2"), sheetFileSpec(run.Fastq2))
			read1s = append(read1s, read1)
			read2s = append(read2s, read2)
			for _, read := range []struct {
				name   string
				handle gobble.Handle
			}{{name: "r1", handle: read1}, {name: "r2", handle: read2}} {
				options := config.FastQC
				options.OutDir = config.Results.Join("samples", sample.Name, "raw_qc", run.ID, read.name)
				ports, addErr := fastqc.Add(sampleTaskParent(runModule.AddModule("raw_fastqc_"+read.name), sample, run.ID), read.handle, options)
				if recordModuleError(pipeline, addErr) {
					return pipeline
				}
				reports = append(reports, ports.HTML, ports.Zip)
			}
		}

		read1, read2 := read1s[0], read2s[0]
		if len(sample.Runs) > 1 {
			options := config.Consolidate
			options.OutDir = gobble.Dir("work/scrnaseq").Join(sample.Name, "consolidated")
			options.Prefix = sample.Name + "_R1"
			consolidated, addErr := catfastq.Add(sampleTaskParent(sampleModule.AddModule("consolidate_r1"), sample, ""), read1s, options)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			read1 = consolidated.FASTQ
			options.Prefix = sample.Name + "_R2"
			consolidated, addErr = catfastq.Add(sampleTaskParent(sampleModule.AddModule("consolidate_r2"), sample, ""), read2s, options)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			read2 = consolidated.FASTQ
		}

		quantOptions := config.SimpleafQuant
		quantOptions.Chemistry = chemistry
		quantOptions.Resolution = string(config.UMIResolution)
		quantOptions.OutDir = config.Results.Join("samples", sample.Name, "simpleaf")
		quant, addErr := simpleafquant.Add(sampleTaskParent(sampleModule, sample, ""), reference.index, reference.t2g, reference.whitelist, read1, read2, quantOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, quant.Quant)

		qcatchOptions := config.QCatch
		qcatchOptions.OutDir = config.Results.Join("samples", sample.Name, "qcatch")
		qc, addErr := qcatch.Add(sampleTaskParent(sampleModule, sample, ""), quant.Quant, qcatchOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, qc.Report, qc.Metrics)

		matrixOptions := config.MatrixToH5AD
		matrixOptions.Sample = sample.Name
		matrixOptions.ExpectedCells = sample.ExpectedCells
		matrixOptions.SeqCenter = sample.SeqCenter
		matrixOptions.OutDir = config.Results.Join("matrices", sample.Name)
		matrixOptions.Prefix = sample.Name + "_raw_matrix"
		raw, addErr := matrixtoh5ad.Add(sampleTaskParent(sampleModule, sample, ""), quant.Quant, matrixOptions)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		rawH5ADs = append(rawH5ADs, raw.H5AD)
		labels = append(labels, sample.Name)

		rdsOptions := config.AnnDataR
		rdsOptions.OutDir = config.Results.Join("matrices", sample.Name)
		rdsOptions.Prefix = sample.Name + "_raw_matrix"
		if _, addErr = anndatarconvert.Add(sampleTaskParent(sampleModule, sample, ""), raw.H5AD, rdsOptions); recordModuleError(pipeline, addErr) {
			return pipeline
		}
	}

	concatOptions := config.H5ADConcat
	concatOptions.Labels = append([]string(nil), labels...)
	concatOptions.OutDir = config.Results.Join("matrices")
	concatOptions.Prefix = "combined_raw_matrix"
	if _, err = h5adconcat.Add(pipeline.AddModule("cohort"), rawH5ADs, concatOptions); recordModuleError(pipeline, err) {
		return pipeline
	}
	multiQCOptions := config.MultiQC
	multiQCOptions.OutDir = config.Results.Join("multiqc")
	if _, err = multiqc.Add(modules.WithDisplay(pipeline, gobble.TaskDisplay{Scope: gobble.DisplayCohort}), reports, multiQCOptions); recordModuleError(pipeline, err) {
		return pipeline
	}
	return pipeline
}

func addReference(pipeline *gobble.Pipeline, config Config) (referenceHandles, error) {
	module := pipeline.AddModule("reference").WithDisplay(gobble.TaskDisplay{Scope: gobble.DisplayShared})
	whitelist := pipeline.AddInput("barcode_whitelist", config.Reference.BarcodeWhitelist.Path)
	if !config.Reference.SimpleafIndex.IsZero() {
		return referenceHandles{
			index:     pipeline.AddInputTree("simpleaf_index", config.Reference.SimpleafIndex),
			t2g:       pipeline.AddInput("transcript_to_gene", config.Reference.TranscriptToGene),
			whitelist: whitelist,
		}, nil
	}
	fasta := pipeline.AddInput("reference_fasta", config.Reference.FASTA)
	annotation := pipeline.AddInput("reference_annotation", config.Reference.Annotation)
	referenceDir := config.Results.Join("reference", "normalized")
	filterOptions := config.GTFFilter
	filterOptions.OutDir, filterOptions.Prefix = referenceDir, "genes.filtered"
	filtered, err := gtfgenefilter.Add(module, fasta, annotation, filterOptions)
	if err != nil {
		return referenceHandles{}, err
	}
	transcriptOptions := config.Transcriptome
	transcriptOptions.OutDir, transcriptOptions.Prefix = referenceDir, "transcripts"
	transcripts, err := gffreadtranscriptome.Add(module, filtered.GTF, fasta, transcriptOptions)
	if err != nil {
		return referenceHandles{}, err
	}
	relationOptions := config.TranscriptToGene
	relationOptions.OutDir, relationOptions.Prefix = referenceDir, "transcript_to_gene"
	relation, err := gtftot2g.Add(module, filtered.GTF, relationOptions)
	if err != nil {
		return referenceHandles{}, err
	}
	indexOptions := config.SimpleafIndex
	indexOptions.OutDir = config.Results.Join("reference", "simpleaf_index")
	index, err := simpleafindex.Add(protocolTaskParent(module, config.Protocol), transcripts.FASTA, indexOptions)
	if err != nil {
		return referenceHandles{}, err
	}
	return referenceHandles{index: index.Index, t2g: relation.T2G, whitelist: whitelist}, nil
}

type parameterizedParent struct {
	parent modules.Parent
	params []gobble.Param
}

func (p parameterizedParent) AddTask(spec gobble.TaskSpec) *gobble.Task {
	spec.Params = append(append([]gobble.Param(nil), spec.Params...), p.params...)
	return p.parent.AddTask(spec)
}

func protocolTaskParent(parent modules.Parent, protocol Protocol) modules.Parent {
	return parameterizedParent{parent: parent, params: []gobble.Param{{Name: "protocol", Value: string(protocol)}}}
}

func sampleTaskParent(parent modules.Parent, sample Sample, run string) modules.Parent {
	params := []gobble.Param{
		{Name: "sample", Value: sample.Name},
		{Name: "expected_cells", Value: sampleParamValue(sample.ExpectedCells)},
		{Name: "seq_center", Value: sample.SeqCenter},
	}
	if run != "" {
		params = append(params, gobble.Param{Name: "technical_run", Value: run})
	}
	return parameterizedParent{parent: parent, params: params}
}

func recordModuleError(pipeline *gobble.Pipeline, err error) bool {
	if err == nil {
		return false
	}
	pipeline.RecordComposeError(err)
	return true
}

func inputName(sample, run, mate string) string {
	return "s" + strconv.Itoa(len(sample)) + "_" + sample + "_r" + strconv.Itoa(len(run)) + "_" + run + "_" + mate
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
