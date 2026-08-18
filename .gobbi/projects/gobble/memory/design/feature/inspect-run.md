# inspect-run — Feature

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. A refused use is not a feature file.

Open Actors, Scope, or Behavior on a feature that can sit in the first horizon
is blocking. Ask before leaving those Open.

## Purpose

- Statement: Let an agent or human operator read structured run and task state, events, errors, logs, timing, DAG, and artifact lineage without changing the run.
- Source: `core-tasks`

## Actors

- Actor: Agent operator
- Role: Reads structured run and task state. Does not change the run by inspecting.
- Actor: Human operator
- Role: Same as the agent operator.
- Source: `task-actors`

## Scope

- In scope: Structured run and task state, events, errors, stdout/stderr, timing, DAG, and artifact lineage through the library, and through the CLI at first-horizon exit.
- Out of scope: A GUI dashboard; human-only log archaeology as the primary interface.
- Source: `task-scope`, `refused-use`

## Behavior

### Normal

- Read current run and task state as JSON or JSONL.
- Source: `task-behavior`

### Alternate

- Fetch logs and events for one named task.
- Source: `task-behavior`

### Invalid

- Requiring a GUI, or treating unstructured logs as the only status channel. An unknown run or task id returns a structured not-found error.
- Source: `task-behavior`

## Failure / Recovery

### Failure

- The consumer sees a structured not-found or read error. Inspect does not mutate state.
- Source: `failure-recovery`

### Recovery

- Correct the run or task id, or wait until state exists. Use recover-run to change the run.
- Source: `failure-recovery`

## Structure

### Parts

- Statement: Read path over persisted run state and logs. This feature does not change the run.
- Source: `shape`

### Data

- Statement: Reads run state, logs, and lineage. Does not change them. Authoritative source is the run workspace.
- Source: `data`

### Interfaces

- Statement: Library inspect operations. CLI for the same operations is required at first-horizon exit. CLI command names stay Open (`invocation-contract`). Default response encoding is JSON or JSONL, not a pipeline language.
- Current: `Inspect(workspace, view, instance)` is the shipped read-only library verb. Views are `run`, `instances`, `errors`, `logs`, `timing`, `DAG`, `lineage`, `remaining`, and `reuse`. `run`, `errors`, `logs`, `timing`, `DAG`, and `lineage` return one JSON object. `instances`, `remaining`, and `reuse` return JSONL. Empty instance selects every reserved identity. Logs are pointers plus an optional bounded tail. Remaining is classified on all latest attempts, then instance-filtered for emit. Inspect does not occupy, create, or rewrite control files. There is no CLI.
- Source: `interfaces`

## Constraints and qualities

- Statement: Structured and explainable by default. Color-only status is assumed insufficient. Agent-operability wins.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| visual-principles | Which few visual principles must stay recognizable if human text is shown? | no | A recorded CLI visual rule |
| invocation-contract | Which inspect CLI names must stay compatible? Names are not locked. | no | An accepted CLI contract |

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
- `visual-principles`
- `invocation-contract`
- Source: the topic ids actually cited
