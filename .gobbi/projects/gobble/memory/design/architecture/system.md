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

- Statement: Major parts are the Go core library (pipeline model: task, pipeline, input, output, resource, environment), engine services (validator, planner, scheduler), executors (local process and Docker), and interfaces (Go API, then CLI). The seam between scheduler and executors is required. The first-horizon scheduler is DAG- and resource-aware: per-task CPU and memory, and a run-level concurrency cap. Fairness, quotas, and job arrays wait. Each task declares its own environment, including its Docker image, and must be runnable by itself. Construction order is library, then engine, then CLI.
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

- Statement: Pipeline definitions live in Go source. First-horizon run state, cache or reuse decisions, artifacts, logs, and provenance live as local files in the run workspace. A run creates isolated task work directories, stages inputs, writes intermediates, publishes outputs on success, and keeps enough state to resume. Secrets must not be persisted in logs or metadata. Retention policy is unset. A later store may replace the file implementation.
- Authoritative source: the run workspace on the local filesystem for that run
- Source: `data`, `data-lifecycle`

## Interfaces

### Gobble

- Statement: Internal seams are library to engine, scheduler to executor, and engine to state and artifact files. External seams are the public Go API first and the CLI second. Later agent APIs share the same model. JSON or YAML may appear later as interchange, not as a second language.
- Source: `interfaces`

## Stack

Local-stack rows and `## First check` are the only Bootstrap inputs from design.

### Gobble

| Product | Entry | Category | Responsibility | Local or cloud | Version policy | Rationale | Constraints |
|---|---|---|---|---|---|---|---|
| Gobble | Go | language | Author and build the library, engine, and CLI | local | 1.26 or newer | Forced language (`stack`) | Module path `github.com/HahyeonJeon/gobble` |
| Gobble | Go toolchain | toolchain | Compile and run `go test ./...` | local | 1.26 or newer | First-check and change path (`first-check`, `stack`) | Missing Go blocks build and import (`dependency-failure`) |
| Gobble | Docker | runtime | Run each container task in its declared image | local | unknown | First-horizon container runtime (`stack`); required for first success | Missing Docker blocks container tasks and first-horizon success (`dependency-failure`) |
| Gobble | Local files | datastore | Hold run state, cache decisions, artifacts, and logs | local | none | First-horizon store; later stores replaceable (`stack`, `data`) | Run workspace is authoritative (`state-authority`) |
| Gobble | Podman | runtime | Later local container executor | local | unknown | Replaceable executor after Docker (`dependency-exit`); not first-horizon | Not required to exit first horizon |
| Gobble | Slurm | runtime | First HPC backend | later-cloud | unknown | After local and container backends (`horizon-direction`) | Must not force the core model |
| Gobble | Cloud batch | runtime | Later cloud backend | later-cloud | unknown | After Slurm (`horizon-direction`) | Must not force the core model |
| Gobble | Kubernetes | runtime | Later orchestrated backend | later-cloud | unknown | After Slurm (`horizon-direction`) | Must not force the core model |

- Source: `stack`, `local-or-cloud`, `horizon-direction`

## First check

Do not use a no-op, `echo`, or any command that ignores the local toolchain.

### Gobble

- Command: `go test ./...`
- What it proves: Go 1.26 or newer is installed and the module `github.com/HahyeonJeon/gobble` builds and its tests run. It is not proof of agent-operability or Docker.
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

- Verification: Inherited — A change is safe to keep when `go test ./...` passes and, once tests exist, package tests for touched packages pass. Agent-operability is not proved by compile alone.
- Build risk: Inherited — The part most likely to be wrong is the pipeline model: whether it can express modules, branch, and merge so an agent can plan, run, and resume without a DSL. Early evidence is the synthetic workflow-case pipeline, then WGS end-to-end on a small dataset.
- Source: `verification`, `build-risk`

## Change path

### Gobble

- Statement: Inherited — The solo maintainer changes a topic branch, runs `go test ./...` locally, and commits. There is no named CI, review, or release path yet.
- Source: `change-path`

## Open questions

| Id | Question | What would resolve it |
|---|---|---|
| cache-inputs | Which inputs participate in reuse and “affected work”? | An accepted cache fingerprint rule |
| retention-deletion | How long is run state kept, and what deletes it? | An accepted retention policy |
| processing-model | Do pipeline results arrive live, in batches, or on demand? | A recorded processing model |
| public-contract | Which public types or functions must current callers keep? | Project Design public API after first library surface |
| invocation-contract | Which command names, inputs, and outputs must stay compatible? | Project Design CLI contract after first CLI |
| dependency-unavailable | What does the consumer see while Docker is down? | A recorded Docker-down status shape |

- Source: open topic ids
