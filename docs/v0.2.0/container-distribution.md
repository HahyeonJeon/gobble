# Common container distribution

Status: implementation plan accepted through the container-first discussion.

## Goal

Use one Docker installation and command contract on Linux, Windows, and macOS.
An external coding agent edits local Go files and follows the same procedure a
person uses. Go, Git, Gobble, and the authoring dependencies live in the runtime
image. Analysis tasks keep their individually pinned Linux images.

## Review

- The runtime image already includes the compiler, engine, source identity, and
  dependency cache. The image currently has to be built on the user's machine.
- The native launcher owns runtime/daemon selection, controller discovery,
  path translation, permissions, and signal forwarding. A bare Docker command
  cannot yet replace it safely.
- Workspace ownership uses a hostname. Compose creates a new hostname per
  command, so recovery needs an identity derived from the verified Docker daemon.
- The sibling mount probe assumes the command runs at the mount root; it fails
  when a project is below that root. Task mounts already use longest-prefix
  translation and reject read-only or unshared attempts.
- Runtime locks retain a local image ID but no downloadable registry reference.
- Foreground execution belongs to the calling terminal. Docker's detached
  one-off containers can own long runs while independent commands inspect or
  stop them. Engine checkpoints and reconciliation remain necessary.
- Real RNA-seq and WGS CI exists, but currently enters through the launcher.
  Real macOS/Windows Docker Desktop acceptance is still outstanding.

## Decisions

1. Publish a Linux/amd64 runtime and a common Compose configuration. Do not
   classify installations by beginner/advanced users. Retain the native launcher
   and direct Linux entry points for existing users and development.
2. Prepare runtime identity and project permissions inside the runtime process.
   Keep the host Docker daemon; analysis containers are siblings. Do not run a
   second daemon inside Gobble.
3. Generate a project Compose file with the exact runtime registry digest when
   available, otherwise the exact local image ID. Preserve daemon and image
   checks for existing runs. A different daemon is not automatic migration.
4. Use `docker compose run --rm gobble ...` for short commands and
   `docker compose run -d gobble run ...` for detached analysis. Keep failed
   controller containers for diagnostics. Use Gobble Stop before cleanup.
5. Keep pipeline code, inputs, checkpoints, and results in a writable bind mount.
   Workspaces outside shared mounts fail explicitly. Map nested paths using the
   actual daemon mount description, including Desktop paths.
6. Keep Linux/amd64 explicit for the controller and each analysis task. ARM
   emulation needs real pipeline acceptance; native ARM tool support is separate.

## Sequential work and acceptance

1. Implement container bootstrap, immutable project configuration, nested mount
   probing, daemon-based occupancy identity, and ownership handling.
2. Add a tested publication workflow and remove local builds from the normal
   installation instructions. Publication must follow the Docker test gates.
3. Rewrite common installation, generated project guidance, Agent instructions,
   README, and real-assay examples around Compose.
4. Test fresh setup without host Go or a launcher, nested/non-ASCII paths,
   detached client exit, live logs, duplicate ownership, Stop, controller death,
   Resume reuse, runtime mismatch, and non-root project writes. Run RNA-seq and
   WGS through Compose. Check anonymous image access after publication.

Host reboot/Docker Desktop restart and interactive terminal behavior require
real Desktop acceptance. Containerization alone does not prove these scenarios.
