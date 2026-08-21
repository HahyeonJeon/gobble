# compose-pipeline backlog

## CLI --samples

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Add a list-valued CLI `--samples` flag.

**Why backlogged:** D4 replaced that idea with singular `--sample PATH`. `--samples` is an unknown flag (exit 2).

**Context:** `--sample PATH` is the samplesheet CSV on compose, validate, plan, run, and resume. It is not a list of sample ids.

## Engine scatter / reservedIdentity expansion

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Expand samples as engine scatter / plan-time Document expansion of `reservedIdentity` instead of a compose-time Go loop.

**Why backlogged:** D1 locked compose-time CSV parse that emits one `AddModule` per sample. First-horizon compose stays modules, branch, and merge.

**Context:** `WGS()`, `RNASeq()`, and `MethylSeq()` already expand named modules at compose. Roadmap stop condition still names scatter as a later breaking rewrite if attached as Document expansion.

## TSV auto-detect

**Backlogged at:** 2026-08-21T00:25:00Z

**What:** Accept tab-separated samplesheets.

**Why backlogged:** D1 locked CSV only. A tab-separated file is malformed CSV.

**Context:** `ParseSampleSheet` uses `encoding/csv` with comma separator. No comment lines. No TSV dialect.
