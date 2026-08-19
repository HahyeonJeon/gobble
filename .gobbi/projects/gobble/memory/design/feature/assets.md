# assets — First-party proofs

Same-module package `github.com/HahyeonJeon/gobble/assets` holds dual-entry first-party assets. They are proofs, not Gobble product tools. Package `gobble` must not import `assets`. Reverse import is a defect.

Constructors: `WGS`, `RNASeq`, `MethylSeq`, and independent `LinkedQC`. Each asset is one tool command: typed options, extra-args, image pin, named ports, parent adder, and standalone wrapper. Proof constructors call parent adders and wire Handles. `LinkedQC` `AddInput`s one official RNA FASTQ and one official Methyl FASTQ and calls FastQC and MultiQC only.

Live WGS product proof is `tests/wgs-e2e` executing `assets.WGS()` (two-sample BWA+QC), including Inspect remaining, Release, and Resume after success. Thin and spine inlined graphs in that pack are not the product proof. Live RNASeq and MethylSeq remain live Run packs, not the Inspect+Release+Resume assay. LinkedQC is plan-only.

STAR `--genomeDir` is a Tree dest via `DeclareTree`. Directory remains placement. Bismark genome folder is still a token plus declared regular files.

See [compose-pipeline](compose-pipeline.md) for `AddInput` / `AddInputGroup` / `AddInputTree` and File, Group, Tree. See [run-local](run-local.md) for isolate restage (`IO.Source`) and link-then-copy staging.
