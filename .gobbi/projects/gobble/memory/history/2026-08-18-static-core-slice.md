# Static-core slice shipped

**Completed at:** 2026-08-18T06:55:32Z

## Changes

- Shipped public `Group`, `Script` XOR `Command`, and `Env` literals on `refactor/design-improve`. `BuildPlan` records wait paths. Unproducible waits are `never-ready`. Unparseable Memory is `invalid-memory`.
- Run applies `--cpus` and `--memory` when non-zero, admits by remaining CPU, memory, and the count cap, stages Group members by name, and reads wait paths from the plan. Isolate and occupy stay.
- Moved WGS consumer proofs to `tests/wgs-e2e` as a public-API package. First-check remains `go test ./...` and includes that package. Inspect, Resume, Collection, scatter, and CLI were not shipped.
- Current design is in [Design Memory](../design/README.md). Deferred items remain in [compose-pipeline backlog](../backlogs/compose-pipeline.md) and the [later run-local disposition](../reports/note/2026-09-02-gobble-multiomics-products.md#deferred-outcomes). The session note is [Static-core slice](../reports/note/2026-08-18-static-core-slice.md).
