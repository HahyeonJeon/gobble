# Work Mistakes

## Treating a checked evaluation item as optional

**Context:** Gobbi checklists mark an item when the problem is present.

**Mistake:** Issuing PASS, or carrying the item as a note, while correctness, verification, or documentation-contract boxes are checked.

**Correction:** A checked item in those categories is a Problem. Optional Improvements and Strengths do not cancel it.

## Sending Partner How discussion to files outside the session

**Context:** Partner Claude Code How discussion under `--safe-mode`.

**Mistake:** Pointing the partner at worktree or repository files outside the session directory, or giving only `git show --stat` plus a few file bodies. `--safe-mode` denies those reads, so the partner returns NEEDS_CONTEXT and writes no result file. A partial `commit.show.txt` leaves tests Unverified even when the partner can still find defects in the files that were present.

**Correction:** Copy or write every required source under the session root and name only those session-local paths in the Partner brief. The brief must include every file the partner must cite. Put every file the partner must judge in the session, or `git show` the full commit. Do not use a partial `git show`. This review missed `compose_test.go`.

## Waiting under 300s for a Partner CLI and losing EXIT

**Context:** Partner Claude Code runs that last several minutes.

**Mistake:** An observer wait under 300 seconds dies while the Partner process continues. The wrapper then loses EXIT. A 120s kill can leave stdout `Execution error` with no report.

**Correction:** Keep the wait at least 300 seconds. Long design reviews may need about 900 seconds. Capture EXIT in a file written by the same long command that runs the Partner CLI.

## Do not average conflicting prior-art studies

**Context:** An nf-core-as-written study and an isolate-keep Nextflow/Snakemake study disagreed.

**Mistake:** Merging them into one feature list.

**Correction:** They answer different questions. Keep isolate. Take from nf-core only declared regular files plus declared literals.

## Cached go test is not a live Docker e2e

**Context:** First-check is hermetic `go test ./...`. Live tests use build tag `live` and fail closed without Docker. Go may cache a prior package result.

**Mistake:** Treating a cached `go test ./...` as proof that a live Docker WGS e2e ran, or treating a skipped live test as pass.

**Correction:** First-check cannot skip for Docker. Live proof is `go test -tags=live` (WGS assay executes `assets.WGS()`). A skip is not a live pass. Cached ok is not a fresh live run.
