# Go Tips

## Keep From identity on the pipeline pointer

**Context:** Gobble snapshots binds into an id-keyed engine graph.

**Tip:** A `From` handle belongs to a pipeline only when `h.task.pipe` or `h.pipe` is the composed pipeline. Task-id equality is not enough. Module reuse produces colliding dotted ids across pipelines.

**Application:** Apply one `foreignFrom` predicate at every site that records a spec or a DAG identity, including output binds.

## From readiness comes from the plan wait path

**Context:** A task becomes ready from a `From` bind, including `FromIn`, related-file output `From`, and Group members.

**Tip:** Resolve the wait-path set at `BuildPlan`. `upstreamReady` reads `Edge.Wait` only. Do not guess from `ToPort`. FromIn waits on the consuming input path. Output-port `From` waits on the published from-path. A Group edge waits on every named member path.

**Application:** An empty or unproducible wait set is `never-ready` at plan. After the named upstream task succeeds, a File or Group wait path must be a regular file. A Tree wait path is ready when the declared directory holds dest `.gobble-tree.json`.

## Leftover not-started is not success

**Context:** Public `Run` returns nil only when every task succeeded.

**Tip:** A leftover `not-started` task is a failed defect. Do not treat unfinished or unpublished work as success.

**Application:** After the scheduler stops, any task that is not `succeeded` must become a `failed` defect so `Run` cannot return nil.

## Resources apply; Params stay unused

**Context:** `Resources` and `Params` are on `TaskSpec` and persist in plan JSON and `tasks.json`.

**Tip:** Validate parses Memory. Non-zero CPU and Memory emit `--cpus` and `--memory` and consume remaining admission. Zero is unspecified. `Params` stay recorded and unused. They are not interpolated into Command or Script.

**Application:** Treat CPU and Memory as real limits. Do not treat Params as command inputs.

## Isolate scratch is not a directory port

**Context:** Isolate already lets a tool write undeclared directories. Tree is the declared directory artifact.

**Tip:** Tools may write undeclared dirs; only declared File, Group members, or Tree contents plus dest `.gobble-tree.json` publish. PathSpec.Dir and `Directory` are placement, not artifacts. STAR `--genomeDir` is Tree via `DeclareTree`.

**Application:** Do not treat isolate scratch or implicit cwd as a port. Do not revive glob Set for directory tools that can declare Tree.

## Claim occupy with a lock plus owner record

**Context:** After `Release`, `.gobble/run.json` stays.

**Tip:** Do not use `O_EXCL` on `run.json` to claim occupancy. Claim with a lock file plus the owner record.

**Application:** `Run` and `Resume` after Release use `occupy.lock` and write the owner on `run.json`.

## Pin cache keys include content

**Context:** Assay-owned and module-owned test manifests can contain distinct
bytes with the same filename.

**Tip:** The shared test-only fixture helper uses
`cache/<sha256[:16]>/<name>`. The manifest destination, not the cache basename,
owns the workspace path. Fetch into a temporary file, verify exact size and
SHA-256, then rename.

**Application:** Keep the same-name/different-hash collision test in
`tests/internal/fixture`. Recheck staged bytes after copying them into a
workspace.

## Isolate restage stages Source onto dest

**Context:** A pipeline-input bind may restage so dest Dir differs from the authored From path.

**Tip:** Record `IO.Source` (and `IOMember.Source`) when the From rendered path
differs from dest. Empty Source keeps Path as both. Stage and publish copy through
complete temporary files; staged bytes must not share an inode with their source
or rely on a symlink.

**Application:** Bismark and BWA can restage a FASTA out of `in/` into a dest Dir the tool writes beside.

## Schedule keys reservedIdentity

**Context:** Runtime Scatter needs instance and shard slots. Product sample,
lane, run, and replicate lists remain compose-time typed construction.

**Tip:** Scheduler maps, `tasks.json` latest attempts, Inspect instance filter, and Release incomplete lists key `reservedIdentity`. Document is the only engine payload. Empty Image is process; non-empty is docker.

**Application:** Adding product membership adds authored IDs and a graph Change.
Do not revive `task.ID`-only maps or Snapshot as a parallel contract.

## Inspect remaining uses dest cheap keys

**Context:** Remaining used to SHA-256 dataset bytes. Content digest is still stored at publish.

**Tip:** Use recorded size, mtime, device, and inode as an Inspect fast path.
When cheap identity is absent or differs, compare the recorded SHA-256 with
current bytes. Docker reuse also requires the current resolved image identity to
match the attempt's recorded image digest.

**Application:** Cheap dest mismatch is `output-missing`. Schema 0 and 1 workspaces are `unsupported-schema`.

## Staged replace never unlinks a published dest

**Context:** Resume may replace an authorized dest after a rerun.

**Tip:** Never unlink, truncate, or overwrite a published dest in place. Exclusive-create a temp, write all bytes, then rename over the dest after complete isolate outputs.

**Application:** A failed new attempt that never renamed leaves the prior dest in place.

## testdata packages are importable

**Context:** CLI graph-verb tests need fixture packages that export `Pipeline()`.

**Tip:** `cmd/gobble/testdata/<pkg>` is an explicit import path. Do not assume testdata is unlistable.

**Application:** `go test ./...` skips testdata, but `go list` of that explicit path still resolves the fixture.

## Exec the built driver, not go run

**Context:** Graph verbs compile a generated driver and must forward child stderr byte-exact.

**Tip:** Exec the built driver binary. Do not use `go run`. `go run` prints `exit status N` to stderr on a non-zero child.

**Application:** Panic and signal tests compare stderr bytes against the child, not a `go run` wrapper.
