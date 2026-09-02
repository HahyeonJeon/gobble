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

Run and Resume require a usable local Docker client and a reachable daemon. The
engine invokes Docker with a client environment containing only
`PATH=/usr/bin:/bin`. The command `env -i PATH=/usr/bin:/bin docker info` must
succeed for the local user running Gobble. The task-container `--network=none`
flag is applied by `docker run`. It does not disable registry access used by the
Docker client before container launch.

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
gate does not replace the backend-state gate in [Recovery](#recovery).

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

Use **Inspect → Release → Resume** after a contained failure, cancellation, or
controller death:

1. Run `inspect identity`. It remains available on identity mismatch and shows
   `required`, `have`, `match`, `goos`, `goarch`, and `identity_mode`.
2. With a matching identity, inspect `run`, `errors`, `logs`, `instances`, and
   `remaining`. Use `reuse` and `lineage` to understand selective reruns. If
   `run` reports `"unknown": true`, use `instances` to identify the affected
   executor. An `unknown-backend` Docker identity has unproved backend
   disposition; do not call Release or Resume. Stop and investigate it as
   described below.
3. Correct caller-owned inputs or configuration without deleting state or
   artifacts. Keep the same accepted graph generation unless starting a new
   workspace for a named graph break.
4. Select the Release caller:
   - An embedded Go process that called Run or Resume may call Release after
     that call returns while it still holds occupancy. It does not wait for
     either occupancy field to become `false`.
   - A generic CLI or packed runner executes one command and exits. A separate
     `release` invocation is therefore a different later process. Call it only
     after `occupancy.live` becomes `false`; `occupancy.active` remains `true`
     until Release succeeds.
5. Run Release only after its actor gate passes and Docker disposition is
   proved. Confirm **after successful Release** that `occupancy.active` and
   `occupancy.live` are both `false`.
6. Resume with the same package, compatible graph, sheet meaning, typed config,
   and required execution identity. Inspect `remaining` until it is empty.
7. After Resume returns, inspect again and close its new occupancy through the
   same actor and backend gates. An embedded Go owner may call Release in the
   same process even while both occupancy fields remain `true`. A CLI `resume`
   process exits first, so its later, separate `release` invocation waits for
   `occupancy.live` to become `false`.

Other Inspect views, Release, and Resume refuse an identity mismatch without
mutating the workspace.

If Docker disposition cannot be proved, the affected identity is
`unknown-backend`. Occupancy remains active, Release cannot close it, and Resume
is refused. Stop and investigate the backend; do not relabel it as an ordinary
failed task, delete controls, force occupancy closed, or signal an unproved
PID.

File destinations must be regular files. Every Group member must be regular.
A Tree requires its directory and root manifest. Resume re-evaluates `When`,
input fingerprints, task identity, and published destination checksums. Missing
or changed outputs rerun through the normal dependency graph; no hidden repair
or old-output acceptance occurs.

## Migration

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
