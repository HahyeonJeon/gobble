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
- Current: Recovery is Inspect, Release occupancy, then Resume remaining work. `Resume(ctx, *Graph, workspace, cap, opts ...OccupyOption)` requires one effective `WithIdentity`, re-validates, and classifies each reserved identity as Added, Removed, Rewired, Repathed, IdentityChanged, or Unchanged. `Release(workspace, opts ...OccupyOption)` may omit identity and then binds the current executable. Remaining is unsuccessful latest work excluding skipped and `published-unfinalized`. A reuse miss affects the identity plus downstream work. Succeeded reuse requires reservedIdentity, command or script, params, env digest, Docker image digest or process executable SHA, staged-input fingerprints, and published-destination checksums. Missing digests or hashes miss; Resume re-evaluates `When`. Mixed control snapshot tokens fail Resume and Release. Cancellation leaves occupancy active. A Docker task whose stopped state and exit code were proved may retain a RuntimeID when log copy or container removal fails; later Poll, Reconcile, or Release retries removal. That leftover is terminal, does not reopen `unknown-backend`, and does not keep occupancy active. If disposition remains unproved, `unknown-backend` keeps occupancy active and Resume refuses. A later process must acquire the occupancy lock; a live occupier is `live-occupancy`. Release never signals an unproved PID and performs no PID adoption. Dest-complete unproved process work persists `published-unfinalized`; incomplete process work reruns. Release is not deletion. There is no public Cancel, named retry, repair verb, PID adoption, or guarded Clean. First-horizon remaining-work recovery is proved on API, installed CLI, and packed WGS paths by `go test -tags=live ./tests/install-e2e`. The published agent install is `github.com/HahyeonJeon/gobble@v0.1.0`.
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
