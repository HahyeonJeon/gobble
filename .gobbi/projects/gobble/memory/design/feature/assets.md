# assets — Module and pipeline ownership

First-party command adders live in one command package below `assets/modules`.
The graph-stable WGS, RNA-seq, and Methyl-seq checkpoints live below
`assets/pipelines/wgs`, `assets/pipelines/rnaseq`, and
`assets/pipelines/methylseq`. Package `gobble` and `cmd/gobble` do not import
asset or product packages.

Package `assets` temporarily exposes only `WGS`, `RNASeq`, and `MethylSeq`.
Each constructor delegates directly to its pipeline owner. The mechanical move
preserves task ids, edges, commands, images, parameters, resources, inputs,
binds, and destinations. The pre-move plan SHA-256 values are locked by pipeline
tests: WGS `fd762650d4fcfb4f14b862a67cc123777e98a3b2cd291b76196d94472295e2f1`,
RNA-seq `827931c2a6addaf716b8a9ee62057177b3a1838d135d967dc575906fbd948667`,
and Methyl-seq `d6cd91ea0e4962f3cea1eec9633912a1f6f9b01261445b28f356c9420b23e517`.

This checkpoint preserves unchanged-graph workspace meaning only. The later
RNA, Methyl, and WGS main-path lifts are separate graph generations and require
new workspaces. The shims preserve source names, not old graph bytes after a
lift.

Module evidence follows each command below `tests/modules/<module>`. Pipeline
graph and fixture evidence follows each assay below `tests/pipelines/<assay>`.
The WGS JSON manifest is the sole WGS pin authority. RNA and Methyl pin records
and sheets have one assay owner each. Shared fetch and plan-check mechanics live
under `tests/internal` and contain no fixture facts.

`LinkedQC` is design evidence under `tests/scenarios/design`. `OptionalMate` and
the synthetic Scatter, Gather, When, Tree, fan-out, and cohort proofs are engine
and resume evidence under `tests/scenarios/resume`. None is a product.
