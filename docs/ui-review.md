# Pre-push monitor review

Scope: all changes on `codex/gobble-ui` since develop
`37aab275d6de0bc3ac9324b3c68c27e647680557`, including public display metadata,
engine snapshots, product builders, CLI and packed commands, terminal views,
requirements coverage, and tests. This is a local review; no push is performed.

## Findings and fixes

| Finding | Impact | Resolution |
| --- | --- | --- |
| A library-only consumer could not build the generated packed runner without adding TUI dependencies | Packaging failed with missing checksum entries | Resolve runner-only dependencies using a temporary module/lock copy; regression test verifies consumer files remain unchanged |
| Some RNA-seq, scRNA-seq and ATAC-seq cohort tasks had no display scope | Real graphs displayed unassigned stages and could not distinguish cohort consequences | Label every cohort builder boundary; validate all tasks in all five product plans |
| Asset display wrapper overwrote explicit task fields and shared its sample slice | Stage overrides disappeared and a child could mutate future defaults | Reuse copied, field-wise inheritance through `TaskDisplay.WithDefaults` |
| Bracketed paste was ignored; exact matching ignored case | Pasted IDs did nothing; `s01` could select `S01` | Handle paste as sanitized, bounded input and prefer the exact case-sensitive ID |
| Search refresh and inspector refresh retained row numbers | New samples/tasks could move focus to another item | Preserve selected identities through refresh, help and cancelled search; retire removed details safely |
| Long commands, scripts and reasons were permanently truncated | Users could locate a failure but could not inspect its full recorded facts | Add a scrollable facts pane on `3`, including full identity, arguments, script, timing, error unit and reuse information |
| Short-terminal help had unreachable lower rows | Some controls and limitations could not be discovered | Add help scrolling and wrapping; retain run status explicitly in narrow headers |
| Failure to read a selected log froze otherwise readable global progress | A damaged or rejected log path could leave the whole monitor stale | Independently re-read the gated global snapshot and show a log-specific error; never read rejected bytes |
| Skipped scatter templates counted as pending expansion | A deliberately skipped branch appeared to be expanding forever | Count skipped templates separately from executable instances and pending expansion |
| Graph text discarded combining characters and could split joined Unicode glyphs | Some names differed from their recorded identity | Render terminal grapheme clusters and preserve combining marks |

## Requirements coverage

| User flow | Coverage |
| --- | --- |
| Understand overall progress immediately | Dashboard opens with global counts, state distribution, directed stages, attention and sample progress |
| Locate a failed or blocked area | Semantic stage counts and a global attention list remain available while a sample is selected |
| Inspect one sample | Explicit sample ownership, substring discovery, exact selection and pasted input; shared/cohort work remains separate |
| Inspect task details and output | Stage/sample task lists, full scrollable facts, stdout/stderr selection, bounded live tails and follow controls |
| Monitor changing work safely | Latest attempts, scatter membership, resume/reuse, preserved selection and explicit stale/unavailable states |
| Close monitoring without stopping execution | Separate read-only watch, terminal restoration, pipeline lifecycle tests |
| Keep the accepted visual direction | Existing dashboard surfaces and palette are retained; full facts use the inspector pane |
| Keep code maintainable | Engine projection, pure monitor aggregation, UI state, graph, dashboard and inspector remain separate |

## Verification

Regression tests reproduce the discovered input, selection, metadata, labeling
and rendering failures. Product checks inspect complete plans for WGS, RNA-seq,
ATAC-seq, methyl-seq and scRNA-seq. Local process integration exercises failure,
blocked descendants, exact sample scope, rejected log containment, identity
mismatch, release/resume, unchanged-work reuse after a display-only edit, and
runtime scatter membership. The real-PTY test checks terminal restoration and
continued execution after leaving watch.

The complete hermetic suite (`go test -count=1 ./...`) passed. Monitor/engine race checks and the command suite also passed. The final
packed-runner terminal check is recorded after verification.

## Deliberate limits

- `watch` attaches separately; automatic TUI launch from `run`/`resume` is not
  part of the accepted first entry point.
- Progress measures known task counts. ETA, measured CPU/RAM use and internal
  tool percentages need engine telemetry that is not currently available.
- Logs expose the latest 4 KiB per stream. Full-file paths are available in the
  facts pane; paused tails do not retain unlimited history.
- Older workspaces without display metadata support task/graph inspection but
  cannot infer sample ownership. New labels require a newly authored plan.
- Aggregated stages expand into task lists. Large graphs require panning; there
  is no graphical zoom, mouse navigation or freeform task-text filter in this
  iteration. Sample search is implemented.
- Scientific tool/container validation is separate from monitoring behavior.
