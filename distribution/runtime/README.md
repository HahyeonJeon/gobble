# Run Gobble with Docker

Use one container workflow on Linux, Windows, and macOS. Your coding agent edits
local files; Go, Git, Gobble, and compilation run inside the runtime. Analysis
tools run in separate containers on the same local Docker engine.

You need Docker with Compose v2. No host Go, Git, or Gobble installation is
required for this path. An AI agent is optional and is not bundled in the image.

## Prepare Docker

- Windows: install and start [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/)
  with Linux containers. Use PowerShell; a separate Ubuntu terminal is unnecessary.
- macOS: install [Docker Desktop](https://docs.docker.com/desktop/setup/install/mac-install/)
  for Intel or Apple Silicon and start it.
- Linux: install Docker Engine and the Compose plugin. Your user must be able to
  access the local Docker socket.

Check `docker info` and `docker compose version`. Use a local engine: remote
contexts cannot share the files on your computer through this configuration.
Both the runtime and current analysis images target **linux/amd64**. Apple
Silicon uses emulation; native ARM analysis support is not implied.

## Download the common configuration

Create a local folder for projects and save the official
[compose.yaml](https://raw.githubusercontent.com/HahyeonJeon/gobble/develop/distribution/runtime/compose.yaml)
there as `compose.yaml`. Your agent can download this file for you. Then, from
that folder, run these same commands in PowerShell, macOS, or Linux:

```sh
docker compose pull
docker compose run --rm gobble doctor
docker compose run --rm gobble demo rnaseq my-rnaseq
cd my-rnaseq
docker compose run --rm gobble doctor
docker compose run --rm gobble validate .
docker compose run --rm gobble plan .
```

The development image is `ghcr.io/hahyeonjeon/gobble:develop`. Its publication is
conditional on Docker CI, including real RNA-seq and WGS runs. A failed or not-yet
completed publication is not a release; check the **publish** job in
[Docker tests](https://github.com/HahyeonJeon/gobble/actions/workflows/docker.yml).
The workflow's distribution artifact contains a Compose file pinned to that
build's registry digest. Prefer that artifact when sharing an exact version.

`demo` creates a project with a pinned `compose.yaml`, Go source, official test
data, README, and AGENTS.md. Continue from the **new project directory** so its
Compose file selects the original runtime even after `develop` changes. Start a
new project from a fresh parent folder when choosing a newer runtime.

For a small installation exercise without analysis-image downloads, use
`docker compose run --rm gobble init my-pipeline`, enter `my-pipeline`, and run
`docker compose run --rm gobble run . --workspace runs/hello`. The result
`runs/hello/results/sequence-count.txt` must contain `2`.

## Execute and reconnect

For a long analysis, let Docker own the controller in the background:

```sh
docker compose run -d gobble run . --workspace runs/demo --cap 1
```

Save the returned container ID. A successful launch only means Docker started
the controller; it does **not** prove the pipeline succeeded. Check startup with
`docker logs CONTAINER_ID` and pipeline state with:

```sh
docker compose run --rm gobble inspect run --workspace runs/demo
docker compose run --rm gobble inspect errors --workspace runs/demo
docker compose run --rm gobble watch --workspace runs/demo
```

The terminal or Agent session can close after detached launch. The controller
continues while Docker and the computer remain running. `q` closes the monitor
without stopping the analysis. For scripts, add `-T` before the `gobble` service.
`docker wait CONTAINER_ID` prints the controller exit code; its own shell exit
status does not represent the pipeline outcome.

```sh
docker compose run --rm gobble stop --workspace runs/demo
docker compose run -d gobble resume . --workspace runs/demo --cap 1
```

Wait for Stop to report `settled` before resuming. Resume verifies completed
outputs and retries unfinished or changed tasks. A killed controller or Docker
restart requires reconciliation; there is no automatic rerun on restart. If
state is unknown, restore the original Docker engine and use the
[recovery guide](../../docs/operations.md#recovery).

After confirming completion, remove only your stopped controller with
`docker rm CONTAINER_ID`. Keep the project and `runs/`. Do not use Compose down,
image pruning, or deletion of run locks as substitutes for Gobble Stop.

## Files and permissions

Keep projects in writable local folders shared with Docker Desktop. Project
source, input files, checkpoints, and results remain on your computer. Container
paths are Linux paths; use relative paths in samplesheets and pipeline code.
The runtime translates shared paths for sibling containers using Docker's mount
metadata. An external workspace must be explicitly mounted too.

On Linux the runtime creates files as the bind mount's owner and retains access
to the Docker socket group. On Desktop, Docker translates host file access. If
Linux uses a non-default socket, set `GOBBLE_DOCKER_SOCKET` to its local path
before invoking Compose. Desktop uses the VM's `/var/run/docker.sock` in this
configuration, not the macOS client socket under your home directory.

The container uses the host daemon's API to launch tasks. Run trusted pipeline
code and images. The image does not contain an independent Docker daemon.

## Runtime identity

A project's `.gobble-runtime.json` records the exact local image ID, Docker
engine ID, and registry digest when available. Its generated `compose.yaml`
uses that digest (or the local image ID for source builds). Never edit the lock
to force an upgrade. If the image was removed, `docker compose pull` restores the
pinned registry image; locally built images need the original exact build.
A different engine or factory reset is not automatic migration of an old run.

## Development and compatibility

To test uncommitted implementation work, first commit it on a development branch;
the runtime deliberately verifies source identity. Build with:

```sh
docker build --platform linux/amd64 -f distribution/runtime/Dockerfile --target runtime -t gobble-runtime:local .
```

Copy the common Compose configuration into a **fresh** project folder and change
its image to `gobble-runtime:local`. Then use the same doctor/init/demo/run flow.
This build step is for developing Gobble, not normal installation.

The old `setup.sh` and `cmd/gobble-container` launcher remain compatible entry
points for existing projects. New installations use Compose. The
[direct Linux build](../../docs/installation.md) remains available for library
and engine development, without a separate beginner/advanced classification.

## Acceptance status

Linux Docker CI tests Compose without host Go or a launcher, detached execution,
Stop, controller death, Resume, nested paths, runtime mismatch, and actual
RNA-seq/WGS outputs. Native launcher tests cover the compatibility command.
Real Mac/Windows Docker Desktop, interactive Ctrl+C, and Desktop restart still
require host-specific acceptance. See [test data](../../docs/tutorials.md) and
[Agent installation and operation](../../docs/agent-guide.md).
