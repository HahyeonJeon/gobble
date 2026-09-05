# Operations

This runbook covers the current trusted-local lifecycle for all five products.
Read [Authoring](authoring.md) first for product-specific inputs. Read
[Provenance](provenance.md) before staging public fixture bytes or changing
default images.

## Boundary

The supported tuple is engineering-only, trusted-local `linux/amd64` execution.
The pipeline source, typed config, local operating-system user, and caller-owned
workspace are trusted. Docker runs selected tasks with network disabled and the
caller's UID/GID, but those flags are conveniences, not a sandbox. A hostile
pipeline or image is outside the threat model.

Docker-backed tasks require a usable local Docker client and reachable daemon.
The engine honors the user's Docker context/configuration and selects a local
Unix-socket endpoint. It records that endpoint and the daemon ID, then keeps
submission and recovery on that endpoint. Run `docker info` as the local user
who will run Gobble. See [Docker execution](docker-execution.md).

The Docker client receives selected host configuration variables; task `Env`
values are passed only to the task container. Its `--network=none` flag is set
at container creation. That flag does not disable registry access used to
prepare the selected image. Process-only pipelines do not require Docker.

Inputs, outputs, task isolates, state, and logs stay in local caller-owned
files. Gobble adds no account, upload, telemetry, remote analysis, object store,
cloud, cluster, or service. Keep secrets out of samplesheets, config values,
commands, parameters, scripts, logs, and manifests.

## Preparation

1. Select one exact product package and graph generation from
   [Products](products.md#product-index).
2. Use an engine command built from the same exact module graph. The current
   product family is unreleased; `v0.1.0` does not contain it.
3. Create a new, empty, exclusive workspace. Gobble will not create it.
4. Stage local regular files at every workspace-relative path named by the
   sheet and typed reference config. Do not put URLs in product sheets.
5. For a ready Tree, stage the complete directory and its regular root
   `.gobble-tree.json`. A directory alone is not a Tree.
6. Compose, validate, and review Plan JSON before Run. Confirm the package,
   graph, commands, images, resources, inputs, and destinations.
7. Make every exact Plan image available to the local daemon. Gobble runs
   `docker image inspect` on each exact image and runs `docker pull` when that
   image is absent. Offline use requires pre-staging every exact
   `registry/repository:tag@sha256:digest` image before disconnecting.

The default package adapter reads `--sample PATH`; without it, the adapter reads
`samplesheet.csv` in the process current directory. The sheet file itself is a
host path. Cells inside it identify workspace-relative product inputs.

## Commands

For a generic agent command:

```sh
gobble compose PACKAGE --sample SHEET
gobble validate PACKAGE --sample SHEET
gobble plan PACKAGE --sample SHEET > plan.json
gobble run PACKAGE --workspace WORKSPACE --cap N --sample SHEET
gobble inspect VIEW --workspace WORKSPACE [--instance ID]
gobble release --workspace WORKSPACE
gobble resume PACKAGE --workspace WORKSPACE --cap N --sample SHEET
```

For a packed runner, omit `PACKAGE`. Omit `--cap`, or pass explicit `0`, to use
the engine default of one. Positive explicit caps are 1 through 64. Negative
values and values above 64 are refused. Success output is JSON or JSONL. Exit 0
means success, 1 means a domain or operational failure, and 2 means invocation
or input-shape failure.

Inspect views are `run`, `instances`, `errors`, `logs`, `timing`, `dag`,
`lineage`, `remaining`, `reuse`, and `identity`. Inspect is read-only and does
not create, occupy, or rewrite a workspace.

Generic Inspect and Release use the installed command identity and need no
package operand or consumer-module working directory. Keep the exact occupying
command available. Packed Inspect and Release use the runner's embedded
identity.

### Failures

Run and Resume first apply start preflight to the graph, workspace, cap, and
other start conditions. A defect there prevents occupancy and task execution.

Image acquisition is separate from start preflight. After occupancy, each
Docker task submission inspects its exact image. A cache miss causes a pull. A
non-cancellation Docker client, daemon, inspection, or pull error fails that
task before its command starts. A task-command failure instead means the
container started and the command exited unsuccessfully. Inspect `run`,
`errors`, and `logs` to tell which occurred. Occupancy remains active after
either contained failure; follow [Recovery](#recovery) rather than starting
another Run.

## Lifecycle

The same seven outcomes apply to every product:

| Outcome | Reader action | Observable contract |
|---|---|---|
| Design | Find the product, typed values, selected stages, outputs, and owner | One assay package owns one graph; unsupported routes have explicit boundaries |
| Build | Load typed data, apply config, compose, validate, and inspect a plan | `Build` and the default `Pipeline` adapter represent the same default graph; defects precede execution |
| Customize | Change a named typed field or safe argv extra | The plan shows the exact effect; defaults stay fresh; data, analysis config, and run controls remain separate |
| Run | Execute the selected path in an exclusive workspace | Required artifacts, reports, state, and provenance are present only after all required members succeed |
| Resume | Inspect, release, then continue a compatible graph | Matching work may be reused; changed and downstream work reruns; identity and backend state fail closed |
| Stop | Cancel Run or Resume through its context | A structured `canceled` defect is recorded; state is inspectable; occupancy stays active |
| Failure | Inspect a contained input, command, backend, or publication defect | Failed work, logs, reusable successes, and blocked descendants remain distinguishable; no fallback or partial fan-in is invented |

Stop and failure are different. Stop is caller cancellation. Failure is a
command, input, backend, or publication defect. Neither adds a public Cancel,
Retry, or assay-specific recovery verb.

## Occupancy

Run and Resume require one execution identity and exclusive occupancy. Their
return—success, contained failure, or cancellation—does not close occupancy.
The process boundary determines when Release is allowed. Passing this actor
gate permits Release to reconcile; it does not guarantee that reconciliation
can close occupancy. See [Recovery](#recovery) for the fail-closed result.

| Release caller | Actor gate | `inspect run` before Release |
|---|---|---|
| Owner process, through an embedded Go caller | The same process called Run or Resume, that call has returned, and the process still holds occupancy | `occupancy.active` and `occupancy.live` may both be `true` |
| Different later process, including every generic CLI or packed-runner `release` invocation | The former owner is no longer live | `occupancy.active` is `true` and `occupancy.live` is `false` |

Do not wait for `occupancy.active` to become `false` before Release. That value
means occupancy is already closed, and another Release returns
`already-released`.

Release always reconciles first. If every backend identity is known stopped, it
closes occupancy. A proved-stopped Docker task may retain a `runtime_id` when
log copy or container removal failed; later reconciliation or Release may
attempt those terminal actions again. This is backend reconciliation, not task
retry or a general cleanup operation. Release does not execute product work,
remove control documents, or delete artifacts.

A different later process must not call Release while `occupancy.live` is
`true`; if it does, it receives `live-occupancy`. A foreign recorded host
yields `foreign-host`. Never signal or adopt an unproved process PID.

## Recovery

Use `stop` to end active work and `resume` to reconcile and continue it:

```sh
gobble stop --workspace runs/analysis
gobble inspect run --workspace runs/analysis
gobble resume . --workspace runs/analysis
```

Stop writes a durable request addressed to the current owner lease. It never
signals an unproved PID. The owner stops admitting tasks, cancels active work,
collects available logs, and records the outcome. A delayed request for a prior
lease cannot stop a new Resume. Repeated Stop is safe.

| Result / state | Meaning and next action |
|---|---|
| Stop `settled` | Owned work is known stopped; successful results remain. |
| Stop `requested` | The CLI's 40-second wait ended before settlement. The request remains; inspect or repeat Stop. |
| Run `stopping` | The owner is settling active tasks. |
| Run `stopped` | Cancellation completed; unfinished tasks can be retried. |
| Run `interrupted` | The owner died or backend state could not be established. Resume reconciles before doing new work. |
| `backend: recovery-required` / `unknown-backend` | Restore access to the recorded Docker daemon, then retry Resume. No task launches until reconciliation proves it is safe. |
| `occupied-workspace` | A scheduler still holds the run lock. Use Watch or Stop; no second owner starts. |

Resume holds one continuous run lock across reconciliation and acquiring its
new lease. A separate Release command is unnecessary for routine recovery.
Completed work is reused only after checking task identity, inputs, and output
checksums. Unfinished or changed tasks run as new attempts; Resume does not
continue inside a bioinformatics tool's memory or promise tool-level checkpoints.

For identity errors, `inspect identity` shows the required and current build.
Use the same pinned runtime or matching direct/packed executable. Other views
and mutations refuse a mismatching identity. A different host or Docker daemon
cannot establish what happened to old jobs. Keep the recorded state intact.

Advanced library callers can still use Release after Run/Resume returns to
reconcile and close the retained run lock. A separate process may Release only
when that owner is gone. Successful explicit Stop closes its own lock; Ctrl+C
and normal Run/Resume retain the existing library lease contract until Release,
auto-reconciling Resume, or process exit. Run outcome and ownership are separate.

For Docker attempts, reconciliation checks the recorded endpoint, daemon ID,
and ownership label, then removes the exact owned container to fence an
outstanding start. An observation error is never interpreted as absence.
See [Docker execution](docker-execution.md) and [checkpoints](checkpoints.md).

File destinations and every Group member must be regular files. A Tree requires
its directory and root manifest. Resume re-evaluates `When`, fingerprints,
identity, and checksums. Missing or changed outputs rerun through the dependency
graph; there is no hidden repair or unchecked acceptance of old output.

## Migration

Run state now uses [atomic checkpoint generations](checkpoints.md). Compatible
legacy flat controls remain readable and are converted on the next allowed
state-changing operation. This storage transition does not bypass engine or
pipeline identity checks.

Migration has two historical boundaries. They must not be conflated:

| Boundary | Preserved behavior | Current workspace rule |
|---|---|---|
| Mechanical move from flat assets into module and pipeline owners | At that checkpoint only, task IDs, edges, commands, images, parameters, resources, inputs, and destinations remained graph-stable | A workspace could cross only that move when its graph was otherwise unchanged |
| Named product lift | WGS became joint germline; RNA became STAR-Salmon; Methyl gained Trim Galore, deduplication, truthful Tree indexing, extraction, and reports | Pre-lift workspaces do **not** resume with current defaults; create a new workspace |
| New products | ATAC and scRNA established their first graph generations | No earlier workspace exists to migrate or resume |

Temporary deprecated source shims `assets.WGS`, `assets.RNASeq`, and
`assets.MethylSeq` now call the lifted product defaults. They preserve source
names only. They do not preserve old graph bytes or make old workspaces
compatible. New code imports `assets/pipelines/wgs`, `rnaseq`, or `methylseq`
directly. ATAC and scRNA have no top-level constructor shims.

For all five current generations, ordinary selective Resume applies only while
the graph and execution identity remain compatible. A changed task ID, edge,
command, image, resource, input, destination, sample membership, or selected
stage can invalidate work. Default graph changes must be announced as
compatibility events. There is no workspace migration, repair, force-resume, or
hidden cleanup command.

## Non-features

The current products do not provide automatic retries, backoff, keep-going,
route fallback, resource escalation, partial cohort joins, cloud bursting,
remote execution, sandboxing, or artifact deletion. Resource exhaustion is a
contained failure; it never changes scientific settings automatically.
