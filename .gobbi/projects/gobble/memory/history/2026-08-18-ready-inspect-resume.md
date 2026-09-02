# Ready inspect and resume shipped

**Completed at:** 2026-08-18T15:48:00Z

## Changes

- Shipped public `Inspect`, `Resume`, and `Release` on `feat/full-engine`. Occupancy is an owner record. `Release` closes occupancy and is not deletion.
- Resume uses dest-scope `output-exists`, checksum or lineage dest attribution, staged replace plus per-attempt isolate, and persist of script and env on the attempt.
- First-check remains `go test ./...`. Live proof at `13c42a6` included gobble, engine, and `tests/wgs-e2e`. CLI and WGS-as-resume-proof remain required for local-core exit.
- Current design is in [Design Memory](../design/README.md). Deferred items remain in [project backlog](../backlogs/project.md), [recover-run backlog](../backlogs/recover-run.md), and the [later run-local disposition](../reports/note/2026-09-02-gobble-multiomics-products.md#deferred-outcomes). The session note is [Ready inspect and resume](../reports/note/2026-08-18-ready-inspect-resume.md).
