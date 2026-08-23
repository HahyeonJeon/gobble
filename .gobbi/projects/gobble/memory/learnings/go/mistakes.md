# Go Mistakes

## Compose and Run must share one edge contract

**Context:** Related-file output `From` is legal at compose. Thin-slice BAI used it.

**Mistake:** `upstreamReady` only looked up inputs. Output-port `From` composed cleanly and deadlocked at run (`not-started`).

**Correction:** Resolve wait paths at `BuildPlan`. Run reads `Edge.Wait` only. Do not guess from `ToPort`. Add a process-level regression.

## WGS FASTQ is not RNA or bisulfite proof

**Context:** First-party RNA-seq and Methyl-seq live Runs need assay-shaped reads.

**Mistake:** Reusing WGS homo_sapiens FASTQ and FASTA as STAR or Bismark stand-ins proves graph shape only. Non-bisulfite WGS reads aligned 0.00% through Bismark.

**Correction:** Pin official nf-core rnaseq and methylseq test-profile files. Drop WGS stand-in from RNA and Methyl proofs.

## Basename-only pin cache collides

**Context:** Two pins can share `Pin.Name` (`test_1.fastq.gz` for WGS and SARS-CoV-2).

**Mistake:** Writing `CacheDir/<Name>` lets the second pin overwrite or reuse the first file.

**Correction:** Use `CacheDir/<sha256[:16]>/<Name>` and a shared fetch helper. Check size and sha256 after download.

## Bismark image swap needs a CLI study

**Context:** Gobble first used `quay.io/biocontainers/bismark:3.1.0`. Official methylseq 4.2.0 uses Seqera Bismark 0.25.1.

**Mistake:** Swapping the image without mapping argv. A rumor that Perl 0.25.1 wants `--output` instead of `--output_dir` is not evidence.

**Correction:** Study the 0.25.1 CLI (and live `--help` when available) before the swap. v0.25.1 documents `-o/--output_dir`. Keep `--basename aligned`. Do not combine `--basename` with `--multicore`. Re-run live after the image change.

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

## Binding sheet FASTQ cells as Literal

**Context:** Samplesheet read cells must feed adders such as fastp that call `PathSpec.AppendSuffix`.

**Mistake:** Binding those cells with `Literal(cell)` stores an opaque filename. `AppendSuffix` marks a Literal invalid, so cleaned-read dests cannot render.

**Correction:** Split a validated workspace-relative cell into Dir, Base, and Ext (`sheetFileSpec`). Keep `Literal` for opaque shared reference or GTF cells that are not suffixed. Match WGS pin PathSpecs.

## Treating PID as occupancy owner

**Context:** Run, Resume, Release, Inspect `live`, and live recover tests.

**Mistake:** Treating PID existence as occupancy liveness, closing occupancy while identities remain unknown, treating skip as remaining work, or faking a dead PID in tests as recovery proof.

**Correction:** Occupancy owner is the occupying process. Private liveness is a held occupancy flock and lease. PID and host are diagnostic Inspect fields. Occupancy does not close and Resume does not occupy while any identity remains unknown. Skip is a known terminal status and is not remaining. Live tests must not fake a dead PID as recovery.
