# Checkpoint storage

Gobble publishes a run's plan, task attempts, and run identity as one complete
checkpoint. The public `inspect` commands resolve the current checkpoint;
scripts should use those commands rather than hard-code internal filenames.

| Workspace path | Purpose |
|---|---|
| `.gobble/current.json` | Storage format, current generation ID, and previous generation ID |
| `.gobble/checkpoints/<id>/plan.json` | Recorded graph |
| `.gobble/checkpoints/<id>/tasks.json` | Task attempts and their decisions |
| `.gobble/checkpoints/<id>/run.json` | Run identity, outcome, and ownership |
| `.gobble/checkpoint.lock` | Coordinates checkpoint readers, publication, and retention |
| `.gobble/occupy.lock` | Exclusive execution owner; independent of checkpoint reads |
| `.gobble/tasks/` | Attempt work directories and logs; unaffected by generation retention |

Each generation uses one opaque ID, also recorded in every member's `snapshot`
field. Generation files are immutable after publication. The pointer's storage
format is `1`; the JSON document schema remains `2`. These are separate versions.

## Publication and recovery

The writer creates a new generation, writes all three documents, syncs their
contents and directory entries, then atomically replaces and syncs the current
pointer. Readers pin that generation under a shared lock. Retention runs under
the exclusive checkpoint lock and keeps the current and previous generations.
An abandoned, unpublished generation is removed on the next successful commit.

| Interruption | Result |
|---|---|
| Before publishing the pointer | The previous committed generation remains readable |
| After publishing the pointer | The complete new generation is readable |
| Publication returns an error after rename | The new generation is kept because the pointer may already reference it |
| A committed document is missing, corrupt, or outside the workspace | Inspection/recovery reports an error; it does not substitute stale state |

The previous generation is retained for diagnosis. It is **not an automatic
rollback point**: rolling back state cannot roll back a process or container
that has already started. Do not manually edit the pointer to authorize reruns.

Atomic file publication fixes mixed plan/task/run snapshots. It does not yet
close the separate window between a Docker submission and recording its runtime
ID. Durable submission intents and backend identity reconciliation are the next
part of the [v0.2.0 plan](v0.2.0/plan.md). The current public recovery sequence
remains [Inspect → Release → Resume](operations.md#recovery).

## Legacy layout

When no current pointer exists, Gobble can read the old flat `.gobble/plan.json`,
`tasks.json`, and `run.json` layout. Existing schema, snapshot-coherence, and
execution-identity checks still apply. Read-only inspection does not convert a
workspace or create a checkpoint lock.

The next allowed state-changing operation, such as Release, publishes a
generation and its pointer. Original flat files are kept unchanged as migration
evidence and are no longer consulted. Once a pointer exists, malformed or
missing committed state is an error even if valid old flat files remain.

This is a storage-layout transition, **not permission to resume across different
engine builds or pipeline generations**. Identity mismatch is still refused.
Do not use an older binary that cannot read generations on a converted
workspace: the retained flat files are stale and are not a downgrade mechanism.

## Validation

The engine tests terminate a separate writer process without running its cleanup
after each document write, after generation sync, and after pointer publication.
They then inspect, release, and resume through normal engine operations, checking
that already completed work remains on the original attempt and is reused.

Additional tests inject returned storage errors at the same boundaries, read
while generations are pruned, convert legacy controls, and reject damaged or
escaping committed state. These tests exercise process death on the local Linux
filesystem. They do not claim physical power-loss, Windows, network-filesystem,
or real-Docker recovery validation.
