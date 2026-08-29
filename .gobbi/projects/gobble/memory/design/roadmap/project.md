# Gobble — Roadmap

Record project-level direction and local-then-cloud horizons. Do not write
dates, estimates, assignees, sprints, or tasks. Fill each heading, write
`Not applicable — {reason}`, or write
`Open — {what would resolve it}`.

## Direction

Build a local agent-operable core first: Go library, then engine, then CLI. First observable success is an agent composing a modules-branch-merge pipeline, inspecting the plan, running it locally in per-task Docker images, and resuming affected work. An early falsifying test may use the Go API only. First-horizon exit also requires the same loop through structured CLI verbs. Engine-class Scatter, Gather, and When are authored operators with runtime instance and shard occupancy. Plan-time Document expansion of reservedIdentity stays deferred. After the local core, add HPC-ready execution (Slurm), then cloud and service-ready execution, then an agent-native ecosystem. Local and container backends come before Slurm, and Slurm comes before cloud batch or Kubernetes.

- Source: `horizon-direction`, `success-and-stop`

## Current position

- Now: Gobble is a pre-1.0 trusted-local `linux/amd64` preview licensed under MIT. The supported contract is the seven library and CLI verbs; Module, Branch, Merge, Scatter, Gather, and When; PathSpec and File, Group, or Tree binds; explicit-path samplesheet parse and load; and structured errors. Other exports remain provisional. Agents use the library and generic command through an explicit local path pin. Humans receive one packed runner for one pipeline and need no Go at run time. Graph verbs and pack require `go` on PATH and reject consumer `internal/` packages. Install identity handshakes before `Pipeline()` and before workspace mutation. Run workspaces are caller-owned and exclusive. Stage and publish are copy-only. Recovery is Inspect, later-process Release without an unproved PID signal, then Resume remaining work. Dest-complete process work persists `published-unfinalized`; incomplete process work reruns. A proved-stopped Docker task may retain a RuntimeID for cleanup retry without wedging occupancy; unproved Docker remains `unknown-backend` and blocks close. Hermetic `go test ./...` remains the first check. First-horizon installed-path exit is proved for the local-pin agent and packed human families by `go test -tags=live ./tests/install-e2e`; no exact-tag release is claimed. The module remains `github.com/HahyeonJeon/gobble` at Go 1.26 and SchemaVersion remains 2. Future pre-1.0 publication uses one immutable repository-root `v0.x.y` tag for the root module, library, and `cmd/gobble`, with no `/v0` path, retag, or supported `@latest`. Patch means no intended Go API, CLI protocol, workspace-schema, or recovery break. Minor may add features and may declare a pre-1.0 break, naming Go API, CLI, workspace, and recovery effects. The first number is Deferred; `v0.x.y-rc.1` is used only if later requested. This session creates no tag, push, GitHub Release, or other remote publication. A future tag still needs release notes, MIT bytes, an exact commit, and installed external-consumer proof.
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
- Deliberately deferred: Plan-time Document expansion of reservedIdentity, Podman executor, Slurm, cloud batch, Kubernetes, GUI, a Gobble DSL, fairness, quotas, job arrays, and a component registry.
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
| assets | Not scheduled |

- Source: Feature index and `feature/<feature>.md`

## Replan and stop

- Replan: Later scatter cannot attach as plan-time Document expansion of `reservedIdentity` without a breaking rewrite. Or the CLI cannot express the same loop as the Go API.
- Stop: After the local core exists, agents still cannot validate, run, diagnose, and resume without a human translating logs. Or a Gobble DSL becomes required.
- Source: `success-and-stop`

## Not scheduled

| Id | Why not scheduled |
|---|---|
| assets | First-party proofs in package `assets`, not a core-task or product feature. |

- Source: Feature files not placed in a horizon

## Open questions

| Id | Question | What would resolve it |
|---|---|---|
| cache-inputs | Which long-term inputs participate in reuse? First-horizon workspace reuse uses reservedIdentity, command or script, params, env digest, authored image plus image digest or resolved executable SHA, and content hashes of dests and inputs. Cheap keys are diagnostic. Cross-workspace cache is excluded. Remaining work is unsuccessful latest attempts excluding skipped. Changed work is tasks whose reuse check failed. | An accepted long-term cache fingerprint rule |
| later-features | Which remaining engine-class features enter HPC, cloud, or ecosystem horizons beyond authored Scatter, Gather, and When? | A named later-horizon feature list |

- Source: open placement or rule items
