# Design-review engine evolve

The session evolved Gobble from a file-only local core into engine contracts that later scatter and Slurm can fill, without a DSL or CLI.

Result: public verbs stay Compose, Validate, BuildPlan, Run, Inspect, Resume, Release. `Run(ctx)` and `Resume(ctx)` cancel in-flight work and leave occupancy active until `Release`. PathSpec fields are Dir, Prefix, Base, Suffixes, Ext (JSON dir, prefix, base, suffixes, ext). A Bind is File, Group, or Tree. Tree is a declared directory plus dest `.gobble-tree.json`. Directory remains placement. STAR `--genomeDir` is Tree.

Engine payload is Document only. SchemaVersion is 2. Schema 0 and 1 are `unsupported-schema`. Scheduler keys `reservedIdentity`. Executor Submit/Poll/Cancel/Reconcile lives in `internal/engine/exec`. Empty Image is process; non-empty is docker. Path render, restage, and classify live in `internal/path`. `Merge(name)` takes no branch list.

Stage is hardlink, then process-only symlink, then copy. Docker skips symlink. Publish is hardlink then copy; never symlink. Inspect remaining uses dest cheap keys recorded after publish and input cheap keys recorded at success. It does not hash bytes. Content digest is stored at publish. Image digest is recorded on the docker attempt and is not reuse identity. Dest cheap mismatch uses `output-missing`.

Resume classifies Change: Added, Removed, Rewired, Repathed, IdentityChanged, Unchanged. `plan-drift` is gone. Remaining and reuse omit `change` until Resume classifies. Wait-only edge change is Repathed.

First-check is hermetic `go test ./...`. Live is `go test -tags=live` and fails closed without Docker. The WGS assay is `tests/wgs-e2e` executing `assets.WGS()`, including Inspect, Release, and Resume. Thin and spine inlined graphs are not that proof. Assets stay proofs.

Locked keep-list: Go authoring, no DSL, isolate-keep, occupancy owner on `.gobble/run.json`, Group as named regular files, Docker `--network=none`, host env forbidden, cap 0 means 1, cap above 64 refused, no CLI.

User decisions: D1 Evolve; D2 File+Group+Tree; D3 graph-diff Resume; D4 recommended bundle. Routine closures: R1 input cheap keys at success; R2 live handle Cancel then previous-incomplete; R3 empty Image is process else docker; R4 live assay may use a dead-PID helper.

Evidence: work HEAD `9a57abb9a4984bb979558e63ea7913ccf5551c30` tree `b0649020652502d9fa261c6a438c6b84e7527c28`, started from `develop` `2b25ce0d762a31eaeeabeda0e75f38e03dbb5c56`. Seven local commits. Task-07 recorded hermetic `go test ./...` PASS and live WGS, run-local, RNASeq, and MethylSeq PASS.

Limits: no CLI; no public Cancel; named retry and guarded Clean unshipped; dest-rename successful publish, Group branch-merge Resume e2e, and transitive blocked-upstream remain deferred; image digest is recorded not identity; Linux is the supported platform.

See [Design Memory](../../design/README.md) and [history](../../history/2026-08-19-design-review.md).
