# Gobble — Overview

## Purpose

Gobble gives coding agents and Go authors a typed pipeline engine with
machine-readable planning, execution, inspection, failure, and recovery. It also
provides five supported local assay products built on that engine.

Gobble is the shared engine and command surface. It is not a sixth assay. The
five products are WGS joint germline, bulk RNA-seq, Methyl-seq, ATAC-seq, and
scRNA-seq. Their current identities and selected paths live in
[Assay product family and ownership](../feature/assets.md).

## Current outcome

Each product has an assay-owned strict CSV loader, typed sample values, a fresh
`DefaultConfig`, a pure `Build`, a default `Pipeline()` command adapter, selected
command modules, required artifacts, one fixture and image manifest, and all
seven lifecycle outcomes: design, build, customize, run, resume, stop, and
failure.

The products form a consistent family, not an integrated multiomics workflow.
There is no universal sample model, cross-assay graph, join identity,
missing-modality policy, or combined scientific result.

## User outcome

An author can load assay data, change typed command policy, build and inspect a
plan, and use either the Go API or generic command. An operator can run the
selected graph in an exclusive local workspace, inspect structured state,
release occupancy when safe, and resume only work whose identity no longer
matches.

A human operator may instead receive one packed runner containing one selected
pipeline. The packed runner removes the package operand; it does not add a
second pipeline model.

## Scope and non-goals

Current support is engineering-only on trusted-local `linux/amd64` with local
files and Docker. It covers graph construction, declared command execution,
artifacts, provenance, structured failure, and recovery. Docker isolation is a
convenience, not a sandbox.

The family does not claim scientific, clinical, diagnostic, regulatory, or
production-scale validity. It does not imply nf-core support or endorsement.
WGS ends at an indexed, unfiltered joint callset. Integrated cross-assay
analysis, optional nf-core routes, extra assays, serialized product parameters,
a component registry, remote execution, HPC, cloud, and services remain outside
the current result.

## Release position

The published `v0.1.0` tag contains the engine preview and predates the product
family. The five executable product graphs are local and unreleased at baseline
commit `f21a858c66a2d95ce8eff469e6db2bfa3240c3a5`, tree
`c90dfe77192c2528f8fd54d17f4d9547b09a6998`. A consumer must use an exact
trusted checkout and a command built from the same selected module graph.

## Constraints and authority

One maintainer owns product defaults and support decisions. Upstream benchmark
changes do not update Gobble automatically. A default command, image, task ID,
port, destination, sheet meaning, or graph generation changes only through a
deliberate compatibility decision.

## Vocabulary

| Term | Meaning |
|---|---|
| Gobble | The shared Go library, engine, generic command, workspace, and recovery model used by all five products. |
| Product | One supported assay package, selected graph generation, typed contract, required outputs, pinned defaults, and lifecycle evidence. |
| Module | One command or subcommand represented by one Gobble task. It is not a remote registry unit. |
| Pipeline | One assay-owned typed graph under `assets/pipelines/<assay>`. |
| Lifecycle scenario | One cross-product actor outcome under `tests/scenarios/<outcome>`. |
| Graph generation | A named compatibility boundary for task and artifact identity. Named lifts require new workspaces. |
| Run workspace | The caller-owned local authority for one run's state, logs, artifacts, occupancy, and reuse decisions. |
