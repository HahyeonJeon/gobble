# RNA-seq STAR-Salmon testdata

This directory is the sole fixture authority for the supported RNA product.
`manifest.json` resolves the nf-core/rnaseq 3.26.0 test profile and its
transitive `nf-core/test-datasets` references to immutable commits, byte sizes,
SHA-256 values, licenses, stage uses, and linux/amd64 image tags, digests,
commands, versions, module-source revisions, provenance, and license sources.
The manifest records separate tximport and DESeq2-QC runtimes. The selected
DESeq2-QC image was probed at its pinned linux/amd64 digest for R 4.4.2 and
DESeq2 1.46.0. It also records the explicit GTF and single-command Python
sample-retention owners introduced by Gobble's executable product graph.

`rnaseq-samplesheet.csv` and the live-consumer copy
`rnaseq-live-samplesheet.csv` are byte-equivalent in row meaning to the
workspace-relative staged form of the upstream v3.10 test sheet. They retain
every repeated-run membership, sample identity, single- or paired-end mode,
and strandedness value. They change only remote URLs to declared workspace
paths because product Build and tasks never download inputs. `cache/` is an
ignored host preparation cache and is not an alternate authority.

The official yeast fixture proves graph, command, artifact, and lifecycle
engineering only. Its depth and reference do not establish expression
validity, differential expression, study design, or scientific thresholds.
