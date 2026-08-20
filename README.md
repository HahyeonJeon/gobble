# Gobble

Gobble is a Go library for composing, validating, planning, and running bioinformatics pipelines.

PathSpec is the public parameterized path model. Fields are Dir, Prefix, Base, Suffixes, and Ext (JSON `dir`, `prefix`, `base`, `suffixes`, `ext`). Linux is the supported platform this session. macOS is not a support promise.

The public surface is unsupported except these locked PathSpec concepts.

This module can `Run` a valid graph in a caller workspace. `Inspect` reads a workspace, `Resume` continues a released run, and `Release` closes occupancy. `docs/gobble-draft.md` is a non-binding vision draft, not the shipped API.

Import `github.com/HahyeonJeon/gobble`. Authors build a `Pipeline`, call `Compose` for an immutable `Graph`, then `Validate`, `BuildPlan`, or `Run`. Dry run is `BuildPlan`. `Run` occupies the workspace after checks pass. The default concurrency cap is 1. Compose, Validate, BuildPlan, and Run report defects as `*Error` values inspected with `errors.As`. A `WriteTo` failure is the writer's own error.

A Bind is File, Group, or Tree. Tree is a declared directory artifact published with `.gobble-tree.json`. `Directory` is placement, not a directory output.

Guarded Clean is designed and not shipped. Occupancy must be closed first. Default scope is isolate directories under `.gobble/tasks`. Dry-run a manifest before delete. Never unguarded dest delete. Refuse while occupied. Tree remains correct without this verb.

## CLI

Install on Linux:

```sh
go install github.com/HahyeonJeon/gobble/cmd/gobble@<version>
```

Graph verbs compile a Go package that exports `func Pipeline() *gobble.Pipeline`, then Compose. The package operand defaults to `.`.

```sh
gobble compose [package]
gobble validate [package]
gobble plan [package]
gobble run [package] --workspace DIR [--cap N]
gobble inspect VIEW --workspace DIR
gobble resume [package] --workspace DIR [--cap N]
gobble release --workspace DIR
```

`--workspace` is required on `run`, `inspect`, `resume`, and `release`. Gobble does not create DIR.

Success writes JSON or JSONL to stdout. Failures write `*Error` JSON to stderr. Exits are `0` success, `1` domain error, and `2` invocation error.

Recovery after a contained failure or interrupt is `inspect`, then `release`, then `resume`. Linux is the supported platform.

## First check

Requires Go 1.26 or newer. Docker and network are not required.

```sh
go test ./...
```

Hermetic first-check must not skip for Docker or network. Live packs use
build tag `live` and fail when Docker or a required download is missing.

```sh
go test -tags=live ./tests/wgs-e2e
go test -tags=live ./assets
go test -tags=live .
```

`./tests/wgs-e2e` is the live WGS assay: `assets.WGS()` success
Inspect+Release+Resume, plus a pack-local thin fail fixture for contained
failure Inspect+Release+Resume. `./assets` RNASeq and MethylSeq are live
Run packs, not that assay. `testdata/run-local` alpine+process is the
live Run pack in package `gobble`. LinkedQC is plan-only.
