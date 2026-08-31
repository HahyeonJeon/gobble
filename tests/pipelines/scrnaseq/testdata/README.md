# scRNA-seq fixture authority

`manifest.json` is the sole scRNA-seq fixture and image authority. It pins the
nf-core/scrnaseq 4.2.0 Simpleaf benchmark commit and
`nf-core/test-datasets` commit
`d934d6e8367fe2626184496b1889671cf2b02dab`.

The nine staged inputs are the official Sample_X and repeated-run Sample_Y 10x
V2 read pairs, chr19 FASTA and GTF, and the 4.2.0 V2 whitelist. `evidence.go`
fetches only exact commit URLs, verifies size and SHA-256, and stages ordinary
workspace inputs. Pipeline construction never fetches data. The ignored
`cache/` directory is disposable and is not another authority.

The fixture exercises engineering behavior only. Expected-cell values and
QCatch observations are not thresholds or evidence of barcode validity,
suitable filtering, normalization, integration, annotation, clustering, or
scientific interpretation. CellBender is not part of this product.
