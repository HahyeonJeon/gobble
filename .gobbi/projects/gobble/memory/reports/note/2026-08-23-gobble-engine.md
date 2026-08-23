# Gobble Engine session

The session implemented a complete local Engine after the 2026-08-22 adversarial review: one occupancy lifecycle, contained workspace trust, Engine-class Scatter/Gather/When, representative hermetic proofs, and live occupy/release/resume without PID-fake recovery.

Result: Occupancy owner is the occupying process. Private liveness is a held flock and lease. Inspect `live` is that liveness. PID and host are diagnostic. Occupying-process Release is live-owner Release. A later process may Release; while the occupier is live that is `live-occupancy`. Occupancy does not close and Resume does not occupy while any identity remains unknown. Skip is known terminal. `Release` Reconciles first. There is no public Cancel, Retry, or Clean. Public operators include Scatter, Gather, and When. Scatter members occupy runtime instance segments with shard index 0. Plan-time Document expansion stays deferred. Schema 0 and 1, and PID-only schema-2 occupancy missing a lease, are `unsupported-schema`. First-check remains hermetic `go test ./...`. Live is `go test -tags=live` and fails closed without Docker.

Evidence: work started from `develop` `428d650`. Accepted Engine head `4fbb9db`. Wrap-up Memory commit follows. Hermetic `go test -count=1 ./...` PASS at `4fbb9db`. Exclusive live local-e2e, engine, and assets PASS at task-06 trees. Publication is local.

Limits: first-horizon exit is not claimed; hung Start, leftover Docker after killed `docker run -d`, occupancy TOCTOU, unsalted env digest, and `ps` visibility of `docker -e KEY` remain residual; no public Cancel/Retry/Clean.

See [Design Memory](../../design/README.md), [history](../../history/2026-08-23-gobble-engine.md), and [Adversarial codebase and Engine review](../review/2026-08-22-adversarial-codebase-engine-review.md).
