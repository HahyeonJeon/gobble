# recover-run — Feature

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. A refused use is not a feature file.

Open Actors, Scope, or Behavior on a feature that can sit in the first horizon
is blocking. Ask before leaving those Open.

## Purpose

- Statement: Let an agent or human operator retry, resume remaining work, cancel, or guard cleanup after a contained failure, using structured state and reusable outputs.
- Source: `core-tasks`

## Actors

- Actor: Agent operator
- Role: Retries, resumes, cancels, or guards cleanup from structured state without a person translating logs.
- Actor: Human operator
- Role: Same as the agent operator.
- Source: `task-actors`

## Scope

- In scope: Retry, backoff, cancel, resume of remaining work using reusable outputs, and guarded cleanup.
- Out of scope: Unguarded artifact deletion; resume onto HPC or cloud; silent skip of validation after the pipeline changed.
- Source: `task-scope`, `refused-use`

## Behavior

### Normal

- After a contained failure, resume only remaining work using reusable outputs.
- Source: `task-behavior`

### Alternate

- Retry one named task with backoff, or cancel an in-flight local run.
- Source: `task-behavior`

### Invalid

- Unguarded cleanup that deletes artifacts, or resume that silently skips validation after the pipeline changed. A destructive clean or an invalid resume target is rejected with a structured error and no deletion.
- Source: `task-behavior`

## Failure / Recovery

### Failure

- The consumer sees structured task and run state, an error that names the failed unit, and which outputs remain reusable.
- Source: `failure-recovery`

### Recovery

- Retry, resume remaining work, or inspect-then-modify the Go pipeline and resume. Deletion happens only after an explicit guarded clean, or the consumer deleting their own files.
- Source: `failure-recovery`

## Structure

### Parts

- Statement: Engine control over persisted run state, plus executors for retry. Cleanup is guarded at the artifact files.
- Source: `shape`

### Data

- Statement: Reads run state and artifacts, updates task and run state, and may delete artifacts only after an explicit guarded clean. The run workspace is authoritative.
- Source: `data`

### Interfaces

- Statement: Library retry, resume, cancel, and clean operations, then matching CLI verbs.
- Source: `interfaces`

## Constraints and qualities

- Statement: Recoverability wins. Artifacts may be valuable consumer-owned files. Secrets must not appear in logs. No Gobble identity system.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| cache-inputs | Which inputs decide reusable versus affected work? | no | An accepted cache fingerprint rule |
| retention-deletion | How long is run state kept beyond explicit clean? | no | An accepted retention policy |

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
- `cache-inputs`
- `retention-deletion`
- Source: the topic ids actually cited
