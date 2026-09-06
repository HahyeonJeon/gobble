# Agent installation and operation guide

Use the same Docker/Compose workflow as a person. Do not install a separate
beginner edition or generate an ad-hoc source build. Gobble does not bundle the
Agent; the Agent needs local file and terminal access to the user's computer.

## Environment preparation

1. Identify OS, CPU architecture, the intended local project directory, and
   available resources. Explain that current images are linux/amd64 and ARM
   hosts need working emulation.
2. Check `docker info` and `docker compose version`. If unavailable, follow the
   OS-specific official Docker installation links in the
   [installation guide](../distribution/runtime/README.md). Resume these checks
   after any required user action or reboot. Use the local Docker engine.
3. Download the official Compose file to a fresh local project parent directory.
   Do not overwrite an existing Compose file or runtime lock. Use a published
   digest-pinned distribution when a specific version was requested.
4. Run `docker compose pull` and `docker compose run --rm -T gobble doctor`.
   Doctor returns JSON and must verify sibling-container read/write access.
   An image pull error is not a reason to switch revisions or invent a build.
5. Prepare a real assay with `docker compose run --rm -T gobble demo rnaseq my-rnaseq`,
   or use `init my-pipeline` for the tiny first installation check. Enter the
   created directory and use its own pinned Compose file for every next command.

Repeat read-only checks safely. Do not repeat project creation over an existing
project. Failed fixture downloads reuse verified cache entries; if a partial
project was already created, retain it for diagnosis and choose a new name.

## Design and run

Read the generated README and AGENTS.md. Establish the analysis goal, input
samples, organism/reference build, resource budget, and expected outputs before
changing scientific settings. Never guess a reference build. Edit ordinary Go
files on the host; run compilation and Gobble commands inside the runtime.

Run `docker compose run --rm -T gobble validate .` and `plan .` using the same
prefix. Explain the pipeline and resource requirements. Check the tutorial's
largest task CPU/memory requirements; `--cap 1` does not shrink a task.

Launch long runs with `docker compose run -d gobble run . --workspace runs/demo --cap 1`.
Record the returned container ID. Check `docker logs CONTAINER_ID` for startup
failure, then `docker compose run --rm -T gobble inspect run --workspace runs/demo`.
Starting a container is not successful analysis completion. Use `inspect monitor`,
`inspect errors`, and `inspect instances` for machine-readable progress. Reserve
`watch` for a human terminal. Verify expected nonempty result files at completion.

## Stop and recovery

Use `docker compose run --rm -T gobble stop --workspace runs/demo`. A result of
`requested` means settlement is not proven; inspect/repeat Stop. After `settled`,
use `docker compose run -d gobble resume . --workspace runs/demo --cap 1` and record
the new container ID. Completed valid tasks are reused; interrupted tasks restart.

Agent or terminal exit after detached launch does not stop the controller. A
computer/Docker restart does stop execution and requires checking the original
engine and reconciling state. Never start a second owner, remove locks, discard
checkpoints, or switch images to bypass a recovery error. Keep diagnostic logs.
Use `docker rm CONTAINER_ID` only for your stopped controller after inspection.

## Updating

Use a new project parent and selected published runtime for new work. Existing
projects keep their pinned Compose image and runtime lock. Pull their recorded
image if it was removed. An engine replacement needs explicit migration design;
it must not be treated as an ordinary Resume. Keep user data on the local host.
