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

- Statement: Major parts are the Go core library (pipeline model: task, pipeline, input, output, resource, environment, module, branch, merge, scatter, gather, when), engine services (validator, planner, scheduler), executors (local process and Docker), and interfaces (Go API, then CLI). The seam between scheduler and executors is `Executor` Submit/Poll/Cancel/Reconcile in `internal/engine/exec`. Empty Image selects the process adapter; non-empty Image selects docker. Path render lives in `internal/path`. Document is the only engine payload. SchemaVersion is 2. Schema 0 and 1, and PID-only schema-2 occupancy missing a lease, are `unsupported-schema`. The first-horizon scheduler keys `reservedIdentity`. Scatter members fill the instance segment with shard index 0. BuildPlan does not explode Document.Tasks into per-shard identities. Admission uses remaining CPU, memory, and the count cap. Docker receives `--cpus` and `--memory` when non-zero. Zero stays unspecified. Fairness, quotas, and job arrays wait. Each task declares its environment and must be runnable by itself. A container task must declare its Docker image. A local-process task must not require an image. Construction order is library, then engine, then CLI. First-horizon exit still requires the same loop on the CLI.
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

- Statement: Pipeline definitions live in trusted Go source. Run state, reuse decisions, artifacts, logs, and provenance live in the exclusive caller-owned workspace. A run creates isolated task work directories, copies inputs into the isolate, writes intermediates, and copies published outputs to their destinations. Staging and publication do not use hardlinks or symlinks. Input fingerprints cover staged isolate bytes after staging and before Submit. Destination checksums are recorded only after successful publication. Resume reuses a succeeded identity only when reservedIdentity, command or script, params, env digest, authored image plus image digest for Docker or resolved executable SHA for process, and content hashes of consumed inputs and published destinations all match. Missing digests or hashes are reuse misses. `published-unfinalized` is not reuse. Cross-workspace cache stays excluded. The run workspace remains authoritative. Occupancy owner is the occupying process; private liveness is a held flock and lease on `.gobble/run.json`; PID and host are diagnostic Inspect fields. Secrets must not be persisted in commands, parameters, scripts, or logs. Retention policy is unset.
- Authoritative source: the run workspace on the local filesystem for that run
- Source: `data`, `data-lifecycle`

## Interfaces

### Gobble

- Statement: Internal seams are library to engine, scheduler to executor, and engine to state and artifact files. External seams are the public Go API first and the CLI second. First-horizon exit still requires the same loop on the CLI. Later agent APIs share the same model. JSON or JSONL is the default library and CLI response encoding. JSON or YAML as a pipeline interchange document is later, not a first-horizon pipeline language.
- Current: Gobble is a pre-1.0 trusted-local `linux/amd64` preview licensed under MIT. The public contract is `Compose`, `Validate`, `BuildPlan`, `Run`, `Inspect`, `Release`, and `Resume`; Module, Branch, Merge, Scatter, Gather, and When; PathSpec and File, Group, or Tree binds; explicit-path samplesheet parse and load; and structured `Error`, `Defect`, and `DefectCode` values. Other exports remain provisional. Agents use the library and generic command through an explicit local path pin. Humans receive one packed runner for one embedded pipeline and need no Go at run time. Graph verbs and pack require `go` on PATH; consumer `internal/` packages are unsupported. The parent-selected Gobble module and generated child handshake before `Pipeline()`. Run and Resume require `WithIdentity` through the external API. Inspect and Release may omit it and then bind the current executable. `inspect identity` reports required, have, match, goos, goarch, and identity mode; other reads and mutations fail closed on mismatch. The trusted caller owns one exclusive workspace. Docker `--network=none` and UID/GID are conveniences, not a sandbox. Recovery is Inspect, Release, then Resume remaining work. Later-process Release never signals an unproved PID. Dest-complete unproved process work persists `published-unfinalized`; incomplete process work reruns. A proved-stopped Docker task may retain a RuntimeID for cleanup retry without remaining unknown or wedging occupancy. Unproved Docker remains `unknown-backend`, keeps occupancy active, and blocks Resume. There is no PID adoption, repair verb, public Cancel, Diff, Retry, or Clean. Success stdout is JSON or JSONL only, generated-child user stdout is isolated, failure stdout is empty, and exits are 0, 1, or 2. First-horizon installed-path exit is proved for the local-pin agent and packed human families by `go test -tags=live ./tests/install-e2e`. The published agent install is `github.com/HahyeonJeon/gobble@v0.1.0`.
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
- What it proves: Go 1.26 or newer is installed, the module `github.com/HahyeonJeon/gobble` builds, and hermetic package tests for `gobble`, `cmd/gobble`, `assets`, `internal/engine`, `internal/path`, `internal/engine/exec`, `tests/local-e2e` (spine/thin/manifest/omitted-sample), and `tests/cli-valid` pass. Live tests use build tag `live` and are not in this command. It cannot skip for Docker. It is not proof of a live Docker assay.
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

- Verification: Assumption — a change is safe to keep when hermetic `go test ./...` passes, including package tests for `gobble`, `cmd/gobble`, `assets`, `internal/engine`, `internal/path`, `internal/engine/exec`, hermetic `tests/local-e2e`, and hermetic `tests/cli-valid` when those packages change. Agent-operability of live Docker run, inspect, and resume is not proved by first-check. Live is `go test -tags=live` and fails closed without Docker. Live scenario assays are `tests/local-e2e` (run-local, `assets.WGS()`, samplesheet RNA through DEG, samplesheet Methyl through extract; API and CLI occupy/release/resume passed). Remaining package-local live stays `assets` tool proofs and `internal/engine/docker_live_test.go` and passed with Docker. First-check remains `go test ./...` without the live tag. Unrun live tests are not evidence.
- Build risk: Assumption — the part most likely to be wrong is the pipeline model: whether it can express modules, branch, and merge so an agent can plan, run, and resume without a DSL. Early evidence is the synthetic workflow-case pipeline, then WGS end-to-end on a small dataset.
- Source: `verification`, `build-risk`

## Change path

### Gobble

- Statement: Assumption — the solo maintainer changes a topic branch, runs `go test ./...` locally, and commits. There is no named CI, review, or release path yet.
- Source: `change-path`

## Open questions

| Id | Question | What would resolve it |
|---|---|---|
| cache-inputs | Which long-term inputs participate in reuse? The first-horizon workspace rule is dest and input content hashes plus env digest and image digest or resolved executable SHA; cheap keys are diagnostic; cross-workspace cache is excluded. | An accepted long-term cache fingerprint rule |
| retention-deletion | How long is run state kept, and what deletes it? | An accepted retention policy |
| processing-model | Do pipeline results arrive live, in batches, or on demand? | A recorded processing model |
| dependency-unavailable | What does the consumer see while Docker is down? | A recorded Docker-down status shape |

- Source: open topic ids
