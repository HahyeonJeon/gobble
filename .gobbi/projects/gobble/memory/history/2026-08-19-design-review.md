# Design-review engine evolve completed

**Completed at:** 2026-08-19T23:25:00Z

## Changes

- Evolved the library engine on `refactor/design-review` from `develop` `2b25ce0d`. Work HEAD `9a57abb9`. Public verbs stay Compose, Validate, BuildPlan, Run, Inspect, Resume, Release. `Run(ctx)` and `Resume(ctx)` cancel through context; occupancy stays until Release. No public Cancel, Diff, Retry, Clean, or CLI.
- PathSpec fields are Dir, Prefix, Base, Suffixes, Ext. A Bind is File, Group, or Tree. Tree is a declared directory plus dest `.gobble-tree.json`. Path render lives in `internal/path`. `Merge(name)` takes no branch list.
- Engine payload is Document only. SchemaVersion is 2. Schema 0 and 1 are not resume-compatible. Scheduler keys `reservedIdentity`. Executor is Submit/Poll/Cancel/Reconcile in `internal/engine/exec`. Empty Image is process; non-empty is docker.
- Staging is hardlink, then process-only symlink, then copy. Publish is hardlink then copy. Inspect remaining uses dest cheap keys after publish and input cheap keys at success. Content digest is stored at publish. Image digest is recorded on the attempt and is not identity.
- Resume classifies graph-diff Change (Added, Removed, Rewired, Repathed, IdentityChanged, Unchanged) and no longer returns `plan-drift`.
- First-check is hermetic `go test ./...`. Live is `go test -tags=live` and fails closed without Docker. Live WGS assay executes `assets.WGS()`, including Inspect, Release, and Resume.
- Commits oldest first: `359b3d87` PathSpec rename; `23ed3010` Document-only and reservedIdentity; `7599d8b7` Executor and context cancel; `ab4dbb78` Tree; `6ae3e4bc` link-then-copy and dest cheap keys; `3e1c2820` graph-diff Resume; `9a57abb9` hermetic first-check, live tag, and WGS assay.
- Current design is in [Design Memory](../design/README.md). This session removed `backlogs/compose-pipeline.md` after completing One path model (`internal/path`) and Merge branch list (`Merge(name)`). It also completed Scheduler keyed by task ID, Wait-only edges in plan drift, and WGS live Docker resume proof. Remaining deferred items are in [project](../backlogs/project.md), [recover-run](../backlogs/recover-run.md), and the [later run-local disposition](../reports/note/2026-09-02-gobble-multiomics-products.md#deferred-outcomes). Earlier history that still names the compose-pipeline backlog is the 2026-08-17 and 2026-08-18 point-in-time account; this record is the correction. The session note is [Design-review evolve](../reports/note/2026-08-19-design-review.md).
