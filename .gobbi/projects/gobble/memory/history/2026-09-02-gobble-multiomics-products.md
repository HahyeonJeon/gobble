# Gobble multiomics products completed

**Completed at:** 2026-09-02T14:51:01Z

## Changes

- Replaced the flat proof-only asset model with five supported local assay
  products: WGS joint germline, bulk RNA-seq, Methyl-seq, ATAC-seq, and
  scRNA-seq. Gobble remains their shared engine, not a sixth assay or integrated
  multiomics orchestrator.
- Added typed assay-owned construction, selected dated paths, immutable image
  and fixture authority, required outputs, and design, build, customize, run,
  resume, stop, and failure outcomes for every product.
- Established new graph generations for the three lifted products and first
  generations for ATAC-seq and scRNA-seq. Temporary WGS, RNA-seq, and Methyl-seq
  constructor shims preserve names only; old proof workspaces require new
  workspaces.
- Kept support engineering-only on trusted-local `linux/amd64`. WGS ends at an
  unfiltered joint VCF, the product family remains unreleased, and live
  Docker/network/third-party command execution was not claimed at closure. The
  completed account is [Gobble multiomics products](../reports/note/2026-09-02-gobble-multiomics-products.md).
