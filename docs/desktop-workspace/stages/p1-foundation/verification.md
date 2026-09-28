# P1 verification

Date: 2026-09-09. Author-run implementation checks on the existing dirty repository;
no commit, release or packaged-artifact qualification. The owner approved P1 only.
See [implementation and next checkpoint](README.md).

## Final results

| Check                                                               | Result and scope                                                                                                                                                                                                                |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `npm run check` from `app/`                                         | Passed: formatting, TypeScript checks, generated schema consistency, 29 unit/contract files with **323 tests**, development build and **67 native Electron scenarios**. [Complete final log](app-check.txt).                    |
| `go test -race ./internal/appservice ./cmd/gobble-service -count=1` | Passed. Service suite 2.559s; command package has no tests. [Final service log](go-service-check.txt).                                                                                                                          |
| Focused new native Pipeline scenarios                               | Both passed in the focused run and the final full suite: registration/source/selection/restart and visible failure/retry. Native Electron with real local service and isolated profiles.                                        |
| Public contract declaration emit                                    | Passed with `--declaration --emitDeclarationOnly --noEmit false`; inspected Pipeline DTO declarations for unintended host/provider types. [Retained declaration](pipeline-declaration.txt). Owned temporary output was removed. |
| Supported Linux Plan baseline                                       | Passed actual root-package `TestBuildPlanReject`, `TestBuildPlanWaitPaths`, `TestBuildPlanWriteToKeepsPlan` selection in Linux/amd64. Exact argv, runtime image and result in [Linux evidence](linux-plan-tests.json).          |
| Visual review                                                       | Inspected final actual [source/Chat capture](pipeline-source.png) and [compact capture](pipeline-compact.png). Primary source, second image, draft, selection and compact navigation remain usable.                             |
| Source preservation                                                 | Compared original stage source hashes, historical bundles and the prior frozen engine subject. Exact results and final source identities are in [after.json](after.json).                                                       |

Host tuple: macOS 26.5.2 / arm64, Node 25.7.0, TypeScript 5.9.3, Electron 44.2.0,
React 19.2.8, Vitest 4.1.11 and Go 1.27.1. Linux Plan qualification used cached
image `sha256:bccd458a724b795ac507e3dd0be43e8431269db2819fa544784d73dc751ba22b`
with Go 1.26.8, one CPU, user `gobble`, a read-only repository mount, no network
and no Docker socket. Its temporary container was removed. This exercised Plan
tests, not an analysis task or a production pipeline.

## Behaviors covered

- Go registration: source parsing does not run `init` or require a valid return
  type; source bytes stay unchanged; definitions survive restart. Idempotent
  retries and concurrent duplicate requests produce one definition.
- Unsupported source: missing/invalid/ambiguous declarations, arguments, generic
  entries, multiple results, oversized files/folders and outside-root symlinks
  fail explicitly without publishing a catalog mutation. Cross-Project resource
  use is rejected.
- API: unknown request fields, command/path payloads, wrong Project/package
  associations and duplicate list identities are rejected at their boundaries.
  Source/Plan/execution effect routes remain unavailable.
- Catalog migration: v1 reads are non-writing; exact v1 bytes, Projects, Runs and
  prior receipts survive the first v2 publication. Archive conflict and missing
  primary with either retained backup fail without silently creating fresh state.
- Existing suite: exact references, stale source/render rejection, preserved
  evidence and draft state, Show/Return, questions, two Pane restoration, native
  lifecycle and provider protocol behavior continue to pass.

## Corrections during verification

The first unit run found the old hard-coded capability list; it was updated for
the two additive service capabilities. The first complete native run reported
65/67 passing: the foundation test expected the old preload method list, and the
PDF question test clicked Send before the asynchronous reply target reached the
composer. That host rejection preserved the draft and sent no reply. The test
now waits for the visible Reply target, then exercises changed evidence. No
production question or sender behavior was changed to hide that result.
[The first complete-run log](app-check-first.txt) is retained separately; the
final full rerun passed all 67 scenarios.

An earlier formatting check entered a pre-existing Jupyter qualification virtual
environment and tried to parse third-party templates. Added only
`qualification/jupyter/.venv/` to the App formatter exclusions; owned qualification
source remains checked. Vendor files and dependencies were not edited. The original
ignore contents and hash are preserved in [additional-before.json](additional-before.json).

During the bounded catalog self-review, the new v1 archive exposed one missing
recovery guard: primary absent with only the v1 archive remaining could start a
fresh catalog. The open path now checks both backup names; the existing recovery
test covers each independently. Final Go race and complete App checks ran after
this correction. No production code changed after the final full run began.

## Limits and retained evidence

P1 qualifies registration, source reading and presentation ownership. It does not
claim Agent source authoring, an immutable multi-file revision, Go validation,
analysis execution, Run-to-definition inference, rename/unregister or automatic
source rebinding. Those require consumers and separate stage contracts.

The new native tests use an owned declaration fixture. Existing provider scenarios
use controlled Codex fixtures; this stage made no live signed-in Agent request.
The normal App profile, real analysis Projects/Runs and synced `sources/` were not
used for mutation tests. There is no packaged App result.

The previously recorded native macOS root-library compile failure
(`undefined: useProjectOwner`) is a target-support limitation, not repaired by
this slice. The two baseline packed-executable failures in
[Stage 6](../06-integration-evidence.md) remain separate unresolved engine work.
The focused passing Linux tests do not establish that the entire engine suite or
packaged runtime passes. Existing engine source was preserved.

The [before manifest](before.json) binds 381 stage inputs to the approved-review
subject. [after.json](after.json) identifies modified/new source, hashes final
inputs and records unchanged engine/schema/package evidence. Stage documentation
and logs are recorded separately from source hashes, avoiding self-referential
hashes. The frozen prior review, subject and plan artifacts were not rewritten.
