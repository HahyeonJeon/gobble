# Pipeline collaboration — Review checkpoint

2026-09-09. **Proposed design; implementation awaits the owner's stage review.**
The accepted product direction is Agent-authored pipeline design/change/execution/
monitoring through shared views. This checkpoint makes that direction concrete.

## Read for the decision

1. [Design and ownership diagrams](design.md): definitions, source-change lifecycle,
   Plan admission, process/API boundaries, shared-view sketch and code organization.
2. [Architecture review](architecture-review.md): capability gaps, focused
   improvements, strengths and explicit uninspected areas. Author self-review;
   the long checklist is folded.
3. [Five-stage plan](plan/plan-index.md): indexed task hierarchy, execution groups,
   writer boundaries, handoffs and observable completion conditions.
4. [Study](study.md) and [verification](verification.md): local/upstream evidence,
   alternatives, actual checks and their limits.

## Decision proposed now

Approve the direction and **stage 1: pipeline concepts and ownership foundation**.
This adds a distinct pipeline registration/read path and a focused presentation
lifetime boundary while retaining the existing single workspace writer. Changes,
Plans and Operations are defined now; executable modules appear only when a later
stage has a real consumer.

Then review each completed stage before proceeding:

| Stage            | Observable result                                                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 1. Foundation    | Pipeline/Project/Run/Pane ownership is explicit; registered pipeline sources can be read; existing reference/layout behavior is preserved. |
| 2. Agent changes | Exact Agent-authored source candidate, shared read-only diff, refinement and controlled source acceptance.                                 |
| 3. Plan          | Isolated Gobble validation, structured defects and exact shared Plan review.                                                               |
| 4. Execution     | Authorized Start, durable operation recovery and engine-conditional Stop.                                                                  |
| 5. Feedback      | Failed task/log → Agent change → Plan → engine-owned reuse preview/Resume → monitored result.                                              |

## Boundaries retained

- Agent authors source. App makes intentions, changes and execution consequences
  discussable through the same Panes and right Chat.
- Gobble remains the execution/validation/recovery authority. The service owns
  workflow records; Main owns presentation and references.
- No Notebook editor, CSV chart generation, new viewer family, general coding shell,
  visual Pipeline DSL or extension framework in this plan.
- Source writes and execution controls have later concrete design/verification
  gates. The initial source-apply path has an explicit managed-writer limit;
  arbitrary simultaneous external-editor safety is not claimed.
- Local packaging remains unfinished, but is not the proposed next priority.

## Evidence limits

The native App service tests and 32 focused App tests passed. Linux/amd64 root
cross-compilation passed. Native macOS root Go tests did not execute because of a
Linux-specific dependency; the documented engine target is Linux containers.
No new authoring/effect flow, live Agent turn or complete product suite ran here.

Original production/governing source bytes were preserved; this task produced
review, study, design, plan and navigation documents only. Exact identities and
document validation are in [subject.json](subject.json),
[dependency-evidence.json](dependency-evidence.json),
[review-method.json](review-method.json) and
[handoff-checks.json](handoff-checks.json).
