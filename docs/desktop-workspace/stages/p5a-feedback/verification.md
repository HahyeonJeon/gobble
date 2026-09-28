# P5A verification — p5a-feedback-20260912

Requesting owner: Electron Development, construction/behavior exchange. Consumer: Project owner. Record date: 2026-09-13 Asia/Seoul. Subject is the inherited P4 tree plus [P5A paths](changed-paths.json); final file digests are recorded in source-sha256.json. This is development-app evidence, not release or scientific validation.

## Target and observations

macOS arm64, Electron 44.2.0, Node 25.7.0, TypeScript 5.9.3, React 19. Installed Playwright runner, one visible foreground Electron instance and a disposable private profile. Native service is built locally. Docker Desktop uses the existing qualified linux/amd64 engine `sha256:5d7b7247b407844149bc80173e8da2bf656077b827dd3f2ae9149ee490258e76`; no image or engine policy is changed. Inputs are 100 synthetic 80-base FASTQ records. Native file chooser selection and Agent responses are deterministic test doubles; native pipeline checking, evidence storage, adoption, preparation, data checks, admission and Trim Galore/FastQC execution are real.

| Claim | Lowest sufficient layer | Result |
| --- | --- | --- |
| Wrong Project/Pipeline/Run, missing capture, hash, duplicate or oversized evidence is refused | Native service tests | Passed |
| Proposal replay retains its original sent-message identity; changed-origin replay conflicts | Native service test | Passed |
| Exact Current selects the saved proposal relation; launch checksum covers provenance | Native service tests | Passed |
| Main derives only from sent evidence and verifies stored bytes; mixed origins/stale session/unavailable evidence fail | Main unit tests | Passed |
| Agent cannot inject follow-up metadata through its proposal arguments | Closed tool schema test | Passed |
| Input equality/disagreement and exact attempt coordinates remain truthful | Contract tests | Passed |
| Existing affected evidence/proposal/adoption/launch behavior | 39 Vitest files / 369 tests | Passed |
| Native race and static checks | App-service package | Passed |
| Types, current schema and application build | Construction checks | Passed |
| Full linked change and two analyses, then restored evidence preview | Real Electron + real synthetic engine, deterministic Agent | Passed: electron-live-final.log, 6.2 minutes |
| Compact desktop visual review and missing capture recovery | Actual Electron, 1600 × 1000 and 1100 × 800 native window sizes | Passed: electron-retention-final.log, 5.6 seconds |
| Real provider, representative researcher/assistive-input usability, packaged/installed/update paths and Windows/Linux desktop | Outside this development acceptance target | Not run |
| Same-design Resume and reuse | Explicitly excluded P5B | Unsupported target or claim |

## Failures preserved and corrections

1. electron-live.log / test-results/p5a-feedback: test defect. P4's restart leaves the provider dormant; the extension tried to send without refreshing the test connection. The existing explicit Account → Refresh connection flow now runs before the follow-up.
2. schema check and electron-live-2.log: product defect in the in-progress contract refinement. Importing the broad evidence validator created a schema initialization cycle. The new contract now checks its own capture association invariants without a runtime import cycle. Schema/build and dependent tests were rerun. The non-started test Electron process was terminated; no analysis had begun.
3. electron-live-3.log / test-results/p5a-feedback-3: test defect. The deterministic protocol peer omitted pipeline-feedback from its scenario dispatch list and returned a generic completed response. The exact sent Run evidence was present in persisted Chat. Added the missing dispatch entry; no product behavior was weakened.

4. electron-live-4.log / test-results/p5a-feedback-4: test defect. After adoption the retained preparation correctly becomes Earlier review, with Prepare again. The test expected the first-use Prepare run label. Updated the action selector to the retained-review state; visual adoption and original evidence preview had passed.

5. electron-live-5.log: full workflow reached two successful Runs, saved association and restored preview. The final compact assertion incorrectly expected Chat and Workspace simultaneously below the existing 1120px breakpoint. The test now operates the existing region switch. The outer failure recorder also tried to read the already closed original page, masking the compact failure; error capture now preserves the actual exception.
6. Visual review of feedback-restored.png found a product defect: expanded provenance exceeded the fixed 240px retained-Flow box and overlapped tasks. The Flow region now sizes with its content, clips/scrolls within a bounded share of the work area, and reserves its graph height. Final build plus restored wide/compact checks passed; the complete workflow rerun uses this corrected build.
7. electron-retention.log: test defect. The unavailable preview retains its descriptive footer, so absence must assert no captured content element, not no footer label. The corrected case verifies the error, zero content, restores the same blob, retries and checks the actual task text. electron-retention-2.log exposed the same compact-switch expectation as item 5. Both are corrected in the passing retention-3 run.

Failures are not counted as passed; only the corrected candidate's completed reruns support the results. Prior test profiles and logs remain local for diagnosis.

## Actual desktop visual review

The passing final retention run shows the changed Trim step and 40/20 bp values in B comparison, the earlier design notice on Analysis 1, exact retained evidence, and Analysis 2 after restart. Missing capture displays an explicit error and never substitutes the new Run. The 1100px compact layout uses the existing Workspace/Chat region switch; these regions are intentionally reviewed separately. Buttons are visible before hover. Flow and tasks no longer overlap when provenance is opened; the Flow region has its own bounded scrolling.

An inherited navigation limitation remains: after creating a Run, the sidebar list refreshes on its existing Refresh runs action or reopening the Project/app. Open Run and retained Run tabs provide immediate navigation; P5A does not introduce a separate catalog subscription. Representative-user and full assistive-input evaluation remain unclaimed.

## Final acceptance

Both final Electron cases passed on the corrected build: electron-live-final.log and electron-retention-final.log. The latter also asserts separation of Flow/task bounds and proper clipping of expanded content. Original capture bytes were restored after the missing-content case. Screenshots and actual launch review results are retained under [evidence](evidence/feedback-result.json). No test process or actual analysis remains running from this acceptance run.

Final construction: typecheck.log, schema.log, build.log and format-check.log. Behavior: unit-all.log (39 files / 369 tests), unit-followup-final.log (6 focused tests after capture validation refinement), native-race.log, native-vet.log, and the two final Electron logs. Environment is recorded in environment.json; P5A source/test identity is source-sha256.json. Earlier logs preserve failures without replacing their classification.
