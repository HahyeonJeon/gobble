# Gobble — Roadmap

Record project-level direction and local-then-cloud horizons. Do not write
dates, estimates, assignees, sprints, or tasks. Fill each heading, write
`Not applicable — {reason}`, or write
`Open — {what would resolve it}`.

## Direction

Build a local agent-operable core first: Go library, then engine, then CLI. First observable success is an agent composing a modules-branch-merge pipeline, inspecting the plan, running it locally in per-task Docker images, and resuming affected work. An early falsifying test may use the Go API only. First-horizon exit also requires the same loop through structured CLI verbs. Engine-class Nextflow and Snakemake features must stay designable and are implemented later. After the local core, add HPC-ready execution (Slurm), then cloud and service-ready execution, then an agent-native ecosystem. Local and container backends come before Slurm, and Slurm comes before cloud batch or Kubernetes.

- Source: `horizon-direction`, `success-and-stop`

## Current position

- Now: The static-core slice is shipped on `e8b4720f8cb594be2c165960f86b901dfd2315be` in module `github.com/HahyeonJeon/gobble` (`go 1.26`) with `internal/engine`. First-check is `go test ./...` and includes `tests/wgs-e2e`. Public verbs stay Compose, Validate, BuildPlan, and Run. Additive names are `Group`, `Script`, and `Env`. `Run(graph, workspace, cap)` occupies after checks, isolates under `.gobble/`, stages plan-relative paths, applies CPU and memory to Docker and admission, and runs process and Docker tasks. Wait paths come from the plan. Group members stage and publish by name. Proofs are `testdata/run-local/` (`alpine:3.21` plus process) and `tests/wgs-e2e/` (WGS is not a product feature). There is no CLI, no `cmd/`, and no inspect or resume. This session started from `develop` `5fe7b88652c1c19e2fd60dc215a4042af27eb8b4`. Pre-bootstrap baseline remains `a2d561fea2846bc1c55213d66a7025dac980f330`.
- Local or cloud: local
- Bootstrap: run; committed as `3389767462780dc0cfc436de864580959dfbb0ce` after user authorization to replace the existing README.
- Source: `local-or-cloud`, `first-check`, `current-baseline`

## Horizons

Three to six blocks in order. First included horizon is local when cloud is in
scope later. Do not place a feature with Blocking-open Actors, Scope, or
Behavior in the first included horizon.

### Local agent-operable core

- Name and outcome: An agent can compose, validate, plan, run locally in per-task Docker images, inspect, and resume a modules-branch-merge pipeline, including through structured CLI verbs. Proof is a synthetic workflow-case fixture, then WGS end-to-end on a small dataset.
- Included feature ids: compose-pipeline, validate-plan, run-local, inspect-run, recover-run
- Entry condition: Accepted Startup design drafts and a first-check of `go test ./...` on module `github.com/HahyeonJeon/gobble`.
- Exit evidence: An agent completes the local loop on the synthetic fixture and on a small WGS pipeline, using the Go API and the CLI, with at least one real Docker task. Compile or `go test ./...` alone is not exit evidence.
- Deliberately deferred: Scatter/gather, conditionals, dynamic expansion, Podman executor, Slurm, cloud batch, Kubernetes, GUI, a Gobble DSL, fairness, quotas, job arrays, and a component registry.
- Costly decision: One backend-independent pipeline model, scheduler/executor split, per-task Docker images, local-files run workspace, and no proprietary DSL.
- Source: `horizon-direction`

### HPC-ready engine

- Name and outcome: The same pipeline model runs on a first HPC adapter such as Slurm, with shared and node-local storage, queue and resource mapping, and backend reconciliation.
- Included feature ids: none
- Entry condition: Local agent-operable core exit evidence exists.
- Exit evidence: A recorded Slurm run of a Gobble pipeline that can be inspected and resumed without changing pipeline code.
- Deliberately deferred: Cloud batch, Kubernetes, object storage as the primary store, and multi-run service state.
- Costly decision: How Gobble’s scheduler maps onto Slurm without making Slurm the core model.
- Source: `horizon-direction`

### Cloud and service-ready engine

- Name and outcome: The same model runs on a cloud batch or Kubernetes adapter, with object storage and persistent service state for long-running operations.
- Included feature ids: none
- Entry condition: HPC-ready engine exit evidence exists.
- Exit evidence: A recorded cloud or Kubernetes run that uses the same pipeline model and inspect/resume loop.
- Deliberately deferred: A full agent-native catalog and application integration ecosystem.
- Costly decision: Whether a standardized remote API is required to keep the core backend-independent.
- Source: `horizon-direction`

### Agent-native ecosystem

- Name and outcome: Reusable component and pipeline catalogs, capability discovery, plan diffs, automated diagnosis and recovery assistance, and application integration on the same model.
- Included feature ids: none
- Entry condition: Cloud and service-ready engine exit evidence exists.
- Exit evidence: An agent can discover components, apply a plan diff, and recover a modified pipeline through the recorded operations.
- Deliberately deferred: A GUI or desktop product.
- Costly decision: Catalog and compatibility policy.
- Source: `horizon-direction`

## Feature order

Every feature file appears in exactly one horizon or in Not scheduled.

| Id | Horizon or not scheduled |
|---|---|
| compose-pipeline | Local agent-operable core |
| validate-plan | Local agent-operable core |
| run-local | Local agent-operable core |
| inspect-run | Local agent-operable core |
| recover-run | Local agent-operable core |

- Source: Feature index and `feature/<feature>.md`

## Replan and stop

- Replan: The pipeline model cannot leave room for later scatter/gather, conditionals, or dynamic expansion without a breaking rewrite. Or the CLI cannot express the same loop as the Go API.
- Stop: After the local core exists, agents still cannot validate, run, diagnose, and resume without a human translating logs. Or a Gobble DSL becomes required.
- Source: `success-and-stop`

## Not scheduled

| Id | Why not scheduled |
|---|---|
| none | Every Feature index id is placed in Local agent-operable core. |

- Source: Feature index rows not placed in a horizon

## Open questions

| Id | Question | What would resolve it |
|---|---|---|
| cache-inputs | Which long-term inputs participate in reuse? A temporary first-horizon rule is in force: reuse only when task identity, declared command or image, declared parameters, and recorded input path plus content fingerprints match and published outputs still exist; otherwise treat the task and its downstream dependents as affected. Remaining work is tasks not yet successful in this run. Changed work is tasks whose reuse check failed. | An accepted long-term cache fingerprint rule |
| later-features | Which engine-class features enter HPC, cloud, or ecosystem horizons? | A named later-horizon feature list |

- Source: open placement or rule items
