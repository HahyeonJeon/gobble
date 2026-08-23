# compose-pipeline backlog

## CLI --samples

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Add a list-valued CLI `--samples` flag.

**Why backlogged:** D4 replaced that idea with singular `--sample PATH`. `--samples` is an unknown flag (exit 2).

**Context:** `--sample PATH` is the samplesheet CSV on compose, validate, plan, run, and resume. It is not a list of sample ids.

## Plan-time reservedIdentity expansion

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Expand samples as plan-time Document expansion of `reservedIdentity` instead of a compose-time Go loop.

**Why backlogged:** D1 locked compose-time CSV parse that emits one `AddModule` per sample. Authored Scatter, Gather, and When occupy runtime instance and shard slots. They are not plan-time Document expansion.

**Context:** `WGS()`, `RNASeq()`, and `MethylSeq()` already expand named modules at compose. Empty `read2` is allowed at parse. Mate-only constructors return `invalid-samplesheet`. Roadmap stop condition still names plan-time Document expansion as a later breaking rewrite.

## TSV auto-detect

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Accept tab-separated samplesheets.

**Why backlogged:** D1 locked CSV only. A tab-separated file is malformed CSV.

**Context:** `ParseSampleSheet` uses `encoding/csv` with comma separator. No comment lines. No TSV dialect.
