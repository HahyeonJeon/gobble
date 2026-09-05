# Design: macOS installation and guided pipeline runs

Status: option A accepted. Tutorials use existing assay pipelines and official
manifest-pinned test data, as requested by the user.
Baseline: `ca1396c92c447e71705369a2a97e0d23631b3375`.

## Goal

A Mac user installs Docker Desktop and Gobble, opens a terminal or coding
agent, and completes small, reproducible pipeline exercises. Host Go and
manual bioinformatics tool installation are unnecessary. Direct Linux remains
the advanced installation route.

## Review findings

- `cmd/gobble-container/main.go` rejects Darwin and non-amd64 hosts.
- `distribution/runtime/Dockerfile` only exports Linux and Windows launchers.
  Building it with native Apple Silicon defaults would produce an ARM runtime,
  but CLI and engine installation identity checks currently require
  `linux/amd64`.
- Existing analysis modules pin linux/amd64 images. A native ARM launcher alone
  does not make the runtime and analysis images ARM-compatible.
- Docker Desktop's client socket and its VM-side socket have distinct paths.
  Mac support must discover the client endpoint and mount the daemon-side
  socket, while preserving the existing daemon identity checks.
- `tests/runtime-e2e/smoke.py` replaces Unix PATH with `/usr/bin:/bin`, hiding
  Docker Desktop's usual Mac CLI location. Its no-host-Go check needs a
  portable, explicitly prepared command path.
- Existing assays already own immutable fixture manifests with data checksums,
  provenance, and default image pins. They are the tutorial authority.
- Some existing tests replace tool execution. Those tests cannot establish that
  a novice can run the real tools after installation.

## Architecture decision

| Option | Structure | Tradeoff |
|---|---|---|
| A — accepted for the first Mac delivery | Native Intel/Apple Silicon launcher; explicit linux/amd64 runtime and supported analysis images | Reuses the existing engine, identity, and tool-image contracts. Apple Silicon needs emulation and may run more slowly. |
| B — native ARM runtime in the same delivery | Native launcher and linux/arm64 controller; per-tool architecture handling | Faster native authoring is possible, but expands identity, packing, runtime distribution, and mixed-architecture testing. Existing amd64 tool images still need emulation or separately validated ARM replacements. |

Accepted initial target matrix for option A:

| Host | Launcher | Execution environment |
|---|---|---|
| Intel Mac | darwin/amd64 | Docker Desktop, linux/amd64 |
| Apple Silicon Mac | darwin/arm64 | Docker Desktop, emulated linux/amd64 |
| Linux x64 | linux/amd64 | Local Docker Engine, linux/amd64 |
| Windows x64 | windows/amd64 | Docker Desktop, linux/amd64 |

The launcher should select and validate the runtime platform explicitly, report
an incompatible image before occupying a run, and preserve the exact runtime
image and daemon identity used for Stop and Resume. Architecture upgrades must
not silently migrate an existing workspace.

## Tutorial design

`gobble demo NAME DIR` prepares a fresh ordinary Go project. NAME selects one of
`rnaseq`, `wgs`, `methylseq`, `atacseq`, or `scrnaseq`. The generated entry point
calls that existing assay's `Pipeline()` and `DefaultConfig`; no tutorial-only
graph or fake command replaces it. Run, watch, Stop, and Resume remain the normal
commands. `gobble init` remains a tiny installation check.

Input manifests and samplesheets stay owned by `tests/pipelines/<assay>/testdata`.
The runtime includes its exact source checkout and reads those files. Shared
verification/downloading code moved from `tests/internal/fixture` to
`internal/fixture`, so both tests and the CLI use the same byte checks. Failed or
cancelled downloads do not publish unverified cache entries; retry reuses valid
bytes. WGS and its evidence tests share interval-member materialization.

The user-facing [walkthrough](../tutorials.md) documents default resources,
input download sizes, much larger image downloads, expected result paths, agent
prompts, and recovery. It distinguishes a validated graph from full real-tool
execution. RNA-seq and WGS gain installed-runtime Docker CI; other full assay
runs can use the same acceptance script on sufficiently provisioned hosts.

## Implementation

1. Native Darwin launchers, explicit runtime platform checks, Desktop VM socket
   handling, and symlink/path regression tests. Both task pulls and task creates
   explicitly select linux/amd64.
2. A Mac/Linux setup script builds inside Docker and installs the native launcher.
   It probes amd64 execution before installation, writes an opt-in shell
   environment file, and leaves shell startup files to the user.
3. `demo` prepares actual assay projects and verified inputs without host Go.
4. Linux CI exercises installed RNA-seq/WGS and unchanged Resume. Native Intel
   and Apple Silicon CI tests the launcher. Real Desktop acceptance remains
   separate, using `tests/runtime-e2e/smoke.py` and `demo.py`.
5. README links installation directly to the existing-assay walkthrough.

## Acceptance criteria

- Native launchers build and execute on both Mac architectures.
- From a clean project, doctor proves sibling-container reads and writes.
- The automatic RNA-seq/WGS walkthroughs run without host Go; required final
  files are non-empty and unchanged Resume does not increase attempts.
- Spaces, Unicode, symlink-resolved Mac paths, Docker CLI discovery, and
  project-local/external workspaces behave consistently.
- Stop, repeated Stop, Ctrl+C, and Resume after controller loss preserve the
  existing ownership and checkpoint rules. TUI attachment is tested on a real
  Mac terminal.
- Missing Docker, incompatible runtime images, and missing emulation produce
  actionable errors. Measured timings accompany actual-host results.

This development environment currently has neither a Mac host nor Docker.
GitHub documents that its arm64 macOS runners do not support nested
virtualization; native launcher CI cannot stand in for Docker Desktop tests.
A real Mac or an appropriately configured Mac runner is required for that gate.

## References

- [Docker Desktop for Mac installation](https://docs.docker.com/desktop/setup/install/mac-install/)
- [Docker Desktop Mac sockets and permissions](https://docs.docker.com/desktop/setup/install/mac-permission-requirements/)
- [Docker multi-platform execution and emulation tradeoffs](https://docs.docker.com/build/building/multi-platform/)
- [GitHub-hosted runner architectures and limitations](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)
