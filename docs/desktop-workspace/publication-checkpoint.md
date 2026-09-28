# Desktop collaboration publication checkpoint

Date: 2026-09-29. The owner requested ending the current implementation session, committing and pushing its accumulated work to GitHub. Target: `origin/codex/project-workspace-design`, based on `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`. No develop merge, release, installer, or further feature is included.

## Included scope

1. Gobble pipeline inspection/creation/review, exact execution preparation/admission, Stop/continuation/history, output evidence, native service and corresponding tests.
2. Electron/React/TypeScript Project workspace; shared Flow/Chat, visual proposals, execution/continuation, saved FastQC reports, exact evidence and session comparison navigation; contracts, tests and desktop CI.
3. Accepted design, stage verification, synthetic qualification evidence and current remaining-work inventory.

Source and preserved schemas remain authoritative. The 30 schema bundles total about 233 MB uncompressed and are retained as documented compatibility records; no individual candidate file exceeds 16 MB. Dependencies, builds, full profiles and live credentials are not included. Synthetic evidence can include original developer machine paths and recorded temporary fixture locations. They are provenance, not portable runtime configuration. Disposable Playwright `.last-run.json` markers are ignored without deleting local copies.

The read-only publication audit found no credential/private-key payload or full browser/profile directory among candidate files. Apparent token matches were test-name fragments; execution ownership tokens are non-authentication container identities. Automated scans are bounded checks, not a claim of comprehensive security review.

## Closure corrections

- Updated the App/desktop checkpoint and remaining-work inventory to include the completed report and navigation repair, with current Workspace21/bundle30/catalog5/toolset15 versions.
- Formatted two existing JSON test/storage fixtures to satisfy the standard check. Parsed JSON values were asserted identical before/after; no schema or stored-data change.
- Historical stage evidence is retained with its original pass/fail limits. Earlier failures and opt-in live results are not relabelled as new successful executions.
- The legacy package `./schema` export still points to v1; the current generated bundle is v30 and documented by explicit path. Changing that public export is a separate compatibility decision.
- A closure script initially resolved the repository relative to app/; it stopped on a missing file. The misplaced ignore entry was removed and applied at the repository root. No production file was changed by that failed step.

## Verification

Final commands and their outcomes are appended below before commit. All development artifacts remain distinct from packaged/distribution acceptance. Existing engine packed-executable qualification gaps and future reading UX/packaging work remain visible in [remaining work](remaining-work.md).

### Final-suite findings

The first complete App check passed formatting/types/schema, 397 unit tests and the build; Electron reported 63 passed, 15 failed and 12 explicit opt-in skips. The failures exposed a stale bridge-key expectation (missing continuations) and native macOS focus prerequisites in older shared-view tests. The existing focusWindow helper is now used immediately before scripted Agent gates; no product authority condition is relaxed. The suite's original output is retained in stages/publication-check/app-first.log.

The first Linux full Go run reproduced the two recorded packed-executable failures. It also exposed architectural scanners misclassifying external test imports and nested/generated fixture projects as production dependencies. Those scanners now inspect production .go files of the module; a fixture verifies that a real forbidden production import still fails. Both fixes affect tests only. A mistaken attempt to run these architecture packages on macOS hit the already documented useProjectOwner build limit; final engine checks use Linux. No native engine support or packed-executable repair is claimed.

### Final verification outcomes

- Standard App check: formatting, process/tool TypeScript, schema consistency, all 397 unit tests and production development build passed. Initial full Electron run: 63 passed, 15 failed, 12 skipped (opt-in actual-runtime/signed-in scenarios).
- After the test-harness corrections, all 24 affected Electron scenarios have passing results across the scoped runs: dependency/foundation/Notebook/observed-reference/PDF/shared-context in electron-repair.log, Notebook/PDF in electron-final.log, questions in questions-final.log. An intermediate PDF failure required native focus before Send (its peer has no gate); the question peer was then gated after Send settled. Failed attempts are retained. This is not a claim that the initial full suite was green or that skipped live scenarios ran again.
- Native race checks passed for appservice, service/container launchers and collaboration contract packages; native service/launcher vet passed.
- Linux full suite completed with four failing checks: the two architecture checks subsequently repaired and verified, plus the two existing CLI packed-executable cases. Other reported package results passed. Final Linux checks passed for assets, architecture, engine, executor and preparation. The source was mounted read-only into the previously qualified Linux/amd64 engine with networking disabled.
- The native architecture-check attempt hit the documented macOS root-library compile limitation. It is retained separately from Linux evidence.
- Source formatting: changed Go files are gofmt-clean; final App format check passes. Staged whitespace checks exclude the original FastQC HTML fixture and verbatim historical evidence, whose exact bytes are preserved intentionally.

Logs are in [publication-check](stages/publication-check/). No installation, release or fresh biological analysis was performed. The successful scope remains the bounded synthetic single-FASTQ development workflow.

## Commit boundaries

- Engine/native foundation: `11258b6` (`feat(engine): add pipeline preparation and continuation lifecycle`).
- Desktop workspace and CI: `cf6be35` (`feat(desktop): add the project workspace application`), including test-harness corrections.
- Accepted design/evidence/checkpoint: separate documentation commit.

Push the three commits together to the existing local topic branch's matching GitHub ref. The final chat reports the exact published tip after the remote check; this document does not pre-claim successful network publication.
