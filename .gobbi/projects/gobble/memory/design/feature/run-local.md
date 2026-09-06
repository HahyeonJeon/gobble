# Gobble — Run Locally

## Purpose and scope

Schedule a valid graph on the local machine using dependencies, requested CPU
and memory, and a run-level concurrency cap. Persist task attempts, state,
logs, inputs, outputs, and provenance in a caller-owned workspace.

Agents and people choose a graph and execution limits. The scheduler decides
readiness. Tasks with images use Docker; tasks with an empty image use the
process executor. Slurm, cloud batch, Kubernetes, quotas, and remote scheduling
remain outside current support.

## Distribution and lifetime

The common installation is a Docker runtime containing Go, Git, Gobble, and
authoring dependencies. Projects receive a pinned Compose file. No host Go or
Gobble installation is required for this route. External agents edit local Go
files. Direct Linux development and compatibility launchers remain available.

Run long analyses with `docker compose run -d gobble run ...`. Docker owns the
detached controller, and analysis tools run as sibling containers on the same
local daemon. A terminal or Agent can close after launch. Docker or computer
shutdown interrupts this lifetime and requires reconciliation before recovery.

Current containers target linux/amd64. Apple Silicon uses emulation. Linux
Docker evidence and native launcher tests must be distinguished from actual
Windows/macOS Docker Desktop acceptance, which remains outstanding.

## Engine and workspace contract

Run requires an effective execution identity and exclusive workspace ownership.
Preflight defects prevent task launch. Exact image inspection/acquisition can
still fail during task submission. The engine records structured task failures;
analysis-command failure and image preparation failure remain distinguishable.

Scheduler/executor integration uses Submit, Poll, Cancel, and Reconcile. Docker
tasks use the configured user identity and disabled task networking. Registry
access during image acquisition is separate. These conveniences are not an
untrusted-code sandbox. Task environment values do not become host Docker
client configuration.

Ready files must be regular files. Group members stage and publish by name.
Ready Trees require a directory and their regular root manifest. Run copies
staged inputs into isolates and publishes declared outputs by copy. It records
runtime identity, staged-input fingerprints, and published-output checksums.
Missing digest/hash evidence cannot establish reuse.

A second Run cannot acquire an occupied workspace. Stop requests settlement;
Resume reconciles before continuing. Unknown backend disposition blocks new
work. [Recovery](recover-run.md) owns the detailed lifetime and reuse contract.

Cap 0 selects the default of one; explicit positive caps are 1 through 64.
The cap limits concurrency, not each task's CPU or memory request. Current
resource scheduling does not imply measured utilization, fairness, or quotas.

## Monitoring and evidence

Persistent attempt logs and coherent control snapshots feed Inspect and Watch.
Monitoring never owns execution. The browser/service extension follows
[Application and monitoring](../architecture/application.md).

Hermetic tests prove contracts without running third-party analysis tools.
Installed Linux Docker tests exercise the common Compose route, detached
controllers, Stop/Resume, and actual WGS/RNA-seq outputs. Supported-platform and
assay claims require their own execution evidence.

See [Container distribution](../../../../../../distribution/runtime/README.md)
for setup, relative-path requirements, mounts, pinning, and recovery limits.
