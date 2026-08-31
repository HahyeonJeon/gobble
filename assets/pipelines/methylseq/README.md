# Methyl-seq directional Bismark product

`assets/pipelines/methylseq` owns Gobble graph generation
`methylseq-bismark-v1`, benchmarked against nf-core/methylseq 4.2.0. Pre-lift
FastP and enumerated-Group workspaces are not resumable with this generation.
The temporary `assets.MethylSeq` constructor now points here.

## Typed entry points

- `Parse` and `Load` accept the exact columns
  `sample,fastq_1,fastq_2`. Unknown columns are errors. Repeated rows become
  ordered typed `Run` values. One sample cannot mix single- and paired-end
  runs. Reference and library settings are never samplesheet cells.
- `DefaultConfig` returns fresh typed reference, directional library mode,
  output, command, image,
  resource, argv, trimming, alignment, extraction, and publication policy. It
  selects directional Bismark with Bowtie2 only.
- `Build(samples, config)` copies caller data and performs no network I/O. For a
  caller-supplied ready Tree, call `Build` from the workspace root. It requires
  a readable regular root `.gobble-tree.json` at the configured relative path.
  Invalid data, paths, images, ready Trees, option conflicts, and route-changing
  extras are compose defects.
- `Pipeline` is only the process-exclusive CLI adapter. It loads the injected
  sheet path and delegates to `Build` with fresh defaults.

## Selected path and outputs

The graph performs repeated-run FASTQ consolidation, raw FastQC, Trim Galore,
post-trim FastQC, Bismark genome preparation when no ready Tree is supplied,
directional alignment, deduplication, comprehensive CpG/CHG/CHH methylation
extraction, one HTML report per sample, one run summary, and MultiQC. The
generated index is one complete Tree rooted at the exact directory consumed by
Bismark. A ready Tree skips genome preparation and retains the same port kind.

Declared results include deduplicated BAMs and reports, comprehensive context
calls, CpG coverage and bedGraph, M-bias and splitting reports, per-sample
Bismark HTML, run-summary HTML and text, and MultiQC HTML/data. Required BAM,
methylation-call, and report publication cannot be disabled. Typed policy can
also publish trimmed reads and a generated Bismark Tree.

## Lifecycle and support boundary

A changed run invalidates consolidation and that sample's downstream branch. A
changed FASTA or ready Tree invalidates every alignment and downstream result.
Extractor-only changes preserve reference, trimming, alignment, and
deduplication under ordinary Gobble identity rules. Aggregate reports follow
their changed inputs. Stop, failure, Inspect, Release, and Resume use the shared
engine contract; this product adds no retry, fallback, cleanup, or migration
verb.

The product supports engineering construction, local execution, declared
artifacts, provenance, and recovery. It does not support PBAT, RRBS, EM-seq,
TAPS, BWA-Meth, targeted or single-cell libraries, coverage-to-cytosine, NOMe,
or DMR analysis. It does not recommend trimming, interpret conversion or
methylation values, validate a study, provide clinical interpretation, or
claim nf-core endorsement. Official small data proves engineering behavior
only.
