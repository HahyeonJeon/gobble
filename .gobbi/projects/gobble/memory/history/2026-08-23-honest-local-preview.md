# Honest local preview completed

**Completed at:** 2026-08-23T15:54:00Z

## Changes

- Made Compose, Validate, Plan, Run, Inspect, Release, and Resume an honest pre-1.0 trusted-local Linux preview on `feat/gobble-release-ready` from `develop` `5da1386`.
- Narrowed the supported public contract to those seven verbs; Module, Branch, Merge, Scatter, Gather, and When; PathSpec, File, Group, and Tree; samplesheet parse; and structured errors. Docker stays isolation convenience. The license remains unset and grants no redistribution.
- Recovery is Inspect, later-process Release without unproved PID kill, then Resume remaining work. Later-process dest-complete process work persists `published-unfinalized` and is omitted from remaining; `classifyReuse` is never applied. Later-process incomplete leftovers that still store a RuntimeID rerun. Resume may replace dest-incomplete present dests of that incomplete or succeeded recorded producer; failed, blocked, and repathed foreign dests remain `output-exists`. Later-process Docker `unknown-backend` keeps occupancy active and blocks Resume.
- Fingerprints cover staged isolate bytes. Missing hashes miss. Resume re-evaluates When.
- Docker client Env is not task Env. After Wait returns, Cancel does not signal. Log-copy and `docker rm` failures are visible. Caching a stopped Docker terminal only after both cleanups succeed stays deferred.
- Concurrent samplesheet load is `LoadSampleSheetFile`. The provisional setter is process-owned.
- Combined hermetic `go test ./...` PASS at `4b08ef8`.
- Accepted heads oldest first: `0dad6ad` public and CLI contract; `c8437ff` executor host boundary; `4ba88ed` workspace honesty; `d29805c` remaining-work Resume; `9602b75` process-owned samplesheet; `4b08ef8` Cancel after Wait.
- Current design is in [Design Memory](../design/README.md). The session note is [Honest local preview](../reports/note/2026-08-23-honest-local-preview.md). Code Review of `4ba88ed` is historical.
