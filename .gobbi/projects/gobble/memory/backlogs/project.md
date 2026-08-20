# project backlog

## CLI live and WGS first-horizon proof

**Backlogged at:** 2026-08-20T05:09:58Z

**What:** Prove the shipped CLI on the synthetic fixture and on a small WGS pipeline, with at least one real Docker task.

**Why backlogged:** This session shipped `cmd/gobble` and hermetic tests only. Live CLI tests were excluded.

**Context:** Local-core exit evidence in [roadmap](../design/roadmap/project.md) is unchanged: an agent completes the local loop on the synthetic fixture and on a small WGS pipeline, using the Go API and the CLI, with at least one real Docker task. Compile or `go test ./...` is not exit evidence. Product CLI is accepted at `4f5e6b8`. Temp compile leak is a separate item.

## Temp driver cleanup on interrupt during compile

**Backlogged at:** 2026-08-20T05:09:58Z

**What:** Remove leftover generated driver files if the user interrupts while a graph verb is compiling.

**Why backlogged:** Wrap-up-readiness iteration 2 accepted this as a non-blocker.

**Context:** The leak is during compile, before the driver runs. Occupancy and signal tests do not cover it.
