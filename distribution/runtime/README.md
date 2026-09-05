# Install Gobble with Docker

The default installation is a native **Gobble launcher + Docker**. Your coding
agent edits local files and calls `gobble`; Go, Git, and the pipeline engine run
inside the selected Linux runtime. Analysis tools run in sibling containers.
You do not need Go or bioinformatics tools installed on the host.

This is a development preview built from an exact Git checkout. Release images,
signed installers, and checksummed release downloads are not yet published.

| Computer | Launcher | Runtime and analysis tools |
|---|---|---|
| macOS, Apple Silicon | Native `darwin/arm64` | `linux/amd64` through Docker Desktop emulation |
| macOS, Intel | Native `darwin/amd64` | `linux/amd64` in Docker Desktop |
| Windows, x64 | Native `windows/amd64` | `linux/amd64` in Docker Desktop |
| Linux, x64 | Native `linux/amd64` | `linux/amd64` in local Docker Engine |

Native Mac launcher tests run on both architectures. Linux Docker CI covers the
installed launcher, file sharing, logs, Stop, controller loss, and Resume.
**Real Mac and Windows Docker Desktop acceptance is still a release gate.**
Cross-compiling a launcher or passing Linux CI does not establish Desktop support.

## macOS: Apple Silicon and Intel

1. Install [Docker Desktop for Mac](https://docs.docker.com/desktop/setup/install/mac-install/)
   for your processor and open it. Wait for the engine to start. macOS uses
   Docker's Linux VM and does not need WSL2.
2. Install Git if needed (`git --version` prompts for Apple's command line tools),
   then obtain this branch:

   ```sh
   git clone --branch develop https://github.com/HahyeonJeon/gobble.git
   cd gobble
   bash distribution/runtime/setup.sh
   ```

3. Activate the installed launcher in this terminal:

   ```sh
   source "$HOME/.local/share/gobble/env.sh"
   gobble demo rnaseq my-rnaseq
   cd my-rnaseq
   gobble doctor
   ```

Continue with the [actual pipeline walkthrough](../../docs/tutorials.md).
Use the same `source` command in each new terminal. If desired, add that line to
your shell startup file after checking its contents. The setup script does not
edit shell startup files or require administrator access.

The script detects Intel/Apple Silicon, builds inside Docker, checks that the
amd64 runtime starts, and installs the matching launcher in `~/.local/bin`.
It finds Docker in PATH or Docker Desktop's usual Mac CLI locations. The runtime
is always amd64; building the launcher natively does not change analysis-tool
architecture. Emulation can be slower, especially on first compilation or large
analyses. If the runtime smoke check fails, check Docker Desktop's amd64
emulation configuration before retrying.

Allocate Docker enough CPU, memory, and disk for your chosen assay; see the
[resource table](../../docs/tutorials.md#choose-an-example). Keep active projects
in a local folder shared with Docker. Doctor checks actual sibling-container
read and write access, including paths resolved through macOS symlinks.

## Linux x64

Install and start a local Docker Engine, ensure your user can run `docker info`,
then use the same checkout and setup commands:

```sh
git clone --branch develop https://github.com/HahyeonJeon/gobble.git
cd gobble
bash distribution/runtime/setup.sh
source "$HOME/.local/share/gobble/env.sh"
gobble demo rnaseq my-rnaseq
cd my-rnaseq
gobble doctor
```

The controller uses your UID/GID and socket group. Advanced users can instead
use the [direct Linux installation](../../docs/installation.md).

## Windows x64: PowerShell

Install Git and [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/),
then start Docker with Linux containers. A separate Ubuntu terminal is
unnecessary. Docker Desktop itself requires its supported WSL2 or Hyper-V setup.

```powershell
git clone --branch develop https://github.com/HahyeonJeon/gobble.git
Set-Location gobble
docker build --platform linux/amd64 -f distribution/runtime/Dockerfile --target runtime -t gobble-runtime:preview .
docker build --platform linux/amd64 -f distribution/runtime/Dockerfile --target launcher-artifacts --output ./dist .
$env:PATH = "$PWD\dist\windows-amd64;$env:PATH"
$env:GOBBLE_RUNTIME_IMAGE = "gobble-runtime:preview"
gobble demo rnaseq my-rnaseq
Set-Location my-rnaseq
gobble doctor
```

In each new PowerShell terminal, set PATH using the absolute path to
`gobble\dist\windows-amd64`. Existing projects remember their runtime image;
`GOBBLE_RUNTIME_IMAGE` selects the initial image for new project roots.

## Build artifacts manually

From a clean, committed checkout:

```sh
docker build --platform linux/amd64 -f distribution/runtime/Dockerfile --target runtime -t gobble-runtime:preview .
docker build --platform linux/amd64 -f distribution/runtime/Dockerfile --target launcher-artifacts --output ./dist .
```

Artifacts are `dist/{linux-amd64,windows-amd64,darwin-amd64,darwin-arm64}/gobble`
(`gobble.exe` on Windows). Add your platform's directory to PATH and set
`GOBBLE_RUNTIME_IMAGE=gobble-runtime:preview` in your shell. Keep `.git` in the
build context: the runtime checks the exact source identity. The Dockerfile
excludes run data, downloads, and output binaries from the image.

## Your first results

The [test-data guide](../../docs/tutorials.md) walks through existing RNA-seq and
WGS pipelines, monitoring, Stop, Resume, and additional assays. For a very small
installation check that does not download analysis images:

```sh
gobble init my-pipeline
cd my-pipeline
gobble doctor
gobble plan .
gobble run . --workspace runs/hello
```

`runs/hello/results/sequence-count.txt` should contain `2`. This small check uses
`sh` and `awk` in the runtime; `demo` runs the actual assay tools.

## Runtime identity and recovery

The launcher saves the exact local runtime image ID and daemon ID in the
project. It checks `linux/amd64` before selecting or reusing the image. Keep that
image and `.gobble-runtime.json` for existing runs. A tag change or reinstall
does not upgrade an existing project's runtime. A different daemon is refused
because it cannot prove the state of previous tasks.

The Mac client socket may live in `~/.docker/run/docker.sock`; the controller
mounts the socket inside Desktop's Linux VM. No host `/var/run/docker.sock`
symlink or socket permission workaround is required by the launcher.

| Concern | Behavior |
|---|---|
| Local paths | Run commands from the same project directory. External existing workspaces can be passed with `--workspace`; samplesheets and packed outputs stay inside the project. |
| Input staging | Each attempt keeps Gobble's existing input-copy semantics. Large external dataset roots do not yet have a no-copy mode. |
| Terminal close | Execution is foreground. Ctrl+C requests cancellation; Resume recovers unexpected controller loss. Keep the run terminal open. |
| Dependencies | Gobble dependencies are cached. New Go dependencies require another matching prepared runtime; no toolchain is silently downloaded. |
| Offline use | Prepare test data and tool images while online first. Missing analysis images may be pulled during a run. |
| Trust | The controller has the Docker socket and executes trusted pipeline code. Analysis containers do not receive that socket. |

## Platform acceptance

Contributors with Docker and Python 3 can run:

```sh
python3 tests/runtime-e2e/smoke.py /absolute/path/to/gobble
python3 tests/runtime-e2e/demo.py /absolute/path/to/gobble rnaseq
python3 tests/runtime-e2e/demo.py /absolute/path/to/gobble wgs
```

Set `GOBBLE_RUNTIME_IMAGE` first. The scripts exclude host Go and exercise actual
containers in fresh projects with spaces and Unicode paths. `smoke.py` checks
live logs, Stop, repeated Stop, controller death, and Resume; `demo.py` checks
real assay outputs and reuse. Record host/processor, Docker version, commit,
and elapsed times. On real Desktop hosts also test interactive watch, Ctrl+C,
Docker restart, external workspace sharing, and reopening terminals.

GitHub's hosted Apple Silicon runners do not support nested virtualization;
the native Mac CI job is deliberately separate from these Desktop gates.

Docker references: [Mac permissions](https://docs.docker.com/desktop/setup/install/mac-permission-requirements/),
[multi-platform execution](https://docs.docker.com/build/building/multi-platform/),
[Windows setup](https://docs.docker.com/desktop/setup/install/windows-install/),
[bind mounts](https://docs.docker.com/engine/storage/bind-mounts/).
