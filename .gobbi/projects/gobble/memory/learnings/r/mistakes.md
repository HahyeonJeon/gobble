# R Mistakes

## Reusing one FASTQ pair as four DEG samples

**Context:** A wiring proof wants four RNA samples and a two-group DESeq2 contrast without claiming biology.

**Mistake:** Reusing one official PE pair on four rows produces identical count columns. DESeq2 cannot fit that matrix and exits non-zero.

**Correction:** Pin four distinct tiny official pairs in the same two groups. Keep result-shape assertions. Do not stub DESeq2. Do not skip the task. Do not assert biologically DE genes.
