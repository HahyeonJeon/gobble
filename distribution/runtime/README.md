# Docker installation preview

The default v0.2.0 route is a small **Gobble launcher + Docker**. The launcher
runs Gobble, Go, and Git in a Linux runtime. Your coding agent edits ordinary
files on your computer and calls `gobble` from the project directory. Analysis
tools run in sibling containers. Direct Linux installation remains available.

**Validation status:** implementation and hermetic tests are in this branch.
The Windows x64 launcher cross-compiles. No release image or installer is
published, and actual Docker Desktop/Windows results are still required. These
instructions build the preview; they are not a claim of tested Windows support.

## Prepare the preview

Start Docker Desktop in Linux-container mode on Windows, or a local Docker
Engine on Linux. From a Git checkout of this branch, build both artifacts:

```sh
docker build -f distribution/runtime/Dockerfile --target runtime -t gobble-runtime:preview .
docker build -f distribution/runtime/Dockerfile --target launcher-artifacts --output ./dist .
```

Both builds use Go inside Docker. Host Go is unnecessary. Keep Git metadata in
the checkout: Gobble validates the source identity used by the runtime. The
Dockerfile-specific ignore file keeps run data and output binaries out of the
image. Base images are build inputs; a release must resolve and record their
digests, publish checksummed launcher artifacts, and publish a runtime digest.

On **Windows PowerShell**, from this checkout:

```powershell
$env:PATH = "$PWD\dist\windows-amd64;$env:PATH"
$env:GOBBLE_RUNTIME_IMAGE = "gobble-runtime:preview"
gobble init my-pipeline
Set-Location my-pipeline
```

On **Linux**:

```sh
export PATH="$PWD/dist/linux-amd64:$PATH"
export GOBBLE_RUNTIME_IMAGE=gobble-runtime:preview
gobble init my-pipeline
cd my-pipeline
```

The launcher saves the exact local image ID and daemon ID in the project.
Subsequent invocations use that image even if the original tag changes. Do not
delete the lock or the pinned image to upgrade an existing run. A different
Docker daemon is refused, because it cannot prove what happened to old tasks.

## First pipeline

These commands are identical in PowerShell and a Linux terminal:

```sh
gobble doctor
gobble plan .
gobble run . --workspace runs/hello
gobble inspect run --workspace runs/hello
```

The result is `runs/hello/results/sequence-count.txt`, containing `2`.
The generated `AGENTS.md` explains the authoring and recovery workflow to your
coding agent. The example uses `sh` and `awk` in the runtime; your agent should
select explicit Docker tool images for real analysis steps.

For a longer pipeline, open another terminal **in the same project directory**:

```sh
gobble watch --workspace runs/hello
gobble stop --workspace runs/hello
gobble resume . --workspace runs/hello
```

Doctor checks tools and proves that a sibling container can read and write the
intended project directory. The controller translates task paths from Docker's
actual mount descriptions; it does not guess Windows drive mappings.

## Current boundaries

| Concern | Behavior |
|---|---|
| Windows setup | Docker Desktop provides Linux containers. A separate Ubuntu terminal is unnecessary. WSL2 or a supported Hyper-V arrangement is Docker's setup decision. |
| Local files | Project files are mounted; an external existing workspace can be passed with `--workspace`. Samplesheets and packed outputs must be inside the project. Run from the same project directory each time. |
| Input staging | Tasks keep Gobble's existing per-attempt staging semantics. A no-copy path for large external dataset roots is not yet implemented. |
| Ownership | Linux uses the invoking UID/GID and socket group. Windows Desktop mount permissions require real-host validation. |
| Recovery | Controller host identity is stable for a daemon. Task submission records verify that daemon and the exact owned container before a rerun. |
| Terminal close | Foreground execution. Ctrl+C requests cancellation; unexpected controller loss is recovered by Resume. Detached supervision is not provided. |
| Runtime dependencies | Gobble's dependencies are cached in the image. Arbitrary new Go dependencies require preparing another matching runtime; the runtime does not silently download a new Go toolchain. |
| Tool images | Prepare tool images for offline execution. The existing task adapter may pull a missing image when online. |
| Trust | The controller has the Docker socket and executes trusted pipeline code. Analysis containers do not receive that socket. |
| Platforms | Linux/amd64 implementation; Windows/x64 launcher compiled. Real Linux Docker and Windows Desktop journeys are release gates, not local test results. |

The [runtime smoke script](../../tests/runtime-e2e/smoke.py) exercises init,
doctor, shared files, live logs, Stop, controller death, and Resume without host
Go. The Docker CI workflow runs it on Linux. Execute it separately on a real
Windows x64 Docker Desktop host, then test PowerShell Ctrl+C, spaces/Unicode,
Docker restart, and closing/reopening terminals before claiming support.

Docker references: [Windows installation](https://docs.docker.com/desktop/setup/install/windows-install/),
[WSL behavior](https://docs.docker.com/desktop/features/wsl/),
[daemon-host bind mounts](https://docs.docker.com/engine/storage/bind-mounts/),
[build contexts and ignore files](https://docs.docker.com/build/concepts/context/).
