# WGS checkpoint testdata

`manifest.json` pins the official nf-core human R1, R2, FASTA, and FAI URL, byte size, and SHA256.

The live WGS checkpoint imports `assets/pipelines/wgs` and stages these same
pin records through `tests/pipelines/wgs`. Thin and spine graphs remain engine
evidence and are not WGS graph authority.

`cache/` holds host-downloaded bytes. The cache must not be committed.
