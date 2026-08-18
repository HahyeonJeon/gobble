# WGS e2e isolate proof

**Completed at:** 2026-08-18T02:43:23Z

## Changes

- Added the WGS consumer-test fixture at `testdata/wgs-e2e/` and a thin-slice alignment graph that occupies a workspace and publishes named SAM/BAM/BAI.
- Fixed Run readiness so a related-file output `From` waits on the published from-path after the upstream task succeeds (`d1319ff`).
- Extended the consumer graph through Strelka; live run staged 266,736 pairs, mapped 9857 reads, and published `work/sample.variants.vcf.gz`.
- WGS is a consumer test, not a product feature. Current design is in [run-local](../design/feature/run-local.md) and [compose-pipeline](../design/feature/compose-pipeline.md). The session note is [WGS e2e isolate proof](../reports/note/2026-08-18-wgs-e2e.md).
