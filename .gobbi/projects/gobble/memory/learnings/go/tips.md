# Go Tips

## Keep From identity on the pipeline pointer

**Context:** Gobble snapshots binds into an id-keyed engine graph.

**Tip:** A `From` handle belongs to a pipeline only when `h.task.pipe` or `h.pipe` is the composed pipeline. Task-id equality is not enough. Module reuse produces colliding dotted ids across pipelines.

**Application:** Apply one `foreignFrom` predicate at every site that records a spec or a DAG identity, including output binds.

## From readiness comes from the plan wait path

**Context:** A task becomes ready from a `From` bind, including `FromIn`, related-file output `From`, and Group members.

**Tip:** Resolve the wait-path set at `BuildPlan`. `upstreamReady` reads `Edge.Wait` only. Do not guess from `ToPort`. FromIn waits on the consuming input path. Output-port `From` waits on the published from-path. A Group edge waits on every named member path.

**Application:** An empty or unproducible wait set is `never-ready` at plan. After the named upstream task succeeds, every wait path must be a regular file.

## Leftover not-started is not success

**Context:** Public `Run` returns nil only when every task succeeded.

**Tip:** A leftover `not-started` task is a failed defect. Do not treat unfinished or unpublished work as success.

**Application:** After the scheduler stops, any task that is not `succeeded` must become a `failed` defect so `Run` cannot return nil.

## Resources apply; Params stay unused

**Context:** `Resources` and `Params` are on `TaskSpec` and persist in plan JSON and `tasks.json`.

**Tip:** Validate parses Memory. Non-zero CPU and Memory emit `--cpus` and `--memory` and consume remaining admission. Zero is unspecified. `Params` stay recorded and unused. They are not interpolated into Command or Script.

**Application:** Treat CPU and Memory as real limits. Do not treat Params as command inputs.

## Isolate scratch is not a directory port

**Context:** Isolate already lets a tool write undeclared directories.

**Tip:** Tools may write undeclared dirs; only declared regular files publish. Strelka wrote `strelka/` then `mv` to a declared VCF.

**Application:** Reject first-class directory artifacts until a required consumer needs an index directory in place.
