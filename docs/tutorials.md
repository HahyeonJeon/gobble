# Run an existing pipeline with test data

Use the same WGS, RNA-seq, Methyl-seq, ATAC-seq, and scRNA-seq pipelines that
Gobble already provides. `gobble demo` prepares a new project, its samplesheet,
and the official test inputs; it does not substitute fake analysis commands.
Your agent can then inspect, run, and adapt that project.

Complete the [Docker installation](../distribution/runtime/README.md) first.
These commands work with Docker Compose in macOS/Linux terminals and Windows
PowerShell. No host Gobble or Go installation is needed. Use the project's own
Compose file after entering its directory.

## Choose an example

| Command name | Existing pipeline and test data | Input download | CPUs needed by largest task | Largest task memory |
|---|---|---:|---:|---:|
| `rnaseq` | STAR/Salmon with yeast reads and reference | 23.5 MiB | 4 | 8 GiB |
| `wgs` | BWA/GATK joint germline with small human reference intervals | 63.7 MiB | 4 | 6 GiB |
| `methylseq` | Bismark with small methylation test reads and reference | 10.7 MiB | 6 | 15 GiB |
| `atacseq` | BWA/MACS2 with yeast reads and reference | 63.9 MiB | 6 | 4 GiB |
| `scrnaseq` | Simpleaf/QCatch with mouse chromosome 19 data | 95.7 MiB | 4 | 8 GiB |

These are the unchanged pipeline defaults. In Docker Desktop, allocate at least
the listed CPUs and **more memory than the largest task** to leave room for the
controller and Linux VM. For example, start RNA-seq with 4 CPUs and 10–12 GiB
assigned to Docker on a machine with sufficient additional host memory. Use
`--cap 1` for these exercises: it limits concurrent tasks, not the resources of
one task. A tiny input does not automatically lower a task's request.

The table counts input bytes only. Pinned analysis images, temporary task
copies, and results require additional disk space, often many gigabytes.
Downloads need internet. First runs can spend considerable time pulling images;
Apple Silicon also runs the analysis under amd64 emulation. Elapsed times are
reported by the acceptance scripts; no fixed completion time is promised.

RNA-seq and WGS have automatic **installed-runtime, real-tool** CI checks in
[Docker tests](https://github.com/HahyeonJeon/gobble/actions/workflows/docker.yml).
The other three projects are prepared and their graphs validated by the
hermetic suite; their full installed-runtime execution remains an acceptance
gate. See each workflow run for the result at your selected commit. Linux CI
does not establish Docker Desktop behavior on Mac or Windows.

## 1. Prepare RNA-seq

From a directory containing the downloaded common `compose.yaml`:

```sh
docker compose run --rm gobble demo rnaseq my-rnaseq
cd my-rnaseq
docker compose run --rm gobble doctor
```

`demo` downloads inputs from the existing assay's immutable upstream URLs,
checks their exact sizes and SHA-256 hashes, and creates:

| File or directory | Purpose |
|---|---|
| `compose.yaml` | Selects the exact runtime image that created this project |
| `pipeline.go` | Calls the existing RNA-seq pipeline with its default configuration |
| `samplesheet.csv` | Sample names, paired/single read paths, and strandedness |
| `fixture-manifest.json` | Data checksums, provenance, licenses, and tool-image references |
| `runs/demo/in/` | Prepared FASTQs and reference files |
| `README.md`, `AGENTS.md` | Local instructions for you and your coding agent |

Go and Git run inside the runtime. The project gets its own local Git history
and exact runtime lock. No analysis has started yet. If a download fails,
repeat the same command from the parent directory: completed verified downloads
are reused from `.gobble-cache/fixtures`. An existing project is never replaced.
If setup failed after creating the directory, keep it for diagnosis and retry
with a new project name.

`doctor` must succeed, including its sibling-container read/write check. Keep
the project in a folder shared with Docker Desktop. Avoid synced/network folders
for active runs, and use a path with enough free local disk space.

## 2. Ask your agent to explain and plan

Open `my-rnaseq` in your coding agent and give it this prompt:

> Read AGENTS.md and README.md. This is the existing Gobble RNA-seq pipeline
> with official test data. Explain the samplesheet, reference, main analysis
> steps, and expected reports in beginner-friendly language. Run doctor,
> validate, and plan. Show me the plan and resource requirements before running
> it with workspace runs/demo and cap 1. Keep the existing analysis settings
> for this first exercise. Use inspect to explain failures and Stop/Resume for
> recovery; preserve the runtime lock and run state.

You can run the same preparation yourself:

```sh
docker compose run --rm gobble validate .
docker compose run --rm gobble plan .
docker compose run -d gobble run . --workspace runs/demo --cap 1
```

Save the returned controller ID and check `docker logs CONTAINER_ID` for startup errors.
Detached execution continues after the terminal closes. Use inspect to establish
success; a successful Docker launch is not a completed analysis. Gobble runs the
real tools in pinned Docker images.
The samplesheet uses paths relative to `runs/demo`; do not prepend the project
path or change them to macOS/Windows host paths.

## 3. Monitor and open the results

In another terminal, activate the Gobble installation environment, change to
the **same project directory**, and run:

```sh
docker compose run --rm gobble watch --workspace runs/demo
```

Press `/` to find a sample, `!` for problems, and `q` to close the monitor.
Closing the monitor leaves the analysis running. For an agent or a text report:

```sh
docker compose run --rm gobble inspect run --workspace runs/demo
docker compose run --rm gobble inspect errors --workspace runs/demo
```

After a successful RNA-seq run, open these local files:

- `runs/demo/results/rnaseq/multiqc/multiqc_report.html` — combined QC report.
- `runs/demo/results/rnaseq/matrices/gene_counts.tsv` — gene count matrix.
- `runs/demo/results/rnaseq/matrices/transcript_tpm.tsv` — transcript abundance matrix.
- `runs/demo/results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam` — processed alignment.

The report opens in your normal browser. Ask the agent to explain warnings and
what the test data can show. This exercise checks software execution; successful
files alone do not establish biological accuracy or suitability for real data.

## 4. Try Stop and Resume

While the run is active, from another terminal:

```sh
docker compose run --rm gobble stop --workspace runs/demo
docker compose run --rm gobble inspect run --workspace runs/demo
```

Wait for Stop to report `settled`. If it reports `requested`, repeat Stop or
inspect until the owner has settled. Then:

```sh
docker compose run -d gobble resume . --workspace runs/demo --cap 1
```

Completed compatible tasks are reused. Interrupted tasks start a new attempt;
Gobble does not continue inside an interrupted STAR, GATK, or other tool.
Repeated Stop is safe. If the small run has already finished, Resume should
reuse its completed work without rerunning it.

Closing a terminal after detached launch leaves the analysis running. If Docker
or the computer restarts, restore Docker access, return to the same project,
and Resume. Preserve the project's runtime lock, pinned image,
and workspace. An unknown backend must be reconciled before another run can own
the workspace; follow [recovery](operations.md#recovery) if Resume reports it.

## 5. Run WGS or another existing assay

From the parent directory, create a separate project:

```sh
docker compose run --rm gobble demo wgs my-wgs
cd my-wgs
docker compose run --rm gobble doctor
docker compose run --rm gobble validate .
docker compose run --rm gobble plan .
docker compose run -d gobble run . --workspace runs/demo --cap 1
```

WGS preparation also splits the pinned interval BED into the two existing
scatter members expected by its default configuration. Expected results include:

- `runs/demo/results/wgs/joint/joint_germline.vcf.gz` — unfiltered joint callset.
- `runs/demo/results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam`.
- `runs/demo/results/wgs/samples/patient2/testT/alignment/testT.recalibrated.bam`.
- `runs/demo/results/wgs/multiqc/multiqc_report.html`.

Use the same sequence for `methylseq`, `atacseq`, or `scrnaseq`, with a new project
name and the resources in the table. The generated README lists that assay's
expected results. Methyl-seq uses its generated Bismark-index route; the fixture
manifest also contains a ready-index archive for separate engineering tests.

For a fresh repeat, create another project with `demo`. For your own data, ask
the agent to configure samples and matching references using the
[pipeline guides](../README.md#bioinformatics-pipelines). Keep the working test
project as a baseline; scientific configuration changes require review.

## Troubleshooting

| Symptom | What to check |
|---|---|
| `Docker is unavailable` | Start Docker Desktop/Engine and select the same local Linux engine used by the project. |
| Runtime image is `linux/arm64` | Use the official pinned linux/amd64 runtime; Apple Silicon needs working Docker Desktop emulation. |
| SHA-256, HTTP, or download failure | Restore internet access and repeat `demo` from the parent directory. Never bypass verification. |
| Resource validation rejects a task | Allocate enough CPUs/memory to Docker; lowering `--cap` does not shrink a task. |
| Image pull or registry failure | Restore registry access and Resume. Preserve the failed attempt for diagnosis. |
| A task fails | Read `inspect errors`, then its logs in the monitor. Ask the agent to explain the exact failure before editing the pipeline. |
| Files are not visible to a sibling container | Run doctor in the project, check Desktop file sharing and folder permissions. |

For reproducible acceptance on a machine with Docker and Python 3:

```sh
python3 tests/runtime-e2e/demo.py compose rnaseq
python3 tests/runtime-e2e/demo.py compose wgs
```

Run those from the Gobble checkout. They exclude host Go, prepare fresh projects,
run real tools, require non-empty expected files, and verify that unchanged
Resume does not increase task attempts. Failed workspaces remain for diagnosis.
