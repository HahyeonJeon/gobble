# assets backlog

## Methyl DMR

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Add a DMR analysis task on the Methyl-seq proof graph.

**Why backlogged:** D9 limited Phase 1 Methyl to per-sample Bismark align+extract plus merged MultiQC. Official methylseq 4.2.0 has no DMR.

**Context:** `MethylSeq()` has no DMR, DSS, metilene, or methylKit task. A later item needs a named dataset, image, and contract.

## RNA multi-group DEG

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Accept RNA sheets with more than two distinct `group` values and a contrasts file.

**Why backlogged:** This horizon is one two-group contrast. `RNASeq()` rejects other group counts as `invalid-samplesheet`. Manager authorized exactly two RNA groups after ideation iteration 1.

**Context:** Contrast reference is the lexicographically first group. Live fixture groups are `ctrl` and `treat`. Biological DEG correctness remains rejected.

## Salmon/tximport Phase 1 quant

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Replace featureCounts-on-STAR-BAM with nf-core-faithful Salmon/tximport quantification in the RNA proof graph.

**Why backlogged:** D7 chose featureCounts on STAR BAM for Phase 1 DEG wiring. Salmon/tximport was deferred.

**Context:** Live RNA is STAR BAM → featureCounts → merge counts → two-group DESeq2. Official rnaseq 3.26.0 uses Salmon. A later item needs a named image, tximport contract, and constructor change.

## WGS samplesheet conversion

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Convert `WGS()` from authored two-sample modules to samplesheet expansion.

**Why backlogged:** D11 kept WGS as existing authored two-sample modules. A samplesheet conversion was allowed only as trivial reuse with no extra pins.

**Context:** `RNASeq()` and `MethylSeq()` already load the sheet and emit one module per sample. Empty `read2` is allowed at parse. Those constructors still require a mate. `WGS()` still hard-codes two sample modules and shared FASTQ PathSpecs. The library samplesheet API already exists.
