# P2B-1 design and code review

Reviewed the implemented dependency path and the actual B interaction, not the
historical sketch. No production dependency was added.

| Boundary                         | Review and resulting design                                                                                                                                                                                                                                                                                   |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Gobble construction → comparison | Actual Trim/FastQC modules and recipe verification share pure command construction. Complete exported TaskPlan fingerprints retain command/script/env/resource/control facts absent from ordinary Plan JSON. Unknown changed behavior produces gaps.                                                          |
| Engine → native service          | `internal/pipelinereview` is a portable leaf; dependency tests prohibit platform/provider/engine imports. The service receives bounded checked definitions and retains both versions. It does not infer source semantics from labels or parse Go into a substitute execution model.                           |
| Source → current                 | Source scope, proposal checking, adoption and HTTP routing have separate owners/files. Retained files are hashed before publication. Expected current and a durable receipt publish together through the existing catalog writer. Concurrent proposals cannot both replace the same base.                     |
| Main → provider                  | A feature-specific service client validates response association. `PipelineReviewHost` controls per-message authoring and comparison references. Current source permission is neither inherited from shared-view access nor carried into another submission. No Agent adoption tool exists.                   |
| Main → renderer                  | Named IPC exposes list/select/adopt/outcome, with trusted renderer validation. Source is not exposed through preload. The existing Workspace writer owns Chat context and independent Agent references. User composer selection uses a dedicated serialized command; it is not an Agent viewport observation. |
| React                            | `PipelineReviewPanel` owns review interaction and asynchronous status; `PipelineChangeDetails` renders checked facts only. Existing flow layout/routing/zoom/list components are reused. Changes are presentation inputs, not a second graph model.                                                           |
| Historical contracts             | Workspace v16 is frozen; v17 adds explicit source permission and exact comparison references. Earlier schemas cannot gain new authority through reused current Submission fields. v1/v2 catalog archives and old evidence remain retained.                                                                    |

Concrete corrections made during review:

- Corrected a Trim argument offset and confirmed values against actual module output.
- Rejected extra FastQC settings and hidden TaskPlan differences; accepted pipeline
  input edges with their explicit empty source-task identity.
- Fixed a circular schema import that otherwise produced an empty validator during
  startup, and removed new authority fields from a historical schema dependency.
- Rechecked imported source immediately before adoption; bounded history to eight
  records and each completed comparison to 1.75 MiB; pinned the User's chosen
  history item while later checks finish.
- Refused adoption outcome lookup from a failed catalog writer: after a successful
  rename but uncertain directory sync, only the restarted durable reader can
  confirm the outcome. A fault-state/restart test covers this ambiguity.
- Kept changed cards amber after selection, strengthened numbered badges and marked
  the added connection green. Unchanged nodes are disabled in review; an added
  edge opens the same change by keyboard or pointer. Normal flow navigation remains.
- Preserved visible feedback when selecting current-flow discussion without a local
  flow target. Extracted lower comparison facts from the review state owner.
- Added the existing license notice to packed `review`/`flow` help. The broader CLI
  run exposed this omission separately from two already recorded packed-runner failures.

Qualification is intentionally bounded: supported module recipes and declared check
inputs, one registered Pipeline per Project, eight retained proposals, and one actual
signed-in model interaction. There is no new editing canvas, Run admission, source
synchronization, new-Pipeline creation or external MCP endpoint. These are separate
product checkpoints, not extension points fabricated for this slice.
