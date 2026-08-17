# Gobble

Gobble is a Go library for composing, validating, planning, and running bioinformatics pipelines.

PathSpec is the public parameterized path model. DirName, Prefix, BaseName, Suffixes, and Extension map to `Dir`, `Lead`, `Name`, `Steps`, and `Ext` (JSON `dir`, `lead`, `name`, `steps`, `ext`). The public surface is unsupported except these locked PathSpec concepts.

This module can `Run` a valid graph in a caller workspace. It does not inspect or resume a pipeline. `docs/gobble-draft.md` is a non-binding vision draft, not the shipped API.

Import `github.com/HahyeonJeon/gobble`. Authors build a `Pipeline`, call `Compose` for an immutable `Graph`, then `Validate`, `BuildPlan`, or `Run`. Dry run is `BuildPlan`. `Run` occupies the workspace after checks pass. The default concurrency cap is 1. Compose, Validate, BuildPlan, and Run report defects as `*Error` values inspected with `errors.As`. A `WriteTo` failure is the writer's own error.

## First check

Requires Go 1.26 or newer.

```sh
go test ./...
```
