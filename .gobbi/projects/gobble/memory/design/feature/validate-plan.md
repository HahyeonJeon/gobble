# validate-plan — Feature

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. A refused use is not a feature file.

Open Actors, Scope, or Behavior on a feature that can sit in the first horizon
is blocking. Ask before leaving those Open.

## Purpose

- Statement: Re-check missing inputs, cycles, conflicting outputs, and unsupported backend requirements before execution, and expose a dry run, DAG, and plan. Compose-time rejection of a cycle or missing I/O does not replace this check.
- Source: `core-tasks`

## Actors

- Actor: Agent operator
- Role: Asks Gobble to validate and plan. Does not execute tasks in this feature.
- Actor: Human operator
- Role: Same as the agent operator.
- Source: `task-actors`

## Scope

- In scope: Re-check missing inputs, cycles, conflicting outputs, and unsupported backend requirements; expose dry run, DAG, and plan through the library, and through the CLI at first-horizon exit.
- Out of scope: Executing tasks; submitting to HPC or cloud; treating a drawn DAG as agent-operable success.
- Source: `task-scope`, `refused-use`

## Behavior

### Normal

- Validate a composed pipeline, including a dry run that does not execute tasks, and emit a structured plan and DAG.
- Source: `task-behavior`

### Alternate

- Show the impact of a proposed change against the last plan.
- Source: `task-behavior`

### Invalid

- A request to emit a successful plan for a pipeline that still has cycles or missing inputs. The library or CLI returns a structured error listing each defect, not only a human paragraph.
- Source: `task-behavior`

## Failure / Recovery

### Failure

- The consumer sees a structured error that names each defect.
- Source: `failure-recovery`

### Recovery

- Fix the pipeline and validate again. Do not execute a refused plan.
- Source: `failure-recovery`

## Structure

### Parts

- Statement: Implemented by the validator and planner in the engine, called from the library, and from the CLI at first-horizon exit.
- Source: `shape`

### Data

- Statement: Reads pipeline definitions and may write an inspectable plan artifact. Does not execute or publish outputs. Authoritative source for the plan is the emitted plan artifact or structured response.
- Source: `data`

### Interfaces

- Statement: Library validate and plan operations. CLI for the same operations is required at first-horizon exit. CLI command names stay Open (`invocation-contract`). Default response encoding is JSON or JSONL, not a pipeline language.
- Current: `Validate` re-checks compose defects and rejects path conflicts, unsupported backends, and non-finite CPU. `BuildPlan` validates first and returns a `Plan`. Dry run is `BuildPlan`. `WriteTo` is optional and keeps the plan if the writer fails. Plan JSON keys are `pipeline`, `tasks`, and `dag`. The workflow-case golden is `testdata/workflow-case/plan.json`.
- Source: `interfaces`

## Constraints and qualities

- Statement: Structured, explainable, and inspectable before run. Agent-operability wins. Simple Go API is the means.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| invocation-contract | Which validate and plan CLI names must stay compatible? Names are not locked. | no | An accepted CLI contract |

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
- `invocation-contract`
- Source: the topic ids actually cited
