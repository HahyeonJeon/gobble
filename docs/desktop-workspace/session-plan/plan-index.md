# Project Workspace — Session Implementation Plan

> **Document role:** Authoritative Planning result locator.
> **Reading rule:** Task hierarchy first, then the plan part.
> **Status:** User-approved implementation plan; no implementation completion claimed here.
> **Output boundary:** Decomposition and execution contracts, not execution progress.

Project concept and multiple-agent shared-workspace direction are user-approved.
The user approved development and requested sequential stage gates. Before each
group, present a sketch; implement and test that group; summarize its result and
obtain user approval before the next group. Progress and evidence belong in
[stage records](../stages/03-workspace.md), outside this planning directory.
Detailed contract decisions are in the linked design records. All groups
run in one ordered implementation stream; role labels do not authorize spawning
subagents or posting to external services.

App-owned text, menus, errors and accessibility labels are English; user data
retains its original language. This applies to every implementation group.

## Task hierarchy

| View | Description |
|---|---|
| [Tasks](tasks/tasks-index.md) | All first-slice outcomes and responsibility boundaries |

## Plan parts

| Part | Description |
|---|---|
| [Plan 01](plan-01.md) | Seven dependency-ordered groups from contracts to a local Mac package |

## Governing design

- [Review and first-slice acceptance](../README.md)
- [Project interaction design](../../../.gobbi/projects/gobble/memory/design/feature/agent-workspace.md)
- [Architecture and communication](../../../.gobbi/projects/gobble/memory/design/architecture/project-workspace-contract.md)

The directory contains exactly this index, one plan part, the task index and one
task part. Changes to any member require rechecking coverage, dependencies and
cross-view consistency. Environment-dependent outcomes remain required; an
unavailable runtime is recorded as incomplete evidence, not a passing result.
