# P2B-2.1 verification

2026-09-12, author mode. Module `github.com/HahyeonJeon/gobble`; language1.26,
selected go1.27.1 on darwin/arm64. No dependencies added. User-authorized local
implementation/tests use ordinary build caches and disposable profiles.

## Passed storage and consumer evidence

- `go test -race -json ./internal/appservice ./cmd/gobble-service`: 55 top-level
  service tests passed (81 including subtests), no race report. See [raw results](go-tests.jsonl).
  Two opt-in Go live-inspection tests were skipped, not counted as passed;
  `cmd/gobble-service` has no tests. Engine-integrated Electron evidence is separate below.
- `go vet ./internal/appservice ./cmd/gobble-service`: passed.
- `npm --prefix app run typecheck`: contracts, Main, preload, renderer and tools passed.
- `npm --prefix app test`: all345 tests in33 files passed. Includes actual service
  process storage/restart/tombstone integration and process-dependency architecture checks.
- `npm --prefix app run schema:check`, `format:check`: passed. Bundle21 generated;
  historical bundle20 unchanged. Changed Go sources formatted.
- `npm --prefix app run build`: native service plus Electron Main/preload/renderer passed.
  Service rebuilt after the final private draft-sorting helper relocation.
- Targeted normal Electron foundation/Pipeline registration suite: eight passed.
  Two engine-opt-in cases initially skipped and run separately with explicit pins.

New storage cases cover generation races, cross-Project access, explicit input clear,
metadata-only observation and missing/outside-symlink rejection, failed-save behavior,
request identity replay/conflict, retained tombstones, current isolation, historical
adoption outcome after migration, exact v3 archive preservation/conflict and mixed
origin rejection. The managed-origin test installs a catalog fixture; it does not
claim a newly authored first Pipeline was engine-qualified.

The initial focused App test run failed an old capabilities-array expectation after
adding three actual draft mutations. Its assertion was updated with the exact new
surface; the full345-test run passed afterward. No production authorization check
was loosened to satisfy the test.

## Engine-integrated regression

Flow: `GOBBLE_FLOW_LIVE=1` with retained fixture runtime
`sha256:1d4dccd578363c104aa65127e33a4d924f1f2e4a15ae3d9e6ee8ab4c46695adb`
on linux/amd64. Actual checked branching flow, keyboard details, preservation of
unsent Chat and other Pane, and historical restart passed in56.5seconds.

B review uses `GOBBLE_PROPOSAL_LIVE=1` and `GOBBLE_REVIEW_IMAGE` pinned to
`sha256:effd239270cd58dc27d97b32b884ab21569a08e26b163d61736ab50c2a85f78d`
on linux/amd64, daemon `bfc24268-2f66-46d9-a1d1-969af8291bec`.
The Agent peer is a deterministic fixture; source check is real Gobble. This is not
an additional signed-in LLM qualification.

The first B run passed source proposal/check and visual change assertions, then the
Agent reference fixture failed before adoption (`Comparison reference refused`).
Inspection identified that publishing shared references requires a focused Project
window, while retained comparison reads do not. The test now explicitly focuses its
owned window and checks that prerequisite before sending the reference request;
the peer includes the full rejected result in any future failure. No production
window-availability guard was weakened. This is a diagnosis of the test precondition;
the initial generic fixture error alone did not capture the precise rejection code.
The catalog assertion was also updated from3 to4 and now verifies Pipeline-keyed
Current and no creation drafts from the existing-Pipeline refinement scenario.

Final B rerun passed in3.4minutes: actual checked proposal, exact User discussion,
Agent comparison mark, User adoption, preserved imported source/unsent text and
restart all passed. The persisted v4 catalog has one Pipeline-keyed Current and
zero Runs. Combined Electron evidence is ten distinct passed scenarios (eight
normal, one actual flow, one actual B review); the initial skipped/failed attempts
above remain recorded rather than relabeled as passes.

## Limits and handoff

This slice exposes native draft storage, not live creation UI/Agent tools, whole-new-
flow qualification or first-version creation adoption. No Run controls, packaging,
minimum-toolchain/Windows qualification or root-engine native-macOS support claim.
Prior unrelated packed-CLI test gaps remain unchanged. The next checkpoint is the
Gobble-owned scaffold and complete creation qualifier, using these generation and
ownership boundaries.

Compatibility: migration supplied for catalogs1/2/3; bundle21 consumers updated;
Workspace17/shared toolset11 unaffected. Recovery preserves historical bytes and
fails closed on archive/write uncertainty. No automatic profile rollback or deletion.

**author mode used no credentials and performed no external mutation**. Test sockets,
local Docker checks, caches/build artifacts and disposable test files are local
validation effects. No research Project, remote repository or synced `sources/` file
was modified; no commit/push/publication. Changed paths and source hashes are in
[before](before.json) and [after](after.json); inherited dirty work remains preserved.
