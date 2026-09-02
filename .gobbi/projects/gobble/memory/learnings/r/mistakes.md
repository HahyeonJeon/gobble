# R Mistakes

## Reusing one input pair as several differential samples

**Context:** A real differential-analysis command needs replicated sample
columns even when the test makes no biological claim.

**Mistake:** Repeating one paired-read input under several sample names can
produce identical count columns. DESeq2 cannot fit that matrix, so a graph-only
fixture becomes a false execution fixture.

**Correction:** Use distinct, provenance-bound inputs for each represented
sample and assert only the supported result shape. Do not stub or skip the
analysis to make the proof pass. The current RNA-seq product has no study design
or differential-expression result; apply this lesson only to a separately
accepted downstream analysis.
