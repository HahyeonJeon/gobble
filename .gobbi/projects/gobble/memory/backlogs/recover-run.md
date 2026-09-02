# recover-run backlog

## Dest-rename successful publish proof

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Prove a destination-rename Resume publishes the new path after a reuse miss.

**Why backlogged:** Current tests prove that a renamed destination is Repathed
and is not reused, but do not prove successful publication to the new path by a
command that follows the renamed destination.

**Context:** `TestResumeDestRenameDoesNotReuse` exists at public and engine
boundaries. A complete proof needs a changed command or typed destination
contract that writes the new path and verifies the old destination remains safe.

## Group and branch-merge resume e2e

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Prove public Resume on one Group-producing branch-and-merge pipeline.

**Why backlogged:** Current evidence separately covers Branch/Merge composition,
Group publication and incomplete rerun, and product selective Resume. It does
not combine those contracts in one public end-to-end scenario.

**Context:** Group members stage and publish by name. A partial Group from an
incomplete producer reruns in engine evidence. `Merge(name)` still takes no
branch list; Bind dependencies define fan-in.

## Guarded clean and retention

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Add explicit guarded clean and a retention policy for run state and artifacts.

**Why backlogged:** Release reconciles and closes occupancy but is not deletion.
Retention and guarded deletion remain unset.

**Context:** The caller owns workspace files. Unguarded destination deletion is
prohibited, and no current verb owns artifact retirement.

## Live cancel

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Add a public Cancel verb, or cancel an in-flight run without a live caller context.

**Why backlogged:** Run and Resume support caller-context cancellation. A
workspace Cancel from another process is not part of the public contract.

**Context:** Cancellation is structured and leaves occupancy active. Recovery is
actor-gated Inspect, Release, and Resume. Executor Cancel remains internal.
Unproved process identity must never be signaled or adopted.

## Named retry with backoff

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Retry one named task with backoff.

**Why backlogged:** Resume covers remaining unsuccessful and invalidated work.
No independent named retry policy was accepted.

**Context:** Current task identity and strict dependency fan-in provide no
forced-rerun or backoff semantics independent of Resume.

## Transitive blocked-upstream

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Assign `blocked-upstream` through a chain of wait producers that never started.

**Why backlogged:** Current recovery evidence classifies direct descendants of
rerun unsuccessful work. A complete transitive never-started chain has no
separate public proof.

**Context:** A direct dependent that cannot start because its required producer
is unsuccessful becomes `blocked-upstream` and receives no new attempt. The
deferred outcome covers deeper never-started chains.
