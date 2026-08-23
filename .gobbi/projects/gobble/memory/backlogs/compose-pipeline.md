# compose-pipeline backlog

## Additional When predicates

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Let `When` skip or run on predicates other than `SkipIfMissing` and `SkipIfFalse`.

**Why backlogged:** First-horizon `When` is those two predicates only. This Engine session did not add more.

**Context:** A `When` with no predicate never skips. `SkipIfMissing` takes a File Handle. `SkipIfFalse` takes a declared boolean param. Design Current in [`compose-pipeline`](../design/feature/compose-pipeline.md) records that contract.

## CLI --samples

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Add a list-valued CLI `--samples` flag.

**Why backlogged:** D4 replaced that idea with singular `--sample PATH`. `--samples` is an unknown flag (exit 2).

**Context:** `--sample PATH` is the samplesheet CSV on compose, validate, plan, run, and resume. It is not a list of sample ids.

## Multi-lane samples

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Represent one sample with more than one `read1`/`read2` pair (lanes or libraries) on the samplesheet.

**Why backlogged:** D1 locked unique sample names. A second row with the same `sample` is `invalid-name` duplicate sample name.

**Context:** `SampleRow` is one `Sample`, `Read1`, and optional `Read2`. Extra lane files must be `AddInput` in Go or pre-merged FASTQs. `RNASeq()` and `MethylSeq()` emit one module per unique sample row.

## Partial-success fan-in

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Let a join task run on the succeeded subset of wired `From`s instead of waiting for every `From` to succeed.

**Why backlogged:** `Run` `upstreamReady` requires every wait producer `StatusSucceeded`. That matches a declared-cohort joint call. "Genotype whoever finished" was not first-horizon fan-in.

**Context:** `Merge` and `Gather` do not auto-wire; edges come from `Bind.From`. A join task with N file `From`s never becomes ready if one producer fails. An empty Gather membership is not gather-ready. Failed scatter members already fail the run.

## Plan-time reservedIdentity expansion

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Expand samples as plan-time Document expansion of `reservedIdentity` instead of a compose-time Go loop.

**Why backlogged:** D1 locked compose-time CSV parse that emits one `AddModule` per sample. Authored Scatter, Gather, and When occupy runtime instance and shard slots. They are not plan-time Document expansion.

**Context:** `WGS()`, `RNASeq()`, and `MethylSeq()` already expand named modules at compose. Empty `read2` is allowed at parse. Mate-only constructors return `invalid-samplesheet`. Roadmap stop condition still names plan-time Document expansion as a later breaking rewrite. N and any compose-time batching stay in that Go loop. Runtime membership remains `Scatter.From` a Group, Tree, or File. Hierarchical gather is not a separate operator.

## Samplesheet extra columns

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Accept samplesheet columns beyond the locked set and expose them to the author.

**Why backlogged:** D1 locked the header set. An unknown header makes the sheet malformed.

**Context:** Locked columns are `sample`, `read1`, `read2`, `reference`, `gtf`, `group`, and `strandedness`. Parse test `unknown header` uses an `extra` column. Extra metadata must be a separate `AddInput` file, not sheet cells.

## TSV auto-detect

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Accept tab-separated samplesheets.

**Why backlogged:** D1 locked CSV only. A tab-separated file is malformed CSV.

**Context:** `ParseSampleSheet` uses `encoding/csv` with comma separator. No comment lines. No TSV dialect.
