# Go Tips

## Keep From identity on the pipeline pointer

**Context:** Gobble snapshots binds into an id-keyed engine graph.

**Tip:** A `From` handle belongs to a pipeline only when `h.task.pipe` or `h.pipe` is the composed pipeline. Task-id equality is not enough. Module reuse produces colliding dotted ids across pipelines.

**Application:** Apply one `foreignFrom` predicate at every site that records a spec or a DAG identity, including output binds.

## From readiness: FromIn on the input path, output port on the published from-path

**Context:** A task becomes ready from a `From` bind, including `FromIn` and related-file output `From`.

**Tip:** Keep FromIn on the downstream input path. An output-port `From` waits on the published from-path after the upstream task succeeds.

**Application:** In `internal/engine/run.go` `upstreamReady` after `d1319ff`, require the named upstream task to have succeeded, then check the consuming input path for FromIn, or the published from-path when `ToPort` is not an input. Do not resolve `FromIn` only against upstream outputs.

## Leftover not-started is not success

**Context:** Public `Run` returns nil only when every task succeeded.

**Tip:** A leftover `not-started` task is a failed defect. Do not treat unfinished or unpublished work as success.

**Application:** After the scheduler stops, any task that is not `succeeded` must become a `failed` defect so `Run` cannot return nil.

## Recorded Resources and Params are unused until applied

**Context:** `Resources` and `Params` are on `TaskSpec` and persist in `tasks.json`.

**Tip:** Recorded is not applied. `docker.go` does not emit `--cpus` or `--memory`. `TestRunUnparseableMemory` accepts junk. Live Strelka saw host memory (`memMb: 47345`).

**Application:** Do not treat recorded CPU or Memory as docker limits until validate parses Memory and the executor emits flags when non-zero.

## Isolate scratch is not a directory port

**Context:** Isolate already lets a tool write undeclared directories.

**Tip:** Tools may write undeclared dirs; only declared regular files publish. Strelka wrote `strelka/` then `mv` to a declared VCF.

**Application:** Reject first-class directory artifacts until a required consumer needs an index directory in place.
