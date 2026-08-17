# compose-pipeline — Feature

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. A refused use is not a feature file.

Open Actors, Scope, or Behavior on a feature that can sit in the first horizon
is blocking. Ask before leaving those Open.

## Purpose

- Statement: Let an agent or human developer express a bioinformatics pipeline in Go with reusable tasks and modules, explicit inputs, outputs, parameters, resources, and environments, and branch and merge.
- Source: `core-tasks`

## Actors

- Actor: Agent author
- Role: Writes or edits library code that composes the pipeline.
- Actor: Human Go developer
- Role: Writes or edits the same library code. Neither actor is a backend operator.
- Source: `task-actors`

## Scope

- In scope: First-horizon implementation of reusable tasks and modules, explicit inputs, outputs, parameters, resources, environments, and branch and merge (fan-out and fan-in) in Go. A container task must declare its Docker image. A local-process task must not require an image. Design must leave room for later scatter/gather, conditionals, and data-dependent dynamic expansion. Compose-time rejection of a cycle, missing input or output, or a DSL-as-source file so the graph cannot become a plan. Validate-plan re-checks those defects and owns conflicting outputs, unsupported backends, dry run, DAG, and plan emission.
- Out of scope: A Gobble DSL, GUI authoring, built-in assay-specific tools, implementing engine-class features in the first horizon, executing or scheduling a run, and emitting a successful plan.
- Source: `task-scope`, `refused-use`

## Behavior

### Normal

- Write a Go pipeline with at least one branch and one merge that the library accepts.
- Source: `task-behavior`

### Alternate

- Modify an existing pipeline and keep the same model.
- Source: `task-behavior`

### Invalid

- A cycle, missing input or output, or a Gobble-specific language file as the source of truth. Rejected composition returns a structured validation error and is not a runnable plan.
- Source: `task-behavior`

## Failure / Recovery

### Failure

- The consumer sees a structured error that names the composition defect.
- Source: `failure-recovery`

### Recovery

- Fix the Go pipeline and compose again. Human translation of raw logs is not the recovery path.
- Source: `failure-recovery`

## Structure

### Parts

- Statement: Implemented by the Go core library pipeline model.
- Source: `shape`

### Data

- Statement: This feature creates pipeline definitions in Go source. It does not write run state. Authoritative source for definitions is the consumer’s source tree.
- Source: `data`

### Interfaces

- Statement: The public Go compose API. The Go API may prove compose before the CLI exists. First-horizon exit still requires the same loop on the CLI. CLI command names stay Open (`invocation-contract`).
- Source: `interfaces`

## Constraints and qualities

- Statement: One maintainer; local first. A simple Go API is required. No proprietary DSL. Agent-operability wins as an outcome.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| public-contract | Which exported compose types must stay compatible? | no | An accepted public Go API |

- Source: open ids used above

## Interview sources

- `core-tasks`
- `task-actors`
- `task-scope`
- `task-behavior`
- `refused-use`
- `failure-recovery`
- `shape`
- `data`
- `interfaces`
- `constraints`
- `quality-priority`
- `public-contract`
- Source: the topic ids actually cited
