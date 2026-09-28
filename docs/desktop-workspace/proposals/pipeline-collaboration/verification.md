# Pipeline collaboration — Verification record

Date: 2026-09-09. This is evidence for research/design review, not acceptance of an
implemented pipeline collaboration workflow.

## Bound identities

- [Source manifest](subject.json): 523 files, aggregate SHA256
  `ec2b38e675f9b6e535839b8fd9a436f7402f6a71ab970ebfff19f9f9ee6d1821`.
- [Added dependency evidence](dependency-evidence.json): containerenv and canonical
  checklist identity, added after the unaided review was locked.
- HEAD: `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`; existing dirty/untracked
  source is part of the manifest. HEAD alone does not identify this App.
- macOS 26.5.2 (25F84), arm64; Go 1.27.1; native CGO_ENABLED=1; Node 25.7.0.
- app/package.json and lock are source-bound: Codex 0.153.4, Electron 44.2.0,
  TypeScript 5.9.3, Vitest 4.1.11. These are review environment facts, not advice
  to upgrade or broaden support.

## Executed checks

Working directories below are repository root unless explicitly marked App.
Terminal completions are summarized; this document does not claim to be a full raw
test transcript.

| Exact command                                                                                       | Observed result                                                                                                                            | What it establishes                                                                                   |
| --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| `go test ./internal/appservice`                                                                     | Pass, cached.                                                                                                                              | Existing result only; followed by the fresh command below.                                            |
| `go test ./internal/appservice -count=1`                                                            | Pass, 1.078s.                                                                                                                              | Fresh service test execution, including the existing read-only capability boundary.                   |
| `go test . -run 'TestBuildPlan(Reject\|WaitPaths\|WriteToKeepsPlan)$' -count=1`                     | Build failed: `internal/containerenv/runtime.go:113:12: undefined: useProjectOwner`.                                                       | Requested root tests did **not execute** on darwin/arm64.                                             |
| App: `node_modules/.bin/vitest run desktop/tests/codex.test.ts desktop/tests/collaboration.test.ts` | 2 files, 32 tests passed, 6.03s.                                                                                                           | Focused protocol/policy/collaboration behavior with test doubles; not a live Agent authoring session. |
| `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c -o <owned-temporary-path>/gobble.test .`          | Compile succeeded; exact argv and output in [linux-compile.json](linux-compile.json). Artifact removed with its owned temporary directory. | Linux/amd64 compilation only. No Linux tests or analysis tasks executed.                              |
| `git status --short internal/containerenv` plus SHA256 of current and HEAD runtime.go               | No modifications; both hashes `66a8de00f30f69d1f68840f8e5f18bf121804feaef413f43ab7de8a268d0f6d7`.                                          | The observed target-specific compile issue predates this review.                                      |

## Reconciliation and limits

The root README specifies Go/Gobble execution in Linux/amd64 containers.
Accordingly, the native-root failure is a recorded host-testing limitation, not
proof that the supported engine runtime or the independent native App service is
broken. The Linux compile result does not erase that failure and does not establish
Linux runtime behavior. Do not add host analysis execution merely to make an
irrelevant test target pass.

No complete App suite, native UI scenario, signed-in Agent turn, actual new
source-write flow, launch/Stop/Resume, packaging or scientific analysis ran during
this review. Stage-6 evidence remains historical; its two packed-runner failures
are not resolved by these checks. No dependency was installed or upgraded.

Production code and governing source bytes are preserved. Final document/link,
plan-index/leaf/dependency and source-preservation checks are recorded in
[handoff-checks.json](handoff-checks.json). That record is document validation,
not a production test result.
