# run-local backlog

## Apply Resources to docker run

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Parse Memory at validate. Emit `--cpus` and `--memory` on `docker run` when non-zero.

**Why backlogged:** Recorded Resources are unused. Live Strelka saw host memory (`memMb: 47345`). `TestRunUnparseableMemory` accepts junk.

**Context:** Synthesis rank 3. Zero stays unspecified. First-horizon scheduler is CPU/memory-aware.

## Plan-time edge wait path

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Resolve the wait path at BuildPlan. Give a distinct never-ready defect.

**Why backlogged:** D5 A is a second guess in `upstreamReady`. The next bind kind can re-enter the same hole.

**Context:** Synthesis rank 2. Keep the shipped from-path wait for output-port `From`.

## Static per-task env map

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Declare per-task env as author literals only. Refuse value-less `-e`.

**Why backlogged:** Image `ENV` is invisible today. Host inheritance stays forbidden. This is not a Nextflow env port.

**Context:** Synthesis rank 5. `Env: {"HOME":"/work"}` is the same class as `Command`. Keep `TestRunProcessEnvIsFixed`.

## Control-plane image digest

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Pull out of task stdout. Record the image digest. Keep docker-client env separate from task env.

**Why backlogged:** `strelka/stdout` starts with `docker pull`. Digest lives only in the test helper. `runDockerCLI` inherits stripped task env.

**Context:** Synthesis rank 4. Cheap hygiene before resume or a stdout port. Needs an explicit Prepare vs Check decision.
