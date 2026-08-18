# Static-core slice

The session shipped the static-core slice: `Group`, `Script` XOR `Command`, `Env` literals, plan-time wait paths, applied Resources, and Group I/O by name. Isolate and occupy stay. Inspect, Resume, Collection, scatter, and CLI were not shipped.

WGS consumer proofs moved to `tests/wgs-e2e` (public API only). First-check remains `go test ./...` and includes that package. Live Docker tests skip if the daemon is down. Cached `go test ./...` is not proof a live WGS e2e ran. A later live `go test ./tests/wgs-e2e -count=1` PASS included `TestWGSThinSliceRun` and `TestWGSSpineRun`.

Accepted commits: `f39b4bf` Group/Script/Env/wait, `d2c81f2` Resources/Script/Env/Group I/O, `e8b4720` WGS move. Head `e8b4720f8cb594be2c165960f86b901dfd2315be`. See [Design Memory](../../design/README.md) and [history](../../history/2026-08-18-static-core-slice.md).
