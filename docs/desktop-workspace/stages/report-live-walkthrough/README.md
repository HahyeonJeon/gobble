# Live report collaboration walkthrough

Owner approved on 2026-09-28. Completed qualification with a reproduced navigation defect. See [result, evidence and next design boundary](verification.md). This is not independent review acceptance or scientific validation.

## Intent and reach

Researcher and actual Agent create and execute a synthetic single-FASTQ workflow, open the attributed FastQC report, discuss original charts, and request a checked proposal through the UI. Create a fresh temporary Project/profile and Run. Reuse only the known signed-in App credential locally without displaying it; remove the temporary credential copy at closure. Delete no earlier Project, Run or history. No commits or publication.

Use existing creation, preparation, launch, report and comparison UI, installed report-capable engine and actual bundled Agent. Only native folder chooser selection is supplied by automation. Source and schemas remain unchanged unless a reproduced in-scope defect needs repair.

## Ordered guide

1. Verify prerequisites, generate 100 synthetic reads; actual Agent creates Trim Galore → FastQC. Inspect comparison and adopt this test design.
2. Prepare/check/start through App; verify successful producer; open report from step, attach and preview.
3. Actual Agent reads original charts and points to whole report. Record calls, coverage and UI; inventory alone is not visual evidence.
4. Request one explicit setting change using Current proposal controls, inspect B comparison; retain proposal without adopting or rerunning. Check compact layout and restart retention. Stop with evidence and limitations.

## Findings and failures

- Initial discovery printed oversized prior evidence JSON, without credentials. Subsequent reads use selected fields.
- Initial harness creation used app/ as cwd with root-relative paths. Moved only the newly authored harness into its intended stage and removed those new empty directories; no product files changed.
- Existing creation tests use a real engine but simulated Agent. Reuse UI steps only. Read-only subagent live_report_advice confirmed this and recommended current-message reattachment and separate observation before pointing.
- First interactive harness launch used closed stdin. Closed its owned Electron process and removed its temporary credential copy, then relaunched with a PTY. No Agent message or Run had started.
- Harness status originally called workspace.connect, which creates a new renderer session and invalidated the UI session. This caused the unavailable-View error. Changed observation to workspace.read and relaunched the same profile; this is a harness failure, not a production defect. Also replaced an eval reference to a TS-transformed imported helper with direct UI calls.
- The qualification driver used a 45-second wait for preparation; the real compiler was still running. Preserved the timeout screenshot and continued waiting for the same operation; no new preparation or lowered checks. The installed engine source manifest differs only in runtime packaging and native App-service report files; execution-core files match. Main uses the current native service build.
- Live report turn delivered and displayed 8 images; all model-facing PNG hashes equal original saved bytes. Actual shared report pointer succeeded after fixing the harness session read. The earlier window-unavailable observation cannot be attributed solely to desktop focus because that harness bug was active.
- One oversized interactive PTY command hit macOS line length; it was erased before execution and resent as short steps. Durable action log begins with the preparation/account-refresh stage.

- Maximizing a Pipeline comparison then trying to scroll to its changed value failed because the presenter returned to Current. Unlike the earlier harness session error, this is a reproduced product navigation defect. Compact Chat return and restart show the same loss; underlying data remains exact.

- A stable compact recheck initially failed to resolve @playwright/test from the docs directory. Removed that unnecessary direct dependency and used existing launch support plus native locator assertions; preserved the failed import log.
- Final subagent discussion recommends lifting the existing FlowNavigation owner above presentation remounts for session-only retention. This is smaller than immediately adding persisted Workspace fields. Restart-location restoration remains a separate decision; saved data already survives.
