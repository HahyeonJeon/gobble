# compose-pipeline — Feature

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. A refused use is not a feature file.

Open Actors, Scope, or Behavior on a feature that can sit in the first horizon
is blocking. Ask before leaving those Open.

## Purpose

- Statement: Let an agent or human developer express a bioinformatics pipeline in Go with reusable tasks and modules, explicit inputs, outputs, parameters, resources, and environments, branch and merge, and scatter, gather, and when.
- Source: `core-tasks`

## Actors

- Actor: Agent author
- Role: Writes or edits library code that composes the pipeline.
- Actor: Human Go developer
- Role: Writes or edits the same library code. Neither actor is a backend operator.
- Source: `task-actors`

## Scope

- In scope: First-horizon implementation of reusable tasks and modules, explicit inputs, outputs, parameters, resources, environments, branch and merge (fan-out and fan-in), scatter, gather, and when in Go. A container task must declare its Docker image. A local-process task must not require an image. Compose-time rejection of a cycle, missing input or output, or a DSL-as-source file so the graph cannot become a plan. Validate-plan re-checks those defects and owns conflicting outputs, unsupported backends, dry run, DAG, and plan emission.
- Out of scope: A Gobble DSL, GUI authoring, built-in assay-specific tools, plan-time Document expansion of reservedIdentity, executing or scheduling a run, and emitting a successful plan.
- Source: `task-scope`, `refused-use`

## Behavior

### Normal

- Write a Go pipeline with at least one branch and one merge that the library accepts.
- Source: `task-behavior`

### Alternate

- Modify an existing pipeline and keep the same model.
- Source: `task-behavior`

### Invalid

- A cycle, missing input or output, or a Gobble-specific language file as the source of truth. Rejected composition returns a structured validation error and is not a runnable plan.
- Source: `task-behavior`

## Failure / Recovery

### Failure

- The consumer sees a structured error that names the composition defect.
- Source: `failure-recovery`

### Recovery

- Fix the Go pipeline and compose again. Human translation of raw logs is not the recovery path.
- Source: `failure-recovery`

## Structure

### Parts

- Statement: Implemented by the Go core library pipeline model.
- Source: `shape`

### Data

- Statement: This feature creates pipeline definitions in Go source. It does not write run state. Authoritative source for definitions is the consumer’s source tree.
- Source: `data`

### Interfaces

- Statement: The public Go compose API. First-horizon exit still requires the same loop on the CLI. Shipped `gobble compose [PKG] [--sample PATH]`; contract in [architecture/system.md](../architecture/system.md) Interfaces Current.
- Current: Authors build a `Pipeline` and call `Compose` for an immutable `Graph`. The supported preview contract is `Compose`, `Validate`, `BuildPlan`, `Run`, `Inspect`, `Release`, and `Resume`; Module, Branch, Merge, Scatter, Gather, and When; PathSpec and File, Group, or Tree binds; samplesheet parse types with `ParseSampleSheet` and explicit `LoadSampleSheetFile`; and structured `Error`, `Defect`, and `DefectCode` values. `WriteTo` is the supported Plan option constructor. Graph readers `Name`, `TaskIDs`, `InputNames`, and `Edges`, plus Plan JSON, support the loop. Other exported names are provisional. `SetSampleSheetPath`, `SampleSheetPath`, and `LoadSampleSheet` remain provisional process-owned helpers for generated CLI children and exclusive proof pipelines. They are not goroutine identity. Supported concurrent callers pass an explicit path to `LoadSampleSheetFile`. The CLI injects `--sample PATH` or process-cwd `samplesheet.csv` before `Pipeline()`. Required sheet columns are `sample` and `read1`; empty `read2` is allowed. Registered Command, Inputs, Outputs, Params, Env, and Group members are copied. Returned recorded errors and Defect paths are caller-owned. Scatter membership is `From` a Group, Tree, or File Handle. Scatter members occupy runtime instance segments with shard index 0. Gather waits for known membership. `When` uses a named producer File or a boolean param and is re-evaluated on Resume. A Bind is exactly one of File, Group, or Tree. File is one regular-file artifact. Group is an ordered set of named regular-file members. Tree is a declared directory plus dest `.gobble-tree.json`; `Directory` is placement, not an artifact. First-party proofs live in [assets](assets.md) and are not product tools. The `public-contract` question is closed for this pre-1.0 preview.
- Source: `interfaces`

## Constraints and qualities

- Statement: One maintainer; local first. A simple Go API is required. No proprietary DSL. Agent-operability wins as an outcome.
- Source: `constraints`, `quality-priority`

## Open questions

- Not applicable — `public-contract` is closed for the narrow pre-1.0 trusted-local preview. Other exports remain provisional.

- Source: open ids used above

## Interview sources

- `core-tasks`
- `task-actors`
- `task-scope`
- `task-behavior`
- `refused-use`
- `failure-recovery`
- `shape`
- `data`
- `interfaces`
- `constraints`
- `quality-priority`
- `public-contract`
- Source: the topic ids actually cited
