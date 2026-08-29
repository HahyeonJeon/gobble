# Gobble

Gobble is a pre-1.0 trusted-local Linux pipeline engine and Go library for coding agents and humans who receive a packed pipeline runner.

Gobble supports two install families:

- Agents use the Go library and a generic `gobble` command selected from the same module graph.
- Humans receive one packed runner for one embedded pipeline. They do not install generic Gobble or need Go at run time.

The supported platform is `linux/amd64`. Agents, library consumers, and the machine that creates a packed runner need Go 1.26 or newer. Docker is required only for pipeline tasks that declare a Docker image and for first-horizon evidence. Docker `--network=none` and the caller's UID/GID are isolation conveniences, not a sandbox.

Gobble is licensed under the [MIT License](LICENSE).

## Agent install

The available install route is a local path pin to a trusted Gobble tree. In the consumer pipeline module, select that tree explicitly:

```sh
go mod edit -require=github.com/HahyeonJeon/gobble@v0.0.0
go mod edit -replace=github.com/HahyeonJeon/gobble=/absolute/path/to/gobble
go list -m github.com/HahyeonJeon/gobble
mkdir -p .gobbin
GOBIN="$PWD/.gobbin" go install github.com/HahyeonJeon/gobble/cmd/gobble
export PATH="$PWD/.gobbin:$PATH"
```

The `go install` command intentionally has no version suffix. The consumer's `go.mod` supplies the selected local revision. `v0.0.0` is a local module-graph placeholder, not a release tag. Keep the installed binary on `PATH` for graph verbs.

Exact-tag install is not yet available because no release tag exists. After the user names and authorizes an exact tag, the future commands will use the same tag for the library and command:

```sh
go get github.com/HahyeonJeon/gobble@v0.x.y
GOBIN="$PWD/.gobbin" go install github.com/HahyeonJeon/gobble/cmd/gobble@v0.x.y
```

`v0.x.y` is a pattern, not a published version. No supported install uses `@latest`.

## Human install

An agent with Go and the consumer pipeline module creates a runner at an explicit path:

```sh
gobble pack [package] --output PATH
```

The agent sends that packed `linux/amd64` binary to the human. The human runs its embedded pipeline without a package operand and without Go:

```text
./pipeline-runner compose [--sample PATH]
./pipeline-runner validate [--sample PATH]
./pipeline-runner plan [--sample PATH]
./pipeline-runner run --workspace DIR [--cap N] [--sample PATH]
./pipeline-runner inspect VIEW --workspace DIR [--instance ID]
./pipeline-runner release --workspace DIR
./pipeline-runner resume --workspace DIR [--cap N] [--sample PATH]
```

A packed runner contains Gobble plus one embedded pipeline. Gobble portions are licensed under MIT. The embedded pipeline may have a different license. Its license is not Gobble's unless its author says so. Packed root help includes Gobble's copyright and permission notice.

The runner has no `pack` command and accepts no package operand. Docker remains required for embedded tasks that use Docker.

## Identity

Gobble fails closed unless the installed CLI, selected Gobble module, pipeline revision, platform, install family, and workspace identity match.

Agent graph verbs compile the selected pipeline package and therefore require `go` on `PATH` and a consumer module. The generated child performs a handshake before calling `Pipeline()`. Consumer packages under an `internal/` directory are unsupported. Generic Inspect and Release use the installed binary identity and do not require a consumer module working directory. Packed verbs use the identity embedded when the runner was created.

`inspect identity --workspace DIR` remains readable on mismatch. It reports `required`, `have`, and `match`, including `goos`, `goarch`, and `identity_mode`. Other Inspect views, Release, and Resume refuse a mismatched workspace without mutation.

## Pipeline contract

The supported operations are `Compose`, `Validate`, `BuildPlan`, `Run`, `Inspect`, `Release`, and `Resume`. Composition uses `Module`, `Branch`, `Merge`, `Scatter`, `Gather`, and `When`. Artifacts use `PathSpec` and File, `Group`, or `Tree` binds. Failures use `Error`, `Defect`, and stable `DefectCode` values.

`WriteTo` is the supported plan option constructor. Graph readers `Name`, `TaskIDs`, `InputNames`, and `Edges`, plus Plan JSON, support the documented loop. Other exported names are provisional. `LoadSampleSheetFile(path)` is the supported concurrent samplesheet API.

The caller supplies trusted pipeline code and an existing, exclusive workspace. Gobble copies staged inputs into task isolates and copies published outputs to destinations. It does not use hardlinks or symlinks for staging or publication. Commands, parameters, scripts, stdout, and stderr may persist caller content. Do not put secrets there. Inspect omits task environment values.

## Recovery

After a contained failure, cancellation, or controller death:

1. Inspect structured run, instance, and remaining-work views.
2. Release occupancy with the same install identity that occupied the workspace.
3. Resume the same pipeline to run remaining work.

Cancellation leaves occupancy active. Release closes occupancy; it does not delete control files or artifacts. There is no public Cancel, Retry, Diff, Clean, PID adoption, or repair verb.

A Docker task whose stopped state and exit code were proved may retain a `runtime_id` when log copy or container removal fails. That leftover is terminal and can be retried during later Poll, Reconcile, or Release without wedging occupancy. If Docker disposition is unproved, the task remains `unknown-backend`, occupancy stays active, and Resume is refused. A later process never signals an unproved process PID.

File destinations must be regular files, Group members must all be regular files, and a Tree destination needs its directory and `.gobble-tree.json`. Directory presence alone is not complete. Resume re-evaluates `When`, input fingerprints, and published destination checksums.

## Checks

The first check is hermetic and omits the live install assay:

```sh
go test ./...
```

First-horizon installed-path exit is proved on `linux/amd64` for the local-pin agent install and packed human runner by:

```sh
go test -tags=live ./tests/install-e2e
```

The assay fails rather than skips when Docker is unavailable. It covers a local-pin external consumer, a `GOBIN` command, a packed runner with Go made uncallable, three distinct WGS workspaces, cancellation after a real Docker task starts, non-empty remaining work, Release, and successful Resume. This is local-pin and packed-artifact evidence. No exact-tag install or published release is claimed.

## Versioning

A future pre-1.0 release uses one immutable repository-root tag matching `v0.x.y` for the root module, library, and `cmd/gobble`. The module path has no `/v0` suffix. A tag is never moved or reused, and supported commands never use `@latest`.

A patch release means no intended break to the Go API, CLI protocol, workspace schema, or recovery behavior. A minor release may add features and may declare a pre-1.0 break. Its release notes must name the effects on the Go API, CLI, workspaces, and recovery.

The first version number remains deferred until the user names the exact tag. A pre-release matching `v0.x.y-rc.1` is used only if the user later requests it. This session creates no tag, push, GitHub Release, or other remote publication.

A future tag still requires release notes, MIT license bytes, an exact commit, and installed external-consumer proof. The current local assay does not publish those bytes.

## Out of scope

The current horizon excludes Slurm, cloud and Kubernetes backends, Podman as a supported executor, a GUI, a Gobble DSL, catalogs, public Cancel/Diff/Retry/Clean verbs, assay expansion beyond first-horizon WGS, GitHub Release binaries, and nested Docker-in-Docker. No support is claimed for musl, Windows, macOS, or `linux/arm64`.
