# Multiomics assets proofs

**Completed at:** 2026-08-19T07:19:06Z

## Changes

- Added same-module package `github.com/HahyeonJeon/gobble/assets` with dual-entry first-party tools and constructors `WGS`, `RNASeq`, `MethylSeq`, and `LinkedQC`. These are proofs, not product tools. Reverse import is forbidden. `tests/wgs-e2e` stays inlined.
- Library: `AddInput` stays one PathSpec; `AddInputGroup` originates a Group pipeline input; Group From a pipeline input is allowed when member names match (`575c85f`). Isolate restage copies workspace From onto dest (`IO.Source`, `6b32e53`).
- Official nf-core RNA and Methyl pins and images. STAR and Bismark publish declared regular files, not directory ports. Pin cache is `CacheDir/<sha256[:16]>/<Name>`.
- Accepted work head `0a1f335e0397b6df1d4f91061d1bd88e043e3050`. Current design is in [assets](../design/feature/assets.md) and [compose-pipeline](../design/feature/compose-pipeline.md). The session note is [Multiomics assets](../reports/note/2026-08-19-multiomics-assets.md).
