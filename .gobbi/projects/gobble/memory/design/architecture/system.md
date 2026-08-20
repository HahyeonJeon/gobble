# Gobble — System

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. Keep one system file. After `## Products`,
write one `### {Product}` subsection under every remaining heading, including
when the project has one product. Start an inherited answer with `Inherited — `.

## Products

- Products:
  - Gobble
- Source: `products`

## Composition

### Gobble

- Statement: Major parts are the Go core library (pipeline model: task, pipeline, input, output, resource, environment), engine services (validator, planner, scheduler), executors (local process and Docker), and interfaces (Go API, then CLI). The seam between scheduler and executors is `Executor` Submit/Poll/Cancel/Reconcile in `internal/engine/exec`. Empty Image selects the process adapter; non-empty Image selects docker. Path render lives in `internal/path`. Document is the only engine payload. SchemaVersion is 2. The first-horizon scheduler keys `reservedIdentity`. Admission uses remaining CPU, memory, and the count cap. Docker receives `--cpus` and `--memory` when non-zero. Zero stays unspecified. Fairness, quotas, and job arrays wait. Each task declares its environment and must be runnable by itself. A container task must declare its Docker image. A local-process task must not require an image. Construction order is library, then engine, then CLI. First-horizon exit still requires the same loop on the CLI.
- Source: `shape`

## Parts and responsibilities

### Gobble

- Component: Go core library
- Responsibility: Own the pipeline model and the public compose API. Built, not bought.
- Component: Validator and planner
- Responsibility: Detect defects before execution and emit an inspectable plan and DAG.
- Component: Scheduler
- Responsibility: Decide task readiness from the DAG, per-task CPU and memory, and run-level concurrency. Do not place jobs on Docker or the host.
- Component: Executors
- Responsibility: Run a ready task as a local process or in that task’s declared Docker image. Contain backend failure. Adopt Docker; keep the interface replaceable for later Podman.
- Component: Run state and artifact files
- Responsibility: Persist run and task state, cache or reuse decisions, work directories, and published outputs as local files. One task or executor failure must not take down the scheduler or this state.
- Source: `shape`, `build-buy-adopt`, `failure-containment`

## Data and flow

### Gobble

- Statement: Assumption — pipeline definitions live in Go source. First-horizon run state, cache or reuse decisions, artifacts, logs, and provenance live as local files in the run workspace. A run creates isolated task work directories, stages inputs (hardlink, then process-only symlink, then copy), writes intermediates, publishes outputs on success (hardlink then copy, never symlink), and keeps enough state to resume. Secrets must not be persisted in logs or metadata. Retention policy is unset. A later store may replace the file implementation. Until `cache-inputs` is accepted, reuse a prior successful task output only when reservedIdentity, command or script, params, env, authored image string, recorded input path plus dest cheap keys, and published dest cheap keys all match; otherwise treat that task and its downstream dependents as affected. Content SHA-256 is stored at publish. Inspect remaining does not hash bytes. Input cheap keys are recorded at task success (R1). Image digest is recorded on the attempt and is not identity. Cross-workspace cache stays excluded.
- Authoritative source: the run workspace on the local filesystem for that run
- Source: `data`, `data-lifecycle`

## Interfaces

### Gobble

- Statement: Internal seams are library to engine, scheduler to executor, and engine to state and artifact files. External seams are the public Go API first and the CLI second. First-horizon exit still requires the same loop on the CLI. Later agent APIs share the same model. JSON or JSONL is the default library and CLI response encoding. JSON or YAML as a pipeline interchange document is later, not a first-horizon pipeline language.
- Current: Public library verbs are `Compose`, `Validate`, `BuildPlan`, `Run`, `Inspect`, `Resume`, and `Release`. `Run(ctx, graph, workspace, cap)` and `Resume(ctx, graph, workspace, cap)` cancel through context and return `canceled`; occupancy stays until `Release`. There is no public Cancel, Diff, Retry, or Clean. Occupancy is an owner record on `.gobble/run.json`. `Release` is the occupancy-break path and is not deletion. Installed product binary is `cmd/gobble`. No public `cli` package. No Graph JSON. Graph verbs take a Go package path (default `.`), require `go` on PATH, compile a generated driver, call `func Pipeline() *gobble.Pipeline`, then Compose. Inspect and release run in-process. Seven product verbs: `compose`, `validate`, `plan`, `run`, `inspect`, `resume`, `release`. Inspect is `gobble inspect VIEW --workspace DIR`. VIEW matches library views. Supporting `help` and `version` exist; they are not the product loop. `--workspace DIR` is required on run, inspect, resume, and release. Do not create DIR. Do not infer from cwd. `--cap N` is optional on run and resume; omit passes `0`. `--instance ID` is optional on inspect; empty selects every reserved identity. SIGINT/SIGTERM on run and resume cancel ctx. Occupancy stays. No `cancel` verb. Success: JSON or JSONL on stdout. Failure: empty stdout, `*Error` JSON on stderr. Exits `0` / `1` / `2`. Domain → `1`. Invocation, unknown command, bad flag, compile, missing constructor → `2`. `Pipeline` or `init` panic is a process abort; stderr is not JSON. Missing-workspace codes stay verb-specific (`invalid-path` vs `not-found`). Do not collapse them. Linux is the supported platform. Deferred: Graph/Document JSON load, public `cli`, human pretty output, color, progress, completions, config files, env workspace, workspace auto-create, macOS support promise, public Cancel/Diff/Retry/Clean, `--format`, `--constructor`.
- Source: `interfaces`

## Stack

Local-stack rows and `## First check` are the only Bootstrap inputs from design.

### Gobble

| Product | Entry | Category | Responsibility | Local or cloud | Version policy | Rationale | Constraints |
|---|---|---|---|---|---|---|---|
| Gobble | Go | language | Author and build the library, engine, and CLI | local | 1.26 or newer | Forced language (`stack`) | Module path `github.com/HahyeonJeon/gobble` |
| Gobble | Go toolchain | toolchain | Compile and run `go test ./...` | local | 1.26 or newer | First-check and change path (`first-check`, `stack`) | Missing Go blocks build and import (`dependency-failure`) |
| Gobble | Docker | runtime | Run each container task in its declared image | local | Open — no Docker version recorded | First-horizon container runtime (`stack`); required for first success | Missing Docker blocks container tasks and first-horizon success (`dependency-failure`) |
| Gobble | Local files | datastore | Hold run state, cache decisions, artifacts, and logs | local | none | First-horizon store; later stores replaceable (`stack`, `data`) | Run workspace is authoritative (`state-authority`) |
| Gobble | Podman | runtime | Later local container executor | later-cloud | Open — not first-horizon | Replaceable executor after Docker (`dependency-exit`). Column has no later-local value. Not a Bootstrap first-horizon local-stack row. | Not required to exit first horizon |
| Gobble | Slurm | runtime | First HPC backend | later-cloud | Open — not first-horizon | After local and container backends (`horizon-direction`). HPC, not cloud; comes before cloud batch. | Must not force the core model |
| Gobble | Cloud batch | runtime | Later cloud backend | later-cloud | unknown | After Slurm (`horizon-direction`) | Must not force the core model |
| Gobble | Kubernetes | runtime | Later orchestrated backend | later-cloud | unknown | After Slurm (`horizon-direction`) | Must not force the core model |

- Source: `stack`, `local-or-cloud`, `horizon-direction`

Bootstrap first-horizon local-stack rows are only Go, Go toolchain, Docker, and local files. Podman is later and is not a Bootstrap input.

## First check

Do not use a no-op, `echo`, or any command that ignores the local toolchain.

### Gobble

- Command: `go test ./...`
- What it proves: Go 1.26 or newer is installed, the module `github.com/HahyeonJeon/gobble` builds, and hermetic package tests for `gobble`, `cmd/gobble`, `assets`, `internal/engine`, `internal/path`, `internal/engine/exec`, and `tests/wgs-e2e` pass. Live tests use build tag `live` and are not in this command. It cannot skip for Docker. It is not proof of a live Docker assay.
- Source: `first-check`

Project command, after every product subsection. If two local products
disagree, mark Open and ask.

- Project command: `go test ./...`
- Taken from product: Gobble
- Default: the first named product that has at least one `local` Stack row
- Source: `first-check`

## Environments

### Gobble

- Environment: developer workstation
- What differs: authoring and first-check; Docker may be absent, in which case first-horizon success cannot be claimed
- Environment: local pipeline run
- What differs: executes tasks in declared images or as local processes; writes the run workspace
- Environment: HPC and cloud profiles
- What differs: later; must not be required to use the library
- Source: `environments`

## Verification and build risk

### Gobble

- Verification: Assumption — a change is safe to keep when hermetic `go test ./...` passes, including package tests for `gobble`, `cmd/gobble`, `assets`, `internal/engine`, `internal/path`, `internal/engine/exec`, and `tests/wgs-e2e` when those packages change. Agent-operability of live Docker run, inspect, and resume is not proved by first-check. Live is `go test -tags=live` and fails closed without Docker. The WGS assay is `tests/wgs-e2e` executing `assets.WGS()`, including Inspect, Release, and Resume. Live RNA/Methyl proofs remain in package `assets`.
- Build risk: Assumption — the part most likely to be wrong is the pipeline model: whether it can express modules, branch, and merge so an agent can plan, run, and resume without a DSL. Early evidence is the synthetic workflow-case pipeline, then WGS end-to-end on a small dataset.
- Source: `verification`, `build-risk`

## Change path

### Gobble

- Statement: Assumption — the solo maintainer changes a topic branch, runs `go test ./...` locally, and commits. There is no named CI, review, or release path yet.
- Source: `change-path`

## Open questions

| Id | Question | What would resolve it |
|---|---|---|
| cache-inputs | Which long-term inputs participate in reuse? The first-horizon workspace rule is dest and input cheap keys plus stored content digest; cross-workspace cache is excluded. | An accepted long-term cache fingerprint rule |
| retention-deletion | How long is run state kept, and what deletes it? | An accepted retention policy |
| processing-model | Do pipeline results arrive live, in batches, or on demand? | A recorded processing model |
| public-contract | Which public types or functions must current callers keep? | Project Design public API after first library surface |
| dependency-unavailable | What does the consumer see while Docker is down? | A recorded Docker-down status shape |

- Source: open topic ids
