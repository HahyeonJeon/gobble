# assets backlog

## Methyl DMR

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Add a supported differentially methylated region analysis outcome.

**Why backlogged:** The current Methyl-seq product ends at directional Bismark
deduplication, extraction, Bismark reports, and MultiQC. The selected
nf-core/methylseq 4.2.0 route does not define a DMR result.

**Context:** `assets/pipelines/methylseq` has no DMR, DSS, metilene, or methylKit
stage. DMR requires study-design and scientific-output semantics that the
current upstream extraction product deliberately does not own.

## RNA multi-group DEG

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Accept RNA data with more than two groups and an explicit contrasts file.

**Why backlogged:** The current STAR-Salmon RNA-seq product owns quantification
and cohort QC only. It has no group column, study design, contrast, or
differential-expression result.

**Context:** Any multi-group DEG result now belongs to a separate downstream
analysis outcome rather than an extension of the assay sheet. Current
`deseq2-qc` remains PCA and sample-distance QC only.

## WGS VQSR

**Backlogged at:** 2026-08-30T08:18:10Z

**What:** Add a supported VQSR filtering route after the gathered WGS joint callset.

**Why backlogged:** The accepted joint-germline product deliberately ends at an
indexed, unfiltered joint VCF. VQSR is optional upstream behavior and was removed
from the selected path before Ideation acceptance.

**Context:** The current graph has no VQSR training resources, filtering tasks,
or filtered output. Adding that route would be a named graph-generation and
workspace compatibility change.
