# R3b1 verification record

Subject: the App source change set relative to [baseline.json](baseline.json), with final paths
and digests in [implementation-subject.json](implementation-subject.json). This is an author-mode
implementation review on the existing checkout. Unrelated pre-existing engine changes are excluded.

## Results

| Evidence | Result and scope |
| --- | --- |
| [Final full App check](full-check-2.log) | Formatting, all process/tooling TypeScript configurations, generated schema check, Go service build, 247 unit/contract tests in 22 files, built renderer/preload/Main, and 40 native Electron scenarios passed. |
| [Final static and unit check](final-static-unit.log) | Formatting, process/tooling type checks, schema consistency and all 248 unit/contract tests passed after adding archived-pointer recovery coverage. |
| [Final changed-owner tests](final-cleanup-unit.log) | 57 tests passed across dependency, historical plot evidence, linked-table, render lifecycle and Agent shared-context owners. |
| [Final CSV native scenarios](chart-retirement-native-2.log) | Three scenarios: exact selected row and attachment retained through sorting; old chart requests refused without mutation; old chart metadata preserved on restart with source-table recovery. Includes visible attachment readiness before the final capture. |
| [Renderer qualification](renderer-qualification.json) | Both isolated candidates passed their type check, exact semantic target assertions, bounded fixtures, native zoom, read-only and teardown checks under production CSP. See [decision and limitations](renderer-decision.md). |
| [Compatibility audit](compatibility-audit.json) | Frozen v1–v9 bundles, v8 workspace shape, old Run presentation/hash/capture files and unrelated baseline engine/service sources checked against their pre-turn bytes. No Plotly or graph candidate dependency remains in the product manifest/lock. |

`full-check-1.log` is the earlier full pass before final dead-chart-path cleanup. The second
full check tested the final product source; only test coverage changed afterward: native screenshot/readiness, isolated renderer exact-target
assertions and one archived-pointer recovery case. Static checks and all unit tests passed again;
the native CSV and isolated qualification subsets were rerun. Product code did not change. A shell wrapper used a reserved status variable after the second
full check completed; the wrapper failed, while the retained App log confirms every check passed.

Native scenario count changed from 43 to 40: six obsolete chart-creation/rendering scenarios were
replaced by three retirement/table/compatibility scenarios. This is the user-authorized removal
of the feature, not skipped or disabled tests. The final unit/contract count is 248.

## Classified failures and corrections

- `chart-retirement-unit-1.log`: test defect. The new tool-refusal assertion used the wrong result
  field; corrected to the host's actual `success` contract. The next focused run passed 42 tests.
- Initial renderer type/scenario logs: harness defects (missing typed props, duplicate fixture
  edge and initial-navigation race). Corrected and rerun; failed artifacts remain available.
- Final test additions initially used the small-fixture destination for the 80-group fixture and
  omitted an optional-value narrowing in a known archived fixture. These test defects were corrected;
  `renderer-qualification-failed-6.json` and `final-static-unit-failed-1.log` retain the evidence.
- Native capture could precede the renderer's visible attachment update after Main persisted it.
  The CSV scenario now waits for the visible attachment and enabled source action, then two paint
  frames. This strengthens visual evidence; it does not change product timing or suppress a failure.
- No unresolved product defect was found in this bounded affected-set review.

## Limits and review obligations

The new dependency adapter/projection/resolver/capture are not yet wired to live workspace reads,
EvidenceRef unions, storage or Agent tools. They are verified with coherent synthetic monitor
responses. R3b2 owns User UI/camera/store integration; R3b3 owns versioned Agent tools and a real
signed-in Agent trial. Current Run/task/log native regression scenarios still pass.

No new Docker integration, whole-engine suite, Windows/Linux UI, screen-reader study, installed
package lifecycle, update, signing or distribution check was performed in this checkpoint. It
makes no release claim. Historical CSV chart metadata remains readable; chart pixels are no
longer rendered. The native screenshots use temporary synthetic Projects, not user data.
