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

**Tip:** Tools may write undeclared dirs; only declared regular files publish. Strelka wrote `strelka/` then `mv` to a declared VCF. STAR and Bismark declare index and align outputs as regular files (Group members); PathSpec.Dir is the parent folder token, not a directory port.

**Application:** Reject first-class directory artifacts until a required consumer needs an index directory in place. Do not add directory ports for STAR genomeGenerate or Bismark genome preparation.

## Claim occupy with a lock plus owner record

**Context:** After `Release`, `.gobble/run.json` stays.

**Tip:** Do not use `O_EXCL` on `run.json` to claim occupancy. Claim with a lock file plus the owner record.

**Application:** `Run` and `Resume` after Release use `occupy.lock` and write the owner on `run.json`.

## Pin cache keys include content

**Context:** Package `assets` caches pinned downloads under `CacheDir`. Distinct pins can share a basename (`test_1.fastq.gz`).

**Tip:** Cache path is `CacheDir/<sha256[:16]>/<Name>`. `Pin.Name` remains the workspace basename. Fetch into that path with tmp+rename.

**Application:** Collision tests must cover two pins with the same Name and different hashes.

## Isolate restage copies Source onto dest

**Context:** A pipeline-input bind may restage so dest Dir differs from the authored From path.

**Tip:** Record `IO.Source` (and `IOMember.Source`) when the From rendered path differs from dest. Empty Source keeps Path as both. Isolate copies `workspace[source]` onto `isolate[dest]`. Plan Wait, `checkInputs`, and identity hash the source file.

**Application:** Bismark and BWA can restage a FASTA out of `in/` into a dest Dir the tool writes beside.

## Staged replace never unlinks a published dest

**Context:** Resume may replace an authorized dest after a rerun.

**Tip:** Never unlink, truncate, or overwrite a published dest in place. Exclusive-create a temp, write all bytes, then rename over the dest after complete isolate outputs.

**Application:** A failed new attempt that never renamed leaves the prior dest in place.
