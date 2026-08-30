# RNA-seq STAR-Salmon testdata

This directory is the sole fixture authority for the supported RNA product.
`manifest.json` resolves the nf-core/rnaseq 3.26.0 test profile and its
transitive `nf-core/test-datasets` references to immutable commits, byte sizes,
SHA-256 values, licenses, stage uses, and linux/amd64 image digests.

`rnaseq-samplesheet.csv` is the workspace-relative staged form of the upstream
v3.10 test sheet. It retains repeated, single-end, paired-end, `auto`, and
explicitly stranded rows. It removes remote URLs because product Build and
tasks never download inputs. `cache/` is an ignored host preparation cache and
is not an alternate authority.

The official yeast fixture proves graph, command, artifact, and lifecycle
engineering only. Its depth and reference do not establish expression
validity, differential expression, study design, or scientific thresholds.
