# Gobble local pipeline engine distribution readiness analysis

This analysis evaluates what is missing before Gobble can be distributed as a local bioinformatics pipeline engine. It derives release implications from the completed [local pipeline lifecycle, code, and test review](../review/2026-08-23-local-pipeline-lifecycle-and-test-review.md) and adds a read-only inspection of module, CLI, compatibility, packaging, operational, and support surfaces.

> **Decision boundary:** This is a readiness analysis, not an implementation plan, release preparation, change approval, compatibility decision, or prioritization. Any concrete mechanism or policy mentioned is a non-binding discussion example. Actual changes, release class, version, tag, artifact, and destination require separate discussion and explicit decisions.

## Analysis record

| Field | Value |
|---|---|
| Completed | 2026-08-23 |
| Frozen subject | `develop` commit `c9ffeb84c611814fbf00a4ec15036b5939fa46b7`, tree `abb9d73f6b1b297e8dc8ba610033905391c96706` |
| Question | What is missing before Gobble can be distributed as a local pipeline engine? |
| Included | Lifecycle correctness, process and Docker safety, workspace recovery, CLI/module/schema compatibility, installation, release identity, public contract, security and data handling, resource behavior, verification, support, and bioinformatics claim boundaries |
| Excluded | Source changes, test changes, version or tag selection, artifact build, network access, credentials, publication, deployment, rollout, and live-health action |
| Release subject | No final module release or binary/archive checksum identity exists |
| Readiness result | Not ready for a stable public or production/research release |
| Bounded current use | Internal technical preview under a narrow trusted environment |
| Record state | Partial — source and prior hermetic evidence are exact; current live Docker, fault, external-consumer, CI, scale, privacy, and artifact evidence are absent |

## Executive verdict

Gobble already has the functional skeleton of a local pipeline engine: Go-based composition, validation, dry-run planning, process and Docker execution, structured inspection, occupancy release, change classification, and reuse-aware resume. The model–engine–executor split and the File, Group, Tree, Scatter, Gather, and When concepts are credible foundations.

The product is not ready for stable distribution because its principal gaps are not missing feature count. They are correctness, recovery, host safety, version coherence, public compatibility, release identity, operational policy, and evidence. The current code can falsely reuse scientifically incorrect provenance, wedge recovery, signal an unrelated process after PID reuse, wait indefinitely after cancellation, and let task configuration influence the host Docker control plane. The CLI can also apply different Gobble versions to different operations on the same workspace.

The recommended classification is:

| Distribution class | Current assessment | Boundary |
|---|---|---|
| Internal source evaluation | Possible | Trusted single user, Linux, disposable workspace, synthetic or non-sensitive data, pinned source revision, no reliability claim |
| External technical alpha | Not ready | Host-safety, provenance, state, version/schema, public-contract, license, and release-identity gaps remain |
| Stable public module or CLI | Not ready | No supported API contract, upgrade policy, release artifact, CI gate, or external-consumer evidence |
| Production research engine | Not ready | Scientific correctness, crash recovery, cancellation, backend reconciliation, and durable provenance are not established |
| Clinical or regulated use | Out of scope and unsupported | Security, permissions, audit, retention, deletion, provenance, and validation obligations are undefined |

## Readiness by product area

| Area | Current condition | Readiness judgment |
|---|---|---|
| Compose, Validate, BuildPlan | Broad public and hermetic behavior exists | Strong foundation |
| Normal Run | Process and Docker execution works across synthetic and live-source scenarios | Functionally present; operational safety is incomplete |
| Inspect, Release, Resume | Structured views and reuse-aware recovery exist | Important paths can wedge or observe incoherent state |
| Scientific provenance | Task fingerprints, environment digest, image digest, and executable hash exist | Actual consumed/published bytes can be recorded incorrectly; run-level version provenance is absent |
| Workspace state | Local JSON control documents and flock/lease occupancy | No single crash-durable generation or migration contract |
| Process backend | Isolated working directory and process group | Trusted-host execution only; PID reuse and restart recovery are unsafe |
| Docker backend | UID/GID execution, network disabled, CPU/memory flags | Control-plane environment, Reconcile cleanup, runtime dependency, and image policy are incomplete |
| Public Go API | Large exported authoring and lifecycle surface | README declares almost all of it unsupported |
| CLI | Structured verbs and source-install instruction | Runtime Go compiler required; binary and generated driver may use different Gobble versions |
| Packaging and publication | Module path and version output exist | No tag, license, artifact, checksum, release policy, or publication evidence |
| Verification | Normal, race, shuffle, vet, and prior coverage runs passed | No current live, fault, CI, installed-consumer, scale, or upgrade evidence |
| Bioinformatics scope | WGS, bulk RNA-seq, methylation, and synthetic proofs | Useful canaries; not broad modality or scientific-validity evidence |

## Distribution blockers

### 1. Scientific reuse cannot yet be trusted

The prior review's P1 shows that input and output provenance can be calculated from mutable workspace paths after the attempt consumed or published different bytes. A task can consume A, record B, and later reuse an A-derived result when B is present. This is a false-reuse risk and a scientific correctness blocker, not merely incomplete metadata.

Task state records command, script, params, environment digest, image digest, executable path and SHA-256, file fingerprints, checksums, and lineage. Those are useful foundations. However, the run does not durably identify the Gobble engine version, pipeline module/source revision, Go toolchain, Docker client/server, or a complete execution-environment identity. A future reader cannot fully bind a result to the pipeline and engine bytes that produced it.

A stable bioinformatics engine needs conservative, attempt-owned provenance and a run-level version identity. The exact authority point and hashing policy remain design decisions.

### 2. Workspace state is not one durable generation

The prior review's P2 and P4 establish that `plan.json`, `tasks.json`, and `run.json` are not committed as one transaction and artifact publication is not one crash-durable result with task success. Same-process concurrent Release calls can also bypass flock serialization through `holdsLease`.

A crash can leave partial outputs, ownerless complete outputs, or a mixture of control documents that never represented one real generation. Resume can then refuse with `output-exists`, reuse the wrong state, overwrite attempts, or remain unable to explain what is safe to keep.

Stable distribution requires one readable-generation and recovery contract across Run, Inspect, Release, and Resume. Generation markers, journals, manifests, or a single snapshot are examples only; this analysis chooses none.

### 3. Interrupt and restart recovery is incomplete

A Process executor proves ownership through an in-memory map. After a controller restart, a new executor cannot prove the persisted PID belongs to Gobble. This fail-closed stance is safer than adopting any live PID, but the product has no complete public transition for adoption, abandonment, or repair. The documented Inspect → Release → Resume path can remain stuck in `unknown-backend` with active occupancy.

Controller crash, terminal closure, `kill -9`, and machine restart are normal operational events for a foreground local CLI. A product that claims recoverability must define the terminal result for each of them.

### 4. Process cancellation can harm the host

Completed child entries remain in `Process.live`, and Cancel can signal the stored PID or process group without first proving the process is still the same operating-system identity. PID reuse can therefore cause an unrelated process to receive SIGKILL.

This blocks distribution on shared workstations, multi-user analysis servers, and hosts running important services. A clean Go race run does not exercise operating-system identity reuse.

The process backend is also not a sandbox. It executes trusted commands as the Gobble OS user and can access any resource available to that user. That trust model can be acceptable for a narrowly documented developer engine, but not for untrusted pipeline execution.

### 5. Cancellation and recovery calls have no whole-operation completion bound

The Engine applies a bound to each Submit, Poll, Cancel, or Reconcile call, but a successful Cancel followed by perpetual `Running` can cause Run, Resume, or Release to poll forever with a new per-call bound each time.

A public recovery operation needs a finite terminal contract, including when the backend becomes `unknown-backend`. Exact time values and caller-versus-engine authority remain discussion topics.

### 6. Docker runtime and control-plane ownership are incomplete

`TaskSpec.Env` is passed both into the container and into the host Docker CLI process. Task-owned values such as `DOCKER_HOST`, `DOCKER_CONTEXT`, `DOCKER_CONFIG`, `HOME`, TLS configuration, or `PATH` can change daemon or credential selection during Submit. Later Poll, Cancel, and Reconcile calls use a different minimal environment and may not find the container.

Normal Docker Poll copies logs and removes a terminal container, while Reconcile of an already-stopped container does not execute the same cleanup path. Controller restart can therefore leave containers, lose logs, or produce a terminal engine record without consistent backend evidence.

Image absence also causes an automatic `docker pull`, but the runtime network, offline, private-registry, credential, supported Docker-version, and image-policy contracts are not documented. Runtime image ID recording is a strength, but authored assets are tag-addressed and no immutable live evidence record is retained for this revision.

### 7. One installed CLI can use two Gobble engine versions

The installed binary executes `inspect` and `release` in-process using the Gobble version compiled into that binary. The graph verbs `compose`, `validate`, `plan`, `run`, and `resume` instead generate and compile a temporary driver in the consumer module. That driver imports both the consumer pipeline and `github.com/HahyeonJeon/gobble`, so its Gobble version can be selected by the consumer's `go.mod` rather than by the installed CLI.

The same CLI can therefore apply version A to Run or Resume and version B to Inspect or Release on the same workspace. `gobble version` reports only the installed binary version, not necessarily the version used by the generated driver.

This is especially risky because workspace schema version 2 is the only supported schema and there is no migration. A future version change can make a workspace writable by one operation and unreadable by another. The CLI, generated driver, public library, and workspace schema need one explicit compatibility relationship before external distribution.

### 8. The public authoring contract is explicitly unsupported

The README states that the public surface is unsupported except for locked PathSpec concepts. In practice, every pipeline author depends on a much larger exported surface: Pipeline, TaskSpec, Bind, Handle, Module, Branch, Merge, Scatter, Gather, When, Compose, Validate, BuildPlan, Run, Inspect, Resume, Release, Error, Defect, and related methods and constants.

CLI consumers are also Go API consumers because graph verbs compile their pipeline package. A stable CLI cannot avoid the public library compatibility question.

The `assets` package adds another ambiguous surface. It is importable as a public Go package, but its package contract says it contains proofs and should not be required by third-party authors. The distributed module needs a clear distinction between supported product packages and examples or proofs.

### 9. Workspace upgrade and migration are undefined

The current Engine writes schema version 2, rejects schema 0 and 1, and supplies no migration. There is no policy for forward compatibility, backward compatibility, downgrade, inspection-only support, in-place versus copy migration, migration failure, or preservation of the original workspace.

A long-lived local engine needs to state whether a newer CLI can Inspect, Release, or Resume an older workspace and how users retain access to old results. The missing engine and pipeline version fields make this harder.

### 10. No exact distributable release exists

The README presents `go install github.com/HahyeonJeon/gobble/cmd/gobble@<version>`, but the repository has no Git tag and no exact version decision. No final binary or archive checksum identity was supplied or built.

The tracked repository also has no license, changelog, security policy, contributing policy, release configuration, checked-in CI workflow, release notes, binary archive, checksum manifest, signing record, SBOM, or publication evidence. A public-use and redistribution policy is therefore undefined, and there is no continuously enforced release gate.

The absence of some community files need not block an internal preview, but license, exact release identity, compatibility, and verified installation are fundamental for public distribution.

### 11. The CLI is not a self-contained binary engine

Graph verbs locate `go`, resolve the consumer package, generate source, compile a driver, and execute it. Users therefore need Linux, Go 1.26 or newer, a resolvable consumer module, the appropriate Gobble module dependency, a caller-created workspace, and Docker for container tasks.

A prebuilt CLI would not remove the runtime Go toolchain requirement under the current architecture. Fresh-cache installation, offline behavior, module download, private module, version-skew, consumer `internal` package, and installed-binary smoke behavior have not been verified in a separate external module. The generated source living under system temp already rejects a valid consumer `internal/pipeline` layout.

The product must either embrace and document a Go-toolchain-based developer engine or adopt a different distribution contract. This analysis does not choose between them.

### 12. Security, privacy, retention, and cleanup policy are insufficient

The Engine commonly creates control and work directories with `0755`, control and log files with `0644`, and published outputs with `0444`. A restrictive parent workspace can still protect them, but a permissive or shared workspace can expose data to other local users. Responsibility for workspace mode, umask, shared groups, and protected data is not documented.

Environment values are represented by a digest rather than persisted directly, which is a useful protection. Commands, scripts, params, stdout, stderr, artifact paths, and lineage are still stored. Secrets or protected identifiers placed in those channels can remain in the workspace. There is no redaction, retention, secure deletion, audit-access, or clinical/genomic data policy.

Guarded Clean is intentionally not shipped and consumer deletion is allowed. Stable distribution still needs an explicit ownership and retention contract even if Gobble never implements a Clean verb.

### 13. Resource behavior is narrower than a general local engine claim

The scheduler accounts for CPU, memory, and a run-level concurrency cap. Docker receives `--cpus` and `--memory`; host process tasks do not receive OS-level CPU or memory enforcement and can exceed their declarations.

There is no task wall-clock timeout, disk or scratch quota, process-count limit, GPU, I/O bound, priority, or graceful-termination window. A minimal local engine does not need every resource type, but the distinction between Docker enforcement and process admission information must be explicit, together with responsibility for runaway tasks and full disks.

### 14. Verification does not support an external release claim

The frozen revision has strong local hermetic evidence: normal tests, race detector, shuffled repeated tests, vet, and prior coverage all passed. The test suite also has valuable direct coverage of composition, operators, attempts, change classification, and structured CLI behavior.

The release claim still exceeds the evidence because:

- Live-tagged Docker and bioinformatics execution was not refreshed at this revision.
- Named WGS, RNA-seq, and methylation Recover scenarios mainly prove unchanged Resume after success, not failure or interrupt followed by successful recovery.
- Reuse helpers accept a non-empty subset rather than the complete expected identity and attempt set.
- WGS and RNA biological output oracles admit materially invalid or incomplete results.
- The current PID reuse, perpetual Poll, stopped-container Reconcile, and concurrent Release defects have no distinguishing regressions.
- No deterministic crash, partial-publication, migration, installed external consumer, fresh-cache install, CI, scale, or non-Linux evidence exists.

Test count and line coverage cannot substitute for those lifecycle distinctions.

## Release-readiness outcomes

These are acceptance outcomes, not an implementation order.

### Correctness and host safety

A distributable engine needs provenance bound to the bytes the attempt actually consumed and published, process cancellation that cannot affect an unrelated OS identity, serialized same-workspace lifecycle operations, bounded recovery calls, and separation between task environment and Docker control configuration.

### Recovery semantics

Contained failure, Ctrl-C, controller crash, machine restart, running/stopped/missing backend jobs, partial control state, and partial output publication each need a finite structured outcome: resumable, safely refused, or explicitly repairable.

### Version and compatibility

Installed CLI version, generated-driver library version, pipeline source/module identity, workspace schema, and Inspect/Release/Resume compatibility need one observable contract. Upgrade, migration, deprecation, and supported public symbols must be decided before a stable claim.

### Distribution identity

Public distribution needs an owner-decided exact version and tag, license, immutable module or artifact identity, installation evidence, supported target, release notes, known limitations, and artifact checksums when binary or archive distribution is selected. No specific version, tag, or artifact form is selected here.

### Verification

Evidence needs distinct hermetic, concurrency-fault, exclusive-live, failure-to-recovery, installed-consumer, schema-compatibility, crash, scale, and biological-artifact layers. CI must make the chosen layers continuously observable or explicitly record which remain manual.

### Operating and security contract

The product needs explicit trusted-code assumptions, Linux/Go/Docker support, filesystem scope, workspace permissions, network and image-pull behavior, registry and credential policy, secret-safe authoring, log and artifact retention, cleanup ownership, unsupported-state recovery, and support or vulnerability-reporting boundaries.

## What is not required for the first local distribution

Release readiness should not be confused with feature breadth. The following are not inherent blockers if the public scope remains local and explicit:

- More assay families such as long-read, single-cell, spatial, metagenomics, or proteomics.
- HPC, Slurm, cloud batch, Kubernetes, or object storage.
- A GUI, TUI, or proprietary DSL.
- Cross-platform support beyond Linux.
- Cross-workspace caching, a public Cancel verb, named retry, or a built-in Clean verb.
- Every possible resource type or scheduler policy.

Adding more pipelines now would increase images, downloads, live-test cost, duplicate weak oracles, and exposure to the same unsafe lifecycle invariants. The engine first needs modality-independent conformance across success, corrected failure, interrupt, restart, mutation, graph change, backend uncertainty, concurrent control, and crash boundaries. Real assays can remain a small set of strong canaries after the supported scenario contract is discussed.

## Bounded current-use recommendation

The current source may be shared only as an internal technical preview with an explicit non-production boundary. A defensible preview environment is a trusted single-user Linux host, disposable or backed-up workspace, synthetic or non-sensitive data, pinned source and toolchain, no shared control operations, and no claim that cancellation, controller restart, provenance, or reuse is production-safe.

External alpha distribution should not be represented as ready while host-safety, provenance, state coherence, version/schema split, public compatibility, license, and exact release identity remain open. Stable or production/research distribution additionally requires the missing fault, live, installed-consumer, migration, operational, privacy, and support evidence.

## Evidence and limits

This analysis inspected the exact frozen source, current README and module declaration, CLI driver and version paths, workspace schema and state records, process and Docker adapters, repository release metadata, Design Memory, and the completed lifecycle/code/test review. The subject remained commit `c9ffeb84c611814fbf00a4ec15036b5939fa46b7`, tree `abb9d73f6b1b297e8dc8ba610033905391c96706`.

No source or test was changed. No build artifact was created, and no tag, version, license, release destination, credential, network access, publication, deployment, or live run was authorized or performed. The version split is a source-grounded inference from `cmd/gobble/driver.go`, `cmd/gobble/main.go`, and `cmd/gobble/streams.go`; it has not yet been demonstrated with two separately tagged Gobble versions in an external consumer module.

## Conclusion

Gobble is a meaningful developing local engine, but it is not a distributable stable product merely because its lifecycle verbs exist. Scientific provenance, state coherence, restart recovery, host-safe cancellation, bounded completion, Docker ownership, CLI/library/schema version alignment, supported public APIs, release identity, security policy, and release-grade evidence remain incomplete.

The best current decision is to keep distribution within a tightly bounded internal technical preview and strengthen existing lifecycle boundaries before expanding assay breadth. Every mechanism and policy in this analysis remains subject to discussion; no change, release, or compatibility decision is approved by this report.
