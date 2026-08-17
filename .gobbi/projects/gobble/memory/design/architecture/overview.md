# Gobble — Overview

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`.

## Purpose

- Statement: Gobble must give agents and human developers a Go pipeline engine they can use to design, validate, plan, schedule, execute, observe, modify, recover, and resume bioinformatics pipelines. First consumers are agents such as Claude Code and Codex, with human developers using the same model. This is the right time because coding agents can now write, run, and recover pipelines well enough to be first-class authors and operators, and existing engines were not designed for that use.
- Source: `purpose`, `why-now`

## Problem

- Statement: No dated incident, cost, or frequency is recorded. The named cause is that Nextflow-, Snakemake-, and Cromwell-class engines are not designed for agent authoring, structured control, and safe recovery, so agents cannot reliably complete that lifecycle today.
- Source: `problem-evidence`

## Outcome

- Statement: Even if every product surface is rebuilt, Gobble must keep one backend-independent pipeline model, a reusable Go core library, no proprietary DSL as a prerequisite, agent-readable structured control, and recoverability after failure or interruption as a normal lifecycle path. First observable success is an agent completing a local loop: write or edit a Go pipeline with modules, branch, and merge; inspect the plan; run it locally in containers; and resume only affected work. First-horizon exit also requires the same loop through structured CLI verbs. An early falsifying test may use the Go API only before the CLI exists. Engine-class Nextflow and Snakemake features must be considered in design and implemented later. A compiling module, passing unit tests, a drawn DAG, or feature count without that agent loop is not success. Evidence to end the project is that after the local core exists, agents still cannot validate, run, diagnose, and resume without a human translating logs. The current run-local work lives on `feat/run-local`. The public Go API can compose, validate, plan, and run; it does not inspect or resume. This session’s immutable base is `5e2c295c3b39733c2c279616bc95130226ffc5f7` on `develop`. The pre-bootstrap baseline is `a2d561fea2846bc1c55213d66a7025dac980f330`.
- Source: `durable-outcome`, `success-and-stop`, `current-baseline`, `first-use`

## Scope and non-goals

- In scope: A local agent-operable core built as one product named Gobble, in this order: Go library, then engine, then CLI. First-horizon compose covers modules, branch, and merge. First-horizon operate covers validate, plan, local Docker and process execution, inspect, and recover. Proof pipelines are a small synthetic workflow-case fixture, then WGS end-to-end on a small dataset. Later planning must respect local agent-operable core, then HPC-ready execution, then cloud and service-ready execution, then an agent-native ecosystem. Local and container backends come before Slurm, and Slurm comes before cloud batch or Kubernetes.
- Non-goals: A proprietary Gobble DSL; reproducing Nextflow or Snakemake syntax; a GUI, desktop, or web application; built-in support for particular bioinformatics tools as product features; settling concrete Go APIs or backend algorithms in this Startup pass; HPC, cloud, object storage, and a component registry in the first horizon; implementing scatter/gather, conditionals, and dynamic expansion in the first horizon.
- Source: `boundary`, `horizon-direction`, `products`, `success-and-stop`

## Product inventory

- Products:
  - Gobble
- Source: `products`

## Constraints

- Statement: One maintainer now. First horizon is local machines and containers. No budget or deadline is recorded. License is unset. Agent-operability and recoverability win as outcomes. A simple Go API is the required means and is not traded against those outcomes. Feature breadth versus Nextflow or Snakemake may degrade first, then scheduler sophistication beyond CPU, memory, and concurrency. Human-only convenience that does not help agents may degrade before API simplicity.
- Source: `constraints`, `quality-priority`

## Authority and maintenance

- Statement: HahyeonJeon decides and maintains. If that person is unavailable, the project pauses. Essential knowledge today lives in one head plus `docs/gobble-draft.md` and the accepted interview. After Startup, Design Memory is the continuity store.
- Source: `authority-continuity`

## Vocabulary

| Term | Meaning | Source |
|---|---|---|
| Gobble | The one independently useful product: Go library, then engine, then CLI. | `products` |
| Agent-operable | An agent can compose, validate, run, inspect, and resume from structured state without a human translating logs. Capability discovery is an ecosystem-horizon outcome. | `purpose`, `success-and-stop` |
| Pipeline model | Shared backend-independent concepts for task, pipeline, input, output, resource, and environment. | `durable-outcome`, `shape` |
| Task | An independently runnable unit. A container task must declare its Docker image. A local-process task must not require an image. | `shape` |
| Run workspace | Local-files directory that is authoritative for one run’s state, artifacts, and logs. | `data`, `state-authority` |
| Engine-class features | Nextflow- and Snakemake-class capabilities such as scatter/gather, conditionals, and dynamic expansion. Designed for now, implemented later. | `success-and-stop` |

- Source: each row's topic-id

## Open questions

| Id | Question | What would resolve it |
|---|---|---|
| none | No Branch 1 topic is Open. Interview assumptions remain: no measured incident, and a later-horizon engine-class feature list. | Named evidence or a later-horizon list |

- Source: open topic ids
