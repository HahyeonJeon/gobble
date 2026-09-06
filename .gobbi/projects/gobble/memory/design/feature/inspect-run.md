# Gobble — Inspect and Monitor a Run

## Purpose and scope

Let agents and people read execution facts without changing a run. The run
workspace is authoritative. Library inspection, structured CLI output, and the
sample-aware TUI share the same read path. The planned browser application is
specified in [Application and monitoring](../architecture/application.md).

## Current interfaces

`Inspect(workspace, View, instance, opts ...OccupyOption)` is read-only.
`gobble inspect VIEW --workspace DIR [--instance ID]` returns its bytes.
Current views are `run`, `instances`, `errors`, `logs`, `timing`, `dag`,
`lineage`, `remaining`, `reuse`, `identity`, and `monitor`. There is no event
history view yet. Object views return JSON; `instances`, `remaining`, and
`reuse` return JSONL. Records carry a schema version.

`gobble watch --workspace DIR` presents the monitor in an interactive terminal.
Closing Watch leaves execution running. It refreshes once per second and can
navigate stage dependencies, explicit sample ownership, tasks, errors, and
logs. Sample selection does not change global counts. Shared/cohort work stays
visible as context and is not attributed to individual sample completion.

The common container entry is `docker compose run --rm gobble ...` from the
project containing its pinned Compose file. Current installation and full
monitor behavior are described in the
[container guide](../../../../../../distribution/runtime/README.md) and
[monitoring guide](../../../../../../docs/monitoring.md).

## Consistency and failure

Inspect requires a coherent control revision and applies workspace, schema,
identity, and file-containment gates. It does not acquire execution occupancy
or create control files. `identity` remains readable on installation mismatch;
other views require a matching effective identity.

Unknown workspaces or task instances produce structured errors. An unreadable
selected log must not freeze otherwise valid global progress. The TUI retains
a last valid snapshot with a stale indicator and observation time when control
facts cannot be read coherently.

Logs are persistent attempt files. Inspection provides bounded tails, currently
4 KiB per stream, and file locations. Log bytes may advance independently of a
control revision. Terminal control characters are stripped before rendering.

## Vocabulary and interpretation

Task-count progress is not remaining compute time. Runtime expansion can
increase the known-task denominator. Reused successes are a subset of successes.
Resource figures are requests, not measured utilization.

Preserve failed, blocked, skipped, incomplete, published-unfinalized, and
unknown-backend distinctions. A proved-stopped Docker task can retain a runtime
ID for final log/cleanup retry without becoming unknown again. Occupancy
liveness is established through lock/lease evidence, not PID presence.

## Extension boundary

Add browser and agent interfaces through the same projections. Event history,
full log-range APIs, measured resources, and multiple-run catalogs are planned
extensions, not current Inspect guarantees. Inspection itself remains read-only;
[recovery](recover-run.md) owns execution changes.
