# compose-pipeline backlog

## One path model

**Backlogged at:** 2026-08-17T12:18:21Z

**What:** Own PathSpec render, classify, and restage in one implementation used by both the library and `internal/engine`.

**Why backlogged:** The session locked two packages and mitigated drift with Render and classify parity tests instead of merging the copies.

**Context:** `PathSpec` lives in package `gobble`. `engine.Path` is a snapshot copy so the engine does not import `gobble`. `TestRenderAgreesWithSnapshot` and `TestClassifyAgreesWithSnapshot` pin agreement. Group and wait-path rules landed in both copies. The copies stay.

## Merge branch list

**Backlogged at:** 2026-08-17T12:18:21Z

**What:** Make `Merge(..., branches)` constrain the graph, or remove the unused stored list.

**Why backlogged:** The constructor is locked. This session documented that the list is author intent and that edges come from `Bind.From`.

**Context:** `Merge.branches` is written and never read. `snapshotNodes` walks `merge.children` only. The static-core slice kept documented unused author intent.
