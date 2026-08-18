# run-local backlog

## Control-plane image digest

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Pull out of task stdout. Record the image digest. Keep docker-client env separate from task env.

**Why backlogged:** `strelka/stdout` starts with `docker pull`. Digest lives only in the test helper. `runDockerCLI` inherits stripped task env.

**Context:** Last-review rank 4. Cheap hygiene before resume or a stdout port. Needs an explicit Prepare vs Check decision. Task `Env` now applies as `-e KEY=VALUE`; docker-client env is still not a separate control plane. Resume shipped on `13c42a6`. D-digest still defers image-digest pinning. Authored image string is reuse identity; digest is not.

## Scheduler keyed by task ID

**Backlogged at:** 2026-08-18T15:07:00Z

**What:** Key the scheduler by reserved identity, not only `task.ID`.

**Why backlogged:** Ready inspect/resume reserved instance, shard, and attempt slots but left the scheduler keyed by `task.ID`.

**Context:** Isolate is `.gobble/tasks/<id>/_/0/1/work` for empty instance, shard `0`, attempt `1`. Distinct reserved keys isolate. The scheduler still looks up `s.tasks[t.ID]`.
