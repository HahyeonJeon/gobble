# Gobble CLI session

The session shipped the product CLI at `cmd/gobble` as a package-path interface over the existing library loop, not Graph JSON and not a public `cli` package.

Result: seven product verbs `compose`, `validate`, `plan`, `run`, `inspect`, `resume`, and `release`. Graph verbs take a Go package path (default `.`), require `go` on PATH, compile a generated driver, call `func Pipeline() *gobble.Pipeline`, then Compose. Inspect is `gobble inspect VIEW --workspace DIR`; VIEW matches library views and library bytes pass through. Inspect and release run in-process. `--workspace DIR` is required on run, inspect, resume, and release; the binary does not create DIR or infer it from cwd. `--cap N` is optional on run and resume; omit passes `0`. SIGINT/SIGTERM on run and resume cancel ctx; occupancy stays. There is no `cancel` verb. Success is JSON or JSONL on stdout. Failure is empty stdout and `*Error` JSON on stderr. Exits `0` / `1` / `2`. Domain → `1`. Invocation, unknown command, bad flag, compile, and missing constructor → `2`. `Pipeline` or `init` panic is a process abort; stderr is not JSON. Missing-workspace codes stay verb-specific (`invalid-path` vs `not-found`).

Evidence: work HEAD `4f5e6b8` from `develop` `13b0664`. Commits `6384547`, `0f3a89b`, `f34006e`, `939f24d`, `4f5e6b8`, plus this Memory commit. Hermetic `go test ./...` PASS, including `cmd/gobble`. Publication is local.

Limits: no live CLI tests this session; first-horizon exit is still open; interrupting a graph verb during compile can leave `/tmp/gobble-driver-*`; no public Cancel, Diff, Retry, or Clean; no color or pretty output; Linux is the supported platform.

See [system.md](../../design/architecture/system.md) Interfaces Current and [history](../../history/2026-08-20-gobble-cli.md).
