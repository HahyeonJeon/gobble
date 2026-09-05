# Pipeline monitoring

The execution controller collects process and Docker stdout/stderr into
persistent attempt files while tasks run. Docker uses one following log client
per running task and gathers final logs during settlement. The monitor only
reads files and state; closing it does not stop the run. Live Docker collection
has hermetic coverage; the real-runtime CI journey is a separate validation gate.

Start a pipeline as usual. In another terminal, use the same installed Gobble
command or the same packed runner to watch its workspace:

```sh
gobble watch --workspace /path/to/workspace
# A packed runner has the same command:
./rnaseq watch --workspace /path/to/workspace
```

The monitor opens on the pipeline graph and global progress statistics. It
refreshes once per second. Terminal stdin and stderr are required; stdout stays
empty. `q` or Ctrl+C closes the monitor and restores the terminal. Execution
continues in its original owning process. Monitoring never occupies, releases,
resumes, or cancels work.

| Key | Action |
| --- | --- |
| Arrows | Move between neighboring graph cards or search results |
| j / k | Traverse stages in dependency order, or move through task lists |
| Enter | Open stage tasks or task details |
| / or s | Search samples; Enter selects an exact sample ID |
| t | List tasks in the current sample scope |
| ! | Open the global attention list |
| 1 / 2 | Show stdout / stderr in task details |
| 3 | Open full scrollable task facts from a task list or inspector |
| f / End | Pause or resume following the newest log tail |
| PgUp / PgDn | Pan the graph or scroll lists, facts, help and logs |
| Esc | Return; on the dashboard, clear sample scope |
| r / ? | Refresh now / show help |
| q / Ctrl+C | Exit the monitor; in sample search, use Ctrl+C |

The charcoal palette uses teal for active work, green for success, rose for
failure, and amber for uncertain or blocked work. State labels remain visible
without color. Set `NO_COLOR=1` to disable color. The layout adapts to terminal
size; below 44 columns or 20 rows it shows a compact status summary.

## Dashboard layout

The dashboard retains the accepted design preview's layout: three summary
panels, a full-width state distribution, a persistent sample search field, a
compact graph of filled stage cards, and an attention/sample-progress sidebar.
The graph folds its topological sequence into alternating two-column rows when
space allows. Arrowheads retain the direction of every original dependency;
long edges route outside cards. Narrow terminals use one column and panning.

Searching opens a two-column result area above the graph while global issues
and sample progress stay visible. Selecting a sample changes stage counts and
ownership context, while the global metrics remain above. Shared/cohort cards
are labeled as context and excluded from sample completion.

On wide terminals, task inspection keeps a task list beside the selected task's
metadata and logs. Esc returns to the same list position. Short terminals use
compact summaries; no layout invents ETA, resource utilization, or task states.
See [UI design notes](ui-design.md) for the comparison to the original preview.

## Reading progress

Global counts stay global when a sample is selected. Sample completion counts
only work explicitly owned by that sample. Shared reference and cohort report
tasks remain visible as graph context and have separate counts.

The denominator is the number of currently known executable instances. Scatter
templates are separate; unexpanded templates are labeled as expanding; skipped templates are counted separately. New
instances can increase the denominator. Reused successes are a subset of
successes. Failed, blocked, skipped, incomplete, unknown, and
published-unfinalized states remain distinct. The percentage is a task count,
not an estimate of remaining compute time. Resource figures are requests,
not measured CPU or RAM usage.

Repeated task names are grouped by display stage, scope, and topological rank.
This keeps dependencies acyclic even when a stage name occurs again downstream.
Enter opens the exact task instances behind a node. Long graphs can be panned;
the selected node's upstream and downstream stages appear in the wide layout.

Task details include identity, attempt, executor, requested resources, command,
failure reason, and logs. Press `3` for full identity, argument boundaries,
multiline scripts, timestamps, error location, reuse reason and full-file log
paths; arrows, PgUp/PgDn and Home/End scroll these facts. Each stream is bounded to its last 4 KiB. Pausing stops
automatic scrolling within that moving tail; it does not retain older history.
Use the paths returned by `inspect logs` for full files. Terminal controls in
task labels and logs are stripped before rendering.

If a control snapshot cannot be read coherently, the last valid snapshot stays
visible with a STALE message and its observation time. If only the selected log cannot be read, a separate
global read repeats the same gates and keeps progress fresh while the log pane
shows the error. Rejected log bytes are never read. Elapsed time stops
advancing when the owner is no longer live or the snapshot is stale. Existing
workspace identity and schema checks also apply to monitoring.

## Labeling custom pipelines

Sample membership is explicit, never inferred from task-name substrings. The
bundled RNA-seq, scRNA-seq, ATAC-seq, methyl-seq, and WGS pipelines supply labels.
Existing workspaces without labels still support graph, task, and log navigation;
they cannot acquire labels without a newly authored plan.

```go
sample := pipeline.AddModule("S01").WithDisplay(gobble.TaskDisplay{
    Samples: []string{"S01"},
    Scope: gobble.DisplaySample,
})
sample.AddTask(gobble.TaskSpec{
    Name: "align",
    Display: gobble.TaskDisplay{Stage: "Alignment"},
    // Command, Inputs, Outputs, ...
})
```

Call `WithDisplay` before registering children. Defaults inherit through nested
modules, branches, and scatter/gather scopes. Task fields override inherited
values; shared or cohort scope clears sample ownership. Sample IDs should
include a patient or assay prefix when required for uniqueness. These copied
labels do not affect commands, artifact paths, scheduling, or reuse decisions.

Sample search supports pasted text. Discovery ignores case; an exact entered ID
retains its case-sensitive identity. Cancelling search restores the prior view.

For tasks registered through asset helpers, `modules.WithDisplay(parent,
display)` provides copied defaults at their `Parent` boundary; explicit task
fields still override those defaults.

## Code boundaries

| Location | Responsibility |
| --- | --- |
| `display.go` | Public authoring labels and inheritance |
| `internal/engine/monitor.go` | Read-only projection of one coherent control revision and selected live log tails |
| `monitor/snapshot.go` | Typed reader through the public Inspect API |
| `monitor/counts.go`, `dashboard.go`, `topology.go` | Pure aggregation, sample indices, dependency-preserving stage grouping |
| `monitor/tui/model.go` | Bubble Tea update loop, navigation, single-flight refresh |
| `monitor/tui/view.go`, `dashboard_view.go`, `inspector_view.go`, `metadata_view.go` | Persistent summary, dashboard/search, and task/log layouts |
| `monitor/tui/components.go`, `graph.go`, `theme.go` | Cell sizing, state bars, graph routing, surfaces and safe text |
| `cmd/gobble/watch.go`, packed templates | CLI and packed-runner integration |

The engine and public graph library do not import terminal packages. Bubble Tea
and Lip Gloss live only in the TUI leaf package. JSON consumers can obtain the
same global facts without opening a terminal:

```sh
gobble inspect monitor --workspace /path/to/workspace
gobble inspect monitor --workspace /path/to/workspace --instance TASK_ID
```

The optional instance selects log tails; task and graph facts remain global.
Control facts share one revision, while log bytes may advance independently.
The monitor view is additive to control schema 2.
