# P5B-4 verification

Date: 2026-09-27 KST. Existing P5B-3 hashes matched before changes. Baseline focused Main continuation/dependency tests passed (22 tests). The initial command used the repository root instead of the App root and found no tests; it was corrected before edits. Existing dirty work remains unstaged and uncommitted.

## Environment and boundary

macOS arm64, Node 25.7.0, npm 11.10.1, Go 1.27.1; installed Electron 44.2.0 / Playwright 1.63.0 / Vitest 4.1.11 / TypeScript 5.9.3. No dependency installation, engine image replacement, real provider inference, scientific analysis or publication.

Electron tests use the real renderer, preload, Main, workspace/evidence storage and native Go service. A test-only Go exporter creates a fresh profile using native serializers/checksums. A standalone Docker protocol peer supplies deterministic engine observations and records every create/start command. A scripted Codex peer exercises authenticated shared observation/pointing without network inference. No product test bypass or runtime seeding endpoint was added. The fixture validates App/service transport and interaction, not actual tool execution.

## Checks

- App typecheck and generated v28 schema consistency: passed.
- All App unit/integration tests: **41 files, 380 tests passed**.
- Native `go test -race ./internal/appservice ./cmd/gobble-service`: passed; service command has no test files.
- App build: passed.
- Real Electron continuation tests: three scenarios passed (details below).
- Existing real Electron continuation transport/restart and Agent Run/log reference regression: both passed. Combined run: **5 tests passed**.
- Scoped Prettier, Go vet, Go formatting, schema preservation and diff whitespace: passed. All baseline files remain; schema bundles v1–v27 are unchanged. The handoff contains 44 changed/new paths in `changed-paths.json`, with final identities in `after.json` (the manifest files themselves are excluded).
- After the final copy/layout and Agent tool-description refinements, the affected shared-context/context tests passed (26 tests), the App rebuilt, and all three continuation Electron scenarios passed again.

### Exact shared context

Unit checks cover checkpoint mismatch, another Project/Run, wrong attempt, stopping state, a newer check superseding an older ready review, immutable captured bytes after recheck, Agent observation facts, tampered/foreign pointing targets and continued v1 capture behavior. The existing frozen Run schema/digest regression also passes.

### Actual Electron scenarios

1. Ready review: retained Flow, keyboard node selection, attach without send, whole-step restart explanation, Current-newer notice, wide/compact layouts, draft/capture preservation across restart, zero create/start before explicit confirmation, exactly one create and one start afterward, Stop addressed to the new lease (not original Start lease).
2. Failure/recovery: unsupported engine disables check, blocked recheck removes confirmation, historical preview preserves the earlier restart plan with no Resume control, lost create acknowledgement shows unknown, refresh/restart never creates or starts a replacement execution.
3. Shared Agent view: scripted Agent reads the exact review from the acknowledged Run, points to the restart step, leaves the User's selected reuse step and new draft unchanged, opens a temporary reference view and returns, creates/starts no execution.

Visual inspection covered ready wide, compact Flow/Chat, Agent reference and unknown/historical states. Buttons are visible without hovering; labels distinguish status from planned action without depending on color. Tests use native buttons and keyboard Enter, not DOM event injection. A refresh-on-mount loop found during Electron work was fixed by reacting only to a changed review key; final scenarios passed after repair. Fixture-only missing environment and invalid synthetic Current metadata were also corrected without product bypasses.

## Execution self-check

Applied the coding checklist to the actual changed surface: Project fit and scope; native/Main/renderer ownership; feature-local modules; small display APIs; bounded versioned data; immutable identity; asynchronous polling/cleanup; explicit User execution; uncertain outcomes; old-schema compatibility; meaningful unit and real-window evidence. No independent-review verdict is claimed.

## Limits and next gate

- Real Trim Galore/FastQC continuation, new capability-bearing engine image, repeated real Stop/Resume and runtime drift qualification remain P5B-5.
- The previously qualified older engine is intentionally unavailable for continuation.
- Native record observations are cached; UI calls them “Last checked”, and refresh only queries the same request.
- Representative research-user and assistive-input comprehension sessions have not been performed. Automated keyboard checks and visual inspection do not substitute for them.
- Broader packaging and recovery work remains outside this increment. No full-repository all-platform pass is claimed; previously documented engine/module platform limitations are unchanged.
