# P2B-1 verification

Implementation and test runs: 2026-09-09; result finalized 2026-09-10 (KST). Evidence is split between deterministic protocol-peer checks, real
Gobble/native App behavior, and the actual signed-in Agent. Historical stage counts
are not reused as current test results. Final suite evidence is listed below.

## Environment and ownership

Native macOS arm64: Go 1.27.1, Node 25.7.0, Electron 44.2.0, React 19.2.8,
TypeScript 5.9.3, Codex 0.153.4. The module language is Go 1.26. Root engine/CLI
checks run on Linux amd64 in the existing local Docker toolchain. No package
installation, registry publication, commit or research Project mutation occurred.

All application scenarios use owned synthetic Project copies and isolated profiles.
The actual-Agent script copied the already authorized account credential into its
protected temporary profile, did not print it, and verified the original unchanged.
The temporary profile and synthetic Project were removed on completion. See
[evidence/actual-agent/isolation.json](evidence/actual-agent/isolation.json).

## Actual signed-in Agent — passed

The App default `gpt-6-astra` used actual Codex dynamic tools, the native service
and pinned Gobble runtime. First, a natural-language request with explicit scoped
proposal permission produced Quality 25→30 and a real FastQC branch, preserving
Minimum length 40 and existing processing. Gobble checked both versions. Next,
a separate read-only message attached the exact setting change; the Agent read the
comparison and published an independent reference. It named 25→30 Phred, retained
40 bp, and distinguished declared reports from completed execution.

The script asserted the unchanged User selection and unsent draft, then used User
adoption controls and verified unchanged imported source, one managed revision and
zero Runs. The run completed successfully; see [result](evidence/actual-agent/result.json),
[proposal turn](evidence/actual-agent/proposal-turn.json),
[review turn](evidence/actual-agent/review-turn.json), screenshots and
[test log](evidence/actual-agent/test.log). This proves one actual model interaction,
not every model or future nondeterministic response.

That run used image
`sha256:4c7a78e6caf180bd098c441dcfd9ba673f262c0b56889ff5dcb2eeb7e322baeb`;
its exact build inputs are [retained separately](evidence/actual-agent/runtime-context.json).
Subsequent final changes bounded oversized comparison storage, tightened the list
schema limit, added storage-failure tests and fixed packed help. They do not alter
the demonstrated supported processing/reference semantics. The final native suite
qualifies the resulting build and separately repeats the complete proposal loop.

## Validation layers

- App formatting, process TypeScript checks and generated bundle v20 consistency.
- 340 unit/contract/integration tests in 32 files. Includes native HTTP contracts,
  source authority expiry, stale context refusal and frozen legacy schemas.
- Native race tests and vet for service, portable review/construction and launch
  boundary packages. Includes concurrent adoption, integrity/current mismatch,
  failed catalog write without partial authority, and interrupted history refusal.
- Focused Linux tests for actual module-based review and complete engine fingerprints.
  Broader CLI regression reported the two previously recorded packed-runner failures.
  It also exposed missing license notices in `review`/`flow` help; those were corrected,
  and `TestPackedHelpLicenseBoundary` passed on the resulting source.
- Final real Electron regression with live checked-flow/discussion/proposal opt-ins.
  The proposal case uses a clearly labeled deterministic Codex protocol peer for
  reproducibility; its source service, checker, comparison, adoption and restart are real.

The two unresolved baseline tests remain `TestPackPrintpipeArtifact` and
`TestPackHostpipeEmptyInspectProtocol`; this is not a fully green root CLI suite.
The existing native macOS root engine support constraint is unchanged. No analysis
processing task was started by the review/adoption scenarios. Separate existing
execution/packaging qualifications are not implied.

## Visual scenarios and checklist

- [x] Identify changed and added steps by label/number as well as color.
- [x] Select a changed step and compare 25/30 in the lower paired area.
- [x] Inspect an added branch, its declared source and outputs; no generated-report claim.
- [x] Keep current-flow navigation and rounded orthogonal routing.
- [x] Attach exact change context through the one composer, preserving draft and recipient.
- [x] Read an independent actual-Agent reference without moving User selection.
- [x] Confirm the managed-copy handoff explicitly before User adoption.
- [x] Preserve source, history and unsent draft across adoption/restart in the protocol-peer case.
- [x] Inspect compact layout; paired facts, Chat switching and lower actions remain readable.

The final broad Electron run executed **70 cases**: 69 passed and one failed only
because the existing named-bridge allowlist lacked the newly authorized
`pipelineReviews` namespace. The test was updated to include that exact namespace
and assert its four operations (`list`, `select`, `adopt`, `outcome`), excluding
source access. All six foundation cases then passed on the same production build.
All 70 distinct cases therefore have passing evidence; the retained broad log still
truthfully records its original failure. After that run, one service-only failure-path guard was added: a stopped catalog
writer cannot report a missing receipt after an uncertain directory sync. A focused
fault-state/restart test, native race/vet and all 340 App unit cases were rerun;
the App was rebuilt. The healthy proposal and renderer paths are unchanged.
See [broad native log](evidence/final/electron.log) and
[corrected foundation rerun](evidence/final/foundation-rerun.log).

Final runtime image:
`sha256:effd239270cd58dc27d97b32b884ab21569a08e26b163d61736ab50c2a85f78d`.
Its [source context](runtime-context.json), [build log](evidence/final/runtime-build.log),
[fixture identities](evidence/final/fixture.json) and final setting/branch/adoption/
compact captures are retained. The final native proposal case also verifies amber
color after selection, disabled unchanged targets, keyboard selection of the added
connection, source-permission refusal in the second message, exact independent
reference, explicit adoption, preserved imported files/draft, zero Runs and restart.

Other final logs: [format](evidence/final/format.log),
[types](evidence/final/types.log), [schemas](evidence/final/schema.log),
[340 unit cases](evidence/final/unit.log), [native race/vet](evidence/final/native.log),
[Linux scope and known failures](evidence/final/linux.log),
[help correction](evidence/final/help-regression.log).
All previously published contract bundles v1–v19 retain their pre-stage hashes.
`git diff --check` passed; [after.json](after.json) records source provenance,
including inherited uncommitted work. No commit or publication was performed.

Native evidence is captured from the application, not the historical HTML sketch.
Long Agent summaries may require scrolling to the adoption footer. Existing Chat
renders message text literally; rich Markdown rendering is outside this slice.
