package scrnaseq

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	simpleafindex "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-index"
	simpleafquant "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-quant"
)

func validateBuild(samples []Sample, config Config) []gobble.Defect {
	var defects []gobble.Defect
	if len(samples) == 0 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "scRNA requires at least one sample"})
	}
	seenSamples := make(map[string]bool, len(samples))
	seenReadPaths := make(map[string]bool)
	for _, sample := range samples {
		if !identityPattern.MatchString(sample.Name) || seenSamples[sample.Name] || len(sample.Runs) == 0 || sample.ExpectedCells < 0 {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "scRNA sample identity is invalid, duplicated, has invalid metadata, or has no runs"})
		}
		seenSamples[sample.Name] = true
		seenRuns := make(map[string]bool, len(sample.Runs))
		seenPairs := make(map[string]bool, len(sample.Runs))
		for i, run := range sample.Runs {
			fastq1Path, fastq1Valid := renderWorkspacePath(run.Fastq1)
			fastq2Path, fastq2Valid := renderWorkspacePath(run.Fastq2)
			pair := fastq1Path + "\x00" + fastq2Path
			pathAlias := (fastq1Valid && fastq2Valid && fastq1Path == fastq2Path) ||
				(fastq1Valid && seenReadPaths[fastq1Path]) ||
				(fastq2Valid && seenReadPaths[fastq2Path])
			if !identityPattern.MatchString(run.ID) || run.ID != "run_"+leftPad3(i+1) || seenRuns[run.ID] || seenPairs[pair] || !fastq1Valid || !fastq2Valid || pathAlias {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "scRNA technical-run identity, order, mates, path, or path ownership is invalid", Paths: []string{run.Fastq1, run.Fastq2}})
			}
			seenRuns[run.ID] = true
			seenPairs[pair] = true
			if fastq1Valid {
				seenReadPaths[fastq1Path] = true
			}
			if fastq2Valid {
				seenReadPaths[fastq2Path] = true
			}
		}
	}

	if _, ok := simpleafChemistry(config.Protocol); !ok {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "protocol", Message: "scRNA protocol must be typed 10x V1, V2, V3, or V4"})
	}
	if config.Reference.BarcodeWhitelist.Protocol != config.Protocol {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.whitelist", Message: "barcode whitelist protocol must equal the explicit product protocol"})
	}
	if !validPathSpec(config.Reference.BarcodeWhitelist.Path) {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.whitelist", Message: "barcode whitelist path must be workspace-relative"})
	}
	ready := !config.Reference.SimpleafIndex.IsZero()
	if ready {
		if config.Reference.SimpleafIndex.Dir.IsZero() || !validWorkspacePath(config.Reference.SimpleafIndex.Dir.String()+"/member") {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.simpleaf_index", Message: "ready Simpleaf index Tree root must be workspace-relative"})
		}
		if !validPathSpec(config.Reference.TranscriptToGene) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.transcript_to_gene", Message: "ready Simpleaf index requires an explicit transcript-to-gene file"})
		}
		if !pathSpecUnset(config.Reference.FASTA) || !pathSpecUnset(config.Reference.Annotation) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference", Message: "ready Simpleaf index and source FASTA/GTF forms cannot be mixed"})
		}
	} else {
		if !validPathSpec(config.Reference.FASTA) || !validPathSpec(config.Reference.Annotation) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference", Message: "source reference requires workspace-relative FASTA and annotation paths"})
		}
		if !pathSpecUnset(config.Reference.TranscriptToGene) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.transcript_to_gene", Message: "source reference produces its transcript-to-gene relation and cannot mix a ready relation"})
		}
	}
	defects = append(defects, crossRolePathDefects(samples, config)...)
	if config.Results.IsZero() || !validWorkspacePath(config.Results.String()+"/result") {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "results", Message: "scRNA results directory must be workspace-relative"})
	}
	if !validResolution(config.UMIResolution) {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "umi_resolution", Message: "unsupported Simpleaf UMI resolution"})
	}
	publication := config.Publication
	if !publication.Index || !publication.Quantification || !publication.QCatch || !publication.RawH5AD || !publication.RawRDS || !publication.CombinedH5AD || !publication.MultiQC {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "publication", Message: "scRNA required index, quantification, QCatch, raw matrix, cohort, and report outputs cannot be disabled"})
	}

	qcatchChemistry, hasQCatchChemistry := qcatchChemistry(config.Protocol)
	if hasQCatchChemistry {
		if config.QCatch.Chemistry != qcatchChemistry || config.QCatch.NPartitions != 0 {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "qcatch", Message: "QCatch chemistry must match the explicit V2-V4 protocol and cannot be combined with partition override"})
		}
	} else if config.Protocol == Protocol10xV1 && (config.QCatch.Chemistry != "" || config.QCatch.NPartitions < 1) {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "qcatch", Message: "10x V1 has no QCatch chemistry mapping and requires an explicit positive partition count"})
	}
	if config.QCatch.VisualizeDoublets && !config.QCatch.RemoveDoublets {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "qcatch", Message: "QCatch doublet visualization requires typed doublet removal"})
	}
	if unit, flag := protectedExtra(config); flag != "" {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: unit, Message: "scRNA ExtraArgs contains protected option " + flag})
	}
	return defects
}

func protectedExtra(config Config) (string, string) {
	if len(config.Consolidate.ExtraArgs) != 0 {
		return "cat_fastq", "typed-input"
	}
	sets := []struct {
		unit  string
		args  []string
		flags []string
	}{
		{unit: "fastqc", args: config.FastQC.ExtraArgs, flags: []string{"--outdir", "--threads", "--extract"}},
		{unit: "gffread_transcriptome", args: config.Transcriptome.ExtraArgs, flags: []string{"-F", "-w", "-g", "-o"}},
	}
	for _, set := range sets {
		if flag := modules.MatchProtectedExtraArg(set.args, set.flags); flag != "" {
			return set.unit, flag
		}
	}
	if flag := simpleafindex.ProtectedExtraArg(config.SimpleafIndex.ExtraArgs); flag != "" {
		return "simpleaf_index", flag
	}
	if flag := simpleafquant.ProtectedExtraArg(config.SimpleafQuant.ExtraArgs); flag != "" {
		return "simpleaf_quant", flag
	}
	for _, set := range []struct {
		unit  string
		args  []string
		flags []string
	}{
		{unit: "qcatch", args: config.QCatch.ExtraArgs, flags: []string{"--input", "-i", "--output", "-o", "--chemistry", "-c", "--n_partitions", "-n", "--save_filtered_h5ad", "-s", "--export_summary_table", "-x", "--remove_doublets", "-d", "--visualize_doublets", "-vd", "--skip_umap_tsne", "-u", "--gene_id2name_file", "-g", "--valid_cell_list", "-l"}},
		{unit: "multiqc", args: config.MultiQC.ExtraArgs, flags: []string{"--outdir", "--filename", "--no-data-dir", "--zip-data-dir"}},
	} {
		if flag := modules.MatchProtectedExtraArg(set.args, set.flags); flag != "" {
			return set.unit, flag
		}
	}
	for _, exact := range []struct {
		unit string
		args []string
	}{
		{unit: "gtf_gene_filter", args: config.GTFFilter.ExtraArgs},
		{unit: "gtf_to_t2g", args: config.TranscriptToGene.ExtraArgs},
		{unit: "matrix_to_h5ad", args: config.MatrixToH5AD.ExtraArgs},
		{unit: "anndatar_convert", args: config.AnnDataR.ExtraArgs},
		{unit: "h5ad_concat", args: config.H5ADConcat.ExtraArgs},
	} {
		if len(exact.args) != 0 {
			return exact.unit, "typed-input"
		}
	}
	return "", ""
}

func simpleafChemistry(protocol Protocol) (string, bool) {
	switch protocol {
	case Protocol10xV1:
		return "10xv1", true
	case Protocol10xV2:
		return "10xv2", true
	case Protocol10xV3:
		return "10xv3", true
	case Protocol10xV4:
		return "10xv4-3p", true
	default:
		return "", false
	}
}

func qcatchChemistry(protocol Protocol) (string, bool) {
	switch protocol {
	case Protocol10xV2:
		return "10X_3p_v2", true
	case Protocol10xV3:
		return "10X_3p_v3", true
	case Protocol10xV4:
		return "10X_3p_v4", true
	default:
		return "", false
	}
}

func validResolution(resolution UMIResolution) bool {
	switch resolution {
	case ResolutionCRLike, ResolutionCRLikeEM, ResolutionParsimony,
		ResolutionParsimonyEM, ResolutionParsimonyGene, ResolutionParsimonyGeneEM:
		return true
	default:
		return false
	}
}

func validPathSpec(spec gobble.PathSpec) bool {
	rendered, err := spec.Render()
	return err == nil && validWorkspacePath(rendered)
}

func crossRolePathDefects(samples []Sample, config Config) []gobble.Defect {
	owners := make(map[string]string)
	var defects []gobble.Defect
	claim := func(unit, role, rendered string) {
		if owner, exists := owners[rendered]; exists && owner != role {
			defects = append(defects, gobble.Defect{
				Code:    gobble.DefectInvalidValue,
				Unit:    unit,
				Message: "scRNA " + role + " path aliases the " + owner + " input role",
				Paths:   []string{rendered},
			})
			return
		}
		owners[rendered] = role
	}
	for _, sample := range samples {
		for _, run := range sample.Runs {
			for _, read := range []string{run.Fastq1, run.Fastq2} {
				if rendered, valid := renderWorkspacePath(read); valid {
					owners[rendered] = "read"
				}
			}
		}
	}

	references := []struct {
		unit string
		role string
		spec gobble.PathSpec
	}{
		{unit: "reference.fasta", role: "FASTA", spec: config.Reference.FASTA},
		{unit: "reference.annotation", role: "GTF", spec: config.Reference.Annotation},
		{unit: "reference.whitelist", role: "whitelist", spec: config.Reference.BarcodeWhitelist.Path},
		{unit: "reference.transcript_to_gene", role: "transcript-to-gene", spec: config.Reference.TranscriptToGene},
	}
	for _, reference := range references {
		if pathSpecUnset(reference.spec) {
			continue
		}
		rendered, err := reference.spec.Render()
		if err != nil || !validWorkspacePath(rendered) {
			continue
		}
		claim(reference.unit, reference.role, rendered)
	}
	if rendered, valid := renderTreeRoot(config.Reference.SimpleafIndex); valid {
		claim("reference.simpleaf_index", "Simpleaf index Tree", rendered)
	}
	return defects
}

func renderTreeRoot(tree gobble.Tree) (string, bool) {
	if tree.IsZero() || tree.Dir.IsZero() {
		return "", false
	}
	// Render a synthetic member so the root uses PathSpec's canonical path rules.
	member, err := (gobble.PathSpec{Dir: tree.Dir, Base: "member"}).Render()
	if err != nil {
		return "", false
	}
	root, ok := strings.CutSuffix(member, "/member")
	return root, ok && validWorkspacePath(root)
}

func pathSpecUnset(spec gobble.PathSpec) bool {
	return spec.Dir.IsZero() && spec.Prefix == "" && spec.Base == "" && len(spec.Suffixes) == 0 && spec.Ext == ""
}

func cloneConfig(config Config) Config {
	clone := func(options *modules.Options) { *options = options.Clone() }
	clone(&config.Consolidate.Options)
	clone(&config.FastQC.Options)
	clone(&config.GTFFilter.Options)
	clone(&config.Transcriptome.Options)
	clone(&config.TranscriptToGene.Options)
	clone(&config.SimpleafIndex.Options)
	clone(&config.SimpleafQuant.Options)
	clone(&config.QCatch.Options)
	clone(&config.MatrixToH5AD.Options)
	clone(&config.AnnDataR.Options)
	clone(&config.H5ADConcat.Options)
	clone(&config.MultiQC.Options)
	config.Reference.FASTA = cloneSpec(config.Reference.FASTA)
	config.Reference.Annotation = cloneSpec(config.Reference.Annotation)
	config.Reference.TranscriptToGene = cloneSpec(config.Reference.TranscriptToGene)
	config.Reference.BarcodeWhitelist.Path = cloneSpec(config.Reference.BarcodeWhitelist.Path)
	config.H5ADConcat.Labels = append([]string(nil), config.H5ADConcat.Labels...)
	return config
}

func cloneSpec(spec gobble.PathSpec) gobble.PathSpec {
	spec.Suffixes = append([]string(nil), spec.Suffixes...)
	return spec
}

func sampleParamValue(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}
