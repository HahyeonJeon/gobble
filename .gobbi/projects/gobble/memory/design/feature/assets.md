# assets — First-party proofs

Same-module package `github.com/HahyeonJeon/gobble/assets` holds dual-entry first-party assets. They are proofs, not Gobble product tools. Package `gobble` and `cmd/gobble` must not import `assets`. Reverse import is a defect.

Constructors: `WGS`, `RNASeq`, `MethylSeq`, and independent `LinkedQC`. Each asset is one tool command: typed options, extra-args, image pin, named ports, parent adder, and standalone wrapper. Proof constructors call parent adders and wire Handles. `LinkedQC` `AddInput`s one official RNA FASTQ and one official Methyl FASTQ and calls FastQC and MultiQC only. `LinkedQC` is plan-only.

`RNASeq()` loads the samplesheet, expands one module per sample, shares one STAR genome, runs featureCounts, merge counts, two-group DESeq2 (`work/deseq2/results.csv`), and merged MultiQC. Live RNA is four samples and two groups. New adders: `AddFeatureCounts`, `AddMergeCounts`, `AddDESeq2`. Images: `quay.io/biocontainers/subread:2.1.1--h577a1d6_0`, `quay.io/biocontainers/bioconductor-deseq2:1.50.2--r45ha27e39d_0`. Live RNA pins are four distinct GSE110004 pairs SRR6357070–SRR6357073 (identical-count fallback). RNA sheet rules require `group` on every row and exactly two groups. Replacing one-sample `RNASeq()` task ids is an authorized proof-constructor break.

`MethylSeq()` loads the samplesheet, expands one module per sample, shares one Bismark genome, runs per-sample extract, and merged MultiQC. There is no DMR. Live Methyl is two samples. Group and gtf are not required. Replacing one-sample `MethylSeq()` task ids is an authorized proof-constructor break.

Read cells use Dir/Base/Ext via `sheetFileSpec` because `AppendSuffix` refuses Literal. Shared reference/GTF cells may stay Literal when not suffixed.

`WGS()` stays authored two-sample modules. Scenario live home is `tests/local-e2e`, not `tests/wgs-e2e`. Live recover (occupy, remaining empty, occupied second Run, Release, Resume, reuse `reused-identity-matched`) is proved on WGS, RNA, and Methyl in that pack through API and CLI.

STAR `--genomeDir` is a Tree dest via `DeclareTree`. Directory remains placement. Bismark genome folder is still a token plus declared regular files.

See [compose-pipeline](compose-pipeline.md) for `AddInput` / `AddInputGroup` / `AddInputTree` and File, Group, Tree. See [run-local](run-local.md) for isolate restage (`IO.Source`) and link-then-copy staging.
