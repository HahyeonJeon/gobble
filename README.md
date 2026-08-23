# Gobble

Gobble is a pre-1.0 trusted-local Linux pipeline engine and Go library. It is a source preview for coding agents and Go developers. First-horizon exit is not claimed.

The supported preview contract is narrow:

- Operations: `Compose`, `Validate`, `BuildPlan`, `Run`, `Inspect`, `Release`, and `Resume`.
- Composition: `Module`, `Branch`, `Merge`, `Scatter`, `Gather`, and `When`.
- Artifacts: `PathSpec` and File, `Group`, or `Tree` binds.
- Samplesheets: `SampleRow`, `SampleSheet`, `ParseSampleSheet`, `LoadSampleSheetFile`, and structured samplesheet errors.
- Failures: `Error`, `Defect`, and stable `DefectCode` values.

`WriteTo` is the supported plan option constructor. Graph readers `Name`, `TaskIDs`, `InputNames`, and `Edges`, plus Plan JSON, support the documented loop. Other exported names are provisional. In particular, `SetSampleSheetPath`, `SampleSheetPath`, `LoadSampleSheet`, raw Inspect Go types, `PlanOption` construction, `TaskSpec.Backend`, and package `assets` are not supported compatibility promises.

The license is unset. No redistribution license is granted. This preview does not provide a supported public-module or versioned-install route.

## Trust and workspace boundary

Gobble supports a trusted pipeline author, trusted pipeline code, and one caller-owned workspace used exclusively by one OS user. Untrusted pipelines, adversarial co-tenants, multi-user workspaces, and regulated or clinical deployment are unsupported.

Process tasks run as the Gobble user. Docker tasks use `--network=none` and the caller's UID/GID as isolation conveniences. Docker is not a sandbox. Commands, parameters, scripts, stdout, and stderr may persist caller content. Do not put secrets there. Inspect omits task environment values.

Gobble stages inputs by copying them into the task isolate and publishes outputs by copying them to their destinations. Staged and published files are independent regular files; staging and publication do not use hardlinks or symlinks. Input fingerprints cover the staged isolate bytes after staging and before execution. Destination checksums are recorded only after successful publication. A missing hash is a reuse miss, and Resume re-evaluates `When` predicates.

## Library loop

Import `github.com/HahyeonJeon/gobble` from a trusted local source revision. Authors build a `Pipeline`, call `Compose` for an immutable `Graph`, then call `Validate`, `BuildPlan`, or `Run`. Dry run is `BuildPlan`. `Run` occupies an existing workspace after checks pass. The default concurrency cap is 1. Supported operations return structured `*Error` values inspected with `errors.As`. A `WriteTo` failure is the writer's own error.

Use `LoadSampleSheetFile(path)` when a caller supplies a samplesheet. It is the supported concurrent API because the path is explicit. The CLI injects `--sample PATH`, or process-cwd `samplesheet.csv` when the flag is omitted, before it calls the pipeline constructor. The provisional hidden-path helpers remain only for proof-pipeline compatibility.

A File bind names one regular-file artifact. A Group names its regular-file members. A Tree is a declared directory artifact published with `.gobble-tree.json`; `Directory` is placement, not a directory output.

## Recovery

Recovery after a contained failure, cancellation, or controller death is `Inspect`, then `Release`, then `Resume` remaining work. A later process may Release after the occupying process is gone. It never signals an unproved process PID. There is no PID adoption or repair verb.

File dest path is a present regular file, not a symlink. Group: every named member is a present regular file, not a symlink. Tree: dest directory exists as a directory and dest `.gobble-tree.json` is a present regular file. Directory presence alone is not dest-complete. The Tree check does not walk every member.

After later-process Release, a process identity whose every destination is dest-complete is recorded as `published-unfinalized` and omitted from remaining work. If any destination is not complete, it is `incomplete` and Resume reruns it. A later-process Docker identity that remains unproved is `unknown-backend`; occupancy stays active and Resume refuses the workspace.

Ordinary Run and Resume execution is bounded only by caller context. The engine settlement bound begins after stop or cancellation starts and during Release reconciliation or polling. It does not limit an uncancelled task's ordinary lifetime.

Release closes occupancy. It does not delete control files or artifacts. Guarded Clean is designed but not shipped.

## CLI

The `gobble` command is a developer engine. Graph verbs compile a Go package that exports `func Pipeline() *gobble.Pipeline`; they require `go` on `PATH` and a module that resolves the package. Consumer packages under an `internal/` directory are unsupported for CLI graph verbs. Graph verbs use the consumer module's Gobble source. Inspect and Release use the installed binary's Gobble source. A cross-version handshake is not provided.

```text
gobble compose [package] [--sample PATH]
gobble validate [package] [--sample PATH]
gobble plan [package] [--sample PATH]
gobble run [package] --workspace DIR [--cap N] [--sample PATH]
gobble inspect VIEW --workspace DIR [--instance ID]
gobble resume [package] --workspace DIR [--cap N] [--sample PATH]
gobble release --workspace DIR
```

`--workspace` is required on `run`, `inspect`, `resume`, and `release`. Gobble does not create it. A valued flag written with a space requires a following token that does not start with `-`; use `--flag=value` when the value itself starts with `-`.

Success writes only protocol JSON or JSONL to stdout. Output written by generated-child package initialization or `Pipeline` is isolated from parent stdout. Failure leaves stdout empty and writes `*Error` JSON to stderr. Child and parent exits are `0` for success, `1` for a domain or operational error, and `2` for an invocation, option, samplesheet, package-list, compile, or constructor error. A panic in package initialization or `Pipeline` remains a process abort, so its stderr is not JSON.

## First check

Go 1.26 or newer is required. Docker and network are not required.

```sh
go test ./...
```

The first check is hermetic and does not use the `live` build tag. Live Docker packs are canaries, not release or D4 recovery evidence.
