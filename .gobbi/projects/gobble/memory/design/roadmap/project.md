# Gobble — Project Direction

## Direction

Keep one backend-independent Go engine, one structured local lifecycle, and one
bounded supported family of assay products. Product growth must follow typed
contracts, explicit graph generations, exact provenance, and complete recovery
rather than command count or upstream feature parity.

The local engine came first. The present direction is to maintain the five
accepted products as one coherent family without turning Gobble into a DSL,
module registry, nf-core clone, or integrated multiomics analysis.

## Current position

- The immutable `v0.1.0` tag is the released engine preview.
- Five local product graphs are accepted in current source: WGS joint germline,
  bulk RNA-seq, Methyl-seq, ATAC-seq, and scRNA-seq.
- Their executable baseline is commit
  `f21a858c66a2d95ce8eff469e6db2bfa3240c3a5`, tree
  `c90dfe77192c2528f8fd54d17f4d9547b09a6998`.
- The product family is unreleased. `v0.1.0` and `@latest` do not provide it.
- Support remains engineering-only on trusted-local `linux/amd64` Docker
  execution with caller-owned local workspaces.
- The five products and their graph generations are current in
  [Assay product family and ownership](../feature/assets.md).

## Completed local outcomes

### Agent-operable engine

The Go API and generic command compose, validate, plan, run, inspect, release,
and resume structured local pipelines. Module, Branch, Merge, Scatter, Gather,
and When use File, Group, and Tree artifacts. Recovery preserves fail-closed
occupancy and backend uncertainty.

### Five-product family

Every accepted product now has typed construction, a selected dated path,
strict assay data, required outputs, one manifest authority, and design, build,
customize, run, resume, stop, and failure evidence. WGS ends at an unfiltered
joint callset. RNA uses STAR-Salmon. Methyl uses a truthful Bismark Tree. ATAC
reaches consensus counts. scRNA reaches QCatch and combined raw h5ad.

## Current maintenance horizon

Maintain the accepted package paths, graph generations, typed data and config,
task and artifact identities, required outputs, image digests, fixture
authority, and lifecycle behavior together. A default change is a compatibility
event. A release carrying the family must name all five graph generations and
their Go API, CLI, workspace, image, and recovery effects.

This is a maintenance and release boundary, not a commitment to add every
excluded route. Deferred outcomes remain in [Backlog Memory](../../backlogs/README.md)
only where a specific useful outcome already has durable deferral context.

## Directional later horizons

### HPC-ready engine

The same model may later run through a first HPC adapter such as Slurm, with
shared and node-local storage, queue mapping, and backend reconciliation. This
requires a separate accepted design and must not make Slurm the core model.

### Cloud and service-ready engine

Cloud batch or Kubernetes, object storage, and persistent service state remain
after an HPC-ready model. No current product requires them.

### Agent-native ecosystem

Component discovery, catalogs, plan comparison, assisted diagnosis, and
application integration remain later ecosystem possibilities. The current
`assets/modules` tree is deliberately not that registry.

These later horizons preserve earlier project direction. They are not current
tasks, schedules, or commitments.

## Replan and stop

- Replan a product when an accepted stage cannot keep truthful artifact
  membership, exact command identity, singular fixture authority, or complete
  strict fan-in.
- Replan compatibility when a default task, image, port, path, sheet meaning,
  or graph generation changes.
- Stop a support claim rather than describe a partial assay, simulated command,
  missing backend disposition, or small-data result as complete evidence.
- Return to user design before adding integrated cross-assay semantics, a new
  assay, an optional route such as WGS VQSR, serialized parameters, or a remote
  backend.
