# Gobble — Stop and Recover a Run

## Purpose and actors

Agents and human operators can inspect, stop, and resume local analysis using
structured state and reusable outputs. Recovery preserves workspace authority,
exclusive ownership, software identity, and backend uncertainty.

## Current public behavior

Use `gobble stop --workspace DIR` to request termination of active work, then
`gobble resume PACKAGE --workspace DIR` to reconcile and continue. The common
container workflow uses short Compose commands for Stop/Inspect and a detached
controller for Resume. A separate Release is unnecessary for routine recovery.

| Operation or result | Meaning |
|---|---|
| Stop | Writes a durable request addressed to the current owner lease; never signals an unproved PID |
| Stop requested | The caller stopped waiting before settlement; the request remains durable |
| Stop settled | Owned work is proved stopped; valid completed results remain |
| Run stopping / stopped | Settlement is in progress / cancellation has completed |
| Run interrupted | The controller died or backend disposition requires reconciliation |
| Resume | Holds one continuous run lock across reconciliation and acquiring the new owner lease |
| Release | Lower-level reconciliation and occupancy release, subject to owner/liveness gates; not data deletion |

Repeated Stop is safe. A delayed request for an old lease cannot stop a new
Resume. A live scheduler prevents a second owner. An interrupted API context
can also cancel its own Run/Resume; that is distinct from disconnecting a
read-only monitor or closing an Agent after a detached launch.

## Reuse and change

Resume revalidates the graph and classifies identity/graph changes. Successful
reuse requires matching reserved task identity, command or script, parameters,
environment digest, runtime software identity, staged-input fingerprints, and
published-destination checksums. Missing proof is a reuse miss. Affected work
and its downstream dependencies rerun. Resume reevaluates When conditions.

Unfinished work receives a new attempt. Resume does not restore process memory
or promise automatic execution after Docker or computer restart. Changing the
Docker daemon or forcing a runtime lock is not a supported migration.

## Uncertainty and settlement

Mixed control revisions, incompatible identity, or unproved backend state
block unsafe mutation. An unresolved Docker identity remains unknown-backend;
restore access to the recorded daemon before trying recovery again. Never
signal or adopt an unproved PID.

A Docker task with proved stop and exit status can retain its runtime ID when
final log collection or container removal fails. Later reconciliation retries
those terminal actions without treating the task as running again. Unproved
process work with published outputs can remain published-unfinalized; incomplete
work must be retried according to the engine's recovery decision.

Recovery and Release preserve controls and artifacts. There is no general
public Retry, Clean, process checkpoint, or automatic retention policy.

## Application extension

The planned application separates request acceptance, execution outcome,
connection freshness, and backend observability. It adds durable operation
identities, request deduplication, and resume impact preview through one shared
API. Those contracts are proposed in
[Application and monitoring](../architecture/application.md).

The integrated agent's turn interruption, sign-out, usage limit, or crash does
not request pipeline Stop. Desktop monitoring and direct Stop/Resume remain
available independently of provider authentication. Agent tool calls and direct
UI actions share operation identity and duplicate-request protection.

A finish-active-tasks-and-wait action remains a separate future design. It is
not current Stop, Docker pause, or a process-memory checkpoint.

## Evidence and references

Current Linux Docker installation tests exercise Stop, repeated Stop,
controller interruption, duplicate ownership refusal, and Resume. WGS and
RNA-seq installed runs check actual outputs and unchanged-work reuse. These do
not establish real Windows/macOS Docker Desktop restart acceptance.

See the [runtime guide](../../../../../../distribution/runtime/README.md) and
[operations recovery](../../../../../../docs/operations.md#recovery).
