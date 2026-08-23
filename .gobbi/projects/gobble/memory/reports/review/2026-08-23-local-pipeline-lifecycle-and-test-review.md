# Gobble local pipeline lifecycle, code, and test review

This report consolidates the production-code and test-code reviews completed on 2026-08-23. It evaluates how well Gobble covers the lifecycle of local bioinformatics pipelines and how reliably the implementation and tests support that coverage. It is a point-in-time review of a frozen revision and does not replace the [2026-08-22 adversarial codebase and Engine review](2026-08-22-adversarial-codebase-engine-review.md).

> **Decision boundary:** This report is not an implementation plan, change approval, compatibility decision, or prioritization. Every concrete mechanism, name, test shape, file arrangement, and threshold mentioned below is a non-binding example intended to support discussion. Any actual change or decision must be discussed, revised as needed, and explicitly accepted before implementation.

## Review record

| Field | Value |
|---|---|
| Review completed | 2026-08-23 |
| Frozen production subject | `develop` commit `c9ffeb84c611814fbf00a4ec15036b5939fa46b7`, tree `abb9d73f6b1b297e8dc8ba610033905391c96706` |
| Frozen test subject | 85 tracked test files, 23,334 test lines, 449 `Test` functions, 99-path manifest SHA-256 `fea070108029e9366c392bfb51d56e53c95b6e740780e846bf45f3756464e797` |
| Scope | Public Go API and CLI; graph and plan; process and Docker execution; workspace, artifacts, Inspect, Release, and Resume; samplesheets and proof assets; test structure, lifecycle scenarios, helpers, oracles, and evidence boundaries |
| Acceptance threshold | None supplied by the user |
| Gate verdict | Not issued |
| Quality opinion | Mixed |
| Record state | Partial |
| Source review artifacts | Production review SHA-256 `f8a96e936b85c1575d6d17a830e160732ada83a7fee3fae5ffe5ccae2dd44456`; test review SHA-256 `516fa11226712e0ab70d62573fd7e180ebe570eddf88959dc386a84e4b6c71d0` |

The record is partial because the static and hermetic evidence is strong, but the review did not refresh live Docker execution, run destructive crash or fault injection, establish external CI enforcement, measure representative scale, or validate an accepted taxonomy of supported bioinformatics scenarios.

## Executive assessment

Gobble has a credible foundation for a local, static bioinformatics pipeline engine. The product connects Go-based composition, validation, dry-run planning, process and Docker execution, structured inspection, occupancy release, graph-change classification, and reuse-aware resume. File, Group, Tree, Scatter, Gather, and When provide useful workflow primitives, and the model–engine–executor separation is worth preserving.

That functional breadth is not yet equivalent to a complete or production-safe lifecycle. The most serious production risks concern scientific provenance, workspace state coherence, interruption and restart recovery, artifact publication durability, backend identity, Docker cleanup, and bounded cancellation. Several ordinary states can produce false reuse, wedge Release or Resume, leak terminal backend state, or signal an unrelated host process.

The test suite is also stronger than a blanket negative assessment would suggest. It is broad, fast, largely hermetic, and contains useful synthetic coverage for operators, attempts, change classification, and API/CLI behavior. Its weakness is architectural and evidentiary: broad unit and synthetic coverage does not consistently establish real recovery, complete reuse, or scientifically meaningful outputs for the named bioinformatics scenarios.

The combined conclusion is therefore:

- Gobble substantially covers the happy path of a local static pipeline lifecycle.
- Gobble only partially covers interruption, restart, crash, concurrency, backend uncertainty, and durable recovery.
- The tests cover many mechanisms, but they do not yet provide trustworthy end-to-end assurance for the full recovery contract or for diverse bioinformatics pipelines.
- The current WGS, bulk RNA-seq, methylation, and synthetic scenarios are useful proofs, not evidence of broad assay, cohort, storage, or scale support.
- The current Design Memory correctly does not claim first-horizon exit.

## Lifecycle and assurance coverage

`Build` is not one unambiguous public lifecycle operation. The CLI compiles a generated Go driver, `Compose` builds the graph, and `BuildPlan` creates a dry-run plan; Gobble does not currently build Docker images or reusable software components. Any product discussion of “build” needs to distinguish those meanings.

| Lifecycle area | Current surface and evidence | Assessment |
|---|---|---|
| Author and compose | `Pipeline`, `Compose`, Module, Branch, Merge, Scatter, Gather, When | Implemented; some structural names imply stronger semantics than the types enforce |
| Validate and plan | Compose-time checks, `Validate`, `BuildPlan`, plan JSON, DAG, CLI goldens | Strong for static and hermetic behavior |
| Run | Process and Docker executors, CPU/memory/cap admission, isolated attempts | Implemented; provenance, trust-boundary, cancellation, and crash-durability risks remain |
| Observe | Structured run, instance, error, log, timing, DAG, lineage, remaining, and reuse views | Useful; accepted ordered event evidence is absent |
| Interrupt | `Run` and `Resume` context cancellation | Partial; operation-level completion is not bounded |
| Release execution authority | `Release` API and CLI | Implemented; name, serialization, reconciliation, and terminal cleanup contracts are unclear or unsafe |
| Resume | Graph-change classification, content-aware reuse, downstream rerun | Functionally substantial; depends on provenance and control-state invariants that can fail |
| Clean and retention | No public operation | Intentionally deferred; retention, deletion, and migration policy remain open |
| Backend portability | Host process and Docker on Linux | Local scope only; HPC, cloud, and object storage remain later horizons |
| Synthetic operator lifecycle | Direct tests for Scatter, Gather, When, Tree, Group, attempts, and reuse reasons | Materially useful hermetic coverage |
| Successful Release and Resume | API and CLI helpers for run-local, WGS, RNA, and methylation | Covered, but mainly as unchanged no-op reuse after success |
| Failure or interrupt followed by successful recovery | Scattered synthetic failure tests; CLI signal test stops after Release | Incomplete as a full documented lifecycle |
| Biological output integrity | Stronger per-sample methyl checks; weaker WGS and RNA checks | Uneven and insufficient for broad scientific assurance |
| Bioinformatics diversity | WGS, paired-end bulk RNA, paired-end methylation, optional-mate synthetic proof | Supported scenario set is undefined; broad diversity is unverified |

## Problems in production code and lifecycle behavior

### P1. Recorded provenance can differ from the bytes an attempt actually used

- **Severity:** High.
- Inputs are staged before execution, but success records hash mutable workspace sources afterward. Outputs are also hashed through mutable destination paths after publication.
- A task can consume bytes A while the workspace changes to B, then record B as provenance. A later workspace containing B may falsely reuse an output produced from A.
- This is a scientific correctness problem, not only a metadata problem.
- Evidence: `internal/engine/run.go:922,1178-1258`; `internal/engine/identity.go:167-203`; `internal/engine/exec/publish.go:28-54`; `internal/engine/reuse.go:230-346`.

### P2. Workspace lifecycle state has no single coherent transaction boundary

- **Severity:** High.
- Plan, task, and run documents are replaced separately rather than committed as one generation. Resume reads some state before locking, and Release and Resume do not consistently apply Inspect's coherent-snapshot validation.
- Concurrent `Release` calls in the owning process can both observe `holdsLease`, skip flock serialization, and interleave distinct snapshot rewrites.
- Readers can observe a combination of control documents that never represented one real run generation, leading to overwritten attempts, duplicate execution, or invalid reuse.
- Evidence: `internal/engine/run.go:1360-1382`; `internal/engine/inspect.go:38-54,201-258`; `internal/engine/resume.go:55-85,179-221,360-365`; `internal/engine/release.go:53-153,221-268`; `internal/engine/occupancy.go:251-256`.

### P3. A normal process interruption or controller restart can wedge Release and Resume

- **Severity:** High.
- A canceled process task can persist `incomplete` with its runtime PID. Release may close after partial reconciliation while retaining that PID, but a later Process executor has an empty process-local ownership map and cannot prove ownership of it.
- The documented Inspect → Release → Resume path can become stuck in `unknown-backend` with active occupancy, without a public adoption, abandonment, or repair transition.
- Evidence: `README.md:41`; `cmd/gobble/help.go:20`; `internal/engine/run.go:405-573`; `internal/engine/release.go:236-310`; `internal/engine/resume.go:368-395`; `internal/engine/exec/process.go:157-171`.

### P4. Artifact publication and control-state commit are not crash-durable as one result

- **Severity:** High.
- Multiple output files or Trees are installed before task success is persisted. There is no durable adoption, rollback, or replay record for termination between those steps, and file plus parent-directory synchronization is not an explicit commit boundary.
- A crash can leave partial outputs or complete but unowned outputs that block Resume with `output-exists`. Stable bytes and recorded success may also diverge after power or filesystem failure.
- Evidence: `internal/engine/exec.go:90-169`; `internal/engine/exec/publish.go:57-146`; `internal/engine/run.go:995-1022,1223-1244,1505-1529`; `internal/engine/tree.go:204-244`.

### P5. Docker terminal evidence and cleanup do not have one consistent owner

- **Severity:** High.
- Normal Poll reads logs and removes a container, while some cleanup failures are discarded. Reconcile of an already-stopped container returns terminal state without running the Poll log-copy and removal path. Cleanup can also precede durable engine acknowledgement.
- A restart can lose evidence, leave stopped containers behind, or persist terminal engine state that does not explain the backend result.
- Evidence: `internal/engine/exec/docker.go:78-107,125-143,275-301`; `internal/engine/run.go:536-570,978-1022`; `internal/engine/release.go:282-310`.

### P6. Process cancellation can signal an unrelated process after PID reuse

- **Severity:** High; host safety.
- Completed child entries remain in `Process.live` after Wait. `Cancel` trusts map membership and signals the stored PID or process group without checking completion or current operating-system identity.
- During the output-publication window, PID reuse can cause Gobble to kill an unrelated host process. Completed entries also accumulate for the executor lifetime.
- Evidence: `internal/engine/exec/process.go:71-92,133-154`; `internal/engine/run.go:376-383,995-1022`.

### P7. Successful cancellation does not impose an operation-level completion bound

- **Severity:** High.
- If Cancel returns success but Poll remains `Running`, Run, Resume, or Release can poll forever. Each per-call timeout uses a fresh background context and therefore does not bound the overall operation.
- Ctrl-C or recovery can leave active occupancy and polling goroutines indefinitely.
- Evidence: `internal/engine/run.go:405-428,541-559,978-993,1492-1502`; `internal/engine/release.go:289-308`.

### P8. Task environment crosses into the host Docker control plane

- **Severity:** High; trust boundary.
- `TaskSpec.Env` is passed both into the container and into the host Docker CLI process. Values such as `DOCKER_HOST`, `DOCKER_CONTEXT`, `DOCKER_CONFIG`, `HOME`, TLS settings, or `PATH` can change which daemon or configuration Submit uses, while later Poll, Cancel, and Reconcile calls use a different minimal environment.
- The engine can start a container it cannot later track, and task-owned configuration can influence a privileged host control plane.
- Evidence: `internal/engine/exec/docker.go:52-53,89-97,115,178-201,233-249,304-317`; `internal/engine/exec/docker_test.go:68-79`.

### P9. Samplesheet configuration depends on unsupported goroutine identity

- **Severity:** Medium.
- `SetSampleSheetPath` parses a human-readable `runtime.Stack` prefix and uses it as a key in a process-global `sync.Map`. Entries are not cleaned when goroutines end, and parse failure collapses callers onto key zero.
- This can leak state, expose a prior caller's path after ID reuse, and behave differently if runtime formatting changes.
- Evidence: `samplesheet.go:53,75-111`.

### P10. The generated CLI driver rejects valid consumer `internal` packages

- **Severity:** Medium.
- The CLI resolves the target with `go list`, but generates importer source under the system temp directory. Go's internal-package rule depends on the importer's physical location, so a valid caller package such as `./internal/pipeline` can fail to compile.
- This contradicts the apparent package-path contract and common Go module layout.
- Evidence: `cmd/gobble/driver.go:27-47,106-130,145-157`; `cmd/gobble/help.go:24-58`.

### P11. The accepted structured event surface is missing

- **Severity:** Medium; product completeness.
- The accepted Inspect design includes structured events and task-specific event retrieval, but the implementation exposes only current snapshots and logs through run, instances, errors, logs, timing, DAG, lineage, remaining, and reuse views.
- An agent cannot reliably reconstruct ordered transitions and decision context without interpreting snapshots or raw logs.
- Evidence: `design/feature/inspect-run.md`; `inspect.go:8-27`; `internal/engine/inspect.go:10-20,98-104`; `cmd/gobble/inspect_test.go:100`.

## Problems in test architecture and lifecycle assurance

### T1. Named bioinformatics recovery scenarios mostly prove unchanged no-op reuse after success

- WGS, RNA-seq, and methylation live tests normally complete successfully, Release, and Resume the unchanged graph. The shared helper proves occupancy refusal, Release, successful Resume, empty remaining, and at least one reuse record.
- The failing local fixture fails again after Resume, the bad-image test stops before Release and Resume, and the CLI signal test stops after Release.
- Interrupt recovery, corrected failure, selective rerun, changed input, changed sample membership, and downstream invalidation can regress while every assay test named `Recover` remains green.
- Evidence: `tests/local-e2e/wgs_e2e_assay_test.go:14-76`; `rna_live_test.go:13-32`; `methyl_live_test.go:14-42`; `helpers_test.go:366-399`; `harness_test.go:168-193`; `cmd/gobble/occupy_test.go:136-203`.

### T2. The common reuse oracle accepts partial evidence

- API and CLI helpers require only a non-empty reuse stream and validate each returned row independently.
- They do not compare the returned identities with the complete expected task or runtime-instance set, attempts, skipped identities, or sample members.
- One valid reused task with every other record silently missing still passes.
- Evidence: `tests/local-e2e/helpers_test.go:390-398`; `tests/local-e2e/harness_test.go:123-135,168-193`.

### T3. Several biological output oracles admit materially invalid results

- `requireRegularFile` accepts an empty regular file, and WGS uses it for BAM and MultiQC outputs.
- DESeq2 checks selected header fields but does not require a result row. STAR parses all samples but requires only one of four samples to meet the mapping and splice threshold. MultiQC content is not validated.
- Empty, corrupted, incomplete, or mostly failed assay outputs can therefore be reported as successful proof. Methylation's per-sample alignment and call-row floors demonstrate a stronger existing pattern.
- Evidence: `tests/local-e2e/helpers_test.go:189-195,455-499`; `wgs_e2e_assay_test.go:26-32`; `rna_live_test.go:27-30`; contrast `methyl_live_test.go:30-39`.

### T4. The highest-risk runtime defects have no distinguishing regressions

- Existing tests do not distinguish post-Wait process cancellation and PID reuse, successful Cancel followed by perpetual `Running`, stopped-container Reconcile cleanup, or same-process concurrent Release.
- Normal, race, and shuffled suites all pass despite those current production defects. This is a test-design gap independent of the defects themselves.
- Evidence: production boundaries in P2, P5, P6, and P7; nearest tests at `internal/engine/exec/process_test.go:118-168`; `internal/engine/lifecycle_test.go:115-183`; `internal/engine/exec/docker_test.go:222-380`; `internal/engine/release_test.go:32-293`.

### T5. Lifecycle coverage has no explicit owner or requirements traceability

- Coverage is distributed across root public tests, engine tests, executor tests, asset tests, CLI tests, and local E2E. `tests/local-e2e/PARITY.md` maps a migration, not product requirements or lifecycle axes.
- The run-local graph exists in three places and workflow-case in two. Harness discovery, build, and run helpers are repeated across packages.
- Maintainers cannot readily determine which scenario authoritatively protects a transition, whether API and CLI use the same graph, or where a missing fault case belongs.
- Evidence: 85 test files, 23,334 lines, 449 tests; `run_local_test.go:34-58`; `tests/local-e2e/run_local_live_test.go:141-165`; `tests/cli-valid/runlocal/pipe.go:5-30`; `compose_test.go:1072-1140`; `tests/cli-valid/workflowcase/pipe.go:5-74`; `tests/local-e2e/PARITY.md:1-23`.

### T6. Demoted WGS proof files retain unreachable live and record machinery

- The WGS thin and spine tests are explicitly plan-only, but their files still contain hundreds of lines for Docker execution, downloads, format checks, state extraction, proof records, fallbacks, and historical “Do not PASS” handling that no active `Test` reaches.
- This compiled historical machinery obscures the active evidence, increases maintenance cost, and can be mistaken for live coverage.
- Evidence: `tests/local-e2e/wgs_e2e_thin_test.go:110-133,273-653`; `wgs_e2e_spine_test.go:158-180,349-564`; `wgs_e2e_fixture_test.go:70-269`.

### T7. Public E2E tests also act as privileged storage-layout tests

- Package `local_e2e_test` imports `internal/engine`, uses internal status and filename constants, and reads `.gobble/tasks.json` directly.
- A valid storage-schema refactor can break a public behavior test, while a public Inspect regression can be hidden by direct file assertions. The protected contract is unclear.
- Evidence: `tests/local-e2e/helpers_test.go:17-20,401-410`; `run_local_live_test.go:13,45-88,110-139`.

### T8. The CLI temp-leak oracle watches a global cross-process namespace

- `watchDriverTemps` snapshots every `os.TempDir()/gobble-driver-*` path and attributes any later path to the current test.
- Other packages, test commands, or real CLI invocations can legitimately create the same prefix. Concurrent ordinary and race suites produced a false failure, while serial reruns passed.
- Evidence: `cmd/gobble/driver_test.go:350-373`; production creation at `cmd/gobble/driver.go:16,31`; 23 tests call the watcher.

## Improvement directions

The following items describe desired outcomes, not implementation choices. Parenthetical mechanisms and names are examples only. They must be challenged, adapted, or rejected through discussion before any concrete change is selected.

### Preserve attempt-owned scientific identity

Recorded input and output identity needs to refer to the immutable bytes that the attempt actually consumed and published, even when workspace paths change concurrently. Examples include capturing identity at completed staging, using attempt-owned artifact descriptors, or linking pre-publication and post-publication identities. The authoritative moment, hashing cost, large-file policy, and filesystem assumptions remain discussion topics.

### Give workspace state one serialization and recovery contract

Run, Resume, Release, and Inspect need a shared rule for which generation is readable and how partial state is detected or recovered. Example mechanisms include a generation marker, manifest commit, journal, single snapshot document, or read-set revalidation inside one lock. No mechanism is selected here; crash behavior, compatibility, and performance must be discussed together.

### Separate backend identity, terminal evidence, acknowledgement, and cleanup

Process and Docker handles need a defensible ownership identity, and terminal evidence needs to survive until the engine durably acknowledges it. Examples include process start identity, backend-specific adoption tokens, or one shared Poll/Reconcile terminal path. The supported restart and adoption model, security boundary, and cleanup guarantees require explicit discussion.

### Bound cancellation and recovery operations as whole calls

The public outcome needs to state when a non-cooperating backend becomes `unknown-backend` rather than creating another per-poll timeout indefinitely. A one-shot deadline or bounded retry budget is an example, not a decision. Caller authority, engine defaults, configurable policy, and exact state transitions remain open.

### Define the crash relationship between publication and success state

Multi-file and Tree publication needs an explicit adoption, rollback, or replay responsibility so a partial install is not confused with a completed task. Example concepts include an attempt manifest, prepared/committed markers, destination adoption checks, or file and parent-directory synchronization. Supported filesystems, consumer-owned artifacts, and performance costs must be discussed before choosing any of them.

### Separate task runtime configuration from backend control configuration

Container environment and host Docker client or daemon selection need different owners, while Submit, Poll, Cancel, and Reconcile need a consistent backend context. An engine-owned Docker configuration or a container-only environment policy are examples. The trusted-code assumption and remote-daemon support are product decisions, not conclusions of this review.

### Make public library configuration caller-owned and explicit

Samplesheet selection and executor configuration should not depend on hidden mutable process or goroutine state. Explicit options, a request value, context-bound configuration, or an Engine instance are examples only. CLI simplicity, library concurrency, and compatibility need to be considered together.

### Expose stable recovery evidence through the public observation boundary

Agents need structured terminal category, uncertainty, execution identity, exit evidence, and ordered transitions without interpreting raw logs. An events view or richer non-secret Inspect records are examples. Public schema stability, retention, and redaction for genomic or clinical data require discussion.

### Give lifecycle assurance an explicit scenario and evidence model

The suite needs a visible relationship between a product promise, trigger, backend, public surface, expected reused and rerun identities, artifact oracle, and evidence tier. A compact scenario matrix is one possible representation. A single mega-harness is not assumed to be the answer, because it could hide the distinctions the model needs to expose.

### Make reuse and biological oracles complete for their stated claims

Reuse checks need to reject missing or extra identities and incorrect attempt decisions. Bioinformatics checks need to reject materially invalid artifacts at the level each scenario claims. Examples include set equality at a stable public boundary, parseable non-empty artifacts, format validators, per-sample signal floors, relationship checks, or deterministic sentinel content. Exact thresholds and tool costs require domain-owner discussion.

### Clarify the supported bioinformatics scenario set

The absence of long-read, single-cell, spatial, metagenomic, proteomic, multi-lane, replicate, trio, or large-cohort evidence cannot be classified uniformly until Gobble states which scenarios belong to the local horizon. A taxonomy of data shape, lifecycle transition, and cost tier is an example discussion aid, not a proposed roadmap.

### Align test ownership with observable contracts

Public API and CLI scenarios, internal persistence checks, deterministic backend fault checks, and live assay evidence need distinct ownership even if they share carefully bounded fixtures. Examples include one authoritative graph per scenario, a narrow internal test utility, or moving historical proof records out of compiled tests. Final file layout and helper boundaries must follow an agreed contract rather than line-count targets.

## Terminology review

Terminology does not substitute for fixing lifecycle behavior. Renaming an exported Go symbol, CLI verb, JSON field, error operation, or persisted-state concept also creates compatibility work. The alternatives below are examples for discussion only; this report selects none of them.

| Current term | Assessment | Why discussion is warranted | Non-binding examples |
|---|---|---|---|
| `Release` / CLI `release` | Highest-priority terminology question | It overlaps software release, but the operation reconciles leftovers and closes workspace execution authority; it neither deletes nor terminally finalizes the run | Keep and clarify; `CloseOccupancy`; `CloseRun`; `DetachRun`—each has semantic drawbacks |
| `Defect`, `DefectCode`, `Defects` | Reconsider | The same family includes validation issues, cancellation, not-found, failure, and unknown backend; “defect” can imply a software bug | `Diagnostic`, `Issue`, or more specific failure families |
| `Directory` and `Tree` | Reconsider as a pair | `Directory` is path placement, while `Tree` is the actual directory-shaped artifact | `PathDir` or `Placement` for location; `DirectoryArtifact` for the artifact |
| `Module` | Contract may be weaker than the name | It is primarily a named structural scope and does not enforce an independently reusable or parameterized module boundary | Keep and strengthen the contract; `Scope`; `Stage`, noting that Stage implies order |
| `Branch` and `Merge` | Contract may be weaker than the names | `Bind.From` creates graph fan-out and fan-in; the named types themselves are scopes | Strengthen operator semantics; scope-oriented or explicit fan-out/fan-in names |
| `Bind` | Could be clearer | It represents a named input or output port and can be confused with a bind mount | `Port`, `ArtifactPort`, `Binding` |
| `Handle` | Too general | It is a wiring reference to a pipeline input or task artifact port | `PortRef`, `ArtifactRef` |
| `Group` | Too general | It is a named collection of regular-file members | `FileGroup`, `ArtifactGroup`, subject to existing enum compatibility |
| `TaskSpec.Backend` | Behavioral mismatch | Executor choice is currently driven by whether `Image` is empty, while `Backend` accepts little meaningful selection | Make it a real runtime selector, reserve it explicitly, or remove it from the current surface |
| `WriteTo` | Conflicts with Go expectations | It returns a plan option rather than writing or implementing `io.WriterTo` | `WithWriter`, `EmitTo`, or rely on a plan serialization method |
| `SetSampleSheetPath` | Contract problem more than a naming problem | It looks like an ordinary setter but emulates goroutine-local state through runtime internals | An explicit request or configuration API; a rename alone does not solve P9 |
| `RecordComposeError` | Lower-priority clarity issue | It exposes an escape hatch for injecting builder error state without expressing the authoring intent | `Fail`, `SetError`, or a narrower constructor error path |

`Compose`, `Validate`, `Plan`, `BuildPlan`, `Run`, `Inspect`, `Resume`, `Pipeline`, `Graph`, `Task`, `TaskSpec`, `Param`, `Resources`, `PathSpec`, `View`, `Scatter`, `Gather`, and `When` generally fit the current workflow model. `BuildPlan` should still be documented clearly as dry-run plan construction rather than image or software build.

`Release` deserves special caution. `CloseRun` may falsely imply a terminal run, `CloseOccupancy` may expose an internal mechanism, and `DetachRun` may imply that work continues detached. The underlying user intent and behavior contract therefore need discussion before any name can be judged better.

## Strengths to preserve

- The backend-independent Go pipeline model and the refusal to require a proprietary DSL.
- The separation between public model, validation and planning, scheduler, and executors.
- A small structured API and CLI verb set.
- File, Group, and Tree as distinct artifact shapes.
- Reserved identity and explainable content- and environment-aware reuse decisions.
- Fail-closed occupancy liveness based on flock and lease rather than PID existence alone.
- Fast hermetic first-checks that do not silently claim live Docker evidence.
- Direct synthetic tests for Scatter, Gather, When, Tree, Group, attempts, failure containment, and resume classification.
- Per-test temporary resources and structured diagnostics in much of the suite.
- The existing API and CLI parity foundation for run-local, WGS, RNA-seq, and methylation scenarios.
- The explicit local-first scope and honest deferral of HPC, cloud, object storage, and broad ecosystem claims.

## Verification evidence and limits

The following commands passed on Go 1.26.5, Linux/amd64, at the frozen revision:

- `go test ./tests/local-e2e`
- `go test ./...`
- `go test -race ./...`
- `go test -shuffle=on -count=3 ./...`
- `go vet ./...`

The production review also recorded a passing `go test -cover ./...`, with observed package coverage of 79.3% for the root package, 92.3% for assets, 87.4% for CLI, 65.7% for engine, and 63.3% for executors. Coverage percentage does not resolve the missing lifecycle distinctions described above.

A concurrent ordinary and race-suite execution exposed T8's global temp-namespace interference; subsequent serial runs passed. The conflicting evidence is retained rather than replaced by the later passes.

The following limits materially constrain the claims of this report:

- Live-tagged Docker, network, and bioinformatics tool execution was not refreshed at this revision.
- Abrupt termination between publication and state writes, power-loss durability, provenance mutation, safe PID-reuse injection, a non-cooperating backend, and concurrent same-workspace Release or Resume were not dynamically exercised.
- The project does not define an accepted minimum taxonomy of supported bioinformatics modalities, data shapes, or recovery scenarios.
- CI configuration and continuous enforcement were not observable.
- Representative multi-gigabyte files, large cohorts, dense graphs, long attempt histories, and repeated recovery were not measured.
- Protected genomic or clinical data handling, redaction, retention, deletion, and shared-filesystem trust policies remain undefined.
- An installed binary and separate external consumer module were not used to validate package-layout, version-skew, and compatibility behavior.
- The evidence is Linux/amd64 only.

## Conclusion

Gobble already has the functional skeleton of a useful local bioinformatics pipeline engine. Compose, Validate, BuildPlan, process and Docker execution, structured Inspect, and content-aware Resume form a coherent product direction, and the test suite provides broad synthetic support for many of those mechanisms.

The lifecycle is not yet sufficiently reliable when “sufficient” includes scientific provenance, interruption, controller restart, partial publication, same-workspace concurrency, backend uncertainty, and bounded recovery. P1 through P8 are material correctness or safety issues. The testing problems compound that risk because the named assay recovery scenarios, reuse oracle, biological output checks, and missing fault regressions can all remain green while important lifecycle behavior is wrong.

The appropriate direction is to strengthen the existing boundaries and evidence model, not to infer a concrete refactor or roadmap from this report. All mechanisms, test architectures, thresholds, and alternative names described here remain discussion examples. They may be changed or rejected, and no source change or public decision should be treated as approved until that discussion is complete.
