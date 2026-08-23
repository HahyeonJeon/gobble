# Honest local preview session

The session made the seven-verb local loop an honest pre-1.0 trusted-local Linux engine and library preview. Locked decisions D1, D2, D4, D5, and D7 stayed in force.

Result: Supported verbs are Compose, Validate, Plan, Run, Inspect, Release, and Resume. Recovery is Inspect, later-process Release without unproved PID kill, then Resume remaining work. Dest-complete File, Group, and Tree persist `published-unfinalized` and are omitted from remaining; `classifyReuse` is never applied. Later-process incomplete leftovers that still store a RuntimeID are remaining work, not `unknown-backend`. Dest-incomplete present dests replace on Resume. After Wait returns, Cancel does not signal. Concurrent samplesheet load is `LoadSampleSheetFile`; the provisional setter is process-owned. Docker client Env is not task Env. License is unset. First-horizon exit is not claimed.

Evidence: work started from `develop` `5da1386`. Accepted heads oldest first: `0dad6ad` public and CLI contract; `c8437ff` executor host boundary; `4ba88ed` workspace honesty; `d29805c` remaining-work Resume; `9602b75` process-owned samplesheet; `4b08ef8` Cancel after Wait. Combined hermetic `go test ./...` PASS at `4b08ef8`. Publication is local. Merge is not this Memory write.

Limits: no tag or push; first-horizon exit unclaimed; Docker leftover recovery beyond visible log/rm failure stays deferred; Code Review of `4ba88ed` is historical and is not the current tree.

See [Design Memory](../../design/README.md), [history](../../history/2026-08-23-honest-local-preview.md), [product-readiness code review](../review/2026-08-23-local-pipeline-engine-and-library-product-readiness-code-review.md), and [lifecycle review](../review/2026-08-23-local-pipeline-lifecycle-and-test-review.md).
