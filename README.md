# Gobble

Gobble is a Go library for composing, validating, planning, and running bioinformatics pipelines.

PathSpec is the public parameterized path model. Fields are Dir, Prefix, Base, Suffixes, and Ext (JSON `dir`, `prefix`, `base`, `suffixes`, `ext`). Linux is the supported platform this session. macOS is not a support promise.

The public surface is unsupported except these locked PathSpec concepts.

This module can `Run` a valid graph in a caller workspace. `Inspect` reads a workspace, `Resume` continues a released run, and `Release` closes occupancy. `docs/gobble-draft.md` is a non-binding vision draft, not the shipped API.

Import `github.com/HahyeonJeon/gobble`. Authors build a `Pipeline`, call `Compose` for an immutable `Graph`, then `Validate`, `BuildPlan`, or `Run`. Dry run is `BuildPlan`. `Run` occupies the workspace after checks pass. The default concurrency cap is 1. Compose, Validate, BuildPlan, and Run report defects as `*Error` values inspected with `errors.As`. A `WriteTo` failure is the writer's own error.

A Bind is File, Group, or Tree. Tree is a declared directory artifact published with `.gobble-tree.json`. `Directory` is placement, not a directory output.

Guarded Clean is designed and not shipped. Occupancy must be closed first. Default scope is isolate directories under `.gobble/tasks`. Dry-run a manifest before delete. Never unguarded dest delete. Refuse while occupied. Tree remains correct without this verb.

## First check

Requires Go 1.26 or newer. Docker is not required.

```sh
go test ./...
```
