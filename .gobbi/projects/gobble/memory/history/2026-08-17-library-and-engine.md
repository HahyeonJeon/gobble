# Library and engine compose and plan

**Completed at:** 2026-08-17T12:18:21Z

## Changes

- Shipped PathSpec, Compose, Validate, and BuildPlan on `feat/library-and-engine` in module `github.com/HahyeonJeon/gobble`, with validator and planner in `internal/engine`.
- First-check `go test ./...` now runs public and engine tests, including the workflow-case golden at `testdata/workflow-case/plan.json`.
- Dry run is `BuildPlan`. The module does not run, inspect, or resume a pipeline. Merge into `develop` was not authorized. Current design position is in [Design Memory](../design/README.md). Deferred items are in [compose-pipeline backlog](../backlogs/compose-pipeline.md). The session note is [Library and engine session](../reports/note/2026-08-17-library-and-engine.md).
