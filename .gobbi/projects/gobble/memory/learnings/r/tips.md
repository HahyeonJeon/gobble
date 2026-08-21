# R Tips

## Skip a leading -- after Rscript trailingOnly

**Context:** Gobble adders invoke `Rscript -e SCRIPT -- operand...` so operands are not treated as Rscript flags.

**Tip:** In the biocontainers R used here, `commandArgs(trailingOnly=TRUE)` can still start with `"--"`. Drop that token before indexing paths, counts, and group labels.

**Application:** Use the same skip in merge-counts and DESeq2 `-e` scripts. Re-check if the image pin changes.
