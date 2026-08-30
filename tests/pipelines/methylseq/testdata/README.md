# Methyl-seq Bismark testdata

This directory is the sole fixture authority for the supported Methyl product.
`manifest.json` resolves nf-core/methylseq 4.2.0 and
`nf-core/test-datasets` `methylseq@e7e1fb8940fc14e2336101147a31ce8e0eda6264`
to immutable source commits, byte sizes, SHA-256 values, provenance, licenses,
stage uses, complete ready-index archive membership, and linux/amd64 image
tags and digests.

`methylseq-samplesheet.csv` localizes the upstream test rows to declared
workspace paths and removes the upstream `genome` column because reference
selection is typed analysis config. It preserves the three official
single-end bytes, the repeated `SRR389222_sub2` membership, and the official
paired E. coli reads. Product construction and tasks never fetch these bytes.
`cache/` is an ignored host preparation cache, not another authority.

The generated-index fixture uses `genome.fa`. The ready-index branch uses the
same exact FASTA plus every member of `Bowtie2_Index.tar.gz` as the declared
`BismarkIndex` Tree. The manifest records the archive and every extracted
member separately. A fixture preparer must reject a missing, extra, wrong-size,
or wrong-hash member rather than rewrite this authority.

The small bacterial data proves graph, command, artifact, and lifecycle
engineering only. Its conversion and methylation values do not define product
thresholds or establish scientific or clinical suitability.
