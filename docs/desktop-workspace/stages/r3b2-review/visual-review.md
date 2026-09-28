# R3b2 visual review

Target: final App source in [final-app-source.json](final-app-source.json), built Electron 44.2.0
on macOS arm64. Captures are native window content; no generated mockup or image editing was used.
The review used temporary synthetic Projects and deterministic provider/monitor fixtures.
Reviewer: implementation author. This is not a representative-user or screen-reader study.

| Scenario/check | Observed result |
| --- | --- |
| One Run Surface plus Chat at 1280×840 | Graph, selected group counts, exact instance rows and one draft attachment are visible together. Discuss and navigation controls are visible without hover. [Overview](dependency-overview.png). |
| Exact attempt logs in a companion Pane | Upper Run keeps its directed selection; lower log identifies `align:S03`, attempt 2. Chat receives only the selected dependency capture. The upper view scrolls in the smaller split region; actions and endpoint summaries remain reachable. [Split view](dependency-discussion.png). |
| Historical preview after changed attempt/topology | Dialog still shows the original `prepare → align` pair and attempt 2 after the source changes to attempt 3 and no edge. Source-version disclosure remains available. [Saved evidence](dependency-frozen-preview.png). |
| 900×650 window at native 150% | Existing compact Workspace/Chat switching is reused. The dependency list can scroll; selected `task-00`, separate counts and Discuss remain readable and reachable. Search-hidden selection recovery is exercised before capture. [Compact](dependency-compact-150.png). |
| 80 versus 81 observed groups | 80-group chain supports native horizontal wheel, mode restoration and Fit. 81 groups uses an explicit bounded list, showing `80 of 81` and returned edge scope. [Bounded fallback](dependency-bounded-list.png). |
| Keyboard and identity | Node and directed-pair buttons activate by keyboard; instance radio selection retains exact identity despite repeated names. Templates cannot open logs. An unfiltered Show action leaves Discuss enabled. |
| Persistence and failure | Prior Tasks filter, graph zoom/pan, semantic selection and draft survive relevant mode/restart paths. Refresh failure retains the previous observation. New source facts invalidate local targeting but never rewrite saved evidence. |

Visual adjustments from the walkthrough: one shared freshness notice under RunView; a fixed-size
arrowhead that does not inflate on selection; collapsed edge endpoint details; a Show selected
task action for a row outside the Tasks filter. Mode panels retain local navigation state while
inactive content remains hidden from the accessibility tree.

The two native dependency scenarios and existing regression suite cover behavior; screenshots
capture selected moments. Normal-size overview was captured with the separate replayable
[capture walkthrough](capture-overview.ts) against the same unchanged production build. It sends
no message, closes the temporary app and deletes its own profile. No real account or Project was
modified. Final owner feedback is still needed before the next checkpoint.
