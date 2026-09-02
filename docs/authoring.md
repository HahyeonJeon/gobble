# Authoring

This guide defines the accepted data and typed customization surface for the
five products. It is not a scientific protocol. Start with
[Products](products.md), then use [Operations](operations.md) to execute the
result.

## Entry roles

Keep three value planes separate:

| Plane | Owns | Must not own |
|---|---|---|
| Experiment data | Sample, patient, lane/run, replicate, control link, read files, strandedness, protocol-related metadata | Tool flags, images, resources, workspace, cap, or skip lists |
| Analysis config | Reference bundle, selected-route policy, typed command options, outputs, task resources, immutable images, `ExtraArgs []string` | Workspace occupancy, controller cancellation, or backend profiles |
| Engine controls | Workspace, concurrency cap, execution identity, Inspect view, Release, and Resume | Biological metadata or tool options |

For the default CLI path, pass a strict assay CSV to the product package's
`Pipeline()` adapter with `--sample`. For reusable Go construction, call the
assay's `Load` or `Parse`, copy and change a fresh `DefaultConfig`, then pass
both values to `Build`. `Build` performs no ambient samplesheet, filesystem,
working-directory, environment, or network lookup.

A custom CLI or packed runner exposes its own process-exclusive adapter. This
example changes one named RNA policy while preserving `--sample` injection and
structured load failure:

```go
package myrnaseq

import (
	"github.com/HahyeonJeon/gobble"
	product "github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

func Pipeline() *gobble.Pipeline {
	samples, err := product.Load(gobble.SampleSheetPath())
	if err != nil {
		pipeline := gobble.NewPipeline("my-rnaseq")
		pipeline.RecordComposeError(err)
		return pipeline
	}
	config := product.DefaultConfig()
	config.SampleRemoval.MinMappedPercent = 10
	return product.Build(samples, config)
}
```

Reusable concurrent callers bypass `SampleSheetPath` and pass an explicit path
to `Load`. Always inspect Plan JSON after customization.

Each command's nested options expose named fields, task resources, a full
immutable image reference, and argv `ExtraArgs`. Extras are literal argv tokens;
they are not split or evaluated as shell text. A named field and extras cannot
own the same flag. Route-changing, input-rebinding, output-rebinding, or
filesystem-discovery flags are rejected where they would violate the product.

An explicit image replacement changes task identity and provenance. It is an
expert software substitution and leaves the supported default tuple. There is
no YAML/JSON parameter overlay, string-keyed extras map, image profile,
arbitrary module list, aligner selector, caller selector, or generic skip list.

## Samplesheets

Headers are exact. Unknown and duplicate columns are errors. FASTQ cells are
workspace-relative local paths, not URLs or host-absolute paths. Repeated rows
become typed lane or run members in row order; they are not duplicate samples.

| Product | Required columns | Optional columns | Typed semantics and invariants |
|---|---|---|---|
| WGS | `patient,sample,lane,fastq_1,fastq_2` | `sex` | `Sample{Patient,Name,Sex,Lanes}`; patient/sample is the sample key; both mates required; lane IDs unique per sample; repeated sex agrees; at least two distinct samples |
| RNA-seq | `sample,fastq_1,fastq_2,strandedness` | `seq_platform,seq_center` | `Sample{Name,Runs,Strandedness,SeqPlatform,SeqCenter}`; `fastq_2` may be empty; one sample cannot mix read modes; repeated metadata agrees; strandedness is `unstranded`, `forward`, `reverse`, or `auto`; at least two samples |
| Methyl-seq | `sample,fastq_1,fastq_2` | none | `Sample{Name,Runs}`; `fastq_2` may be empty; one sample cannot mix read modes; duplicate read pairs are rejected; one or more samples |
| ATAC-seq | `sample,fastq_1,fastq_2,replicate` | `control,control_replicate` | `Sample{Name,Replicates}` with ordered technical `Run` values; one replicate cannot mix read modes; replicates start at 1 without gaps; at least two total replicate members; both control cells appear together and resolve to an existing replicate |
| scRNA-seq | `sample,fastq_1,fastq_2` | `expected_cells,seq_center` | `Sample{Name,Runs,ExpectedCells,SeqCenter}`; both distinct mates required; read paths cannot be reused; repeated metadata agrees; expected cells is empty or positive; one or more samples |

Sheets never select references, images, resources, flags, contrasts, or engine
controls. They do not infer patient, group, protocol, control, or experimental
meaning from names. Examples used by product evidence are linked from each
assay's [manifest owner](provenance.md#authority); they are small-data fixtures,
not templates for scientific design.

Exact localized fixture sheets are available for
[WGS](../tests/pipelines/wgs/testdata/wgs-samplesheet.csv),
[RNA-seq](../tests/pipelines/rnaseq/testdata/rnaseq-samplesheet.csv),
[Methyl-seq](../tests/pipelines/methylseq/testdata/methylseq-samplesheet.csv),
[ATAC-seq](../tests/pipelines/atacseq/testdata/atacseq-samplesheet.csv), and
[scRNA-seq](../tests/pipelines/scrnaseq/testdata/scrnaseq-samplesheet.csv).
Use them to inspect row shape and workspace-relative staging only. Their sample
selection and metadata are not recommendations for a study.

## WGS

**Package:** [`assets/pipelines/wgs`](../assets/pipelines/wgs)

**Default:** `wgs-joint-germline-v1`, benchmarked against nf-core/sarek 3.10.0.

`DefaultConfig` names `in/reference/genome.fasta`, its `.fai` and `.dict`,
indexed dbSNP and Mills known sites, and the non-empty interval members
`in/reference/intervals/interval_001.bed` and `interval_002.bed`. It generates
the BWA index unless a complete ready prefix and sidecar Group is supplied. The
only final alignment format is BAM.

Supported typed changes cover the reference and intervals; FastP; BWA-MEM;
sorting, merge, duplicate marking, BQSR, and alignment QC; HaplotypeCaller,
GenomicsDB, joint genotyping, VCF sorting and gathering; MultiQC; task resources;
images; argv extras; result root; and optional publication of prepared reads or
a generated BWA index. Recalibrated alignments, sample gVCFs, joint callset, and
reports cannot be disabled.

**Stages:** raw FastQC and FastP per lane; BWA-MEM; lane sort/index and sample
merge; GATK duplicate marking; interval BaseRecalibrator, gathered BQSR, and
ApplyBQSR; BAM gather/index and alignment QC; interval HaplotypeCaller and
sample gVCF gather; complete-cohort GenomicsDB; interval GenotypeGVCFs and sort;
joint VCF gather; bcftools statistics; MultiQC. The declared interval Group is
the runtime Scatter/Gather membership. Missing membership fails closed.

**Required outputs:** indexed recalibrated BAMs and indexed gVCFs under
`results/wgs/samples/<patient>/<sample>/`; indexed unfiltered
`results/wgs/joint/joint_germline.vcf.gz`; command and callset metrics; MultiQC
HTML and data.

**Limits:** The joint callset is unfiltered. Gobble does not choose known sites
or claim variant quality, suitable filtering, annotation, study validity, or
clinical validity. Somatic and tumor-normal calling, WES, VQSR, structural
variants, copy-number analysis, alternate callers, UMI, and GPU routes are
absent.

## RNA-seq

**Package:** [`assets/pipelines/rnaseq`](../assets/pipelines/rnaseq)

**Default:** `rnaseq-star-salmon-v1`, benchmarked against nf-core/rnaseq 3.26.0.

`DefaultConfig` names `in/reference/genome.fasta` and compressed
`in/reference/genes_with_empty_tid.gtf.gz`; absent STAR and Salmon Trees are
generated. Its executable retention and inference policy uses 10,000 trimmed
reads, 5% uniquely mapped reads, 0.8 stranded-call support, and 0.1 maximum
forward/reverse difference for an unstranded call. These are configurable
engineering gates, not validated scientific acceptance thresholds. Because
cohort DESeq2 QC requires dispersion estimation, at least two samples are
required.

Supported typed changes cover reference and ready indexes; sample-retention and
strandedness-inference thresholds; Trim Galore; STAR and Salmon; duplicate
marking; StringTie; coverage; selected alignment and biotype QC; tximport;
DESeq2 cohort QC; MultiQC; resources; images; argv extras; result root; and
optional publication of trimmed reads, STAR intermediates, or generated
reference products. Final BAMs, quantification, matrices, tracks, and reports
cannot be disabled.

**Stages:** reference decompression and preparation; FASTQ lint and raw FastQC;
run consolidation; Trim Galore and post-trim FastQC; Salmon-based `auto`
strandedness inference; STAR genome and transcriptome alignment; alignment-mode
Salmon; BAM sort, duplicate marking, index, and statistics; StringTie;
strand-aware coverage; RSeQC, Qualimap, dupRadar, and featureCounts biotype QC;
tximport matrices; DESeq2 PCA and sample-distance QC; MultiQC.

**Required outputs:** marked/indexed BAMs; STAR logs and junctions; per-sample
`quant.sf` and Salmon reports; gene and transcript count, TPM, length, and
scaled matrices plus an R object; reference-guided transcript and abundance
files; BigWig tracks; selected QC; DESeq2 cohort QC; MultiQC HTML and data.

**Limits:** featureCounts is biotype QC, not abundance authority. DESeq2 has no
study design or contrast. Gobble does not perform differential expression,
choose a reference or study design, validate expression biology, or establish
that the configured engineering gates are scientifically suitable.

## Methyl-seq

**Package:** [`assets/pipelines/methylseq`](../assets/pipelines/methylseq)

**Default:** `methylseq-bismark-v1`, benchmarked against nf-core/methylseq 4.2.0.

`DefaultConfig` names `in/reference/genome.fa` and the standard directional
Bismark/Bowtie2 route. It generates one complete Bismark index Tree unless a
ready Tree is supplied. A ready Tree is a workspace-relative root with a
regular root `.gobble-tree.json`; directory presence alone is incomplete.

Supported typed changes cover the source or ready reference; Trim Galore;
directional Bismark alignment; deduplication; extractor overlap, ignored-base,
coverage, and context settings; reports; resources; images; argv extras; result
root; and optional publication of trimmed reads or the generated index.
Deduplicated BAMs, methylation calls, and reports cannot be disabled.

**Stages:** run consolidation; raw FastQC; Trim Galore and post-trim FastQC;
Bismark genome preparation when needed; directional alignment; deduplication;
CpG/CHG/CHH extraction; per-sample Bismark HTML; run summary; MultiQC.

**Required outputs:** deduplicated BAMs and reports; context calls; CpG coverage
and bedGraph; M-bias and splitting reports; per-sample Bismark HTML; summary
HTML and text; MultiQC HTML and data.

**Limits:** Only standard directional bisulfite libraries are supported. PBAT,
RRBS, EM-seq, TAPS, BWA-Meth, targeted and single-cell libraries,
coverage-to-cytosine, NOMe, and DMR analysis are absent. Gobble does not
recommend trimming or interpret conversion or methylation values.

## ATAC-seq

**Package:** [`assets/pipelines/atacseq`](../assets/pipelines/atacseq)

**Default:** `atacseq-bwa-v1`, benchmarked against nf-core/atacseq 2.1.2.

`DefaultConfig` names `in/reference/genome.fa` and
`in/reference/genes.gtf`, generates BWA/FASTA indexes, uses mitochondrial name
`MT`, read length 50, effective genome size `12157105`, minimum MAPQ 20,
orphan and mitochondrial filtering, broad MACS2 peaks, and consensus minimum 1.
The default supplies no blacklist and leaves blacklist filtering disabled;
enabling it requires a typed blacklist.

Supported typed changes cover reference, annotation, ready BWA index,
blacklist, mitochondrial name, organism, read length, and effective genome
size; filters; broad or narrow MACS2 mode and cutoff; consensus minimum;
FeatureCounts; DESeq2 QC; ataqv; coverage and reports; resources; images; argv
extras; and result root. All publication categories—alignments, tracks, peaks,
consensus matrix, QC, reports, and IGV session—are required.

**Stages:** reference indexing and gene/TSS/chromosome assets; raw and post-trim
FastQC; Trim Galore; read-grouped BWA-MEM; technical-run sorting, indexing, and
stats; strict replicate merge; duplicate marking and typed filters; alignment
metrics; RPM BigWigs and deepTools QC; control-aware MACS2; HOMER, peak count,
FRiP, and ataqv; strict consensus BED/SAF/presence tables; PE/SE FeatureCounts
and checked matrix merge; DESeq2 cohort QC; multi-replicate aggregate branches;
ataqv browser report; MultiQC; IGV session. Known runs and replicates expand at
compose time, not through runtime Scatter.

**Required outputs:** filtered indexed BAMs; normalized tracks; alignment and
ATAC QC; replicate and eligible aggregate peaks and annotations; consensus peak
sets and count matrices; PCA and distance QC; ataqv and MultiQC reports; IGV
session.

**Limits:** DESeq2 is cohort QC and accepts no contrast. Gobble does not claim
reproducible peaks, appropriate controls, valid differential accessibility,
assay quality, or study suitability. Alternate aligners, IDR, Preseq, motif
discovery, footprinting, and study-specific differential analysis are absent.

## scRNA-seq

**Package:** [`assets/pipelines/scrnaseq`](../assets/pipelines/scrnaseq)

**Default:** `scrnaseq-simpleaf-v1`, benchmarked against nf-core/scrnaseq 4.2.0.

`DefaultConfig` selects typed `10XV2`, `cr-like` UMI resolution,
`in/reference/genome.fa`, `in/reference/genes.gtf`, and
`in/reference/10x_V2_barcode_whitelist.txt.gz`. A source reference is normalized
and indexed. A ready reference instead requires a complete Simpleaf index Tree
and separate transcript-to-gene file; source and ready forms cannot mix. The
Tree root, relation, whitelist, references, reads, and result root must not
alias or contain one another.

Supported typed protocols are `10XV1`, `10XV2`, `10XV3`, and `10XV4`; none is
inferred. The whitelist protocol must match. V2–V4 require the matching typed
QCatch chemistry. V1 has no QCatch chemistry mapping and needs an explicit
positive partition count. Supported typed changes also cover UMI resolution,
QCatch filtering and optional doublet settings, conversions, resources, images,
argv extras, and result root. Every publication category is required.

**Stages:** raw FastQC per original mate; paired-run consolidation; GTF sequence
filtering, transcript extraction, and transcript-to-gene projection; Simpleaf
index Tree; Simpleaf map and quantification Trees; QCatch report, metrics, and
separate filtered h5ad; raw matrix to h5ad and Seurat/SingleCellExperiment RDS;
strict combined raw h5ad; MultiQC. Samples expand at compose time. Cells and
barcodes remain matrix rows, not scheduled tasks.

**Required outputs:** Simpleaf index, map, and quantification Trees; raw sparse
matrix members; QCatch report, metrics, and filtered h5ad; per-sample raw h5ad
and RDS conversions; `combined_raw_matrix.h5ad`; FastQC; MultiQC.

**Limits:** Gobble does not claim barcode validity, correct expected-cell
metadata, suitable QCatch filtering, or normalized, integrated, annotated, or
clustered matrices. CellBender, Cell Ranger, alternate aligners, custom
chemistries, Feature Barcode, VDJ, multiome, demultiplexing, differential
analysis, trajectory, and RNA velocity are absent.

## Defects

Sheet loading and graph construction fail through `*gobble.Error`, not panic or
silent fallback. Each error names an operation and one or more `Defect` values
with stable `code`, `unit`, `message`, and `paths` fields.

Common product construction codes are `not-found`, `invalid-samplesheet`,
`invalid-path`, and `invalid-value`. They cover missing sheets, malformed or
inconsistent rows, path escape or aliasing, incomplete references and Trees,
unsupported protocols or route settings, disabled required outputs, mutable or
invalid image references, and named-option/`ExtraArgs` conflicts. Engine
validation adds graph and bind defects. Runtime failures remain structured and
contained; [Operations](operations.md#recovery) owns their recovery.

Do not turn a rejected input into another route, drop a failed cohort member,
or accept an incomplete Tree. Correct the stated unit or path, rebuild the same
selected graph, and inspect the resulting plan before running.
