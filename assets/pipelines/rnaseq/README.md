# RNA-seq STAR-Salmon product

`assets/pipelines/rnaseq` owns Gobble graph generation
`rnaseq-star-salmon-v1`, benchmarked against nf-core/rnaseq 3.26.0. Pre-lift
featureCounts/two-group workspaces are not resumable with this generation. The
temporary `assets.RNASeq` constructor now points here.

## Typed entry points

- `Parse` and `Load` accept the exact assay CSV columns
  `sample,fastq_1,fastq_2,strandedness` plus optional `seq_platform` and
  `seq_center`. Unknown columns are errors. Repeated rows become ordered typed
  `Run` values. Single- and paired-end runs cannot be mixed within one sample.
- `DefaultConfig` returns fresh typed reference, output, module, image,
  resource, and argv policy. It selects STAR and Salmon only.
- `Build(samples, config)` copies caller data, reads no process state or
  network location, and records invalid data, path, image, and option conflicts
  as compose defects.
- `Pipeline` is only the process-exclusive CLI adapter. It loads the injected
  sheet path and delegates to `Build` with fresh defaults.

## Selected path and outputs

The default graph performs reference decompression and preparation, FASTQ lint
and FastQC, repeated-run consolidation, Trim Galore, Salmon-based `auto`
strandedness evidence, STAR genome and transcriptome alignment, alignment-mode
Salmon quantification, sorting, duplicate marking, indexing, StringTie,
combined and directional coverage, selected alignment and biotype QC, tximport
cohort matrices, DESeq2 cohort QC, and MultiQC.

Declared results include marked/indexed BAMs, STAR logs and junctions,
`quant.sf` plus Salmon reports, gene and transcript count/TPM/length matrices,
scaled matrices and an R object, reference-guided transcript and abundance
files, BigWig tracks, selected QC reports, DESeq2 PCA and sample-distance
artifacts, and MultiQC HTML/data. featureCounts is biotype QC only. DESeq2 has
no study design or contrast.

## Lifecycle and support boundary

A changed run invalidates that sample branch and cohort work. A STAR option
change invalidates alignment and its downstream products. A Salmon-only option
preserves STAR work and invalidates quantification, matrices, cohort QC, and
reporting. Report-only changes preserve scientific upstream tasks under normal
Gobble identity rules. Stop, failure, Inspect, Release, and Resume retain the
shared engine contract; the product adds no retry, fallback, or cleanup verb.

The product supports engineering construction, local execution, declared
artifacts, provenance, and recovery. It does not perform differential
expression, select a study design or reference, validate expression biology,
set scientific thresholds, provide clinical interpretation, or claim nf-core
endorsement. The official small yeast data proves engineering behavior only.
