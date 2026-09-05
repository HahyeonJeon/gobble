# Gobble UI — discussion draft

Status: dashboard-first direction accepted and implemented on `codex/gobble-ui`.
This document records the design discussion; [monitoring.md](monitoring.md)
describes the implemented commands, package boundaries, and limits.

Session: `mode: Cowork`, `slug: gobble-ui`, `partner: disabled`.
Base: `develop` at `37aab275d6de0bc3ac9324b3c68c27e647680557`.
Working branch: `codex/gobble-ui`.

## Goal

Let a human understand a running pipeline at a glance, locate the work that
needs attention, and inspect its details and output in an attractive terminal
interface. The engine remains usable by agents through structured APIs.

## Accepted direction — 2026-09-05

- The initial screen is a pipeline graph and progress-statistics dashboard.
- A human must quickly understand overall execution and locate problem areas.
- Search supports finding one sample and inspecting its progress and problems.
- Detailed task lists and logs are reached from a stage, sample, or problem.
  They are not the primary landing screen.

The previous task-list-first proposal is superseded by these requirements.

## Proposed experience

Use Bubble Tea v2 for input and asynchronous updates, Lip Gloss v2 for layout
and styling, and Bubbles v2 components where useful. Keep UI state and rendering
outside the scheduler and outside the root library's dependency graph.

Start with an explicit, read-only monitor:

```text
gobble watch --workspace DIR
./rnaseq-runner watch --workspace DIR
```

These are proposed commands. They do not exist at the base revision. They must
use the execution identity already required by Inspect, including the packed
runner's embedded identity. A newly compiled executable must not bypass a
workspace's identity checks to attach to an older run.

An integrated `run` / `resume` presentation is a second entry-point decision.
If selected, specify terminal detection, stdout/stderr behavior, cancellation,
and signal ownership before integrating it with the child-process driver.

## Primary screen and navigation

| Area | Information and interaction |
|---|---|
| Header | Run ID, workspace, run status, elapsed time, reader freshness, owner liveness |
| Run statistics | Successful latest executable instances / known executable instances, all state counts, active work, elapsed time, and sample-work completion; reused success is a subset |
| Pipeline graph | Primary visual area; grouped stages with directed dependencies and per-state task counts, including mixed running/failed states |
| Attention | Root task failures, blocked downstream work, and affected samples, with direct navigation to the corresponding stage and task |
| Sample search | Find exact sample identities through search; select a result to scope the graph and details; show an explicit empty result |
| Scope indicator | All samples or a named sample; global run statistics stay visible and distinct from scoped counts |
| Drill-down | A selected stage lists its task instances; a selected sample shows its tasks and relevant shared dependencies |
| Task inspector | Full identity, command/script, image, requested CPU/memory, timestamps, reuse decision, structured failure, and bounded stdout/stderr tails |
| Footer | Visible keyboard shortcuts and refresh/error state |

The main hierarchy is Dashboard → Stage or Sample → Task → Logs. An attention
item can jump directly to its failing task. Returning restores the graph focus,
sample scope, and position. A sample search does not start another run.

At wide terminal sizes, give most space to the graph and a smaller area to
attention and sample progress. Put compact run statistics above both. Show a
task inspector on selection. At narrower sizes, use a vertical layered graph
and separate detail view. Do not shrink the text or remove edges to make the
graph fit. Large graphs need collapsed stages and viewport navigation. Keep
selected node positions stable during refresh. At very small sizes, preserve
run status, attention counts, selection, and quit.

Proposed keys: arrows / j / k to select, Enter to expand or open detail, Esc to
go back, Tab to change focus, `/` to search samples, `!` for attention, `f` to
follow logs in a log view, `?` for help, and `q` to leave the monitor.
Closing a separate watch process must not cancel, release, or resume the run.

## Visual direction

Use a quiet charcoal theme with a restrained teal accent, thin panel dividers,
consistent padding, aligned monospace columns, and a clear selected row. Pair
each status color with a word or symbol. Keep animation limited to a small
running indicator. Terminal typography is controlled by the user's terminal;
the implementation cannot promise a bundled font, blur, or true transparency.
Support monochrome and reduced-color terminals with readable status labels.

The accompanying dashboard preview uses a simplified synthetic RNA-seq graph,
eight sample identities, and illustrative logs. Counts are derived from the
same synthetic task records as the stage and sample views. It is not connected
to a workspace and does not establish the complete RNA-seq product graph.

## Graph behavior

- Start with stages aggregated across samples, rather than hundreds of repeated
  sample-task nodes. Nodes display counts, not one color that hides a mixture
  of successful, running, and failed work.
- Preserve real directed dependencies, branch/merge points, and shared work.
  A collapsed stage is not one executable task. Expanding reveals its tasks.
- Validate grouping against the DAG. Arbitrary contraction can introduce an
  apparent cycle even when the task graph is acyclic; split such groups or
  retain the task graph instead of fabricating an order or dropping edges.
- Sample selection shows owned tasks as the counted scope. Shared reference
  dependencies and cohort consumers remain context nodes, explicitly labelled
  and excluded from that sample's completion denominator.
- Differentiate a directly failed task from downstream blockage. Claim a root
  cause or affected-sample relationship only when recorded state and dependency
  evidence support it. Otherwise show an unclassified failure or dependency.
- No DAG path is labelled the critical path or an ETA without an established
  timing model. Elapsed duration alone does not prove a task is stalled.

## Sample and stage metadata

The base Plan contains task name, module path, DAG edges, and parameter pairs.
Some products, such as scRNA-seq's `sampleTaskParent`, explicitly add a `sample`
parameter. RNA-seq groups work below `AddModule(sample.Name)`. These are useful
inputs but are not a universal semantic schema for arbitrary Gobble pipelines.

Propose an explicit persisted display description per task: stable stage ID and
label, sample IDs, and a role distinguishing sample-owned, shared-reference,
and cohort work. Exact public API placement is still to be settled. Display
metadata should not alter commands, scheduling, output paths, or computational
identity; persistence/schema compatibility and fingerprint consequences must
be checked before choosing that separation.

Product adapters can populate this description from typed samples/config while
constructing the graph. Generic pipelines without the description still get a
DAG and task search. Show sample association as unavailable; do not guess a
sample from a substring in a task ID. One-to-many sample associations must be
explicit, and global task counts must deduplicate task identity. A search for
`S01` may find several candidates, but selecting `S01` must not select `S010`.

Search indexes identity and supplied aliases once per coherent snapshot.
Search changes the local view only. Preserve global failure visibility while
viewing a healthy sample.

## Findings from the base revision

| Existing code | Consequence for implementation |
|---|---|
| `inspect.go`, `internal/engine/inspect.go` | Read-only run, instances, errors, logs, timing, DAG, lineage, remaining, reuse, and identity views already exist. |
| `readCoherentControl` and `coherentSnapshots` | One Inspect call rejects inconsistent control snapshots. Several separate calls may still represent different snapshots. |
| `runJob` and `sched.apply` in `internal/engine/run.go` | Log paths are carried in the completion report and assigned to persisted task state in apply. Active-task log visibility needs an earlier, validated publication point. |
| `inspectLogTail = 4096` | Current logs inspection supplies only a bounded tail, not an unlimited stream or full-history viewer. |
| `jsonResources` and scheduler budget | Declared CPU/memory are available. Live utilization, peak RSS, ETA, and per-tool internal percentages are not established telemetry. |
| Scatter/Gather expansion and latest attempts | Counts must exclude scatter templates and old attempts and account for growing runtime membership. |
| Occupancy and owner liveness | An active workspace is not proof that its owner is alive or that work is still running. Unknown states must remain visible. |
| Generic driver and packed inner template | Both human entry paths require deliberate integration; generated consumer code cannot import an engine-internal UI package. |
| `jsonTask.Module`, `Name`, and `Params`; product builders | A grouped graph is possible, but accurate cross-product sample/stage semantics need an explicit mapping. |

## Proposed implementation slices

1. Implement the accepted graph/statistics-first presentation and sample
   navigation in the design preview. Keep `watch` as the recommended entry
   point; integrated `run` remains a separate decision.
2. Add a coherent monitor read model using the existing containment, schema,
   and identity checks. Read run, latest instances, timing, and selected logs
   through one projection. Log bytes can advance after the control snapshot;
   label them as a tail read, not a transactional event history.
3. Publish validated log pointers before task completion without moving UI
   dependencies into the scheduler. Preserve execution and recovery behavior.
4. Add explicit stage/sample mapping and tested aggregate selectors. Build the
   Bubble Tea dashboard, layered graph, sample search, attention navigation,
   detail view, bounded logs, responsive layouts, and stale/error states. Use
   async reads with one outstanding refresh, initially around one second apart.
5. Integrate the accepted CLI entry points, help, packed-runner wiring, terminal
   checks, and documentation. Retain existing JSON/JSONL command contracts.
6. Verify with a small deterministic pipeline: concurrent execution, failure
   and blocked descendants, resume/reuse, expansion, reader races, identity
   mismatch, and exiting watch while execution continues. Inspect the actual
   rendered TUI at wide and narrow terminal sizes.

Additional acceptance cases: one failed sample stays visible in global
statistics while a healthy sample is selected; shared work is counted once;
simultaneously running and failed tasks remain visible within one stage; an
exact sample result does not include similarly named samples; aggregate graph
edges preserve topology; and collapse, expansion, search, and refresh preserve
focus. Verify these as behavioral checks rather than screenshot-only tests.

Dependency placement needs care: an internal UI package works for the generic
command, but a packed runner is generated in the consumer's context. Choose a
small public presentation package or an appropriate launcher boundary rather
than importing `internal/` from generated consumer source. Pin compatible
Charm v2 versions after verifying them with the repository's Go 1.26 toolchain.

## Progress semantics

Show counts as the primary truth. A task-count percentage is not a percentage
of elapsed or remaining computation. Reused successful work is counted once.
Sample-task completion and pipeline completion are different: a sample's own
tasks may all succeed while a cohort report is blocked. Define and label the
former metric explicitly, and report shared/cohort completion separately.
Skipped, failed, blocked, incomplete, published-unfinalized, and unknown-backend
states do not become successful tasks. Runtime expansion may increase the
denominator; display known membership and pending expansion rather than a
misleading fixed 100% target. A stale snapshot keeps its last-read time and
must not animate as if fresh. Monitor polling may miss intermediate states;
do not describe observed changes as an authoritative event history.

## Questions for discussion

- Should the initial user experience be a separate `watch`, or a monitor opened
  as part of `run` and `resume`?
- What minimal display-metadata API preserves the engine's authoring and
  persisted-workspace contracts while enabling reliable sample navigation?

## References

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Bubbles](https://github.com/charmbracelet/bubbles)

The initial design pass preceded implementation. Implementation, validation and
remaining limits are recorded in [monitoring.md](monitoring.md) and the
[pre-push review](ui-review.md).
