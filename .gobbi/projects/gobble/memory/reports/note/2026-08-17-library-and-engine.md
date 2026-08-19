# Library and engine session

The session implemented the public compose, validate, and plan surface for Gobble.

Authors build a `Pipeline` and call `Compose` for an immutable `Graph`, then `Validate` or `BuildPlan`. PathSpec fields are `Dir`, `Lead`, `Name`, `Steps`, and `Ext`. A `From` handle must belong to the composed pipeline. `BuildPlan` validates first. Writer failure keeps the built plan.

Verification is `go test ./...`. The work head at wrap-up start was `be00c9ab1d216c0dc11e48bf4a8cdc62f9c49bb5`. Merge and worktree cleanup were not authorized. See [Design Memory](../../design/README.md) and [compose-pipeline](../../design/feature/compose-pipeline.md). The compose-pipeline backlog file was later removed; see [Design-review evolve](../../history/2026-08-19-design-review.md).
