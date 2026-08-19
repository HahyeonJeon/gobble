# project backlog

## CLI and cmd

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Add structured CLI and `cmd/` for the same loop as the Go API.

**Why backlogged:** D4 excluded CLI this session. The library loop evolved without `cmd/`.

**Context:** First-horizon exit still requires the same compose-validate-plan-run-inspect-resume loop on the CLI. Public verbs now include context cancel on Run and Resume. `invocation-contract` stays Open. There is no `cmd/`.
