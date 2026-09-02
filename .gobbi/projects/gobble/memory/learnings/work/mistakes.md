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

**Correction:** The wrapper must outlive the inner `timeout N`, including the `echo $? > exit.txt` tail. A 300s outer wait can kill after Claude already wrote the result when the inner timeout is 900s. Keep the wait at least as long as that whole command. Capture EXIT in a file written by the same long command that runs the Partner CLI.

## Do not average conflicting prior-art studies

**Context:** An nf-core-as-written study and an isolate-keep Nextflow/Snakemake study disagreed.

**Mistake:** Merging them into one feature list.

**Correction:** They answer different questions. Keep isolate. Take from nf-core only declared regular files plus declared literals.

## Hermetic go test is not live Docker evidence

**Context:** The first check is network-free `go test ./...`. Product live suites
have separate fixture, image, Docker, registry, and command prerequisites, and
some use hermetic command boundaries by design.

**Mistake:** Treating a cached or fresh hermetic pass as proof that Docker,
registry access, pinned third-party commands, or scientific outputs ran. A skip
or a command double is not a live tool pass.

**Correction:** Run the exact separately authorized live package with
`-count=1`, wait for it to exit, and report its stated evidence boundary. When
live Docker, network, or third-party commands were not run, record that limit
instead of upgrading the hermetic result.

## Do not spawn general-purpose as Partner

**Context:** Gobbi Partner review or evaluation on Claude Code.

**Mistake:** Spawning `general-purpose` as Partner. That is not the Partner Manual command.

**Correction:** Partner is `claude` with the Partner Manual command, one writing path, and a preimage check.

## Executor DONE is not a running live test

**Context:** Workflow execution of live-tagged `go test` commands that can run for minutes.

**Mistake:** Reporting executor DONE while a live `go test` is still running. The handoff then claims completion before the command exits.

**Correction:** Wait for the live command to exit before the handoff. A still-running live test is not completion.

## Start DISCUSSION Partner before the plan writer

**Context:** Workflow Planning DISCUSSION that uses a Partner grouping pass.

**Mistake:** Starting the plan writer before the DISCUSSION Partner finishes. The writer then groups without independent Partner input.

**Correction:** Finish the DISCUSSION Partner before the plan writer starts. Planning Partner grouping is an input, not a later patch.

## Concurrent live Docker evaluations flake and hide Defects

**Context:** Live-tagged product or install evidence shares Docker, registry,
image, fixture-cache, and host resources.

**Mistake:** Overlapping live suites can make resource interference look like a
product defect. Collapsing failure to a count also removes the structured unit
and path needed to classify it.

**Correction:** Run the named live package exclusively. On failure, print each
`gobble.Error` Defect with code, unit, paths, and message. Do not classify the
first concurrent failure as a product defect without an exclusive reproduction.
