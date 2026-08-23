# recover-run — Feature

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. A refused use is not a feature file.

Open Actors, Scope, or Behavior on a feature that can sit in the first horizon
is blocking. Ask before leaving those Open.

## Purpose

- Statement: Let an agent or human operator inspect structured run state, release occupancy, resume remaining work, or cancel in-flight work through the Run/Resume context after a contained failure, using structured state and reusable outputs.
- Source: `core-tasks`

## Actors

- Actor: Agent operator
- Role: Inspects, releases occupancy, resumes remaining work, or cancels through context from structured state without a person translating logs.
- Actor: Human operator
- Role: Same as the agent operator.
- Source: `task-actors`

## Scope

- In scope: Inspect, release occupancy, resume remaining work using reusable outputs, and context cancel on Run/Resume.
- Out of scope: Named retry, backoff, public cancel, and guarded cleanup; unguarded artifact deletion; resume onto HPC or cloud; silent skip of validation after the pipeline changed.
- Source: `task-scope`, `refused-use`

## Behavior

### Normal

- After a contained failure, resume remaining work using reusable outputs. Remaining work is unsuccessful latest attempts excluding skipped. Reusable outputs follow the dest and input content-hash rule on run-local until `cache-inputs` is accepted. Affected work is the unmatched task plus its downstream dependents. Topology edits classify as Change instead of `plan-drift`.
- Source: `task-behavior`

### Alternate

- Cancel in-flight work through a done Run/Resume context, then Inspect, Release, and Resume remaining work. Occupying-process Release is live-owner Release. A later process may invoke Release; while the occupying process is live that is `live-occupancy`.
- Source: `task-behavior`

### Invalid

- Unguarded cleanup that deletes artifacts, resume that silently skips validation after the pipeline changed, or a public Cancel, named retry, or guarded Clean. A destructive clean or an invalid resume target is rejected with a structured error and no deletion.
- Source: `task-behavior`

## Failure / Recovery

### Failure

- The consumer sees structured task and run state, an error that names the failed unit, and which outputs remain reusable.
- Source: `failure-recovery`

### Recovery

- Inspect, Release occupancy, then Resume remaining work, or inspect-then-modify the Go pipeline and Resume. Deletion is not a recover-run verb. Consumers may delete their own files. Guarded clean stays designed and not shipped.
- Source: `failure-recovery`

## Structure

### Parts

- Statement: Engine control over persisted run state. Release closes occupancy. Cleanup is not a public verb.
- Source: `shape`

### Data

- Statement: Reads run state and artifacts and updates task and run state. Does not delete artifacts. The run workspace is authoritative.
- Source: `data`

### Interfaces

- Statement: Library inspect, release, resume, and context-cancel operations. CLI for the same operations is required at first-horizon exit. Shipped `gobble resume --workspace DIR [--cap N] [--sample PATH] [PKG]` and `gobble release --workspace DIR`; no `cancel` verb; SIGINT/SIGTERM cancel ctx; occupancy stays; contract in [architecture/system.md](../architecture/system.md) Interfaces Current.
- Current: `Resume(ctx, *Graph, workspace, cap)` occupies a released existing run after re-validation and classifies every reserved identity as Change: Added, Removed, Rewired, Repathed, IdentityChanged, or Unchanged. `plan-drift` is not a Resume result. `DefectPlanDrift` is removed. Wait-only edge change is Repathed. Remaining is unsuccessful latest attempts excluding skipped. Affected is a reuse miss plus downstream (`downstream-of-rerun`). Dest-scope `output-exists` applies only to dests this Resume would publish that are not authorized replace dests. Dest attribution is checksum or producer lineage. Replacement is staged replace plus per-attempt isolate. Script persists on the attempt. Env values are omitted from Inspect; env digest is identity. A dest rename is a reuse miss. `blocked-upstream` is assigned only when a wait producer failed. Reuse identity is reservedIdentity, command or script, params, env digest, authored image plus image digest for docker or resolved executable SHA for process, and content hashes of consumed inputs and published dests. Cheap keys are diagnostic. Resources are not identity. Missing digest or hash is a reuse miss. SchemaVersion stays 2. Schema 0 and 1, and PID-only schema-2 occupancy missing a lease, are `unsupported-schema` with no migration. When `ctx` is done, in-flight work is asked to stop. Known stopped identities persist `incomplete`, occupancy stays active, and the error is `canceled`. If stop cannot be proved before the Engine bound, those identities are `unknown` (`unknown-backend`) and occupancy stays. If Reconcile finds `runtime_id` still live, Cancel that handle then classify `previous-incomplete`. Occupancy is exclusive execution authority on `.gobble/run.json` from successful occupy until close. The occupying process is the owner. Private liveness is a held occupancy flock and lease. Inspect `live` is that liveness. PID and host remain diagnostic Inspect fields. There is no public occupancy token. `Release(workspace)` always Reconciles first, then closes occupancy when leftovers are known stopped, and is not deletion. Occupying-process Release is live-owner Release. A later process may invoke Release; while the occupying process is live that is `live-occupancy`. Occupancy does not close and Resume does not occupy while any identity remains unknown. After Release, `run.json` stays. A dead-PID helper is not recovery authority. When Resume re-evaluates a When to skip, identities with a known terminal status become skipped and are not rerun. Unknown stays unknown. There is no public Cancel, named retry, or guarded Clean. Occupy / remaining empty / occupied second Run / occupying-process or later-process Release / Resume / reuse `reused-identity-matched` passed live on run-local, WGS, RNA, and Methyl through API and CLI in `tests/local-e2e`. Inspect and release still do not compose and do not take `--sample`. Resume still takes `--sample` because it recomposes. Dest-rename reuse-miss and Group/branch-merge resume remain deferred. Shipped `gobble resume --workspace DIR [--cap N] [--sample PATH] [PKG]` and `gobble release --workspace DIR`; no `cancel` verb; SIGINT/SIGTERM cancel ctx; occupancy stays; contract in [architecture/system.md](../architecture/system.md) Interfaces Current.
- Source: `interfaces`

## Constraints and qualities

- Statement: Recoverability wins. Artifacts may be valuable consumer-owned files. Secrets must not appear in logs. No Gobble identity system.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| cache-inputs | Which long-term inputs decide reusable versus affected work? First-horizon content-hash rule is recorded on run-local. | no | An accepted long-term cache fingerprint rule |
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
