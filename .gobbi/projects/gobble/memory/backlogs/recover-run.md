# recover-run backlog

## Dest-rename successful publish proof

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Prove a dest-rename Resume publishes the new path after a reuse miss.

**Why backlogged:** Ready inspect/resume treats dest rename as a reuse miss. This session still did not ship a successful new-dest publish proof.

**Context:** Public dest-rename Resume may still fail to publish if the command writes the old path. `TestResumeDestRenameDoesNotReuse` proves the miss. A successful new dest needs a command that writes the new path.

## Group and branch-merge resume e2e

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Prove Resume on a Group and branch-merge pipeline.

**Why backlogged:** Session proofs were occupy-release, two-instance isolation, reuse decisions, graph-diff Change tables, and the WGS assay.

**Context:** Group members already stage and publish by name on Run. Compose covers a branch-merge pipeline. There is no Group or branch-merge Resume e2e.

## Guarded clean and retention

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Add explicit guarded clean and a retention policy for run state and artifacts.

**Why backlogged:** Designed and not shipped. `retention-deletion` stays open.

**Context:** `Release` is not deletion. Isolate-keep. Tree remains correct without this verb. Default designed scope is isolate directories under `.gobble/tasks`. Dry-run a manifest before delete. Never unguarded dest delete. Refuse while occupied.

## Live cancel

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Add a public Cancel verb, or cancel an in-flight run without a live caller context.

**Why backlogged:** This session shipped context cancel on Run and Resume. Occupancy stays until Release. A workspace Cancel without a live ctx was excluded.

**Context:** `Run(ctx)` and `Resume(ctx)` cancel in-flight work, persist incomplete, return `canceled`, and leave occupancy active. Executor.Cancel is internal. Process-group kill and `docker kill` are the adapters. Hermetic cancel tests exist. There is no live Docker ctx-cancel assay and no public Cancel.

## Named retry with backoff

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Retry one named task with backoff.

**Why backlogged:** Resume covers remaining unsuccessful work. Named retry was deferred.

**Context:** There is no forced reuse reason this session. A later named retry would need one.

## Transitive blocked-upstream

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Assign `blocked-upstream` through a chain of wait producers that never started.

**Why backlogged:** Shipped `blocked-upstream` only when a wait producer failed this Resume.

**Context:** A `rerun` that never started because a wait-path producer ended unsuccessful becomes `blocked-upstream` and gets no new attempt. Transitive never-started producers are not that case.
