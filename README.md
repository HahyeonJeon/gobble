<div align="center">

# Gobble

**Design pipelines with a coding agent. Run them locally. See what is happening.**

[![Go tests](https://github.com/HahyeonJeon/gobble/actions/workflows/test.yml/badge.svg?branch=develop)](https://github.com/HahyeonJeon/gobble/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-7cc5c9)](LICENSE)
[![Status: pre-release](https://img.shields.io/badge/status-pre--release-d5b67b)](CHANGELOG.md)

[Get started](#get-started) · [Pipelines](#bioinformatics-pipelines) · [Monitoring](#see-your-run) · [Documentation](#documentation)

</div>

Gobble is a Go pipeline engine for local bioinformatics work. A coding agent or
Go developer assembles the steps, inputs, outputs, and resources. Gobble runs the
work, records what happened, and reuses compatible completed tasks when you resume.

- **Composable pipelines** — connect reusable modules, branches, and parallel tasks.
- **Local execution** — run tools in Docker with declared resources and local files.
- **Visible progress** — follow the pipeline graph, search samples, and inspect failures.
- **Recoverable work** — inspect attempts, validate outputs, and selectively rerun work.
- **Shareable runners** — pack one pipeline into a binary an operator can use without Go.

![Gobble terminal dashboard with a pipeline graph, sample progress, and failed tasks](docs/images/monitor-dashboard.png)

*Actual terminal renderer with illustrative fixture data. See
[Monitoring](docs/monitoring.md) for controls and state meanings.*

> **Development preview:** this branch contains the pipeline family and terminal
> monitor planned for the next release. Published `v0.1.0` predates those features.
> Current execution support is **Linux/amd64**. The accepted v0.2.0 default is
> **Docker-based installation and agent-driven authoring**, with direct Linux
> installation for advanced users. The launcher and Windows route are still
> being implemented; see the [installation design](docs/v0.2.0/container-installation.md).

## Get started

Start with a tiny example that counts two FASTA sequences. It needs **Go 1.26+,
Git, and a Linux/amd64 environment with `sh` and `awk`**. No Docker or sequencing
reference download is needed for this example.

From this checkout:

```sh
go build -o ./bin/gobble ./cmd/gobble
mkdir -p ./runs/hello/inputs
cp examples/hello/sequences.fasta ./runs/hello/inputs/sequences.fasta

./bin/gobble plan ./examples/hello
./bin/gobble run ./examples/hello --workspace ./runs/hello
cat ./runs/hello/results/sequence-count.txt
# 2

./bin/gobble inspect run --workspace ./runs/hello
./bin/gobble release --workspace ./runs/hello
```

Use a new workspace for the example and keep the same command binary for its
lifecycle. [Hello Gobble](examples/hello/README.md) walks through the code,
selective reruns, and packing a runner. For an agent-owned project outside this
repository, follow [Installation and version selection](docs/installation.md).

## Work with your coding agent

Give your agent the analysis goal, sample files, reference organism/build,
available CPU and memory, and desired outputs. Ask it to:

1. Select or assemble a pipeline and explain its inputs and analysis choices.
2. Validate it and show the execution plan before starting.
3. Stage local inputs and check the required tools and resources.
4. Run the pipeline, open its monitor, and explain any failed tasks.

The agent writes ordinary Go using Gobble's library and invokes the CLI. Gobble
currently does not bundle an AI agent. Customization belongs in your project's
pipeline/configuration code. See [Authoring](docs/authoring.md).

## Bioinformatics pipelines

| Pipeline | What it runs | Guide |
|---|---|---|
| Whole-genome sequencing | BWA/GATK germline workflow through an unfiltered joint VCF | [WGS](assets/pipelines/wgs/README.md) |
| Bulk RNA sequencing | STAR-Salmon, quantification, and cohort QC | [RNA-seq](assets/pipelines/rnaseq/README.md) |
| DNA methylation sequencing | Directional Bismark/Bowtie2 workflow | [Methyl-seq](assets/pipelines/methylseq/README.md) |
| Chromatin accessibility | BWA, MACS2 peaks, consensus counts, and cohort QC | [ATAC-seq](assets/pipelines/atacseq/README.md) |
| Single-cell RNA sequencing | Simpleaf, QCatch, and matrix assembly | [scRNA-seq](assets/pipelines/scrnaseq/README.md) |

These pipelines require Docker, prepared samplesheets, and staged reference/data
files. Each guide describes configuration and outputs. The engineering tests do
not establish scientific or clinical validity. Exact reference workflows,
versions, and evidence are in [Products](docs/products.md) and
[Provenance](docs/provenance.md).

## See your run

Open another terminal and use the same command that started the run:

```sh
./bin/gobble watch --workspace ./runs/hello
```

The dashboard starts with the graph and overall progress. Press `/` to search a
sample, `!` for problems, and Enter to inspect tasks. Press `q` to close the
monitor while execution continues. The small Hello example finishes quickly;
longer pipelines make the live progress view useful.

For agents and scripts, use structured output:

```sh
./bin/gobble inspect monitor --workspace ./runs/hello
./bin/gobble inspect errors --workspace ./runs/hello
```

Local process logs can update while tasks run. **Docker task logs are currently
collected after the container stops**; live Docker log collection is a v0.2.0
improvement. The monitor itself is read-only.

## Stop and resume

In the terminal running `run` or `resume`, press **Ctrl+C** to request
cancellation. Completed results remain available. There is currently no
standalone `gobble stop` command.

After the owning process exits, the current recovery sequence is:

```sh
./bin/gobble inspect run --workspace ./runs/hello
./bin/gobble release --workspace ./runs/hello
./bin/gobble resume ./examples/hello --workspace ./runs/hello
./bin/gobble release --workspace ./runs/hello
```

Release checks backend state and closes the run lock; it does not delete results.
Resume reuses valid completed work and reruns unfinished or changed tasks. It
restarts tasks, rather than continuing inside an interrupted analysis tool.
For a fully completed unchanged run, there may be nothing to resume.

If backend state is unknown, restore Docker access and follow the
[recovery guide](docs/operations.md#recovery). Closing the run terminal is not a
supported detached-execution mode. State is published as
[complete checkpoints](docs/checkpoints.md). Docker submissions now record the
owning engine and container ID before starting work; see
[Docker execution and recovery](docs/docker-execution.md) for the tested
boundaries and remaining live validation.

## Share a pipeline

From a clean, committed checkout:

```sh
./bin/gobble pack ./examples/hello --output ./bin/hello-runner
```

An operator can run `./bin/hello-runner run --workspace DIR` with staged inputs,
without Go or a package operand. The runner targets Linux/amd64 and still needs
the tools or Docker used by its pipeline. See the
[example guide](examples/hello/README.md).

## Documentation

| Start here | Details |
|---|---|
| [Installation](docs/installation.md) | Choose a revision and set up an agent-owned project |
| [Authoring](docs/authoring.md) | Samplesheets, typed configuration, modules, and outputs |
| [Operations](docs/operations.md) | Environment preparation, run lifecycle, and recovery |
| [Monitoring](docs/monitoring.md) | Dashboard, sample search, task details, and shortcuts |
| [Provenance](docs/provenance.md) | Pinned tools/data, reference workflows, and test evidence |
| [v0.2.0 review and plan](docs/v0.2.0/review.md) | Findings, proposed designs, and sequential improvements |

## Development

Resolve dependencies once, then run the local suite:

```sh
go mod download all
GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...
go vet ./...
```

[Docker tests](tests/docker/README.md) describe Linux userspace checks and the
separate real-container smoke suite. Windows/WSL requires its own validation.

Gobble is pre-1.0 and its public API may change. Current supported execution is
trusted local Linux/amd64; cloud, cluster, and remote execution are future work.
Pipeline code and container images must be trusted. Data and run state remain
local; Gobble does not provide an account or hosted analysis service.

Gobble is [MIT licensed](LICENSE). Pipeline tools, images, and datasets retain
their own licenses. See [Changelog](CHANGELOG.md) for release history.
