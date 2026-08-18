# run-local backlog

## Control-plane image digest

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Pull out of task stdout. Record the image digest. Keep docker-client env separate from task env.

**Why backlogged:** `strelka/stdout` starts with `docker pull`. Digest lives only in the test helper. `runDockerCLI` inherits stripped task env.

**Context:** Last-review rank 4. Cheap hygiene before resume or a stdout port. Needs an explicit Prepare vs Check decision. Task `Env` now applies as `-e KEY=VALUE`; docker-client env is still not a separate control plane.
