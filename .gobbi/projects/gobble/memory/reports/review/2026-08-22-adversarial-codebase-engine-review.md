# Gobble adversarial codebase and Engine review

This report preserves the completed adversarial review of Gobble as a decision input for later discussion. It consolidates the 2026-08-21 independent evaluation, the earlier Engine design review, the current project design memory, the follow-up architectural assessment, and a second adversarial design-and-code pass completed on 2026-08-22.

It records problems, impacts, evidence, strengths, limits, and improvement directions. It intentionally excludes proposed APIs, types, algorithms, file-level changes, implementation sequences, and other prescriptive repair details.

## Review record

| Item | Value |
|---|---|
| Review completed | 2026-08-22 |
| Frozen target | Git-tracked repository on `develop` at `bff51b8c9bec6e14e7abf60fdb716acf50e15a50` |
| Target source state | Clean at freeze; evaluations preserved source and design files; only this report and its report index changed afterward |
| Review passes | Initial codebase and Engine evaluation, architectural follow-up, and independent second-pass challenge |
| Primary focus | Codebase design and structure, Engine maturity, operational safety, and bioinformatics scenario coverage |
| Acceptance criteria | None supplied |
| Gate verdict | Not issued |
| Quality opinion | Mixed |

No acceptance verdict is inferred from the quality opinion because no decision threshold or aggregation rule was supplied.

## Executive assessment

Gobble has a credible architecture for a static, local, Go-authored workflow engine. Its package boundaries, graph model, validation and planning path, scheduler-to-executor seam, structured state, and machine-readable API and CLI are sound foundations. The codebase does not require a wholesale architectural rewrite.

The Engine is not yet production-safe. Its weakest boundaries are runtime ownership and recovery, workspace state transitions, filesystem containment, artifact isolation and publication, backend cancellation, resource-contract consistency, secret handling, and concurrent access. These are central Engine invariants rather than peripheral feature gaps.

The second pass also found a recurring structural pattern: contracts that cross packages are represented more than once and can drift. Memory syntax differs between validation and Docker launch, process environment construction differs from executable resolution, and individual control-file atomicity does not produce one lifecycle snapshot. The package layout remains viable, but cross-package invariants need a clearer ownership and verification bar.

The current proof pack does not establish broad bioinformatics pipeline coverage. It demonstrates a narrow family of paired-end, local, static workflows. Several broader capabilities are explicitly deferred by the current first-horizon design, so their absence is not necessarily a defect against that limited scope. It does prevent a claim of complete Engine maturity or broad scenario coverage.

The best overall direction is to retain the current model–engine–executor structure while raising the safety and completeness of the boundaries that already exist. Feature and backend breadth should not be treated as evidence of maturity until those boundaries are trustworthy.

## Scope and evidence

The review covered:

- The public Go API and product CLI.
- Pipeline composition, validation, planning, scheduling, execution, inspection, release, and resume.
- Local process and Docker executors.
- Workspace occupancy, persisted run state, reuse decisions, and artifact publication.
- File, Group, and Tree artifact behavior.
- Samplesheet handling and first-party WGS, RNA-seq, Methyl-seq, and LinkedQC proofs.
- Unit, integration, race, and selected live Docker evidence.
- README claims, accepted design memory, roadmap boundaries, backlogs, and the long-term vision draft.

Observed verification evidence:

- `go vet ./...` passed.
- The race-enabled Engine test exposed a real shared-mutation race in occupancy preparation.
- Plain `go test ./...` did not pass in the restricted evaluation environment because an untagged asset test required loopback networking and several exact mode assertions were sensitive to the process umask.
- Engine packages passed with a conventional `umask 022`, confirming that the mode failures were environment-sensitive.
- Targeted local process and Docker execution checks passed, including basic Docker execution and bad-image containment.
- Full live WGS, RNA-seq, and Methyl-seq data assays were not rerun as part of the frozen evaluation. Existing repository evidence for those assays was inspected.
- No scale, load, host-crash, PID-reuse, secret-redaction, or broad scenario suite was available.
- A second independent static evaluation reviewed project design, scheduler and executor lifetimes, path trust, control-state consistency, resource semantics, recovery, and relevant tests without reading this report.
- A repeat `go test ./internal/engine/... -count=1` again failed on the already-recorded umask-sensitive mode assertions. Focused Inspect, capacity, and Docker argument tests passed. The new boundary findings are primarily direct code evidence rather than new dynamic fault-injection results.

## Architectural and structural assessment

### Strengths

#### Clear responsibility boundaries

The public model, Engine services, scheduler, executors, state files, and artifact operations are recognizably separate. The executor interface provides a useful boundary between scheduling and backend behavior. This structure supports local process and Docker execution without embedding backend placement logic in the public pipeline model.

Evidence: `internal/engine/exec/exec.go`, `internal/engine/engine.go`, `internal/engine/run.go`, and [System design](../../design/architecture/system.md).

#### Simple static graph model

The current graph model expresses modules, branches, merges, fan-out, fan-in, explicit dependencies, and named artifacts without introducing a proprietary workflow language. This matches the accepted first-horizon scope and keeps the authored contract understandable.

Evidence: `pipeline.go`, `spec.go`, and [Compose pipeline design](../../design/feature/compose-pipeline.md).

#### Strong pre-execution validation

Gobble rejects many invalid states before task execution, including missing inputs, cycles, output conflicts, path escapes, reserved control paths, unsupported backends, invalid resources, and unproducible waits. The plan remains inspectable and deterministic for the static graph.

Evidence: `internal/engine/check.go`, `internal/engine/plan.go`, and [Validate and plan design](../../design/feature/validate-plan.md).

#### Appropriate artifact concepts

File, Group, and Tree are useful primitives for bioinformatics artifacts such as FASTQ pairs, BAM/index sets, references with sidecars, and directory-shaped indexes. The concept of publishing declared outputs only after successful execution is appropriate even though the current publication implementation has safety gaps.

Evidence: `spec.go`, `internal/engine/tree.go`, and [Design tips](../../learnings/design/tips.md).

#### Agent-readable control surface

Structured errors, plans, task attempts, remaining-work classification, lineage, and reuse explanations are materially better for agent operation than log-only control. API and CLI terminology is mostly consistent across compose, validate, plan, run, inspect, resume, and release.

Evidence: `gobble.go`, `internal/engine/inspect.go`, `internal/engine/reuse.go`, and [Inspect design](../../design/feature/inspect-run.md).

#### Focused implementation

The Go implementation has few external dependencies and generally avoids speculative abstraction. Most packages can be understood and tested within their own responsibility boundary.

### Structural concerns

The overall package structure is not the primary problem. The main structural weakness is that related lifecycle invariants are distributed across independently mutating paths.

- Run, Resume, Release, reconciliation, cancellation, occupancy, and persisted state together form one lifecycle, but they do not consistently share one authoritative state-transition boundary.
- Runtime state, workspace ownership, task state, and filesystem publication can disagree after partial failure or concurrent operations.
- Plan, task, and run documents can each be internally atomic while describing different logical generations.
- The graph is treated as immutable at the public boundary while execution preparation can mutate shared backing data.
- Artifact abstraction is stronger than the filesystem safety contract that currently implements it.
- Logical path normalization is not equivalent to physical containment when ancestor symlinks or persisted artifact member paths are involved.
- Validation, scheduling, and execution sometimes own separate interpretations of the same resource or environment contract.
- Ordinary task configuration and secret-bearing configuration share the same persisted representation.
- Long-lived library callers share process-global samplesheet and executor state whose scope is broader than one request or workspace.

These concerns are concentrated enough that the current top-level architecture remains worth preserving, but serious enough that the Engine cannot yet be considered operationally trustworthy.

## Problems

### 1. Persisted PID is not a safe runtime identity

**Severity:** High  
**Current-scope relation:** In the shipped recovery boundary

Process runtime identity is persisted as a numeric PID. Post-restart reconciliation checks PID liveness and may cancel the reported process. A reused PID can identify an unrelated process, and the persisted record does not establish that the live process is the task Gobble originally started.

This creates a risk of signaling unrelated work and makes recovery decisions unreliable when ownership cannot be proven.

Evidence: `internal/engine/exec/process.go:112-137`, `internal/engine/run.go:386-423`, `internal/engine/inspect.go:236-245`, and `internal/engine/release.go:47-61`.

### 2. Executor uncertainty is hidden during recovery

**Severity:** High  
**Current-scope relation:** In the shipped failure and recovery boundary

Errors from reconciliation, cancellation, and polling are discarded in important recovery paths. Process and Docker cancellation also suppress some backend failures.

The Engine can therefore treat a backend task as incomplete or canceled without knowing whether it is still running, detached, inaccessible, or already gone. A later resume may create duplicate execution or conflict with an existing task.

Evidence: `internal/engine/run.go:298-325,403-413`, `internal/engine/exec/process.go:112-120`, and `internal/engine/exec/docker.go:98-104`.

### 3. The public recovery lifecycle is not coherent for a live Go caller

**Severity:** High  
**Current-scope relation:** In the public Go recovery loop

Run records the caller process PID as workspace owner and leaves occupancy active. Release refuses a live same-host owner. A long-running process that embeds Gobble cannot complete the documented Run–Inspect–Release–Resume loop without exiting or altering the owner record. Existing recovery tests use a helper that makes the owner appear dead before Release.

This means the public Go API and the persisted ownership model do not form a self-contained recovery lifecycle for service-style callers.

Evidence: `internal/engine/run.go:181-187`, `internal/engine/release.go:43-60`, `tests/local-e2e/helpers_test.go:312-375`, and [Recover design](../../design/feature/recover-run.md).

### 4. Release is not protected by the same workspace and runtime boundary

**Severity:** High  
**Current-scope relation:** In the shipped workspace lifecycle

Release changes run and task state without acquiring the occupancy boundary used by Run and Resume. A concurrent lifecycle operation can therefore be overwritten by a stale Release. Release also accepts an empty workspace string at the Go API boundary and can resolve control state relative to the caller's current directory.

Release marks running task records incomplete and closes occupancy without reconciling or canceling the recorded backend work. A process or container can therefore remain live while the workspace becomes available to a later owner.

These behaviors allow state-changing operations to address the wrong workspace, produce a state snapshot that does not correspond to the actual owner, or admit new work while old work remains active.

Evidence: `release.go:9-10`, `internal/engine/release.go:10-100`, `internal/engine/occupancy.go:76-91`, `internal/engine/check.go:105-134`, and `internal/engine/run.go:386-423`.

### 5. Occupancy preparation has a real shared-mutation race

**Severity:** High  
**Current-scope relation:** In concurrent Run ownership

Execution preparation shallow-copies the document and mutates task defaults before ownership is serialized. Concurrent calls can mutate the same task backing data. The race detector reproduced this in the occupancy ownership test.

The race contradicts the public expectation that a composed graph is immutable and introduces undefined behavior in an ownership-critical path.

Evidence: `internal/engine/run.go:171-196`, `internal/engine/identity.go:42-49`, and `internal/engine/occupancy_test.go:43-61`.

### 6. Artifact staging can mutate source data

**Severity:** High  
**Maturity relation:** Safety gap beyond the accepted link-based staging behavior

Staged task inputs may share an inode with the source through hardlinks or may refer to the source through process-only symlinks. Ordinary same-user tasks can write through those aliases. Docker task inputs are also mounted read-write.

A task can unintentionally corrupt caller-owned inputs or upstream outputs. Later tasks, reuse decisions, and provenance can then describe data that no longer matches the original artifact.

Evidence: `internal/engine/exec/publish.go:28-49,174-179`, `internal/engine/exec.go:44-56`, and `internal/engine/exec/docker.go:136-142`.

### 7. Artifact publication is not transactional at artifact scope

**Severity:** High  
**Current-scope relation:** In the shipped publication and recovery boundary

Initial cross-device publication streams directly into the final destination. Process interruption can leave a partial final file that later validation treats as an existing output. Tree replacement changes members individually and does not restore already replaced existing members when a later member or manifest operation fails.

The visible workspace can therefore contain partial files or a mixture of old and new Tree members without a coherent published artifact state.

Evidence: `internal/engine/exec/publish.go:52-90,93-137`, `internal/engine/tree.go:158-201`, and `internal/engine/check.go:377-420`.

### 8. Task secrets can be persisted and exposed

**Severity:** High  
**Design relation:** Conflicts with the recorded no-persisted-secrets promise

Task environment values are unrestricted literals. They can be included in plan data, persisted task state, Inspect output, and Docker command arguments. The model does not distinguish public configuration from credentials or other sensitive values.

This is incompatible with workflows that need private registries, object storage credentials, cloud access, or clinical-data services without exposing secrets in the workspace or process metadata.

Evidence: `spec.go:5-20`, `internal/engine/plan.go:112-130`, `internal/engine/run.go:124-148,239-262`, `internal/engine/inspect.go:247-301`, `internal/engine/exec/docker.go:150-162`, and [Product data promises](../../design/process/product.md).

### 9. Reuse and provenance identity are incomplete

**Severity:** Medium to High, depending on use  
**Maturity relation:** Known open design area

Reuse depends on the authored image string and inexpensive filesystem keys. Recorded image digest and content SHA are not consistently part of the reuse decision. Mutable image tags or in-place content changes that preserve relevant metadata can therefore produce reuse decisions with incomplete biological provenance. The Docker attempt digest is queried by authored image name after container creation rather than from the created container, so concurrent tag movement can also weaken the recorded execution identity.

Parameter identity is sequence-shaped in the plan but is compared through a name-keyed map. Duplicate parameter names are not rejected, and earlier duplicate values can be overwritten during comparison. Distinct accepted parameter lists can therefore compare equal and permit false reuse.

This weakens reproducibility and makes reuse harder to trust for regulated, clinical, or long-lived analytical results.

Evidence: `internal/engine/reuse.go:70-107,131-170,595-610`, `internal/engine/plan.go:55-57`, `internal/engine/exec/docker.go:40-65,186-199`, [System data design](../../design/architecture/system.md), and the [later run-local disposition](../note/2026-09-02-gobble-multiomics-products.md#deferred-outcomes).

### 10. Verification is neither fully portable nor complete for critical boundaries

**Severity:** Medium  
**Current-scope relation:** In the documented verification contract

The untagged test set contains a loopback HTTP server despite the stated hermetic and network-free first check. Several tests also expect exact file modes that vary under normal process umasks.

The first check can fail because of host policy rather than a product regression, reducing its reliability as a contributor, CI, and agent gate. At the same time, the focused suite does not directly exercise fractional memory translation, write-through hardlink mutation, ancestor and control-path symlinks, File/Tree ancestor conflicts, PID reuse, Release-versus-claim concurrency, live work after Release, a noncooperating executor, or interrupted multi-file state updates.

The current verification surface can therefore fail for incidental environment reasons while still passing without proving several material safety and recovery invariants.

Evidence: `README.md`, `assets/pin_test.go:299-370`, exact-mode assertions in `internal/engine/exec` and `internal/engine` tests, `internal/engine/exec/docker_test.go:47-61`, `internal/engine/occupancy_test.go:44-99`, `internal/engine/cancel_test.go:1-60`, and `internal/engine/check_test.go:360-377`.

### 11. Operational state is partial and can be internally inconsistent

**Severity:** High  
**Current-scope relation:** In the shipped persisted-state and inspection boundary

Inspect exposes run state, task instances, errors, log pointers, timing, DAG, lineage, remaining work, and reuse decisions. It does not expose a durable event view, resource measurements, or enough backend transition detail to distinguish all running, detached, canceling, unknown, and partially published states.

Run and Resume write `plan.json`, `tasks.json`, and `run.json` sequentially. Each replacement is individually atomic, but the set has no shared lifecycle identity or completion boundary. A process interruption or concurrent read can leave or observe documents from different logical updates.

Inspect reads those files independently and explicitly permits cross-file combinations that are mid-update. Missing plan or task files can also become empty portions of an otherwise successful Inspect response. The exposed run `id` is the pipeline name, or the literal `run`, rather than a unique execution identity.

This limits safe automated diagnosis and correlation even though the existing structured views are a strong starting point.

Evidence: `internal/engine/inspect.go:25-81,118-180,199-245,247-451`, `internal/engine/run.go:209-221,232-237,953-1003`, `internal/engine/resume.go:279-291`, and [Inspect design](../../design/feature/inspect-run.md).

### 12. Large-cohort and long-lived-process behavior has known scaling pressure

**Severity:** Medium  
**Maturity relation:** Scale gap outside current proof size

Successful task accounting synchronously computes full hashes for consumed inputs. Shared references and indexes can be read repeatedly across many sample tasks. Readiness repeatedly scans tasks and edges, helper lookups are linear, and every material transition rewrites the complete task history. This creates at least quadratic state-serialization growth across a run and potentially higher scheduler work for dense or deeply ordered graphs.

The process-global executor also retains every submitted process record and every completed Docker report without a release path. The current small fixtures do not establish acceptable behavior for cohort pipelines, large references, many shards, repeated runs, or long-running service use.

Evidence: `internal/engine/run.go:426-545,556-562,803-858,961-976`, `internal/engine/identity.go:121-132`, `internal/engine/exec/process.go:14-28,58-110`, `internal/engine/exec/docker.go:21-30,70-129`, `assets/wgs.go`, and `assets/rnaseq.go`.

### 13. Lexical path validation does not establish filesystem containment

**Severity:** High  
**Current-scope relation:** In the shipped workspace and publication boundary

Plan paths reject absolute syntax, `..` escape, and reserved lexical prefixes. The resulting workspace path is nevertheless a plain join, and later directory creation, open, link, rename, and control-file operations follow intermediate filesystem symlinks. A workspace subdirectory or `.gobble` ancestor can therefore redirect reads or writes outside the authoritative workspace even when the authored path is lexically relative.

Tree replacement also uses a predictable sibling temporary path. A pre-existing symlink at that replacement path can redirect the write before rename. Existing checks cover a final-component symlink, not ancestor, control-directory, or replacement-path redirection.

This allows external reads, publication, or control-state writes through entries that still pass the current plan-path contract.

Evidence: `internal/engine/check.go:236-278,323-419,461-472`, `internal/engine/occupancy.go:76-91`, `internal/engine/exec/publish.go:20-76,93-137`, `internal/engine/tree.go:230-245`, and `internal/engine/check_test.go:360-377`.

### 14. Artifact path sets and persisted Tree membership are under-validated

**Severity:** High  
**Current-scope relation:** In validation, publication, and Resume replacement

Conflict validation compares canonical path strings for equality but does not model ancestor relationships. File outputs at `foo` and `foo/bar` can both pass even though one publication requires `foo` to be a file and the other requires it to be a directory. A Tree at `foo` and a separately owned File at `foo/bar` also pass with overlapping artifact ownership.

Tree replacement reads member paths from the existing `.gobble-tree.json` without validating their relative-path or containment properties. Paths not reproduced by the new Tree are joined to the destination root and removed. A damaged or adversarial manifest can therefore extend deletion authority beyond the declared Tree and potentially beyond the workspace.

The artifact model is stronger than the validation applied to its complete path set and persisted membership authority.

Evidence: `internal/engine/validate.go:150-218`, `internal/engine/check.go:197-233`, `internal/engine/tree.go:158-200,272-310`, and `internal/engine/exec/publish.go:52-76`.

### 15. Cancellation does not bound backend operation completion

**Severity:** High  
**Current-scope relation:** In the public Run and Resume context contract

The Executor seam does not carry a context or deadline. Docker image inspection, pull, launch, poll, kill, log, and remove operations use synchronous commands without a cancellation bound. A context can be canceled while Submit or Poll is blocked, but the scheduler cannot complete until the backend call returns.

Resume reconciliation has an additional unbounded loop: after Cancel it polls until the backend reports non-running, ignores backend errors, and does not observe the caller context. A stalled Docker daemon, image pull, or noncooperating executor can therefore prevent canceled Run or Resume from returning and can leave ownership unresolved indefinitely.

Evidence: `internal/engine/exec/exec.go:45-51`, `internal/engine/exec/docker.go:33-60,70-129,165-221,282-295`, `internal/engine/run.go:289-355,386-423,617-665`, and `run.go:20-24`.

### 16. Executable selection still depends on the host environment

**Severity:** High  
**Current-scope relation:** Conflicts with the recorded host-environment exclusion

The process adapter constructs a command from the bare task executable before assigning the declared child environment. Standard Go command construction resolves a bare executable through the parent process `PATH`, not the later `cmd.Env`. A task-provided `PATH` therefore does not control which executable is selected, while an undeclared host `PATH` does.

The Docker control path similarly resolves the `docker` client through the parent environment before giving that child a fixed `PATH`. This creates a reproducibility and trust mismatch: the child environment appears constrained, but executable identity can still be selected by host state.

Evidence: `internal/engine/exec/process.go:31-38,140-152`, `internal/engine/exec/docker.go:282-285`, `internal/engine/exec/process_test.go:50-72`, and [Run-local design](../../design/feature/run-local.md).

### 17. Accepted fractional memory can disappear before Docker launch

**Severity:** High  
**Current-scope relation:** In the shipped resource admission and Docker enforcement contract

Validation and scheduler admission accept a fractional suffixed memory value such as `1.5g`, convert it to bytes, and reserve that capacity. The Docker adapter owns a second, integer-only parser and adds `--memory` only when its parser succeeds.

A task can therefore consume the declared memory in scheduler accounting while its container starts without the declared memory limit. Existing Docker argument tests cover integer syntax and do not expose this cross-package drift.

Evidence: `internal/engine/validate.go:234-273`, `internal/engine/check.go:422-447`, `internal/engine/run.go:86-113`, `internal/engine/exec/docker.go:144-149,252-280`, `internal/engine/exec/docker_test.go:47-61`, and [System design](../../design/architecture/system.md).

### 18. Long-lived library embedding depends on process-global mutable state

**Severity:** Medium  
**Maturity relation:** Public-library concurrency and lifecycle gap

All runs in one Go process use a package-global executor unless internal tests replace it. Its process and Docker records live for the adapter's process lifetime. Samplesheet selection is also a process-global setting: individual reads and writes are mutex-protected, but setting a path and constructing a pipeline are not one caller-owned operation.

Concurrent library callers using different samplesheets can observe another caller's selection, and repeated runs share retained executor state beyond a workspace lifecycle. The short-lived generated CLI driver avoids much of this exposure by process isolation; the public library does not define equivalent request ownership or cleanup semantics.

Evidence: `internal/engine/run.go:38,165-193,282-287`, `internal/engine/exec/exec.go:53-64`, `internal/engine/exec/process.go:14-28,58-110`, `internal/engine/exec/docker.go:21-30,70-129`, `samplesheet.go:51-103`, and `cmd/gobble/driver.go:172`.

### 19. Recovery design memory presents contradictory public scope

**Severity:** Medium  
**Design relation:** Project-design consistency gap

The recovery feature's purpose, role, scope, and user-action sections describe retry, cancel, backoff, and guarded cleanup as part of the feature. The same accepted record's current contract, and the system interface record, state that public Cancel, named retry, and guarded Clean are not shipped.

The code consistently reflects the narrower current contract, but the accepted design source does not present one unambiguous horizon boundary. A maintainer or consumer can reach different conclusions depending on which section is treated as authoritative.

Evidence: [Recover design](../../design/feature/recover-run.md) and [System design](../../design/architecture/system.md).

## Engine completeness

The following table separates implemented capability from complete-Engine maturity. Missing later-horizon capabilities are not automatically current-contract defects.

| Capability | Current assessment | Scope relation |
|---|---|---|
| Static DAG composition | Strong for modules, branch, merge, fan-out, and fan-in | First horizon |
| Deterministic validation and planning | Strong graph semantics; physical containment and ancestor-path validation incomplete | First horizon |
| Local process and Docker execution | Implemented and basically exercised; environment and cancellation contracts incomplete | First horizon |
| CPU, memory, and concurrency admission | Implemented; accepted memory syntax can diverge from Docker enforcement | First horizon |
| File, Group, and Tree artifacts | Expressive model; containment, manifest trust, and publication safety incomplete | First horizon |
| Structured inspection and reuse explanation | Useful; cross-file consistency and operational states incomplete | First horizon |
| Failure containment and resume | Implemented shape; critical lifecycle safety gaps remain | First horizon |
| Retry, backoff, timeout, named retry, and public cancel | Absent or deferred | Later maturity |
| Data-dependent expansion, scatter/gather, and conditionals | Absent and explicitly deferred | Later horizon |
| GPU, disk, walltime, priority, fairness, and queue policies | Absent | Later maturity |
| Slurm, cloud batch, Kubernetes, and object storage | Absent and explicitly deferred | Later horizons |
| Strong content and environment provenance | Partial | Open design area |
| Durable events and resource metrics | Partial or absent | Later maturity |
| Large-graph and large-cohort evidence | Absent | Unproven maturity |
| Long-lived concurrent Go embedding | Public API exists; process-global ownership and lifecycle are incomplete | Unproven maturity |
| Persisted-state upgrade continuity | Schema 0 and 1 are intentionally rejected; no migration or compatibility promise | Explicit current limit |
| Secret-safe remote execution | Absent | Required before credentialed backends |

The Engine is therefore best described as a promising local static-core prototype, not a complete general bioinformatics workflow engine.

## Bioinformatics scenario coverage

The first-party assets are explicitly proofs rather than product tools. They demonstrate graph and executor paths but should not be treated as evidence that the Engine covers the full assay space.

| Scenario family | Evidence | Assessment |
|---|---|---|
| Static local workflow with explicit files | Synthetic and run-local fixtures | Covered |
| Module fan-out and fan-in | Graph and WGS/RNA/Methyl construction | Covered at small scale |
| Two-sample WGS | Authored WGS proof | Narrow proof only |
| Paired-end RNA-seq | Samplesheet graph through feature counts and two-group DESeq2 | Narrow proof; exactly two groups |
| Paired-end methylation | Samplesheet graph through extraction and merged QC | Narrow proof; no DMR |
| Linked-read QC | LinkedQC constructor | Plan-only |
| Single-end reads | No representative scenario evidence | Not covered |
| Multi-lane, technical replicate, or optional-mate inputs | No representative scenario evidence | Not covered |
| Variable contrasts or multi-group differential analysis | Current RNA proof restricts group shape | Not covered |
| Tumor/normal, trio, pedigree, or cohort relationships | No explicit biological relationship model or scenario | Not covered |
| Cohort and joint calling | No scale or scenario evidence | Not covered |
| Long-read sequencing | No scenario evidence | Not covered |
| Single-cell or spatial omics | No scenario evidence | Not covered |
| ChIP-seq or ATAC-seq | No scenario evidence | Not covered |
| Metagenomics | No scenario evidence | Not covered |
| Amplicon workflows | No scenario evidence | Not covered |
| Proteomics | No scenario evidence | Not covered |
| Dynamic sample discovery or data-dependent branching | Not expressible in the current Engine model | Deferred |
| HPC or cloud bioinformatics runs | No backend or storage support | Deferred |

Absence from this table does not always prove that a static graph cannot be authored for the assay. It means the codebase contains no representative contract or execution evidence demonstrating that the model, metadata, recovery, artifacts, and scale are adequate for that scenario.

Evidence: `assets/wgs.go`, `assets/rnaseq.go`, `assets/methylseq.go`, `samplesheet.go`, `tests/local-e2e/PARITY.md`, [Asset design](../../design/feature/assets.md), and [Asset backlog](../../backlogs/assets.md).

## Improvement directions

These directions state desired outcomes only. They intentionally do not prescribe implementation mechanisms or select among design alternatives.

### Recovery and runtime safety

Recovery must distinguish owned, running, stopped, canceled, detached, and unknown work without risking unrelated processes. Public API and CLI recovery should share a coherent lifecycle for both short-lived commands and long-lived callers.

### Workspace state coherence

Every state-changing lifecycle operation should preserve one authoritative workspace history under concurrency, interruption, partial multi-file updates, and stale callers. Invalid workspace targets should be rejected consistently across all public operations.

### Filesystem containment and artifact authority

Workspace-relative paths, control paths, temporary publication paths, artifact path sets, and persisted Tree membership should remain within their declared physical boundaries under symlinks, damaged state, replacement, and rollback.

### Artifact immutability and publication integrity

Task inputs should remain immutable from the source owner's perspective. Publication should preserve complete artifact-level consistency across files, groups, trees, replacement, cross-device filesystems, partial failure, and process interruption.

### Backend completion and resource consistency

Run, Resume, and cancellation should have an explicit completion contract when backend operations stall or remain uncertain. Values accepted by validation and scheduler admission should retain the same meaning at backend launch, and executable identity should match the declared execution environment.

### Secret and trust boundaries

Sensitive execution inputs should be distinct from ordinary persisted configuration. Planning, state, inspection, logs, and backend invocation should respect the project's promise that credentials are not retained or exposed.

### Concurrency and verification confidence

Shared graph and workspace behavior should be race-free. The standard verification gate should be portable across supported Linux environments and should clearly separate hermetic, live, fault-injection, and scale evidence.

### Public-library lifecycle

Concurrent and long-lived Go callers should have explicit ownership semantics for process-scoped configuration, executor state, workspace activity, and cleanup. CLI process isolation should not be assumed to define library behavior.

### Provenance and reuse trust

Reuse decisions should represent the biological inputs, tools, execution environment, and pipeline definition strongly enough for the intended reproducibility level. The project needs an explicit balance between provenance confidence and the cost of examining large data.

### Engine-class contract

The project should define the intended completeness bar for dynamic graph behavior, failure policy, resources, backends, storage, and service operation before advertising broader Engine maturity. Current first-horizon deferrals should remain visible until accepted later-horizon contracts exist.

### Design and state evolution

Accepted design records should present one unambiguous shipped-versus-deferred public scope. Persisted workspace compatibility and run identity should have an explicit maturity boundary for workflows that outlive a binary version or process.

### Bioinformatics scenario evidence

Scenario coverage should distinguish Engine model capability from assay-wrapper completeness. Evidence should represent materially different read shapes, sample relationships, metadata cardinalities, artifact forms, failure paths, and dataset scales rather than counting additional fixed pipelines.

### Observability and operations

Structured control should provide enough information for an agent to diagnose backend uncertainty, resource behavior, partial failure, and recovery consequences without relying on human interpretation of raw logs.

### Scale confidence

The Engine should have explicit evidence for large shared references, many samples, large task graphs, long-running operations, and repeated recovery. Performance claims should remain bounded until that evidence exists.

## Topics for a later decision session

The review leaves these decisions open:

- What safety bar defines completion of the local agent-operable core?
- Is same-process recovery a required public Go API behavior, or is recovery intentionally process-separated?
- What completion guarantee applies when backend submit, poll, cancel, or reconciliation does not return?
- Are workspace symlinks and persisted control or Tree-manifest contents trusted inputs, and what containment guarantee applies?
- What artifact immutability and publication guarantees are part of the public contract?
- What reproducibility level must reuse and provenance support?
- Must validated resource syntax and declared task environment be enforced identically by every executor?
- Which task configuration may be persisted, inspected, or passed to backends?
- Is concurrent, long-lived Go embedding a supported first-class consumer, including samplesheet configuration and executor lifetime?
- What workspace-schema and run-identity continuity is required across releases?
- Which dynamic graph, failure policy, resource, storage, and backend capabilities define the next Engine horizon?
- Which bioinformatics scenario families are required as Engine contracts, and which remain asset-level demonstrations?
- What scale and fault evidence is required before production use is claimed?

No answers are selected in this report.

## Preserve

- The backend-independent Go pipeline model.
- The separation between model, validation and planning, scheduler, and executors.
- The small public verb set and structured API/CLI output.
- The existing lexical path, reserved-path, and output preflight checks as a useful baseline.
- File, Group, and Tree as the artifact vocabulary.
- Inspectable task attempts, lineage, remaining work, and reuse reasons.
- Local-first scope and explicit disclosure of deferred Engine-class capabilities.
- API/CLI parity and contained-failure scenario tests.
- The current bias toward simple code and limited abstraction.

## Limits

- No acceptance criteria were supplied, so this report does not issue a contract gate verdict.
- The review is frozen at the named commit and does not claim to assess later changes.
- Some test failures were environment-specific; they establish portability weaknesses rather than universal product failure.
- The full live assay pack was not independently rerun during the frozen evaluation.
- Several high-severity findings are supported by direct code inspection and race evidence rather than destructive fault injection.
- The second-pass evaluator was intentionally static and ran no project commands. The main review reran selected local tests, but no PID-reuse, crash-interruption, symlink-race, noncooperating-backend, or live orphan experiment was performed.
- Missing scenario evidence is reported as unproven coverage unless the current model clearly lacks the required capability.
- This report is not an implementation plan and intentionally excludes concrete repair choices.

## Conclusion

Gobble's design and code structure are fundamentally viable. The static graph, validation and planning model, executor seam, artifact vocabulary, and structured control surface are worth preserving.

The current Engine is not yet production-safe because runtime ownership, recovery, workspace mutation, physical path containment, artifact publication, backend completion, resource translation, executable identity, secret handling, and concurrency do not consistently preserve their intended invariants. These issues are more important than adding more pipeline wrappers or execution backends.

Bioinformatics coverage is narrow and proof-oriented. WGS, paired-end two-group RNA-seq, paired-end methylation extraction, and local recovery fixtures demonstrate useful paths, but they do not establish broad assay, metadata, storage, backend, failure, or scale coverage.

The durable quality opinion remains **mixed**: strong local-core foundations, significant safety gaps in shipped Engine boundaries, and substantial later-horizon work before complete Engine or broad bioinformatics maturity can be claimed.
