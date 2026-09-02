# compose-pipeline backlog

## Additional When predicates

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Let `When` skip or run on predicates other than `SkipIfMissing` and `SkipIfFalse`.

**Why backlogged:** The current engine contract includes only those two predicates.

**Context:** A `When` with no predicate never skips. `SkipIfMissing` takes a File
Handle. `SkipIfFalse` takes a declared boolean parameter. Resume re-evaluates
the predicate.

## CLI --samples

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Add a list-valued CLI `--samples` flag.

**Why backlogged:** The current command uses singular `--sample PATH` for one
assay-owned CSV. A sample-ID selector was not accepted.

**Context:** `--sample PATH` is available on compose, validate, plan, run, and
resume. It is not a list of sample names. Product packages parse their own
strict schemas.

## Partial-success fan-in

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Let a join task run on the succeeded subset of wired `From`s instead of waiting for every `From` to succeed.

**Why backlogged:** Current product joins require complete declared membership.
Joint calling, assay matrices, consensus peaks, and combined matrices must not
silently accept holes.

**Context:** `Merge` and `Gather` do not auto-wire; edges come from `Bind.From`.
A join with failed required producers remains blocked. Any future partial
semantics need a distinct output meaning and cannot silently change current
product fan-in.

## Plan-time reservedIdentity expansion

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Expand samples as plan-time Document expansion of `reservedIdentity` instead of a compose-time Go loop.

**Why backlogged:** Current product sheets become typed values and expand at
compose time. Runtime Scatter/Gather is reserved for true artifact membership,
such as WGS intervals.

**Context:** The five product builders produce authored sample, run, lane, and
replicate task IDs. BuildPlan does not explode Document tasks into sample
instances. A later expansion model would be a breaking engine and graph rewrite.

## Samplesheet extra columns

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Accept samplesheet columns beyond each product's locked set and expose them to the author.

**Why backlogged:** Every current assay owns a strict exact header set so
misspelled or unused metadata does not disappear silently.

**Context:** Adding a column is a versioned change to that assay's typed data
contract, not a change to one universal Gobble sheet. Analysis config, images,
resources, and engine controls remain outside sheets.

## TSV auto-detect

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Accept tab-separated samplesheets.

**Why backlogged:** Current assay loaders accept strict CSV only.

**Context:** No loader auto-detects dialect, comments, or a tab separator. A
future format must preserve exact schema and error behavior for each assay.
