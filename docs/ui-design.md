# Dashboard design alignment

The first terminal implementation preserved monitoring behavior but lost the
accepted dashboard preview's composition. This revision uses that preview's
original layout and color values as its reference, including the same eight
samples and six stages for visual comparison.

| Accepted preview | Terminal revision |
| --- | --- |
| Successful tasks, running now, and failed tasks in three horizontal regions | Three aligned summary panels with distinct labels, values, and notes |
| Full-width state distribution and exact-count legend | A cell-sized segmented bar and wrapping state legend; tiny states retain their numeric labels |
| Search field stays above the workspace | Persistent search field; results expand above the graph without replacing global context |
| Reference / Read QC, Alignment / Trim reads, Quantification / Cohort report | A folded topological sequence reproduces these two-column rows without hardcoding pipeline names |
| Filled stage cards with scope, title, counts, state, and progress strip | Seven-row terminal cards with all five fields; selected background and a separate failure accent |
| Attention and sample progress beside the graph | Global issues, sample chips, completion counts, and shared/cohort totals in one sidebar |
| Task list beside selected-task details | A master/detail inspector keeps the task list visible beside metadata and log tabs |
| Muted charcoal surfaces with limited color accents | Exact preview palette; semantic symbols and selection also work with NO_COLOR |

## Palette

| Token | Color |
| --- | --- |
| Background | `#10171B` |
| Panel | `#162127` |
| Text | `#DBE8EC` |
| Muted text | `#9AB0BC` |
| Border | `#354B57` |
| Active | `#8CCDDD` |
| Success | `#8AC9AD` |
| Failure | `#F19BAC` |
| Blocked / uncertain | `#D5B78D` |
| Selection | `#203C46` |

## Terminal adaptations

Terminal cells determine dimensions. Bold text, panel contrast, and spacing
replace variable font sizes. The design uses solid surfaces instead of alpha
blending or blur. These adaptations preserve the information hierarchy.

At sufficiently wide sizes the graph uses two columns and the inspector uses
two panes. Narrow terminals use one column or one pane. Short terminals show
compact metrics and a shorter sidebar. Graph navigation reveals the focused
card; long graphs remain pannable. Left/right arrows follow the graph's folded
rows, up/down follow neighboring cards, and j/k follow dependency order.

Stage grouping still includes topological rank. Folding changes only node
placement, and every edge still connects its original pair of stage IDs. Same
row dependencies point left or right; dependencies between rows point down;
long dependencies route outside intervening cards.

## Reproducible review

`monitor/tui/preview_test.go` contains the original preview's 34-task workload:
23 succeeded, 4 running, 1 failed, 4 pending, and 2 blocked; 3 successful tasks
were reused. The fixture is illustrative and does not execute analysis tools.
The captures call the production Bubble Tea model's View method and navigate
through its real key handlers.

```sh
GOBBLE_UI_CAPTURE_DIR=/tmp/gobble-ui-review \
  go test -run TestCaptureScreens -count=1 ./monitor/tui
```

This exports ANSI frames for the dashboard, search, sample scope, task inspector,
and task logs. `GOBBLE_UI_PREVIEW=/tmp/gobble-terminal.ansi` retains the existing
single-frame CI export. Validation covers reference layout, spatial navigation,
exact sample selection, persistent global context, split inspection, terminal
sizes, no-color rendering, and the existing real-PTY lifecycle checks.
