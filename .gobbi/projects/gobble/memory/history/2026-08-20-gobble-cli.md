# Gobble CLI completed

**Completed at:** 2026-08-20T05:09:58Z

## Changes

- Shipped a seven-verb package-path CLI at `cmd/gobble`. Graph verbs take a Go package path (default `.`), require `go` on PATH, compile a generated driver, call `func Pipeline() *gobble.Pipeline`, then Compose. Inspect and release run in-process. No public `cli` package. No Graph JSON.
- Product verbs are `compose`, `validate`, `plan`, `run`, `inspect`, `resume`, and `release`. SIGINT/SIGTERM on run and resume cancel ctx; occupancy stays. No `cancel` verb. Success is JSON or JSONL on stdout. Failure is empty stdout and `*Error` JSON on stderr. Exits `0` / `1` / `2`.
- Commits oldest first: `6384547` launcher inspect and release; `0f3a89b` compile-driver compose validate plan; `f34006e` run resume and signal cancel; `939f24d` document gobble and close first-check; `4f5e6b8` keep import path and signal exits agent-readable; plus this Memory commit. Work started from `develop` `13b0664`.
- Hermetic `go test ./...` PASS, including `cmd/gobble`. No live CLI tests. First-horizon exit is still open.
- Removed backlog `CLI and cmd`. Remaining deferred items are CLI live and WGS first-horizon proof, and temp driver cleanup on interrupt during compile, in [project](../backlogs/project.md). Current design is in [Design Memory](../design/README.md). The session note is [Gobble CLI](../reports/note/2026-08-20-gobble-cli.md).
