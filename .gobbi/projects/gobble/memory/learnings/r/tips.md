# R Tips

## Keep Rscript operands aligned with commandArgs

**Context:** Inline R command modules index paths and metadata from
`commandArgs(trailingOnly=TRUE)`.

**Tip:** Pass operands immediately after the `-e` script, as current modules do.
If a command deliberately inserts a standalone `--`, normalize that token
before indexing because some Rscript images retain it in trailing arguments.

**Application:** Recheck argument positions whenever a pinned R image or command
wrapper changes. Current `tximport`, `deseq2-qc`, plotting, and matrix-conversion
modules do not use the removed proof-era merge-counts convention.
