# local-e2e testdata

`manifest.json` pins the official nf-core human R1, R2, FASTA, and FAI URL, byte size, and SHA256.

The live WGS product proof is `assets.WGS()`, staged from those same pin
records via `assets.FetchPin`. Thin and spine graphs in this pack are
demoted and are not the WGS product proof.

RNA and Methyl live sheets are `rnaseq-samplesheet.csv` and
`methylseq-samplesheet.csv`. The RNA sheet uses four distinct GSE110004
PE pairs because reused-FASTQ DESeq2 cannot fit identical counts.

`cache/` holds host-downloaded bytes. The cache must not be committed.
