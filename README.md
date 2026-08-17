# Gobble

Gobble is a Go library for composing, validating, and planning bioinformatics pipelines.

PathSpec is the public parameterized path model. DirName, Prefix, BaseName, Suffixes, and Extension map to `Dir`, `Lead`, `Name`, `Steps`, and `Ext` (JSON `dir`, `lead`, `name`, `steps`, `ext`). The public surface is unsupported except these locked PathSpec concepts.

This module does not yet run, inspect, or resume a pipeline. `docs/gobble-draft.md` is a non-binding vision draft, not the shipped API.

Authors build a `Pipeline`, call `Compose` for an immutable `Graph`, then `Validate` or `BuildPlan`. Dry run is `BuildPlan`. Failures are `*Error` values inspected with `errors.As`.

## First check

Requires Go 1.26 or newer.

```sh
go test ./...
```
