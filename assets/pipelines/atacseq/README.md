# ATAC-seq BWA product

`assets/pipelines/atacseq` owns graph generation `atacseq-bwa-v1`, benchmarked
against nf-core/atacseq 2.1.2. This generation introduces the product. There is
no earlier ATAC workspace to resume or migrate.

## Typed entry points

- `Parse` and `Load` accept exact columns
  `sample,fastq_1,fastq_2,replicate` plus optional
  `control,control_replicate`. Unknown columns are errors. Repeated
  sample/replicate rows become ordered technical `Run` values. Biological
  replicates start at 1 without gaps. A control link must resolve to an existing
  sample replicate. No control, contrast, condition, or experimental meaning is
  inferred from a name or filename.
- `DefaultConfig` returns fresh reference, annotation, filter, broad/narrow peak,
  consensus, output, command, immutable image, resource, argv, and publication
  policy. It selects only BWA and the atacseq 2.1.2 primary path. This exact
  default image-and-platform tuple is the supported tuple. Replacing a module
  image through its typed config is an expert software substitution: Gobble
  records the replacement in task identity and provenance, but the replacement
  leaves the supported tuple and has no behavioral-parity support claim.
- `Build(samples, config)` copies caller data and reads no process state,
  filesystem content, current working directory, or network location. Invalid
  run membership, replicate gaps, control links, reference members, paths,
  image references, and protected long or short option aliases are structured
  compose defects.
- `Pipeline` is only the process-exclusive CLI adapter. It loads the injected
  samplesheet path and delegates to `Build` with fresh defaults.

## Selected path and strict outputs

The graph prepares BWA and FASTA indexes plus gene, TSS, chromosome-size, and
autosome assets. It runs raw and post-trim FastQC, Trim Galore, read-grouped BWA
MEM, sorting/indexing and library statistics for every technical run. It then
strictly merges each replicate's technical runs, marks duplicates, applies
typed mapping, orphan, mitochondrial, and optional blacklist filters, emits
indexed final BAMs, alignment metrics, RPM-normalized BigWigs, deepTools
coverage QC, and control-aware MACS2 peaks and call tables. Narrow mode also
produces summits; MACS2 2.2.7.1 does not produce summits in broad mode.

Each replicate peak set has HOMER annotation, peak count, FRiP, and ataqv
evidence. Strict replicate-level and aggregate-level fan-in produces MACS2
peak-distribution summaries and PDFs plus HOMER annotation summaries, PDFs, and
MultiQC tables. Every declared replicate peak and filtered BAM is a required input to
the consensus BED/SAF/presence table and FeatureCounts matrix. Mixed PE/SE
cohorts are counted in typed mode-specific FeatureCounts commands, then merged
through one same-consensus matrix check. DESeq2 emits only size-factor, PCA,
distance, and variance-stabilized cohort QC. A sample with
more than one biological replicate gets a separate aggregate BAM, metrics,
track, peak, annotation, FRiP, and ataqv branch. Samples with one replicate do
not create an empty aggregate branch. When at least two aggregate branches
exist, they receive their own strict consensus and matrix QC. MultiQC, the
ataqv browser report, and the IGV session require every known input. No runtime
Scatter represents known run or replicate membership.

## Lifecycle, recovery, and non-claims

A changed technical run affects its replicate and every downstream aggregate
that includes it. Unrelated sample-local work remains eligible for reuse.
Filter changes preserve matching raw QC and alignment; peak mode, cutoff, or
control-link changes affect peak-dependent and cohort work. Required fan-in
never accepts a missing BAM, peak, count, report, or session resource. Stop,
failure, Inspect, Release, and Resume use the shared engine contract; this
product adds no retry, fallback, repair, cleanup, or migration verb.

Live fixture evidence fetches and verifies the ten exact official bytes, stages
them as product inputs, and drives every selected operation through an
input-consuming hermetic command boundary. This proves byte identity, declared
data flow, strict fan-in, lifecycle state, and provenance without Docker. It
does not claim third-party tool output parity or registry availability. Gobble
does not claim reproducible peaks, appropriate controls, valid differential
accessibility, assay quality, study suitability, or nf-core endorsement. DESeq2
is cohort QC and accepts no design or contrast. Alternate aligners, IDR, Preseq,
motif discovery, footprinting, and study-specific differential analysis are
absent.
