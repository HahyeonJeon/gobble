# R3b3 verification record

Subject: the authorized Agent dependency-reference checkpoint, macOS arm64 source checkout.
Implementation and verification were performed in the same task, with implementation, focused
checks, native scenarios, actual Agent trial and final frozen-source checks recorded separately.
No installed/released application, Windows/Linux support or general usability claim is made.

## Source and ownership review

- Gobble monitor facts and the Go service boundary are unchanged. App Main supplies dependency
  observations from the already retained coherent Run read; no new engine execution path exists.
- `dependency-observation.ts` owns bounded semantic output. Group context contains at most 20
  member records, with complete observed counts and explicit truncation. Whole records are dropped
  to fit a 48 KiB content budget; selected targets that cannot fit are refused. A graph has no
  fabricated EvidenceRef. A selected pair includes both endpoint contexts; a filtered-out endpoint
  group cannot acquire pointing authority from its inclusion as context.
- `ObservedReads` holds at most 32 turn-local scopes. Exact targets, directed endpoints, Project,
  Run, revision and acknowledgment are validated again by the current foreground host before
  serialized publication. Source-preview reads do not authorize task/log/dependency pointers.
- Existing `ObservedReferenceViews` owns one temporary retained observation. Its guard includes
  mode, dependency navigation and camera intents. `current-dependency.ts` handles explicit fresh
  searches without mutating historical references. Existing immutable capture storage and exact
  captured-target lookup are reused.
- React's dependency mark helper only derives labels and target matching. DependencyGraph owns
  geometry; DependencyReferenceContent owns a transient camera. The durable base presenter stays
  mounted. A pending camera timer is cleared when the base view becomes unavailable.
- Task/list contents follow committed navigation while a search input holds an unfinished edit.
  This keeps Agent observation scope aligned with visible content during the debounce interval.
- Workspace v9 and Agent v6 request readers are frozen and compared to published v11 definitions.
  Current Workspace v10 / schema v12 / toolset v7 use the existing explicit conversation renewal
  flow. Migration preserves exact original bytes. No package/dependency or process configuration
  changes were needed.

## Executable evidence

**Final complete check passed:** 263 unit/contract tests in 24 files and 43 native Electron
scenarios, plus formatting, per-process TypeScript checks, schema consistency, Go service build
and production Electron build. See [complete log](check-1.log), [exact App source hashes](final-app-source.json)
and [54 changed paths / preserved boundaries](implementation-subject.json). All 78 baseline
non-App engine/service sources, v1–v11 bundles, manifests and lockfile are unchanged. The focused checks
cover exact directions and field-order independence, invented/foreign/stale targets, prior-turn
receipts, filtered scope, hidden representations, response bounds, Show/Return guards, migration,
explicit current search and native delivery.

The new native scenario exercises:

1. Capture a group and pair; select another group and zoom; send to a scripted Agent.
2. Verify exactly two arriving marks, unchanged mode/camera/selection/draft and input focus.
3. Switch to filtered Tasks, Show a pair/group, navigate the temporary camera and Return.
4. Verify read-only controls and Return at 900×650 with 150% page zoom.
5. Change source facts, refuse old Show, open matching sent capture, and explicitly find current
   dependency data without changing the historical pointer or selecting a new target.
6. Restart while Show is open: the temporary view ends and durable navigation/draft remain.

A selected dependency observation also produces an evidenceId through the existing question
capture path. Existing file/table/image, task/log, capture, question, migration, focus and recovery
scenarios run in the complete App suite.

## Actual signed-in Agent trial

[Live delivery](live-delivery.json) and [native state checks](live-review.json) record one actual
GPT-6-Astra conversation through Codex 0.153.4 with `shared-views-v7`. Only the synthetic Project
`prj_RHZGMRPODFSPKEVNYJMW4LLEHH` and a fixture monitor were used; the Agent runtime was real.
The Agent observed the displayed Dependencies mode and published the exact align group and
prepare → align pair. Its explanation distinguished authored groups from observed members and
attempts, without synthesizing aggregate state or instance-level edges.

The native UI verified focus on the local draft after arrival, cross-mode Show, read-only contents,
Return to the S03 filter, exact persisted state equality and a matching immutable capture preview.
The existing Project was restored, the fixture runtime was closed, and the App was reopened with its normal runtime environment. All five preexisting Project
contents remain unchanged: four files are byte-identical; the active Project migrated only its
schemaVersion from 8 to 10 with an exact v8 backup. Original window/Project selection is restored.
See [profile preservation](live-profile-preservation.json).

The real trial preceded a final defensive refinement for a filtered endpoint and JSON field-order
comparison, plus concise current-search labels. That refinement does not change the unfiltered
path exercised by the real trial and is covered by the final pure/host/native checks.

## First failures and corrections

- Construction type checks: extending v4 required explicit file-selector narrowing and frozen
  legacy-origin narrowing in tests. Subsequent process type checks passed.
- New unit fixtures initially assumed unsorted group positions and omitted authored-reference
  provenance. Tests now locate exact IDs and use valid storage fixtures; product validation was
  preserved. Existing source-preview pointer tests were updated to require a displayed read.
- The first native dependency scenario passed. The existing task/log scenario still used a
  source-preview pointer and then exposed a debounce mismatch: rendered task rows could change
  before Main's navigation did. Both the fixture and committed-content presentation were fixed;
  the focused native task/log rerun passed. First failing logs are retained by stage; the failure UI was inspected during debugging.
- The scripted normal-profile launcher timed out waiting for its first macOS window. Native
  activation opened the existing process; the actual trial used CUA for all subsequent actions
  and refreshed the existing login. No authentication data was read or copied.
- Formatting checks initially identified seven modified files; these were formatted before the
  final complete check. No checks were weakened or failures hidden.

## Limits

Only returned, loaded preview facts are available. A pointer does not retain source bytes; a
historical preview exists only when matching sent/question evidence was previously captured.
There is no Agent-controlled mode switching, graph editor, execution/cancellation, sample-level
edge inference, CSV chart creation, extra composer or new MCP server in this checkpoint.
