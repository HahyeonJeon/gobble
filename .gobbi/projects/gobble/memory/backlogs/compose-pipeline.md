# compose-pipeline backlog

## One path model

**Backlogged at:** 2026-08-17T12:18:21Z

**What:** Own PathSpec render, classify, and restage in one implementation used by both the library and `internal/engine`.

**Why backlogged:** The session locked two packages and mitigated drift with Render and classify parity tests instead of merging the copies.

**Context:** `PathSpec` lives in package `gobble`. `engine.Path` is a snapshot copy so the engine does not import `gobble`. `TestRenderAgreesWithSnapshot` and `TestClassifyAgreesWithSnapshot` pin agreement.

## Merge branch list

**Backlogged at:** 2026-08-17T12:18:21Z

**What:** Make `Merge(..., branches)` constrain the graph, or remove the unused stored list.

**Why backlogged:** The constructor is locked. This session documented that the list is author intent and that edges come from `Bind.From`.

**Context:** `Merge.branches` is written and never read. `snapshotNodes` walks `merge.children` only.

## Same-task output From

**Backlogged at:** 2026-08-17T12:18:21Z

**What:** Decide whether a related-file `From` on the same task is a cycle or a valid derive.

**Why backlogged:** The cycle walk includes output binds. The locked text does not settle the same-task case. This session documented that `From` should name another task.

**Context:** The workflow-case index bind uses `bam.Append(".bai")` rather than `From` of the same task.
