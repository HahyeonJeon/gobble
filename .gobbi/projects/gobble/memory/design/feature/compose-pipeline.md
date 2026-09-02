# compose-pipeline — Feature

## Purpose

- Let an agent or human developer express a pipeline in Go with reusable tasks
  and command modules, explicit artifacts, parameters, resources, environments,
  branch and merge, and Scatter, Gather, and When.
- Let first-party assay packages expose strict typed data and configuration over
  that same public model without adding a second pipeline language.

## Actors

- **Agent or Go author:** Builds a graph from explicit values and reviews Plan
  JSON. The author does not decide runtime readiness.
- **Product maintainer:** Owns one selected assay graph and its typed policy.
- **Module maintainer:** Owns one command task and its ports, image, options, and
  argv boundary.

## Scope

- **In scope:** Public Go composition; copied command, input, output, parameter,
  environment, and Group values; File, Group, and Tree artifacts; modules,
  branch, merge, Scatter, Gather, and When; structured compose defects; and the
  shared first-party `Parse`/`Load`/`DefaultConfig`/`Build`/`Pipeline` contract.
- **Out of scope:** A Gobble DSL, YAML or JSON product parameters, a universal
  multiomics sheet, a component registry, arbitrary module lists, plan-time
  expansion of sample data, execution, and scheduler policy.

## Behavior

### Normal

- Build an immutable `Graph` from a `Pipeline`, validate it, and emit Plan JSON
  before execution.
- Product packages build from typed assay samples and config. Sample, lane, run,
  and replicate expansion is compose-time.

### Alternate

- Change a named typed option or safe argv extra and observe the exact command,
  graph, or artifact effect in the plan.
- Use runtime Scatter/Gather when a declared Group or Tree is the homogeneous
  membership source. Current WGS interval work uses this model. Other product
  sample expansion does not.

### Invalid

- Reject cycles, missing or conflicting artifacts, foreign Handles, incomplete
  Trees, invalid typed data, unsupported product routes, mutable image pins,
  protected option collisions, and path escape as structured defects. Rejected
  composition does not produce a runnable plan.

## Failure and recovery

The caller receives `*gobble.Error` with one or more stable `Defect` records
naming the operation, code, unit, message, and affected paths. Correct the
explicit data or config, rebuild the same selected graph, and inspect the new
plan. Do not silently choose another route or drop a required member.

## Structure

Package `gobble` owns the public model. `assets/modules/<command>` owns one
command task. `assets/pipelines/<assay>` owns one product graph. The engine and
generic command do not import product packages.

Pipeline source and typed defaults are graph authority. Strict assay loaders
convert host CSV rows into typed values. Sheet cells identify
workspace-relative inputs; sheets do not contain tool flags, images, resources,
references, or engine controls. Compose writes no run state and performs no
product download.

## Interfaces

- Authors call `Compose`, `Validate`, and `BuildPlan`. `WriteTo` is the supported
  Plan option constructor. Graph readers and Plan JSON expose the result.
- The generic command is `gobble compose PACKAGE [--sample PATH]`. It compiles
  the selected package and calls `Pipeline()`. Product adapters use the injected
  path or process-cwd `samplesheet.csv`; reusable and concurrent callers call the
  assay package's explicit `Load(path)` and `Build` instead.
- A Bind is exactly one of File, Group, or Tree. File is one regular artifact.
  Group is an ordered named set of regular files. Tree is a declared directory
  with a root `.gobble-tree.json`. `Directory` remains path placement, not an
  artifact.
- Scatter members occupy runtime instance identity; Gather waits for complete
  known membership. `When` uses a named File or boolean parameter and is
  re-evaluated on Resume.

## Constraints and qualities

One maintainer; local first. Keep the Go API explicit, defaults fresh, mutable
inputs copied, command identity inspectable, and failures structured. Product
convenience must not hide graph or recovery effects.

## Deferred outcomes

Deferred composition outcomes are indexed under
[compose-pipeline backlog](../../backlogs/compose-pipeline.md). Their presence
does not alter the current typed product contract.
