# run-local backlog

## Control-plane image digest

**Backlogged at:** 2026-08-18T02:43:23Z

**What:** Use recorded image digest as reuse identity, or pin images by digest.

**Why backlogged:** Digest is recorded on the attempt. Authored image string remains reuse identity.

**Context:** Docker adapter returns RepoDigest on Submit. Scheduler writes `image_digest` on the attempt. Process leaves it empty. `runDockerCLI` env is `PATH=/usr/bin:/bin`, separate from task `-e`. Pull is docker CLI, not task stdout. classifyReuse still compares the authored image string.
