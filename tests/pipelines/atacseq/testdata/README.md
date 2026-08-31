# ATAC-seq testdata authority

`manifest.json` is the sole ATAC fixture authority. It records ten staged bytes
from `nf-core/test-datasets` `atacseq@cd022b097372b078a68d8afadb172ad7342fd91f`:
eight official osmotic-stress FASTQs, the matching FASTA, and the matching GTF.
It also records the exact official PE/SE selector, biological design, dataset
license and provenance, atacseq 2.1.2 test config, selected workflow, and source
license. Every URL contains an immutable commit, byte count, and SHA-256.

`atacseq-samplesheet.csv` localizes the official v2.1 rows as ordinary
workspace paths. T0 PE and T100 SE retain two biological replicates. T15 PE and
T150 SE retain the selector's repeated replicate-1 rows as two technical runs.
The sheet does not infer controls or contrasts. Control-link behavior is tested
with typed local paths and never claims those controls are appropriate.

The manifest's stage trace reaches reference preparation, PE and SE raw and
post-trim QC, BWA alignment, technical-run merge, duplicate/filter/QC/track
work, MACS2 and peak QC, strict replicate and aggregate consensus/count
matrices, DESeq2 cohort QC, ataqv, MultiQC, and IGV. Its image table maps every
actual task command name to an exact tag, registry digest, tool/version, source,
license, and `linux/amd64` tuple.

Hermetic tests never fetch. Live test-only preparation may download staged
entries into the ignored cache, verify size and SHA-256, and copy them into a
caller-created workspace. Product source and `Build` never fetch, inspect, or
stage fixture bytes. No peak count, FRiP, PCA, fingerprint, or replicate metric
is a scientific acceptance threshold.
