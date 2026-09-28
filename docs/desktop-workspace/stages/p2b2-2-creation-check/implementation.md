# P2B-2.2 implementation and ownership

2026-09-12. Scope: native creation authoring/checking foundation. The User-approved
B presentation, Project shell, Chat and Agent policy have no new UI behavior yet.

## Concepts and publication

- **Creation scope** is an exact draft snapshot plus its selected profile runtime.
  It returns a digest, draft generation, input descriptor, guidance and one editable
  source file. Changing brief, input or runtime makes a previously issued scope stale.
- **Scaffold** belongs to Gobble. `creation-scaffold` exports the installed module
  setup, dependency checksums, initial source and input-descriptor template. It
  performs no Git initialization, Project writes or installation.
- **Input descriptor** belongs to the service. The fixed `creation-input.json`
  contains version, logical path and explicit single-end layout. Logical paths
  preserve `.fastq`, `.fq`, `.fastq.gz` or `.fq.gz`. Gobble source and the trusted
  check driver read that descriptor; the Agent cannot replace it through source
  submission. Extension metadata does not validate compression or FASTQ content.
- **Candidate** is a retained source submission identified within a draft. It has
  checking/ready/unsupported/failed/cancelled status, never a Pipeline identity.
  Its source and final evidence are retained independently of the active draft.
- **Creation artifact** binds complete checked flow/review facts to retained source,
  the draft snapshot, input observation and exact runtime/scaffold identity. An
  unsupported artifact retains explainable gaps but cannot be treated as ready.

```mermaid
sequenceDiagram
  participant Host as Native host (Part 3 policy pending)
  participant S as App service
  participant C as Profile storage
  participant G as Pinned Gobble runtime
  Host->>S: Select local engine binding
  S->>G: Verify daemon/image; export fixed scaffold
  S->>C: Save binding + scaffold digest
  Host->>S: Read source for exact draft generation
  S->>S: Reobserve selected input metadata
  S-->>Host: Scope ID + editable source + input descriptor
  Host->>S: Submit scope + source + request ID
  S->>C: Retain four files, manifest and checking record
  S->>G: Evaluate retained source only, without research data
  G-->>S: Complete flow + semantic qualification gaps
  S->>S: Verify source/runtime; fence generation and input
  S->>C: Publish final candidate evidence
```

No candidate registers a Pipeline, changes Current or starts a Run. Removing a
View is not a storage operation. The future UI must label input using the bound
Project observation, not present the logical `inputs/reads.*` path as the original
research file. Part 3 will freeze an exact no-current reference for each addition;
Part 4 will revalidate before atomic first adoption.

## File responsibilities and dependency direction

| Code | Owns |
| --- | --- |
| `internal/pipelinereview/creation.go` | Portable scaffold/input vocabulary and fixed authoring source; no App dependency or I/O |
| `cmd/gobble/creation.go`, driver/parse/help | Read-only installed scaffold export and dedicated creation-review dispatch |
| `pipeline_creation_input.go` | Bounded read of fixed input descriptor; never research data |
| `internal/engine/pipeline_creation.go` | Complete TaskPlan reconstruction/equality and exact supported topology |
| `pipeline_creation.go` | Public checked flow/settings/input correspondence over the engine qualifier |
| `internal/appservice/creation_runtime.go` | Profile-owned pinned runtime/scaffold persistence and isolated export |
| `internal/appservice/creation_source.go` | Exact source scope, four-file managed retention and integrity verification |
| `internal/appservice/creation_candidate.go` | Submission, request replay/conflict, history, cancellation and shared check budget |
| `internal/appservice/creation_check.go` | Evaluation, wire/graph correspondence, final generation fence and publication |
| `internal/appservice/creation_check_routes.go` | Authenticated native routing/decoding; no domain decisions |
| Existing draft routes | Serialize updates/discard with check publication and cancel obsolete work |

The engine qualifier does not reuse refinement's unchanged-residual assumption.
Every Trim task field must equal the reconstructed task, except explicitly
represented display/identity and supported quality/length. FastQC is fully
canonical. Inputs/outputs, default paths, resources, images and both complete edges
are checked; control, hidden environment, params, extra steps and other execution
fields fail qualification. Public flow settings must agree with checked recipes.
The service validates bounds and graph correspondence, not command semantics.

## Native protocol surface

All routes use the existing authenticated loopback host and closed JSON envelope.
They are not renderer IPC or Agent tools. No endpoint accepts arbitrary shell
commands, Agent runtime selection, setup files, adoption or execution requests.

| Method/path | Request/result |
| --- | --- |
| GET/POST `/v1/creation-runtime` | Read selected profile binding; native host selects an exact `RuntimeBinding` after daemon/image/scaffold verification |
| GET `/v1/projects/{project}/pipeline-drafts/{draft}/source?generation=N` | Exact generation → scope ID, selected input, guidance, one editable file |
| POST `.../{draft}/candidates` | requestId, expectedGeneration, scopeId, summary, files → candidate |
| GET `.../{draft}/candidates` | Bounded retained candidate history |
| GET `.../{draft}/candidates/{candidate}` | Saved status/evidence, including after discard or restart |
| POST `.../{draft}/candidates/{candidate}/cancel` | Empty object; request cancellation of that flight only |

Additive native capabilities preserve protocol 1 and existing consumers. Workspace
17, App bundle21, catalog4 and shared toolset11 are unchanged. The new native JSON
records will gain Main/TypeScript/UI consumers in Part 3; they are not advertised as
an already accessible in-App authoring workflow. No migration of existing Pipelines
or refinement artifacts occurs in this part.

## Limits, recovery and exact input identity

One editable source file up to 32 KiB; native request up to 64 KiB; eight retained
candidates per draft; at most two concurrent analysis checks across creation,
inspection and refinement. Existing bounded isolated evaluator is reused with a
110-second container deadline and 120-second service lifetime, no network, no Docker
socket and only read-only retained source. Export mounts no Project at all.

Service lock precedes catalog lock. Final generation/metadata validation and result
publication hold the catalog lock together, so draft writers cannot cross that
publication fence. Update/discard cancel an obsolete flight; direct cancellation
keeps its slot until cleanup completes. Repeated candidate identity with the same
intent returns saved state; different intent conflicts. Restart never resumes or
resends checks, and an interrupted checking record reads as cancelled.

Failed/cancelled/stale checks publish no usable artifact. Historical ready artifacts
remain historical after later draft edits/discard; they cannot imply fresh adoption
permission. Unexpected retained bytes, missing files and extra files fail integrity
checks. Incomplete saves preserve the occupied candidate identity rather than
silently overwrite it. An uncertain final save or runtime-profile write disables
further catalog operations until restart/recovery.

Input observations remain metadata, not content hashes. P3 must establish actual
content/format identity and execution mounting/admission. Tool commands are not run
in creation checks. General assays, paired-end reads, arbitrary resources/options,
source editing in React, Run controls, Notebook editing and CSV charts are excluded.
