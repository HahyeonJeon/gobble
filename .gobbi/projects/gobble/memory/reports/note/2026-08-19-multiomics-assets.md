# Multiomics assets session

The session shipped first-party dual-entry proofs in package `github.com/HahyeonJeon/gobble/assets` and two library gaps that blocked those proofs.

Result: constructors `WGS`, `RNASeq`, `MethylSeq`, and independent `LinkedQC`. Assets are proofs, not product tools. Reverse import is forbidden. `AddInput` stays one PathSpec; `AddInputGroup` originates a Group pipeline input. Isolate restage copies workspace From onto dest (`IO.Source`). STAR and Bismark declare regular files. Pin cache is content-addressed.

Evidence: accepted work `0a1f335e0397b6df1d4f91061d1bd88e043e3050`. Library commits `575c85f` (AddInputGroup) and `6b32e53` (isolate restage). First-check `go test ./...` includes `gobble`, `assets`, `internal/engine`, and `tests/wgs-e2e`. Official RNA live Run uniquely mapped 41697/50000. Official Methyl unique PE alignments = 5.

Limits: official methylseq test pins pair E. coli Sherman FASTQs with a lambda FASTA. Unique PE alignment is a tripwire `> 0`, not an assay-quality floor. Directory ports and faidx were not added. CLI remains backlogged.

See [assets](../../design/feature/assets.md), [compose-pipeline](../../design/feature/compose-pipeline.md), and [history](../../history/2026-08-19-multiomics-assets.md).
