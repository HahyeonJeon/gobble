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

- After a contained failure, resume remaining work using reusable outputs. Remaining work is tasks not yet successful in this run. Reusable outputs follow the dest and input cheap-key rule on run-local until `cache-inputs` is accepted. Affected work is the unmatched task plus its downstream dependents. Topology edits classify as Change instead of `plan-drift`.
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

- Statement: Library retry, resume, cancel, and clean operations. CLI for the same operations is required at first-horizon exit. CLI command names stay Open (`invocation-contract`).
- Current: `Resume(ctx, *Graph, workspace, cap)` occupies a released existing run after re-validation and classifies every reserved identity as Change: Added, Removed, Rewired, Repathed, IdentityChanged, or Unchanged. `plan-drift` is not a Resume result. `DefectPlanDrift` is removed. Wait-only edge change is Repathed. Remaining is unsuccessful latest attempts. Affected is a reuse miss plus downstream (`downstream-of-rerun`). Dest-scope `output-exists` applies only to dests this Resume would publish that are not authorized replace dests. Dest attribution is checksum or producer lineage. Replacement is staged replace plus per-attempt isolate. Script and env persist on the attempt. A dest rename is a reuse miss. `blocked-upstream` is assigned only when a wait producer failed. Reuse identity is reservedIdentity, command or script, params, env, authored image string, and cheap fingerprints. Resources and image digest are not identity. Schema 0 and 1 workspaces are `unsupported-schema` and are not resume-compatible. When `ctx` is done, in-flight work is canceled, identities persist incomplete, occupancy stays active, and the error is `canceled`. If Reconcile finds `runtime_id` still live, Cancel that handle then classify `previous-incomplete` (R2). `Release(workspace)` closes occupancy, marks in-flight instances `incomplete`, and is not deletion. Occupancy is an owner record on `.gobble/run.json`. After Release, `run.json` stays; a later claim uses a lock file plus owner record. Production Release requires the occupying process not live; same-process tests may use a dead-PID helper (R4). There is no public Cancel, named retry, or guarded Clean. There is no CLI.
- Source: `interfaces`

## Constraints and qualities

- Statement: Recoverability wins. Artifacts may be valuable consumer-owned files. Secrets must not appear in logs. No Gobble identity system.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| cache-inputs | Which long-term inputs decide reusable versus affected work? First-horizon cheap-key rule is recorded on run-local. | no | An accepted long-term cache fingerprint rule |
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
