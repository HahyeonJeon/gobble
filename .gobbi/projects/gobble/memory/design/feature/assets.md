# assets — First-party proofs

Same-module package `github.com/HahyeonJeon/gobble/assets` holds dual-entry first-party assets. They are proofs, not Gobble product tools. Package `gobble` must not import `assets`. Reverse import is a defect. `tests/wgs-e2e` stays an inlined consumer test.

Constructors: `WGS`, `RNASeq`, `MethylSeq`, and independent `LinkedQC`. Each asset is one tool command: typed options, extra-args, image pin, named regular-file ports, parent adder, and standalone wrapper. Proof constructors call parent adders and wire Handles. `LinkedQC` `AddInput`s one official RNA FASTQ and one official Methyl FASTQ and calls FastQC and MultiQC only.

See [compose-pipeline](compose-pipeline.md) for `AddInput` / `AddInputGroup` and Group From. See [run-local](run-local.md) for isolate restage (`IO.Source`).
