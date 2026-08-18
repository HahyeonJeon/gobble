# recover-run backlog

## Dest-rename successful publish proof

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Prove a dest-rename Resume publishes the new path after a reuse miss.

**Why backlogged:** Ready inspect/resume treats dest rename as a reuse miss. The increment did not ship a successful new-dest publish proof.

**Context:** A public dest-rename Resume may still fail to publish if the command writes the old path. Prove no reuse first. A successful new dest needs a command that writes the new path.

## Group and branch-merge resume e2e

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Prove Resume on a Group and branch-merge pipeline.

**Why backlogged:** Session proofs were occupy-release, two-instance isolation, and reuse decisions.

**Context:** WGS remains a consumer-test Run proof, not a resume proof. Group members already stage and publish by name on Run.

## Guarded clean and retention

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Add explicit guarded clean and a retention policy for run state and artifacts.

**Why backlogged:** Not in the Ready increment. `retention-deletion` stays open.

**Context:** `Release` is not deletion. Staged replace is the only dest mutation this increment allows. The consumer still owns the workspace.

## Live cancel

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Cancel an in-flight local `Run` or `Resume`.

**Why backlogged:** Not in the Ready increment. Release is refused while the owner is live.

**Context:** Occupancy liveness is host plus process id. A live local owner and a foreign host cannot be released. There is no force flag.

## Named retry with backoff

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Retry one named task with backoff.

**Why backlogged:** Resume covers remaining unsuccessful work. Named retry was deferred.

**Context:** There is no forced reuse reason this session. A later named retry would need one.

## Transitive blocked-upstream

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Assign `blocked-upstream` through a chain of wait producers that never started.

**Why backlogged:** Shipped `blocked-upstream` only when a wait producer failed this Resume.

**Context:** A `rerun` that never started because a wait-path producer ended unsuccessful becomes `blocked-upstream` and gets no new attempt. Transitive never-started producers are not that case.

## Wait-only edges in plan drift for older workspaces

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Decide how older workspaces without recorded wait-only edges compare under plan drift.

**Why backlogged:** Plan drift now includes wait-only edges. Older workspaces may not have recorded them as plan edges.

**Context:** Plan drift is a changed task set or edges, including `wait` on those edges. It is an operation error before occupy.

## WGS live Docker resume proof

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Prove Resume on the live WGS Docker consumer test.

**Why backlogged:** Ready inspect/resume used library proofs. WGS remains a Run consumer test.

**Context:** First-check is `go test ./...`. Live WGS Docker is `go test ./tests/wgs-e2e -count=1`. CLI and WGS-as-resume-proof remain required for local-core horizon exit.
