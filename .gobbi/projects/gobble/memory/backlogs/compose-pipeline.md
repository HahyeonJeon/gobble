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

**Context:** This session locked same-task `From` as a cycle (D1). BAI is a second task. D5 A fixed readiness for output-port `From`. The later answer is a sidecar group, not same-task `From`. The workflow-case index bind uses `bam.Append(".bai")` rather than `From` of the same task.

## Sidecar group of regular files

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** One artifact as N declared regular-file paths, with one bind covering the group.

**Why backlogged:** This session paid an authoring tax of one bind per sibling and used BAI as a second task. That D5 class is not a first-class group.

**Context:** Synthesis rank 1. Five BWA siblings and BAM+BAI still need one bind each. A group is not a directory and not related-file `Ext`.

## Named Script XOR Command

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Make `sh -c` a named Script XOR Command. Engine may supply `set -eu`. No interpolation.

**Why backlogged:** The Strelka body is already a script smuggled as argv[2]. Literal Command stays the default.

**Context:** Synthesis rank 6. Do not make bash the only body.
