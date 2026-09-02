# Go Mistakes

## Compose and Run must share one edge contract

**Context:** Related-file output `From` is legal at compose. Thin-slice BAI used it.

**Mistake:** `upstreamReady` only looked up inputs. Output-port `From` composed cleanly and deadlocked at run (`not-started`).

**Correction:** Resolve wait paths at `BuildPlan`. Run reads `Edge.Wait` only. Do not guess from `ToPort`. Add a process-level regression.

## One assay's fixture is not another assay's proof

**Context:** A command may accept another assay's file format while its behavior
still depends on assay-specific read, reference, annotation, or protocol shape.

**Mistake:** Reusing WGS reads and reference bytes as STAR or Bismark stand-ins
proved graph shape but not the selected assay path. Non-bisulfite WGS reads, for
example, produced meaningless Bismark evidence.

**Correction:** Give each assay one manifest with exact official bytes and stage
use. Reuse shared test support, not another assay's fixture authority.

## Basename-only pin cache collides

**Context:** Pins in different assay or module manifests can share a filename.

**Mistake:** Writing `CacheDir/<Name>` lets the second pin overwrite or reuse the first file.

**Correction:** Use `cache/<sha256[:16]>/<name>` through the shared test-only
fixture helper. Verify size and SHA-256 after download and again after staging.

## An image change needs a command-contract study

**Context:** A product default image binds tool version, accepted flags, output
names, ports, provenance, and workspace reuse.

**Mistake:** Swapping an image from a tag or rumor without mapping the exact CLI
and output behavior can preserve compilation while changing task meaning.

**Correction:** Study commit-bound module source and versioned tool help, update
the exact tag and digest authority, recheck protected flags and ports, and rerun
the applicable command and product evidence. Treat the image change as a
compatibility and resume event.

## Do not classify from a record this operation will overwrite

**Context:** Resume occupy overwrites `plan.json`. Inspect remaining can take an instance filter. Dest replace needs an owner.

**Mistake:** Classifying resume identity from the plan after occupy overwrites it. Filtering latest attempts before remaining classification. Treating any executed identity as dest owner.

**Correction:** Persist script and env on the attempt, or classify before replacing `plan.json`. Classify remaining on all latest attempts, then instance-filter emit. Attribute dests by checksum or producer lineage, not executed identity.

## Parse go list from stdout only

**Context:** Graph verbs resolve a user package with `go list -f '{{.ImportPath}}'`.

**Mistake:** `CombinedOutput` plus first-line parse treats `go: downloading` on stderr as the import path.

**Correction:** Read stdout only (`cmd.Output()`). Surface diagnostics from `ExitError.Stderr`.

## Isolate gobble child TMPDIR under package-parallel tests

**Context:** Hermetic tests that exec a built `gobble` while `cmd/gobble` tests call `watchDriverTemps`.

**Mistake:** Leaving the child `TMPDIR` as the shared process temp. Graph verbs then write `gobble-driver-*` where `watchDriverTemps` globs, so package-parallel `go test` fails.

**Correction:** Set the child `TMPDIR` to a test-owned directory before exec. Do not change product `cmd/gobble` for this isolation.

## Signaled ExitCode is not 255

**Context:** Graph verbs wait on the compiled driver and map its status to the launcher exit.

**Mistake:** Passing `exec.ExitError.ExitCode()` of a signaled child to `os.Exit`. That code is `-1`; `os.Exit(-1)` yields 255.

**Correction:** Map a signaled wait to `128+signal`. Keep other mapped codes in `{0,1,2}`.

## Binding suffixable sheet paths as Literal

**Context:** Strict assay loaders produce workspace-relative read paths that
downstream command modules may extend with suffixes.

**Mistake:** Binding a suffixable path with `Literal(cell)` makes it opaque.
`PathSpec.AppendSuffix` then marks the value invalid and destinations cannot
render.

**Correction:** Convert validated sheet paths into structured Dir, Base, and Ext
values before graph construction. Use Literal only for paths that the owning
command contract treats as an opaque unsuffixed token.

## Treating PID as occupancy owner

**Context:** Run, Resume, Release, Inspect `live`, and live recover tests.

**Mistake:** Treating PID existence as occupancy liveness, closing occupancy while identities remain unknown, treating skip as remaining work, or faking a dead PID in tests as recovery proof.

**Correction:** Occupancy owner is the occupying process. Private liveness is a held occupancy flock and lease. PID and host are diagnostic Inspect fields. Occupancy does not close and Resume does not occupy while any identity remains unknown. Skip is a known terminal status and is not remaining. Live tests must not fake a dead PID as recovery.

## Treating incomplete plus RuntimeID as live Resume state

**Context:** Later-process Resume of a released incomplete process leftover that still stores a RuntimeID.

**Mistake:** Treating `StatusIncomplete` plus a non-empty RuntimeID as live Resume state. Empty-executor Reconcile then becomes `unknown-backend` instead of remaining work.

**Correction:** A RuntimeID on incomplete is historical identity, not proof that this Resume owns that process. Incomplete leftovers rerun. Occupying-process running or unknown remains live.

## Publishing Wait completion after cleanup so Cancel can SIGKILL

**Context:** Process adapter Wait and Cancel share one handle. Wait closes logs and records exit under a mutex, then closes done.

**Mistake:** Publishing Wait completion only after file closes and mutex, so Cancel can still SIGKILL a reaped PID while done is open.

**Correction:** Publish Wait return before cleanup. Cancel checks that signal before and after the mutex. After Wait returns, Cancel is a no-op.

## Treating stamped vcs.revision as occupy identity in a linked worktree

**Context:** A linked worktree builds `cmd/gobble`. Go `debug.ReadBuildInfo` stamps `vcs.revision` from the main checkout.

**Mistake:** Treating that stamped revision as local-pin occupy identity. Occupy then refuses a valid pin because the stamp is the main checkout, not the selected module directory.

**Correction:** Local-pin occupy uses git of the selected module Dir plus the executable digest. Ignore BuildInfo `vcs.revision` for that compare. Exact-tag still compares `Main.Version`.

## Treating empty protocol bytes as globally invalid

**Context:** The packed trampoline copies child protocol from a seeked temporary file. `inspect remaining` and `inspect reuse` encode no records as empty JSONL.

**Mistake:** Treating empty protocol bytes as invalid for every child endpoint. Successful remaining or reuse with zero records then fails closed.

**Correction:** Exact empty JSONL is valid only for successful `inspect remaining` and `inspect reuse`. Other child protocol endpoints still require a record. Whitespace-only or malformed bytes stay invalid.
