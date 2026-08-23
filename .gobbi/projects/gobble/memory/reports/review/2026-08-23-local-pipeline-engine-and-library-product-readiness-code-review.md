# Gobble Whole-Product Distribution Readiness Code Review

> **Document role:** Durable point-in-time Code Review report  
> **Record state:** `partial`  
> **Subject:** Tracked Gobble repository, including production Go, tests, `cmd/gobble`, assets, testdata, README/docs, `go.mod`, the absence of `go.sum`, and release-facing metadata  
> **Frozen revision:** commit `c9ffeb84c611814fbf00a4ec15036b5939fa46b7`, tree `abb9d73f6b1b297e8dc8ba610033905391c96706`  
> **Reviewer:** Codex Gobbi evaluator  
> **Relationship:** `not the author`  
> **Checklist:** Canonical Code Review checklist at SHA-256 `89aaf24bad0d5f0deddba8df22103f7380c6aa348effe3468c5573cbc43fce07`  
> **Report template:** Code Review report template at SHA-256 `1ea3d640df200cc807398363a62e7cf7a61b4e980baeeaca1177580645b7c17e`  
> **Reviewed at:** 2026-08-23 UTC  
> **Evaluation timing:** This is prepared review material. A later evaluator must review the same subject unaided before reading it.  
> **Authority:** This report supplies Problems, Improvement directions, Strengths, and Gaps. It does not approve, gate, score, correct, publish, release, or otherwise authorize a product decision.

## Summary

This is a partial, evidence-backed whole-product critique of Gobble as a prospective local bioinformatics pipeline engine and Go library. The frozen source has a coherent compose/validate/plan/run/inspect/resume/release shape, explicit bounded local admission, conservative occupancy handling, useful checksums and lineage, and a good hermetic core. The supplied root verification evidence is also strong for its stated scope:

- Go `go1.26.5 linux/amd64`;
- `go test ./...` passed;
- `go test -race ./...` passed;
- `go test -shuffle=on -count=3 ./...` passed;
- `go vet ./...` passed.

Those results narrow claims but do not establish live Docker or network behavior, crash/power-loss recovery, installed external consumers or final artifacts, scientific truth sets, representative performance workloads, or release/legal policy.

The central product issue is not one isolated function. The distribution boundary is unfinished. The repository has no versioned release/support/legal/artifact contract, its README simultaneously calls the public surface unsupported and instructs consumers to use the broad API, and the generated CLI compiles a caller package without a module/API compatibility envelope. Persisted workspaces also lack a durable process identity handoff and a transaction owner for the three-file control snapshot. These conditions make the ordinary documented recovery and consumer handoff depend on private implementation facts.

The aggregate retains category-specific problems where their earliest roots differ: Docker trust and image identity, protected-data persistence, workspace race/confinement, scientific design/output validation, CLI protocol and argument handling, concurrency ownership, observability, public API ownership, state/artifact provenance, scheduler structure, resource behavior, testing oracles, and evidence/release gates. The 16 isolated category reviewers supplied prepared specialist reconciliation; their symptoms were deduplicated to the earliest evidenced root and assigned one primary category below.

No contract-gate verdict is issued. The record is `partial` because the source and static evidence are complete for the reviewed boundary while important runtime, artifact, consumer, scientific, performance, and policy evidence is absent or explicitly excluded.

**Escalations:** None. This report does not escalate a Problem into an approval or release gate; the materiality labels below are descriptive only.

## Subject and Scope

| Item | Detail |
|---|---|
| Included | All tracked production Go, tests, `cmd/gobble`, assets, testdata, README/docs, `go.mod`, the absent `go.sum` as dependency/release evidence, generated-driver and CLI paths, workspace/control/artifact schemas, process and Docker executors, release-facing metadata, and the product/library integration surface. |
| Intended question | What problems and improvement directions must be discussed and resolved for product-level distribution as a local bioinformatics pipeline engine and Go library? |
| Excluded | `.git`; agent/plugin configuration; all `.gobbi` reports as subject; current dirty/untracked Memory; network and live external runtime; live Docker; external publication, installation, mutation, or coordination. The prepared reports were evidence inputs only, not subject. |
| Governing sources | `AGENTS.md` (`# Gobble`); accepted tracked Design Memory; tracked README/module docs; actual source/tests/assets; Gobbi Code Review skill; canonical Code Review checklist; applicable Go, CLI, concurrency, security, packaging, release, testing, and bioinformatics specialist sources. |
| Frozen at | `HEAD=c9ffeb84c611814fbf00a4ec15036b5939fa46b7`; `HEAD^{tree}=abb9d73f6b1b297e8dc8ba610033905391c96706`. |
| Invalidation | Any change to the source commit/tree, accepted design, support/release contract, governing review source, or included evidence invalidates this aggregate and requires revalidation. |
| Original review write boundary | Only the caller-bound review report was written. No source, tests, Memory, Git state, external state, or other report was changed during the review. |

## Method

The independent Phase 2 review inspected the actual frozen subject and locked a checklist-free record before the canonical checklist, report template, prepared item lists, or prior same-subject reports were opened. That locked record is preserved below without backfill. After the lock, the full canonical checklist and report template were read, every core category was traversed in source order, and the activated specialist surfaces were reconciled.

### Locked checklist-free record — preserved without backfill

These are the five direct observations recorded before checklist or prepared-material exposure. Later specialist evidence is not inserted into this record:

1. The library → engine → CLI lifecycle is coherent enough to inspect, run, resume, and release locally, but the distribution/support/release contract is absent and the README contains unsupported/placeholder language.
2. Local process recovery stores proving state in memory, task environment reaches the host Docker client, generated user-code output can violate the JSON protocol, and live behavior was unverified.
3. Control files are written through separate per-file atomic replacements without a cross-file durability owner; Docker reconciliation/log and cleanup behavior is not a complete lifecycle contract.
4. The public API is explicitly unsupported in the README despite being the stated consumer path, and no release identity, CI, license, changelog, or external-consumer evidence is present; live evidence is absent.
5. Release/support, compatibility/migration, retirement, CI/release artifacts, and reproducibility are lifecycle absences, not merely missing polish.

### Prepared-material reconciliation

Sixteen isolated category reports supplied prepared specialist reconciliation: `architecture.md`, `bioinformatics-domain.md`, `build-packaging-release.md`, `cli-config.md`, `code-quality-refactoring.md`, `compatibility-delivery.md`, `concurrency.md`, `correctness.md`, `dependencies.md`, `observability-operations.md`, `performance-resources.md`, `public-api.md`, `security-privacy.md`, `state-recovery.md`, `testing.md`, and `whole-product.md`. They were held in a temporary review workspace and were not retained in project Memory. They were read as independent evidence after the locked record. Repeated symptoms were merged to the earliest supported root; distinct category-specific roots remain separate.

The canonical Code Review checklist was read in full at SHA-256 `89aaf24bad0d5f0deddba8df22103f7380c6aa348effe3468c5573cbc43fce07`. The Code Review skill was read at SHA-256 `2e29f70020ef602080d281a0d953ff413fbbc1541e3fe9ce759ac224dae53c0b`; the Code Review report template was read at SHA-256 `1ea3d640df200cc807398363a62e7cf7a61b4e980baeeaca1177580645b7c17e`. The checklist-free record above was not amended from either source.

### Verification evidence and limits

The caller supplied the following serialized root verification evidence, reproduced exactly: Go `go1.26.5 linux/amd64`; `go test ./...` passed; `go test -race ./...` passed; `go test -shuffle=on -count=3 ./...` passed; `go vet ./...` passed. These are accepted here as serialized evidence for the frozen identity. They do not cover live Docker/network, crash/power-loss injection, installed external consumers/final artifacts, scientific truth sets, performance workloads, or release/legal policy. Category reports that described their narrower local assignment as not executing tests do not contradict this aggregate evidence; they explain why those reports did not independently claim the root verification.

### Core category applicability and source-order coverage

Every core category was applicable to this product surface. The table records the category result in canonical source order and points to the aggregate evidence; a category result of `problem found` does not mean every checklist sign is negative, and `evidence missing` is not converted into a claim.

| Core category | Result and aggregate coverage |
|---|---|
| Project Fit | `applicable`; README and accepted Design Memory establish a local engine/library purpose, but the unsupported-versus-installable product stance is unresolved (P1–P2). |
| Affected Surfaces | `applicable`; public Go API, generated CLI, control files, executors, assets, testdata, logs, artifacts, and release metadata all affect the result (P1–P5, P18–P19). |
| Project Structure | `applicable`; source, generated driver, assets, and internal path policy have ownership gaps (P27–P30, P38). |
| Architecture | `applicable`; executor/artifact ownership, process recovery, scheduler concentration, and control-state ownership cross boundaries (P4, P5, P22, P27). |
| Design Pattern | `applicable`; leases, adapters, generated driver, snapshots, reuse and publication patterns are material; generated protocol and backend roles are inconsistent (P3, P14, P22). |
| Abstraction | `applicable`; public TaskSpec exposes Docker mechanics and raw inspection/protocol details while hiding durable ownership (P2, P22, P40). |
| Data Model | `applicable`; run identity, runtime identity, snapshots, checksums, lineage, manifest, attempts, errors, and absence states are persisted or exposed (P4–P6, P19, P23, P24). |
| Public API | `applicable`; Compose/Validate/BuildPlan/Run/Inspect/Resume/Release and exported types are the distribution contract, yet support, ownership, typed views, errors, and extension boundaries are unsettled (P2, P20, P22, P39–P41). |
| Parameters | `applicable`; CLI flags, task specs, sample-sheet selection, image/backend/resources, and environment determine behavior; parser and hidden configuration risks remain (P11, P15, P21). |
| Modularization | `applicable`; scheduler, duplicated validation, generated protocol, and PlanOption boundaries have independent change reasons (P27–P30, P41). |
| Reusability | `applicable`; backend reconciliation, compose/graph rules, artifact proof and reuse identity must agree but currently differ or duplicate (P6, P28, P30). |
| Performance | `applicable`; large DAG, full snapshot, copy-only artifact, polling, retention, and sample-sheet paths are workload-sensitive; no representative workload claim is made (P31–P32, Gaps). |
| Optimization | `applicable`; hardlink/symlink design, indexes, adaptive polling, and incremental persistence are relevant improvement discussions, not decisions (P32). |
| Unintended Overengineering | `applicable`; public PlanOption and broad backend/inspection abstractions create compatibility burden without a settled consumer need (P22, P41). No author motive is inferred. |
| Code Complexity | `applicable`; `internal/engine/run.go` is 2,347 lines and owns multiple lifecycle concerns (P27). |
| Readability | `applicable`; lifecycle vocabulary, raw view shapes, protocol ownership, and generated source provenance require cross-file tracing (P2, P3, P14, P41). |
| Vocabulary | `applicable`; `Release`, Branch/Merge, unknown, published, and supported/unsupported carry product consequences (P1–P2, P41). |
| Naming Convention | `applicable`; lifecycle and topology names do not consistently express ownership or effect (P2, P41). |
| Docstring | `applicable`; README/root comments provide a useful path but conflict on support and omit recovery/compatibility boundaries (P1–P2, P4). |
| Correctness | `applicable`; state transitions, artifact proof/publication, CLI protocol, map order, scatter, resume, and empty-run behavior require correction or discussion (P5–P6, P14, P23–P26). |
| Testing | `applicable`; unit and boundary coverage is broad, but fakes, scientific oracles, crash tests, schema skew, path tests, concurrency tests, and live gates leave material gaps (P33–P38). |
| Verification | `applicable`; serialized root checks passed, while excluded runtime/artifact/scientific/performance/release evidence narrows claims (Method and Gaps). |
| Delivery | `applicable`; module installation, generated CLI, assets, final bytes, checksums, license, changelog, support and external consumer handoff are unbound (P1–P3, Gaps). |
| Usability | `applicable`; a consumer can find the ordinary verbs, but cannot reliably classify versions, run identity, recovery ownership, raw views, or errors (P2–P4, P19, P40). |
| Operations | `applicable`; occupancy, reconciliation, logs, release/resume, correlation, retention and cleanup are operational product behavior (P4, P16–P19, P30). |
| Compatibility | `applicable`; Go API, generated CLI, persisted schema 2, runtime identity, platform tuple, assets, and support transitions need one versioned policy (P1–P3, P36, Gaps). |

### Overlay applicability

| Overlay | Applicability | Basis and limitation |
|---|---|---|
| Security | `activated` | User Go/shell code, paths, Docker, images, host executable lookup, URLs, environment and workspace files cross trust boundaries. Static security findings are retained (P7–P9, P11); no vulnerability or hostile-runtime verdict is claimed. |
| Privacy | `activated` | Samplesheets and bioinformatics workspaces may contain sensitive genomic/sample data, and commands/logs/params persist. The source lacks a protected-data policy (P10); privacy classification and legal policy are absent. |
| Concurrency | `activated` | Goroutines, scheduler workers, shared maps, cancellation, leases, process/Docker adapters, and public lifecycle calls are material (P16–P17, P21, P30, P37). Root serialized race evidence does not prove all shutdown/runtime cases. |
| Accessibility | `not applicable` | The bounded product surface is a line-oriented CLI/library with no GUI, focus, motion, contrast, or assistive-widget surface. |
| Localization | `not applicable` | No locale, translation, date/number/currency, collation, or text-direction contract is present or claimed. |
| Dependencies | `activated` | Go module resolution, generated caller imports, Docker images, executable lookup, and downloaded fixtures are external inputs (P3, P8, P11). |
| Build | `activated` | Go compilation, generated driver, toolchain, build tags and final executable identity are product inputs (P3, P29, P35–P36, Gaps). |
| Packaging | `activated` | `go install`, module/package exports, CLI executable, archives/checksums and consumer handoff are intended delivery surfaces (P1–P3, Gaps). |
| Release | `activated` | Version identity, candidate composition, channels, manifest, legal metadata, publication and support handoff are all unresolved (P1). In this report “Release” must be read carefully: current `Release` means reconcile-and-close occupancy, not publish/delete. Naming or ownership changes require product/API discussion. |
| Deployment | `not applicable` | No service deployment, environment promotion, rollout, rollback, or deployed topology is included; this is a local engine/library. |
| Configuration | `activated` | CLI flags, current directory, PATH, image/backend, task environment and hidden sample-sheet state affect behavior (P7, P11, P15, P21). |
| Observability | `activated` | JSON/JSONL streams, logs, Inspect, correlation, errors, tails, timing, cleanup diagnostics and support handoff are product surfaces (P3, P18–P19). |
| Migration | `activated` | Schema exact-match behavior, persisted snapshots, Resume/Release, CLI/library skew, and upgrade/downgrade are in scope (P1, P3, P5, P36). |
| Deprecation | `activated` | README explicitly marks public surface unsupported and Clean not shipped, but supplies no warning, replacement, window or removal contract (P1–P2). |
| Retirement | `activated` | Release leaves metadata/artifacts, Clean is not shipped, and support/data/artifact disposition is unspecified (P1 and Gaps). |
| Bioinformatics domain | `activated` | The product claims WGS/RNA-seq/DESeq2-like pipeline use; cohort/design/reference/output truth and provenance are material (P12–P13). Scientific truth sets were not executed. |

### Reconciliation

| Finding family | Origin | Checklist relation | Resolution |
|---|---|---|---|
| Release, support, legal, artifact, public-contract, and version-skew symptoms | `both` | Project Fit, Public API, Delivery, Compatibility; Build, Packaging, Release, Migration, Deprecation, Retirement overlays | Split into P1–P3 because release identity, public API ownership, and generated CLI/module compatibility have different earliest owners. |
| Process PID reuse, later-process recovery, and unknown occupancy | `both` | Architecture, Data Model, Correctness, Operations; Concurrency overlay | Combined in P4 around the absence of durable backend identity proof; test absence remains secondary under P35/P37. |
| Mixed control snapshots, crash durability, and inconsistent recovery readers | `both` | Architecture and Correctness; Concurrency, Observability, Migration overlays | Combined in P5 around the missing cross-file transaction/recovery owner. |
| Tree proof, multi-output publication, staged-byte identity, and false reuse | `both` | Data Model and Correctness; Bioinformatics domain overlay | Combined in P6; scientific interpretation remains separately owned by P13. |
| Docker host environment, executable lookup, mutable images, logs, and cleanup | `both` | Security, Dependencies, Configuration, Observability, Operations | Split into P7, P8, and P18 because host control, immutable dependency identity, and terminal evidence have distinct owners. |
| Path confinement, Release control-tree handling, redaction, permissions, and unbounded input | `both` | Correctness, Parameters, Testing; Security and Privacy overlays | Split into P9–P11; each represents a different root boundary. |
| DESeq2 design, file-shape success, and scientific provenance | `both` | Parameters, Correctness, Testing; Bioinformatics domain overlay | Split into P12 and P13 because invalid cohort admission differs from output/provenance truth. |
| CLI stream corruption, exit taxonomy, valued flags, and package cardinality | `both` | Parameters, Correctness, Usability, Compatibility; CLI specialist sources | Split into P14–P15 by protocol versus invocation grammar/resolution ownership. |
| Cancellation bounds, same-process Release, Docker lifecycle serialization, and retained terminal records | `both` | Correctness and Operations; Concurrency overlay | Retained as P16–P18 and P30 because completion bounds, workspace ownership, backend terminal effects, and retention are distinct roots. |
| Public API ownership, vocabulary, raw views, errors, hidden configuration, and extension seams | `both` | Public API, Abstraction, Vocabulary, Modularization, Correctness | Retained as P20–P23 and P39–P41; each can change independently and has a distinct consumer effect. |
| Scheduler concentration, duplicated policy/protocol, scale pressure, and missing package tests | `both` | Project Structure, Modularization, Reusability, Code Complexity, Testing | Retained as P27–P32 and P38; measured performance limits remain a Gap where no representative budget exists. |
| Fake fidelity, scientific oracles, synthetic crash tests, version skew, and concurrency evidence | `both` | Testing and Verification; Concurrency, Build, Packaging, Release, Migration overlays | Retained as P33–P37; passing root checks are reconciled as bounded strengths rather than closure of unexercised risks. |

## Checklist Review

The canonical checklist was traversed in source order by the isolated reviewers and reconciled here. The table records the aggregate disposition of each core category. `problem found` links only to the root Problems that carry that category's negative signs; other inspected signs were `no problem found` within the traced reach. Performance/Verification claims remain narrowed by the Gaps rather than being inferred from absent measurements. The aggregate Problems preserve the reconciled root findings; the temporary specialist reports held the full sign-level ledgers and exact source wording at review time.

| Lifecycle | Core category | Aggregate item result | Root evidence |
|---|---|---|---|
| Project | Project Fit | `problem found` | P1–P3 |
| Project | Affected Surfaces | `problem found` | P2–P6, P14, P18–P19, P28–P29 |
| Design and Development | Project Structure | `problem found` | P27–P30, P38 |
| Design and Development | Architecture | `problem found` | P4–P6, P17, P22, P27 |
| Design and Development | Design Pattern | `problem found` | P3, P14, P17, P22 |
| Design and Development | Abstraction | `problem found` | P22, P39–P41 |
| Design and Development | Data Model | `problem found` | P4–P6, P12–P13, P19, P23 |
| Design and Development | Public API | `problem found` | P2–P3, P20–P22, P39–P41 |
| Design and Development | Parameters | `problem found` | P11–P12, P15, P20–P21 |
| Design and Development | Modularization | `problem found` | P27–P30, P41 |
| Design and Development | Reusability | `problem found` | P6, P22, P28–P29 |
| Design and Development | Performance | `evidence missing` for representative budgets; source risks retained | P11, P30–P32 and Performance Gap |
| Design and Development | Optimization | `evidence missing` for representative benefits | P31–P32 and Performance Gap |
| Design and Development | Unintended Overengineering | `problem found` only where current exported/general surfaces lack a usable consumer | P22, P41 |
| Design and Development | Code Complexity | `problem found` | P27 |
| Design and Development | Readability | `problem found` | P14, P27–P29, P39–P41 |
| Design and Development | Vocabulary | `problem found` | P2, P19, P40; `Release` discussion in Improvements/Preserve |
| Design and Development | Naming Convention | `problem found` where names misstate effect or extension | P40–P41 |
| Design and Development | Docstring | `problem found` | P1–P4, P39–P41 |
| Design and Development | Correctness | `problem found` | P4–P6, P9, P14–P18, P23–P26 |
| Design and Development | Testing | `problem found` | P33–P38 |
| Design and Development | Verification | `evidence missing` beyond the serialized Linux hermetic checks | P35–P37 and Gaps |
| Design and Development | Delivery | `problem found` | P1–P3, P8, P35–P36 |
| Product | Usability | `problem found` | P2–P3, P14–P15, P19, P39–P41 |
| Product | Operations | `problem found` | P4–P5, P16–P19, P30 |
| Product | Compatibility | `problem found` | P1–P3, P5, P36, P39–P41 |

Activated specialist overlays were reconciled after the base pass: Security (P7–P11), Privacy (P10), Concurrency (P4–P5, P16–P18, P21, P30, P37), Dependencies (P3, P8, P11), Build/Packaging/Release (P1–P3, P29, P35–P36), Configuration (P7, P11, P15, P21), Observability (P3, P10, P18–P19), Migration/Deprecation/Retirement (P1–P3, P5, P36), and Bioinformatics domain (P6, P12–P13, P34). Accessibility, Localization, and Deployment were not applicable for the exact local CLI/library subject reasons recorded in Method.

## Problems

The materiality labels are descriptive evidence classifications, not scores. “Blocking relation” describes dependency or consequence, not a release gate. Every Problem includes an explicit discussion field as requested.

### P1 — Versioned release, support, legal, and retirement identity is absent

**Primary category:** Release  
**Expectation:** A distributable Go module/CLI needs one exact version/tag, supported platform/toolchain tuple, candidate/artifact identity, installation and consumer route, support window, migration/deprecation position, legal metadata, and retained-state/retirement disposition.  
**Observation:** The tracked tree has no release workflow, tag policy, `VERSION`, `CHANGELOG`, `LICENSE`/`NOTICE`, release manifest, archive/binary/checksum/signature record, CI gate, uninstall route, or support/deprecation/retirement policy. `README.md:19-23` supplies only `go install ...@<version>`, while `cmd/gobble/streams.go:61-75` can report `(devel)` or a dirty-derived BuildInfo version. Design Memory records no CI/release path and schema 0/1 as unsupported with no migration (`internal/engine/engine.go:46-48`).  
**Impact:** Maintainers cannot identify what bytes, module, CLI, assets, workspace schema, platform, or legal/support promise a consumer receives. An installed command or library update can strand a workspace without a documented predecessor/successor, rollback, retention, or removal position.  
**Evidence:** `README.md:7,19-23,43-65`; `go.mod:1-3`; `cmd/gobble/streams.go:61-75`; `internal/engine/engine.go:46-48`; frozen-tree inventory; `build-packaging-release.md` and `compatibility-delivery.md` prepared evidence.  
**Cause:** Product release identity and lifecycle ownership have not been bound to the source/module/CLI/workspace surfaces.  
**Uncertainty:** The project may intentionally be pre-stable or first-horizon; that is a policy choice not currently stated as one coherent consumer contract.  
**Severity (descriptive, not a score):** High.  
**Blocking relation (descriptive, not an approval gate):** Material to any product-level distribution claim.  
**Contract relation:** `in-contract (intended distributable engine/library question)`  
**Improvement direction — discussion required:** Discuss one exact release/support/legal/retirement vocabulary and ownership model for the module, CLI, assets, workspace state and artifacts. This includes deciding what is intentionally unsupported, not assuming that a release workflow or destructive cleanup is desired.  
**Example — discussion only, not a decision:** A support tuple might name a Go/toolchain/OS/architecture combination, an exact module/tag and CLI protocol, a schema window, and data/artifact disposition; this is an example of the decision shape, not a proposed policy.

### P2 — Public API support language and lifecycle ownership are contradictory

**Primary category:** Public API  
**Expectation:** A Go library consumer needs one authoritative statement of which exports, lifecycle states, schema views, errors, and recovery operations are supported and what they own.  
**Observation:** `README.md:7` says “The public surface is unsupported except these locked PathSpec concepts,” while `README.md:9-11,19-45` directs consumers to the broad Compose/Run/Inspect/Resume/Release path and `gobble.go:32-39` calls those types/functions shipped. `docs/gobble-draft.md:1-8,197` remains non-binding/deferred. Public `Graph`/`Plan`/`Inspect`/error/PlanOption shapes and executor/storage extension points are not accompanied by a version window.  
**Impact:** A consumer cannot tell whether a breaking API, raw view, error, lifecycle-name, or workspace change is a defect, allowed unsupported change, deprecation, or migration event. Support and ownership handoffs cannot be coordinated.  
**Evidence:** `README.md:7-11,19-45`; `gobble.go:1-40`; `docs/gobble-draft.md:1-8,197`; `graph.go:21-65`; `inspect.go:5-27`; `plan.go:37-63`; `public-api.md`; `compatibility-delivery.md`.  
**Cause:** Product-facing usage guidance, provisional support language, and implementation API were written without one accepted public-contract decision.  
**Uncertainty:** The current first horizon may intentionally keep APIs provisional, but the installable/consumer instructions then need to state that boundary precisely.  
**Severity (descriptive, not a score):** High.  
**Blocking relation (descriptive, not an approval gate):** Material to library distribution and independent consumer updates.  
**Contract relation:** `in-contract (Go library consumer question)`  
**Improvement direction — discussion required:** Discuss the narrow supported API, typed/raw view policy, error ownership, extension surface, compatibility window and lifecycle terminology before changing names or freezing interfaces.  
**Example — discussion only, not a decision:** A product might support only concrete pipeline construction and seven operations while keeping executors/storage internal, or it might expose a small provider contract; either is a product/API decision rather than an implementation instruction.

### P3 — Generated CLI and library have no module/API or machine-protocol compatibility envelope

**Primary category:** Compatibility  
**Expectation:** A CLI that compiles and runs a consumer package must bind the CLI, generated driver, Gobble module/API, control schema, output protocol and exit taxonomy to a known compatibility identity before mutation or execution.  
**Observation:** `cmd/gobble/driver.go:18-64,141-225` resolves the caller package, writes a temporary driver importing the caller and `github.com/HahyeonJeon/gobble`, builds it, then executes it. No module/version/capability handshake is performed. Success for run/resume/release can be only `{"op":verb}` (`driver.go:210-225`), while version output is separate and may be `(devel)`. The generated child uses ambient module/toolchain resolution.  
**Impact:** An installed CLI can compile against a different Gobble module or API, fail late with a generic build/invocation error, or emit a payload that an automation consumer cannot classify. Independent CLI/library updates have no supported compatibility or migration boundary.  
**Evidence:** `cmd/gobble/driver.go:18-45,106-130,141-225`; `cmd/gobble/main.go:26-55`; `cmd/gobble/streams.go:61-75`; `README.md:25-39`; `dependencies.md`; `compatibility-delivery.md`.  
**Cause:** CLI delivery identity and library/module identity are not joined in a versioned capability/protocol contract; the generated driver is treated as an internal detail despite being the integration boundary.  
**Uncertainty:** A consumer could manually pin a matching module, but no source or documentation requires or proves that pairing.  
**Severity (descriptive, not a score):** High.  
**Blocking relation (descriptive, not an approval gate):** Material to independently installed CLI/library use.  
**Contract relation:** `in-contract (CLI/library integration question)`  
**Improvement direction — discussion required:** Discuss a machine-readable envelope carrying CLI/module identity, control schema, operation/result profile, capability set, stable defect/state codes and exit mapping, with an explicit policy for generated-driver compilation.  
**Example — discussion only, not a decision:** An envelope could expose a run ID and schema/capability profile before Run/Resume/Release; its fields and compatibility window require product/API agreement.

### P4 — Local process recovery cannot prove identity after the owning engine disappears

**Primary category:** Operations  
**Expectation:** The documented Inspect → Release → Resume path needs a durable, backend-specific proof of whether a process is live, stopped, or unknown after the original engine process exits, while avoiding PID reuse or unrelated-process termination.  
**Observation:** `internal/engine/exec/process.go:15-19,89-94,97-171` stores live process handles and wait state only in `Process.live`; `RuntimeID` is a PID. A fresh executor cannot reconcile a persisted PID absent its old map. `internal/engine/release.go:95-153,271-311` therefore marks the task unknown and retains occupancy; `resume.go:81-85` rejects unknown units. Existing lifecycle tests deliberately assert refusal, not successful cross-engine recovery.  
**Impact:** If the CLI/engine dies while a local task is running or has completed, a later process cannot safely wait, stop, release, or resume it. The workspace can remain active/unknown without a supported completion path.  
**Evidence:** `internal/engine/exec/process.go:15-19,89-94,105-171`; `internal/engine/release.go:95-153,271-311`; `internal/engine/resume.go:81-85`; `internal/engine/lifecycle_test.go:81-113,277-315`; `architecture.md`, `state-recovery.md`.  
**Cause:** The persisted handle is a transient PID; proof and ownership are retained in the old process's in-memory adapter.  
**Uncertainty:** Real child-process behavior after a crash was excluded, but the loss of proof across a fresh adapter follows directly from the source and tests.  
**Severity (descriptive, not a score):** High.  
**Blocking relation (descriptive, not an approval gate):** Material to supported local recovery.  
**Contract relation:** `in-contract (accepted recovery loop)`  
**Improvement direction — discussion required:** Discuss a versioned backend recovery handoff with durable owner/session evidence, authenticated stop proof and an explicit unresolved disposition, preserving fail-closed occupancy and avoiding blind PID killing.  
**Example — discussion only, not a decision:** A process-group marker plus start-time/fingerprint and an owner token could be considered; platform authority and safety would need separate product/security discussion.

### P5 — Control snapshots have per-file atomicity but no cross-file transaction owner

**Primary category:** Correctness  
**Expectation:** `plan.json`, `tasks.json`, and `run.json` form one authoritative generation. A write interruption should leave the old complete generation, the new complete generation, or a recoverable transaction record; every recovery verb should reject or repair mixed state consistently.  
**Observation:** `internal/engine/run.go:1360-1383,1505-1529` and `release.go:110-151` write the three files sequentially using independent temp-file rename. There is no directory fsync, journal, generation pointer, or commit marker. `Inspect` rejects mismatched snapshots at `inspect.go:231-265`, but Resume and Release do not require the same coherence gate (`resume.go:33-85,190-225`; `release.go:17-110`).  
**Impact:** Crash, power loss or an ordinary write error can leave a mixed generation. Inspect safely refuses it, while Resume/Release can consume or overwrite mixed state. Occupancy, task status, artifacts and recovery instructions can diverge.  
**Evidence:** `internal/engine/run.go:1360-1433,1505-1529`; `internal/engine/release.go:110-151`; `internal/engine/inspect.go:231-265`; `internal/engine/resume.go:33-85`; `internal/engine/lifecycle_test.go:437-451`; `state-recovery.md`, `observability-operations.md`.  
**Cause:** File-level atomic replacement was used for a three-file consistency unit without a transaction/recovery owner.  
**Uncertainty:** Filesystem crash durability varies by host and was not injected; the cross-file exposure is direct from write order and readers.  
**Severity (descriptive, not a score):** High.  
**Blocking relation (descriptive, not an approval gate):** Material to crash-safe state handoff.  
**Contract relation:** `in-contract (authoritative workspace/recovery question)`  
**Improvement direction — discussion required:** Discuss a generation/journal/commit protocol, fsync assumptions, reader behavior, repair policy, migration of existing `.gobble` workspaces, and whether artifact publication is committed in the same or a deliberately related generation.  
**Example — discussion only, not a decision:** A generation directory with one durable pointer is one possible model; it is not a selected implementation or release requirement.

### P6 — Artifact proof, publication and provenance are split across mutable or incomplete evidence

**Primary category:** Data Model  
**Expectation:** Reuse and lineage must describe the bytes actually consumed and published; missing or altered proof must be a reuse miss; multi-output replacement and task state must have an unambiguous generation.  
**Observation:** `internal/engine/reuse.go:305-347` treats empty checksums as presence-only proof. Tree identity omits `.gobble-tree.json` (`tree.go:68-109`; `identity.go:167-203,231-267`) and can fall back to walking when manifest proof is absent (`tree.go:379-399`). `internal/engine/exec.go:149-175` replaces outputs sequentially without a multi-output rollback owner. Inputs are staged at `exec.go:17-63`, but post-run provenance hashes workspace bytes at `run.go:1247-1259`, so a source mutation during execution can produce fingerprints for bytes not consumed.  
**Impact:** A tree with a deleted/altered manifest or an output set with mixed old/new members can be reused or reported with false provenance. A task can consume old staged bytes and persist new workspace hashes, undermining scientific lineage and Resume decisions.  
**Evidence:** `internal/engine/reuse.go:305-347`; `internal/engine/tree.go:68-109,379-399`; `internal/engine/identity.go:167-203,231-267`; `internal/engine/exec.go:17-63,149-175`; `internal/engine/run.go:1223-1259`; `state-recovery.md`.  
**Cause:** Artifact proof is distributed among member presence, optional manifest, checksums and late workspace hashing rather than one immutable staged/published identity and commit owner.  
**Uncertainty:** No adversarial mutation or crash run was allowed; the source-level proof gap is direct.  
**Severity (descriptive, not a score):** High.  
**Blocking relation (descriptive, not an approval gate):** Material to reproducible scientific reuse and recovery.  
**Contract relation:** `in-contract (artifact/reuse/provenance question)`  
**Improvement direction — discussion required:** Discuss first-class manifest/content identity, immutable staged-byte fingerprints, multi-output generation publication, and the disposition of old incomplete workspaces. Preserve contained paths and fail-closed behavior while deciding this.  
**Example — discussion only, not a decision:** Recording a digest of the manifest and the staged input record alongside task state is an example of a proof relationship, not a concrete implementation plan.

### P7 — Task environment and host Docker execution share an ambient trust boundary

**Primary category:** Security  
**Expectation:** A task's declared environment should not silently control the host-side Docker client, executable selection, or privileged orchestration unless that boundary is explicit and trusted.  
**Observation:** `internal/engine/exec/docker.go:36-63` combines task environment with the host process environment before invoking Docker; `docker.go:157-175` passes task resources/images to the client. The Docker executable is selected as bare `docker` through ambient parent `PATH` (`docker.go`), while `process.go:176-202` uses declared PATH for executable resolution.  
**Impact:** A pipeline task can alter host-side Docker client behavior through variables such as `DOCKER_HOST`, registry settings, proxy settings or credential selection. A different executable on PATH can change the trust boundary. This is materially different from the stronger container isolation users may infer from the product model.  
**Evidence:** `internal/engine/exec/docker.go:36-63,157-175`; `internal/engine/exec/process.go:176-202`; `whole-product.md`, `security-privacy.md`.  
**Cause:** Task configuration and host orchestration configuration are not separated into owned, allow-listed environments and executable identity.  
**Uncertainty:** The target is a local user-owned engine, not a multi-tenant sandbox; the supported trust model is not declared and live Docker was excluded.  
**Severity (descriptive, not a score):** High for untrusted workflows; lower for a fully trusted local caller.  
**Blocking relation (descriptive, not an approval gate):** Material to any claim of isolation or safe execution of untrusted pipeline definitions.  
**Contract relation:** `in-contract (executor trust-boundary question)`  
**Improvement direction — discussion required:** Discuss the local trust model, separate host/client/task environments, executable provenance, and whether Docker is an isolation convenience or a security boundary. Do not infer multi-tenant safety from current container flags.

### P8 — Image and external executable inputs are mutable and not release-bound

**Primary category:** Dependencies  
**Expectation:** A reproducible pipeline run needs immutable or explicitly versioned identity for container images, host executables, Go toolchain/module resolution, and external fixture bytes.  
**Observation:** Asset image values are tags (`assets/fastp.go:9-10`); `ensureImage` can pull and only observes a digest after launch. The generated driver resolves the caller module/toolchain from ambient context. Fixture fetches use URLs and hashes but `FetchPin` writes before hash validation and has no global source/size/URL bound; cache path handling has a Name traversal concern (`assets/fetch.go:12-56`, `pin.go:25-50`).  
**Impact:** The same tag or ambient executable can produce different bytes across runs; a released source tree does not identify the image/data/toolchain inputs actually used. Large or malicious fixture input can consume resources or write outside intended cache semantics before rejection.  
**Evidence:** `assets/fastp.go:9-10`; `internal/engine/exec/docker.go:36-63`; `assets/fetch.go:12-56`; `assets/pin.go:25-50`; `dependencies.md`.  
**Cause:** Dependency identity is partly observed after execution rather than bound before it, and external asset acquisition is a live helper rather than a bounded release input.  
**Uncertainty:** Assets are documented as first-party external-source examples and may intentionally remain outside product release; no registry/network runtime was used.  
**Severity (descriptive, not a score):** Medium to high depending on whether assets are product inputs.  
**Blocking relation (descriptive, not an approval gate):** Material to reproducibility and any artifact claim that includes assets.  
**Contract relation:** `in-contract (dependency/reproducibility question)`  
**Improvement direction — discussion required:** Decide whether assets are product deliverables or explicitly non-product proofs; then discuss immutable image/data/toolchain identity, fetch bounds and provenance ownership.  
**Example — discussion only, not a decision:** A release input manifest could name image digests and fixture byte hashes; this is a possible evidence form, not a selected release design.

### P9 — Workspace confinement is preflight-only and Release omits control-tree containment

**Primary category:** Security  
**Expectation:** Every path effect, including control-state reads/writes and Release, should preserve workspace confinement across symlink/race conditions, not only validate a path string before use.  
**Observation:** `internal/engine/check.go:718-790` and related `containedRel` checks validate paths before later use, without descriptor/object identity. Release checks workspace containment but lacks the same control containment guard before joining `.gobble` paths. The final-leaf and ordinary symlink checks are useful but do not establish race-free confinement against concurrent replacement.  
**Impact:** A path or control subtree changed after preflight can redirect reads, writes, logs or release effects outside the intended workspace. This weakens claims of safe local operation in the presence of hostile or concurrently modified files.  
**Evidence:** `internal/engine/check.go:718-790`; `internal/engine/release.go:95-153`; `internal/engine/occupancy.go`; `correctness.md`, `security-privacy.md`.  
**Cause:** String/path validation and later filesystem effects have no shared descriptor or object-identity owner, and Release has a narrower containment path than other operations.  
**Uncertainty:** The local first horizon may assume trusted workspace ownership; hostile concurrent races were excluded.  
**Severity (descriptive, not a score):** High for hostile/shared workspaces; lower for an owner-only workspace.  
**Blocking relation (descriptive, not an approval gate):** Material to a security or untrusted-input claim.  
**Contract relation:** `in-contract (workspace safety question)`  
**Improvement direction — discussion required:** Discuss the intended workspace trust boundary and a uniform containment/object-identity contract for Run, Inspect, Resume and current Release semantics before expanding security claims.

### P10 — Protected-data persistence and redaction are not a complete policy

**Primary category:** Privacy  
**Expectation:** Sensitive samplesheet, command, parameter, script, stdout/stderr and diagnostic data need classification, redaction, access, retention and disposition rules.  
**Observation:** `jsonTaskState` persists Command, Script, Params, Stdout and Stderr (`internal/engine/run.go:173-194`); plan and Inspect expose command/script/params and log tails (`plan.go:129-171,320-347`; `inspect.go:328-368,439-495`). Only Env receives a digest/omission treatment. Attempt log files are created with ordinary workspace permissions (`exec/process.go:51-64`). The canary test checks Env absence but not secrets in the other fields.  
**Impact:** Credentials or private sample metadata placed in a command, parameter, script or output can persist in control files/logs and be returned through Inspect. Bounded tails limit size, not disclosure or retention.  
**Evidence:** `internal/engine/run.go:173-194`; `internal/engine/plan.go:129-171,320-347`; `internal/engine/inspect.go:328-368,439-495`; `internal/engine/exec/process.go:51-64`; `internal/engine/run_test.go:1055-1088`; `security-privacy.md`, `observability-operations.md`.  
**Cause:** Secret handling is a narrow Env serialization rule rather than a field-level protected-data contract across metadata, logs and views.  
**Uncertainty:** The source does not classify whether parameters/scripts may contain secrets; genomic/sample data makes the omission material for product distribution.  
**Severity (descriptive, not a score):** High for sensitive data.  
**Blocking relation (descriptive, not an approval gate):** Material to privacy/security and clinical/research deployment claims.  
**Contract relation:** `in-contract (bioinformatics data-handling question)`  
**Improvement direction — discussion required:** Discuss data classes, redaction boundaries, workspace permissions, retention/deletion, support bundles and incident handling. This must remain distinct from general operational log availability.

### P11 — External input and resource bounds are incomplete

**Primary category:** Security  
**Expectation:** Samplesheets, URLs, cache names and downloaded bytes should have bounded size, rows, fields, source, path and memory behavior before acceptance.  
**Observation:** `samplesheet.go:177-207` calls `csv.Reader.ReadAll` and then copies rows (`samplesheet.go:277-292`) without a total row/byte/field bound. Asset fetch writes URL content before final hash validation and has no source/size budget (`assets/fetch.go:12-56`). Cache path construction and pin names require careful containment.  
**Impact:** A large or adversarial samplesheet can exhaust memory; a remote fixture or cache name can consume unbounded resources or produce unsafe cache effects. This complicates support and reproducibility because accepted limits are not part of the API/configuration contract.  
**Evidence:** `samplesheet.go:177-207,277-292`; `assets/fetch.go:12-56`; `assets/pin.go:25-50`; `security-privacy.md`, `performance-resources.md`.  
**Cause:** Input validation establishes syntax and content relationships but not a product-level resource/source budget.  
**Uncertainty:** No representative sheet distribution or live download was examined.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to untrusted input and large bioinformatics-data claims.  
**Contract relation:** `in-contract (parameter/resource-bound question)`  
**Improvement direction — discussion required:** Discuss accepted sheet/source limits, bounded parsing, cache ownership and external-fetch authority, with errors that preserve the caller's action without exposing sensitive values.

### P12 — Domain design validation permits invalid or misleading DESeq2 cohorts

**Primary category:** Bioinformatics domain  
**Expectation:** A domain-facing API should represent and validate cohort labels, replication, reference levels and design requirements before execution; a syntactically valid pipeline must not silently request an invalid scientific design.  
**Observation:** `assets/deseq2.go` accepts arbitrary groups, while generated R code indexes `lev[2]`/`lev[1]`. `assets/rnaseq.go` accepts only exactly two labels and does not establish replication or design validity. No product-level scientific owner, accepted assay matrix or truth-set contract is tracked.  
**Impact:** A pipeline can build a command that fails late, drops/assumes levels, or produces a misleading analysis for a cohort with too few/incorrect labels or replication. Go tests can pass while domain validity is absent.  
**Evidence:** `assets/deseq2.go`; `assets/rnaseq.go`; `bioinformatics-domain.md`; README/assets proof scope.  
**Cause:** Domain parameters are strings/slices at the asset boundary rather than typed cohort/design/reference values with explicit scientific invariants.  
**Uncertainty:** The assets may be proof examples rather than supported scientific product features; the boundary is not decided.  
**Severity (descriptive, not a score):** High if the analysis assets are advertised as scientific product capabilities; otherwise a scope gap.  
**Blocking relation (descriptive, not an approval gate):** Material to any domain correctness claim.  
**Contract relation:** `in-contract (local bioinformatics engine question)`  
**Improvement direction — discussion required:** Decide which assays/designs are supported and which are examples, then discuss typed domain values, preflight validation and a scientific owner/truth-set policy.  
**Example — discussion only, not a decision:** An accepted design could require exactly two nonempty levels plus a declared replicate/reference policy; this illustrates a contract shape, not a chosen assay rule.

### P13 — Successful publication proves file shape, not scientific validity or complete provenance

**Primary category:** Bioinformatics domain  
**Expectation:** Product-level assays need semantic output oracles and provenance sufficient to establish sample/reference/design identity, not only regular files and nonempty bytes.  
**Observation:** Engine publication checks regular/nonempty outputs; WGS assay tests regular files without BAM/BGZF/BAI structure, report content, sample identity or read-count truth (`tests/local-e2e/wgs_e2e_assay_test.go:14-34`). Persisted identity omits pipeline/source revision, samplesheet/reference metadata and typed parameters; Inspect omits some environment/image/executable fingerprints. Tree scatter relies on path-only manifest evidence (`tree.go`, P6).  
**Impact:** A truncated, mislabeled, wrong-sample or scientifically invalid artifact can pass the product path and later be reused or reported as successful. Operators cannot reconstruct the exact scientific input/design identity from the persisted run.  
**Evidence:** `tests/local-e2e/wgs_e2e_assay_test.go:14-34`; `tests/local-e2e/wgs_e2e_thin_test.go:290-303`; `internal/engine/run.go:163-225`; `internal/engine/inspect.go:304-386`; `bioinformatics-domain.md`, `testing.md`.  
**Cause:** Generic filesystem publication and latest-attempt identity are used as the product oracle; scientific truth/provenance is not a bound contract.  
**Uncertainty:** Scientific output execution and external truth sets were explicitly excluded.  
**Severity (descriptive, not a score):** High for scientific product claims; evidence gap otherwise.  
**Blocking relation (descriptive, not an approval gate):** Material to bioinformatics distribution claims.  
**Contract relation:** `in-contract (scientific output question)`  
**Improvement direction — discussion required:** Discuss accepted assay/reference proof families, semantic output validators, source/design/reference provenance and truth-set ownership before describing file existence as product validity.  
**Example — discussion only, not a decision:** A BAM/BAI structure and sample/read-count truth set could be one assay family; it is not a selected acceptance criterion here.

### P14 — Generated user-code output and exit behavior can violate the CLI protocol

**Primary category:** Correctness  
**Expectation:** A machine-readable CLI must own stdout/stderr framing and return a documented exit taxonomy regardless of user package init/Pipeline output, child signal or raw exit behavior.  
**Observation:** `cmd/gobble/driver.go:141-225` wires generated child stdout/stderr directly to parent streams. User package initialization or pipeline construction can print bytes before the JSON response. `driverWaitCode` can return raw child codes or `128+signal`, escaping the documented 0/1/2 taxonomy.  
**Impact:** Automation can receive malformed JSON or an unclassifiable exit code for ordinary user-package behavior, making recovery and support decisions unreliable.  
**Evidence:** `cmd/gobble/driver.go:141-225`; `cmd/gobble/main.go:49-55`; `README.md:25-41`; `cli-config.md`.  
**Cause:** The generated driver and CLI protocol share process streams with arbitrary consumer code and forward child status without a final protocol-owned mapping.  
**Uncertainty:** Direct CLI tests cover ordinary paths, panic and signal cases, but user init output and all child exit variants are not an installed-consumer proof.  
**Severity (descriptive, not a score):** Medium to high for machine consumers.  
**Blocking relation (descriptive, not an approval gate):** Material to automation using the CLI protocol.  
**Contract relation:** `in-contract (CLI machine-interface question)`  
**Improvement direction — discussion required:** Discuss stream ownership, child diagnostic routing, protocol framing and a complete exit mapping before expanding automation claims.

### P15 — CLI argument and import resolution can silently accept the wrong invocation

**Primary category:** Parameters  
**Expectation:** Missing valued flags and package operands should fail at the CLI boundary with actionable diagnostics; no valid-looking invocation should silently change its meaning.  
**Observation:** `cmd/gobble/args.go`/`collectArgs` accepts the next token as a flag value even when it is another option. `resolveImport` uses the first line of `go list` output, silently truncating a multi-package operand.  
**Impact:** `run --workspace --sample foo` or a package pattern yielding multiple imports can execute with the wrong workspace/sample/package or report a later generic error. The automation contract is not self-protecting.  
**Evidence:** `cmd/gobble/args.go`; `cmd/gobble/driver.go` import resolution; `cli-config.md`.  
**Cause:** Parser and package resolver assume token/output cardinality without validating missing values or multiple package results.  
**Uncertainty:** The exact user-facing error path depends on the caller package and command; source behavior is direct.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to reliable CLI use.  
**Contract relation:** `in-contract (CLI parameter question)`  
**Improvement direction — discussion required:** Discuss strict valued-flag parsing, explicit package cardinality and task-shaped help examples; do not rely on downstream Go errors to define the CLI contract.

### P16 — Reconciliation and release polling lack an aggregate stop bound

**Primary category:** Concurrency  
**Expectation:** Cancellation and recovery should have an end-to-end deadline or accepted process lifetime, not only per-backend-call timeouts.  
**Observation:** `internal/engine/run.go:520-559` and `release.go:271-310` cancel then loop until Poll says not running. Each `boundedPoll` call has a 30-second context (`run.go:604-609,1483-1503`), but no enclosing deadline limits repeated calls; polling sleeps 20 ms.  
**Impact:** A backend that remains running or repeatedly exceeds each per-call bound can prevent Release/Resume from returning, retaining occupancy, goroutines and resources indefinitely.  
**Evidence:** `internal/engine/run.go:520-609,978-993,1483-1503`; `internal/engine/release.go:271-310`; `state-recovery.md`, `concurrency.md`, `performance-resources.md`.  
**Cause:** Per-call bounding was treated as aggregate lifecycle boundedness.  
**Uncertainty:** Real backend failure timing was excluded; the unbounded control flow is direct.  
**Severity (descriptive, not a score):** High for non-cooperative backends.  
**Blocking relation (descriptive, not an approval gate):** Material to recovery availability.  
**Contract relation:** `in-contract (cancellation/recovery question)`  
**Improvement direction — discussion required:** Discuss an aggregate cancellation/deadline state machine, explicit unresolved ownership and resource disposition, and bounded retry/backoff policy while preserving unknown occupancy.

### P17 — Same-process public Release can bypass scheduler ownership

**Primary category:** Concurrency  
**Expectation:** Public lifecycle calls must serialize or explicitly coordinate ownership of scheduler, executor, lease, control files and artifacts.  
**Observation:** Run owns a scheduler/executor and goroutines, while Release can use a held lease and a newly created executor, mutate files and close occupancy without a public session/operation lock tying it to an in-flight Run. Docker Poll/Cancel/cleanup also have distinct lock scopes and late-submit registry behavior.  
**Impact:** Concurrent Run and Release/Resume calls can race on task state, close or reconcile resources still owned by scheduler goroutines, or produce mixed control/artifact state even when each individual operation has local synchronization.  
**Evidence:** `internal/engine/run.go:357-518`; `internal/engine/release.go:95-153`; `internal/engine/exec/docker.go:26-35,78-149`; `concurrency.md`.  
**Cause:** Lifecycle ownership is represented by a filesystem lease and local scheduler state, not by a public operation/session owner that serializes all mutation.  
**Uncertainty:** No parallel public-call runtime was executed; the ownership split is visible statically.  
**Severity (descriptive, not a score):** Medium to high.  
**Blocking relation (descriptive, not an approval gate):** Material to library callers that may issue concurrent operations.  
**Contract relation:** `in-contract (Go lifecycle/concurrency question)`  
**Improvement direction — discussion required:** Discuss operation serialization/session ownership and what concurrent public calls are supported; keep filesystem lease safety separate from in-process scheduler coordination.

### P18 — Docker log capture and cleanup failures are discarded

**Primary category:** Observability  
**Expectation:** A completed task must distinguish successful diagnostics and cleanup from degraded capture/removal, with an owned record that operators can inspect.  
**Observation:** `internal/engine/exec/docker.go:78-107,275-301` ignores `docker logs` and `docker rm -f` errors and stores a normal report. `exec.Report` (`exec.go:54-64`) has no capture or cleanup outcome.  
**Impact:** Missing logs look like ordinary no-output completion, and stopped containers can remain after reported success. Operators lose failure-localization evidence and external resources may accumulate without an action signal.  
**Evidence:** `internal/engine/exec/docker.go:78-107,275-301`; `internal/engine/exec/exec.go:54-64`; `architecture.md`, `observability-operations.md`, `performance-resources.md`.  
**Cause:** Log transport and container removal are best-effort side effects whose return values are not part of the executor terminal report.  
**Uncertainty:** Live daemon behavior was excluded; ignored error paths are unambiguous.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to operational diagnosis and cleanup.  
**Contract relation:** `in-contract (executor/observability question)`  
**Improvement direction — discussion required:** Discuss capture/cleanup outcome fields, retry/retention policy and operator-visible degraded completion, preserving bounded log tails and safe cleanup ownership.

### P19 — Run identity and inspection history are insufficient for support correlation

**Primary category:** Data Model  
**Expectation:** Every run/attempt/log/error/recovery action should have a stable correlation identity and an inspection path for prior attempts/events when those distinctions affect action.  
**Observation:** `runID` returns `Document.Name` or literal `"run"` (`internal/engine/run.go:256-298`); successful CLI run/resume output omits the run ID. Inspect selects latest attempts across `run`, `instances`, `errors`, `logs`, `timing`, `dag`, `lineage`, `remaining`, `reuse` (`inspect.go:10-21,61-88`) and exposes no event/prior-attempt view even though `tasks.json` retains history.  
**Impact:** Repeated runs cannot be reliably joined to logs or support actions, and retries/resumes can hide chronology needed to distinguish a new failure, reused artifact, or recovery event.  
**Evidence:** `internal/engine/run.go:163-170,250-298`; `internal/engine/resume.go:234-247`; `internal/engine/inspect.go:10-21,53-88`; `internal/engine/run.go:1401-1432`; `observability-operations.md`.  
**Cause:** A display/name value is used as run identity and the inspection contract intentionally projects latest state without a public event/history relation.  
**Uncertainty:** A first-horizon design may intentionally omit event streaming; stable correlation is still needed for a distributable CLI/library handoff.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to supportability and machine handoff.  
**Contract relation:** `in-contract (observability/support question)`  
**Improvement direction — discussion required:** Discuss stable run/attempt identifiers, completion envelopes and bounded history/event visibility, with retention/privacy ownership decided separately.

### P20 — Builder values are shallowly retained across the public ownership boundary

**Primary category:** Public API  
**Expectation:** A reusable Go builder must document or enforce whether callers may mutate or reuse reference-bearing inputs after registration.  
**Observation:** `AddTask`/`newTask` store slices and maps directly (`pipeline.go:219-223,446-449`), and `AddInputGroup` stores the caller's Group slice (`pipeline.go:197-202`). Defensive copies occur only later in Compose (`compose.go:84-132`).  
**Impact:** Mutating Command, Inputs, Outputs, Params, Env or Group after registration can change a later Compose result; concurrent mutation is an unsound race boundary.  
**Evidence:** `spec.go:5-20`; `pipeline.go:197-202,219-223,446-449`; `compose.go:84-132`; `public-api.md`.  
**Cause:** A shallow value copy was treated as sufficient until the Compose boundary without a clearly documented ownership interval.  
**Uncertainty:** No runtime reproduction was needed for the static aliasing observation.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to safe library composition and concurrent callers.  
**Contract relation:** `in-contract (Go ownership question)`  
**Improvement direction — discussion required:** Discuss copy-on-registration versus an explicit immutable/transfer ownership contract, and align external-package tests with the selected boundary.

### P21 — Samplesheet configuration and test seams use hidden goroutine/global state

**Primary category:** Parameters  
**Expectation:** Reusable library configuration and test dependencies should be explicit, scoped and reproducible, not keyed to runtime goroutine identity or mutable package globals.  
**Observation:** `SetSampleSheetPath` stores a string in a `sync.Map` keyed by parsed `runtime.Stack` goroutine ID; `LoadSampleSheet` takes no path argument (`samplesheet.go:53,75-122`). Production/test seams such as `runExecutor`, `readHostCapacity` and Docker CLI hooks are package-level globals restored serially.  
**Impact:** Behavior changes by calling goroutine and current directory; stale entries can survive lifecycle assumptions, and parallel callers/tests can interfere. Reproduction and support require hidden knowledge.  
**Evidence:** `samplesheet.go:53,75-122`; `occupancy_test.go:124-168`; `internal/engine/exec_hook_test.go`; `code-quality-refactoring.md`, `public-api.md`.  
**Cause:** A per-call CLI isolation need was implemented as runtime-stack state instead of an explicit scoped configuration/dependency object.  
**Uncertainty:** No dynamic stale-entry/parallel test was run; the hidden ownership is directly visible.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to concurrent library use and reliable tests.  
**Contract relation:** `in-contract (configuration/concurrency question)`  
**Improvement direction — discussion required:** Discuss explicit sample-sheet/configuration inputs and dependency injection seams, with a supported concurrent-call policy before exposing the library broadly.

### P22 — Backend-independent pipeline claims leak Docker mechanics and have no public executor/artifact seam

**Primary category:** Abstraction  
**Expectation:** A reusable backend-independent core should keep executor, artifact, storage, log and reconciliation mechanics behind a stable owned boundary, or state a deliberately local-only model.  
**Observation:** Public `TaskSpec` uses empty/nonempty Image for local process/Docker, arbitrary Backend strings and Docker `--memory` syntax (`spec.go:5-20,87-95`). `exec.Executor` accepts an absolute local isolate and reports runtime/exit fields while the engine owns staging, output validation, publication and log paths (`internal/engine/exec/exec.go:35-81`; `run.go:896-1024`; `exec.go:17-175`). Nonlocal backends are rejected (`check.go:346-357`); no public executor/storage/options seam exists.  
**Impact:** Adding Podman, Slurm, remote storage or a different artifact backend would require changing caller-visible models and engine lifecycle semantics. Consumers are coupled to Docker syntax despite architecture language promising a replaceable seam.  
**Evidence:** `spec.go:5-20,87-95`; `internal/engine/exec/exec.go:35-81`; `internal/engine/run.go:896-1024`; `internal/engine/check.go:346-357`; `architecture.md`, `public-api.md`.  
**Cause:** The first local implementation defined the public TaskSpec instead of isolating backend/artifact ownership.  
**Uncertainty:** Docker/process may intentionally be the permanent first-horizon boundary; that product decision is unresolved.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to claims of reusable backend-independent library architecture.  
**Contract relation:** `in-contract (architecture/public API question)`  
**Improvement direction — discussion required:** Discuss whether to promise a narrow local backend or define a small provider boundary for execution, artifacts, state and reconciliation; avoid freezing speculative backend interfaces.

### P23 — Map iteration makes defects and persisted history nondeterministic

**Primary category:** Correctness  
**Expectation:** Identical inputs should produce deterministic defect ordering, task/history ordering and machine-readable views unless variation is explicitly part of the contract.  
**Observation:** Environment defects are appended while ranging maps (`compose_check.go:342-365`; `internal/engine/validate.go:121-145`). Resume ranges `byIdent` and appends history in map order (`resume.go:304-359`); persisted history and Inspect expose that order (`run.go:1401-1432`; `inspect.go:56-88`).  
**Impact:** Golden files, agents, support diffs and downstream consumers cannot rely on stable order; repeated resume/inspect output can differ without a semantic input change.  
**Evidence:** Exact paths above; `public-api.md` and testing evidence.  
**Cause:** Unspecified Go map traversal is used directly at output boundaries without deterministic sorting.  
**Uncertainty:** The source behavior is deterministic-risk evidence; serialized repetition passed does not prove byte-order stability for all cases.  
**Severity (descriptive, not a score):** Low to medium.  
**Blocking relation (descriptive, not an approval gate):** Material to stable machine/readable handoff.  
**Contract relation:** `in-contract (determinism/correctness question)`  
**Improvement direction — discussion required:** Discuss which outputs promise ordering and place deterministic ordering at those boundaries, with golden/repeated tests for the chosen contract.

### P24 — Static scatter can skip missing source members before input validation

**Primary category:** Correctness  
**Expectation:** Every static scatter member must validate required source inputs before occupying or producing a terminal result; an empty/missing member must not silently bypass input checks.  
**Observation:** Static scatter expansion can create an empty member whose IO path bypasses `checkInputs`; the scheduler occupies the member and later reaches a missing-input/not-ready path.  
**Impact:** A manifest with missing source members can enter scheduling or remain in an ambiguous state rather than fail with a clear preflight defect.  
**Evidence:** `internal/engine/tree.go` scatter/manifest paths; `internal/engine/run.go` readiness/input paths; `correctness.md`.  
**Cause:** Static topology expansion trusts member path structure before applying the ordinary task input preflight.  
**Uncertainty:** Exact dynamic state depends on manifest/workspace contents; no adversarial run was executed.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to reliable large-tree bioinformatics workflows.  
**Contract relation:** `in-contract (scatter/input correctness question)`  
**Improvement direction — discussion required:** Discuss one validation order and an explicit missing-member state before admission, preserving path containment and manifest safety.

### P25 — Resume can reuse a skipped state after its `When` condition changes

**Primary category:** Correctness  
**Expectation:** Resume reuse must re-evaluate conditions that can change with current inputs/state; a prior skip must not permanently suppress a now-eligible task.  
**Observation:** Resume reuses a skipped task state without re-evaluating the current `When` condition after its controlling state changes.  
**Impact:** A pipeline can remain incomplete or silently omit work that became eligible after a resume, while the persisted task history suggests the prior skip was authoritative.  
**Evidence:** `internal/engine/resume.go:304-359`; `internal/engine/run.go` `when` evaluation around `2277-2347`; `correctness.md`.  
**Cause:** Reuse classification treats prior skipped state as terminal without coupling it to current condition inputs.  
**Uncertainty:** The source path is static; no changing-condition resume case was executed.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to conditional workflow semantics.  
**Contract relation:** `in-contract (Resume correctness question)`  
**Improvement direction — discussion required:** Discuss which conditional states are reusable and what input fingerprint must invalidate a skip, then encode that contract in recovery evidence.

### P26 — Empty successful Run and Release status disagree

**Primary category:** Correctness  
**Expectation:** A zero-task/empty accepted run should have one consistent terminal status across Run, persisted state, Inspect and Release.  
**Observation:** Run can return success for a zero-task document, while `runStatusFromTasks` in Release returns failed for `len(tasks)==0`.  
**Impact:** A caller can observe success before Release and failure after Release for the same workspace, making automation and support classification inconsistent.  
**Evidence:** `internal/engine/run.go` zero-task completion path; `internal/engine/release.go` `runStatusFromTasks`; `correctness.md`.  
**Cause:** Empty-run status rules are implemented separately in Run and Release without one lifecycle status owner.  
**Uncertainty:** The desired empty-run policy is a product decision; the inconsistency is source-level.  
**Severity (descriptive, not a score):** Low to medium.  
**Blocking relation (descriptive, not an approval gate):** Material to stable lifecycle API semantics.  
**Contract relation:** `in-contract (Run/Release status question)`  
**Improvement direction — discussion required:** Discuss whether an empty graph is success, no-op or invalid, then make the same vocabulary and state rule apply across all verbs.

### P27 — The scheduler is a 2,347-line cross-lifecycle state owner

**Primary category:** Modularization  
**Expectation:** Independent lifecycle responsibilities should have ownership and failure boundaries that can be reasoned about and tested separately.  
**Observation:** `sched` in `internal/engine/run.go:108-122` owns document/task/history/reuse maps, condition state, launch/report loops, persistence errors, resources, executor, staging/publication, scatter/gather expansion and `when` evaluation through line 2347.  
**Impact:** A change to persistence, artifacts, backend behavior, dynamic topology or conditional evaluation crosses one mutable owner and its effects. Focused tests and recovery ownership become difficult; failures in one concern can spread through the same state.  
**Evidence:** `internal/engine/run.go:108-122,357-518,896-1024,1360-1530,1681-1904,2277-2347`; `code-quality-refactoring.md`, `architecture.md`.  
**Cause:** End-to-end first-horizon implementation remained in one scheduler unit after the accepted architecture named separate validator/planner/scheduler/executor/state/artifact responsibilities.  
**Uncertainty:** The scope may intentionally trade modularity for simplicity; this is a changeability/lifecycle risk, not an inferred author motive.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to sustained library/product evolution.  
**Contract relation:** `in-contract (architecture/changeability question)`  
**Improvement direction — discussion required:** Discuss responsibility ownership and failure boundaries for readiness/expansion, state persistence, artifact effects and executor lifecycle, guided by observed change coupling rather than a prescribed decomposition.

### P28 — Compose and Graph validation/resolution duplicate policy

**Primary category:** Reusability  
**Expectation:** One domain rule should have one proportionate owner when Compose and Graph validation must remain consistent.  
**Observation:** `compose_check.go` has separate `composeChecker` and `graphChecker` policy paths and resolver logic for overlapping graph/task/environment rules.  
**Impact:** A validation or resolution rule can drift between pre-Compose and post-Compose paths, producing different defects or acceptance for the same pipeline.  
**Evidence:** `compose_check.go`; `graph.go`; `internal/engine/validate.go`; `code-quality-refactoring.md`.  
**Cause:** The same policy is represented in parallel boundary-specific implementations without a single immutable rule source.  
**Uncertainty:** No contradictory output was observed in the supplied evidence; duplication itself is direct and future drift risk is the concern.  
**Severity (descriptive, not a score):** Low to medium.  
**Blocking relation (descriptive, not an approval gate):** Material to maintainability and validation predictability.  
**Contract relation:** `in-contract (validation/reuse question)`  
**Improvement direction — discussion required:** Discuss one internal policy source or an explicit separation of pre/post-Compose responsibilities, with tests that prove equivalence where intended.

### P29 — Host CLI and generated driver duplicate the protocol boundary without provenance ownership

**Primary category:** Project Structure  
**Expectation:** Generated code needs a clear source, owner, regeneration path, protocol version and artifact provenance.  
**Observation:** Host CLI stream/exit handling and runtime-generated driver logic both encode operation framing, child execution and result handling (`cmd/gobble/main.go`, `driver.go`, `streams.go`). Assets also carry duplicate `Parent`/`ModuleParent` proof interfaces (`assets/parent.go:5-29`) without a tracked generator/manifest owner.  
**Impact:** A protocol change can require coordinated edits in duplicate locations; generated source, asset inputs and final bytes cannot be traced to one immutable release input.  
**Evidence:** `cmd/gobble/main.go`; `cmd/gobble/driver.go:141-225`; `cmd/gobble/streams.go`; `assets/parent.go:5-29`; `code-quality-refactoring.md`, `build-packaging-release.md`.  
**Cause:** Runtime generation and host protocol code evolved as parallel owners without a versioned generated-source boundary.  
**Uncertainty:** Assets may be explicitly proof-only, which would narrow the delivery claim but not create a product artifact.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to generated CLI and asset delivery.  
**Contract relation:** `in-contract (build/delivery structure question)`  
**Improvement direction — discussion required:** Discuss protocol ownership and whether generated assets are release inputs; retain one source/manifest identity if they remain in product scope.

### P30 — Completed executor records and external resources have no bounded retirement policy

**Primary category:** Operations  
**Expectation:** Process/Docker completion records, temporary files, isolates, logs and stopped containers need an explicit lifetime owner and bounded retention where long-running workflows are supported.  
**Observation:** Process `live` and Docker `done` maps retain every terminal task entry (`internal/engine/exec/process.go:15-19,71-115`; `docker.go:26-35,78-149`) with no eviction. Docker removal errors are discarded (P18); accepted Design Memory says `Isolate-keep` without a size/time cleanup contract.  
**Impact:** Long-lived executor instances can retain heap references for every task, while failed cleanup can accumulate external Docker resources. Operators cannot tell whether retention is intended recovery evidence or residue.  
**Evidence:** Exact paths above; `run-local.md:77`; `performance-resources.md`, `state-recovery.md`.  
**Cause:** Terminal-state caches and workspace retention lack a product-owned lifetime/cleanup contract.  
**Uncertainty:** No heap/disk profile or live cleanup behavior was measured.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to long-running/local resource claims.  
**Contract relation:** `in-contract (resource-lifecycle question)`  
**Improvement direction — discussion required:** Discuss retention owner, size/time limits, cleanup trigger, diagnostic preservation and safe orphan disposition before introducing deletion or eviction. Note that current `Release` is reconcile-and-close occupancy, not publish/delete.

### P31 — Ready-candidate traversal and full control snapshots scale with total state

**Primary category:** Performance  
**Expectation:** A local engine should have a known workload envelope and avoid unbounded repeated whole-state traversal/serialization relative to that envelope.  
**Observation:** `readyCandidates` scans all document tasks and, inside each, scans all task state and sorts member lists (`internal/engine/run.go:676-705`), while `taskByIdent/taskByID` linearly scan state (`run.go:715-737,859-865`). Each launch/report persists full plan/tasks/run snapshots, rebuilding history and marshaling the full document (`run.go:887-893,1178-1245,1360-1432`).  
**Impact:** Large DAGs and long histories can consume increasing scheduler CPU, allocations and disk I/O per transition. No accepted size/latency budget or representative baseline says when this becomes material.  
**Evidence:** Exact paths above; `performance-resources.md`.  
**Cause:** Readiness and persistence derive full state on each event rather than maintaining bounded indexes/deltas where recovery permits.  
**Uncertainty:** No performance workload was run; this is structural cost evidence, not a measured SLA failure.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to scale claims, not to the small first-horizon proof.  
**Contract relation:** `in-contract (performance/resource question)`  
**Improvement direction — discussion required:** Discuss a representative DAG/history/filesystem workload and recovery atomicity before choosing indexes, snapshot cadence or delta state; do not infer a benchmark from asymptotic shape alone.

### P32 — Copy-only large-file staging and fixed polling amplify I/O and backend probes

**Primary category:** Performance  
**Expectation:** Bioinformatics-scale files and long-running tasks need an accepted I/O/probe budget, while preserving containment and publication safety.  
**Observation:** `internal/engine/exec/publish.go:28-75,104-146` copies instead of using accepted hardlink/process-symlink fast paths; engine stages and publishes each file and hashes inputs/outputs (`exec.go:44-60,90-147`; `tree.go:122-217,267-320`; `run.go:1247-1259`). Running tasks poll every 20 ms (`run.go:47,978-993`), and Docker Poll invokes an external inspect command.  
**Impact:** FASTQ/reference trees can incur multiple full byte copies and hashes, while long-running caps can multiply backend probes and process overhead. No throughput/disk budget is established.  
**Evidence:** Exact paths above; accepted `run-local.md:76-77`; `performance-resources.md`, `state-recovery.md`.  
**Cause:** Simplicity and isolation are implemented as copy-only staging and fixed polling without a measured adaptive policy.  
**Uncertainty:** Filesystem/device boundaries and workload size determine actual savings; no performance workload was run.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to large-data product claims.  
**Contract relation:** `in-contract (performance/resource question)`  
**Improvement direction — discussion required:** Discuss representative data/probe workloads and aliasing semantics before considering hardlinks, symlinks, adaptive polling or bounded backpressure; preserve containment and copy fallback.

### P33 — Scheduler fakes can bypass the production output oracle

**Primary category:** Testing  
**Expectation:** A scheduler test double should exercise or pair with the real staging/inspect/publish boundary for any claim about outputs, resources, persistence or cancellation.  
**Observation:** `internal/engine/exec_hook_test.go:20-104` lets `fnExec` return `Published=true`; `run.go:997-1018` then skips inspect/publish. Scheduler tests can therefore pass without real input staging, process/container start, output validation, log capture or publication.  
**Impact:** Scheduler/resource/persistence tests can pass while production artifact effects are broken. The fake result is stronger than the exercised production boundary.  
**Evidence:** `internal/engine/exec_hook_test.go:20-104`; `internal/engine/exec/exec.go:54-64`; `internal/engine/run_test.go:239-279,316-327,436-441,554-618`; `testing.md`.  
**Cause:** A fast scheduler seam has an escape hatch with no paired bridge assay proving equivalence to a real adapter that leaves Published false.  
**Uncertainty:** Process/Docker adapter tests exist; the missing bridge is the evidence gap.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to interpreting scheduler test coverage.  
**Contract relation:** `in-contract (testing oracle question)`  
**Improvement direction — discussion required:** Discuss which tests are scheduler-only and which claim artifact behavior, then preserve a real-adapter bridge oracle rather than treating `Published=true` as production equivalence.

### P34 — Product WGS assay checks file existence instead of domain output truth

**Primary category:** Testing  
**Expectation:** A product assay should reject materially wrong bioinformatics outputs, not merely empty/nonempty regular paths.  
**Observation:** `tests/local-e2e/wgs_e2e_assay_test.go:14-34` checks three regular files. The thin-slice helper has stronger magic checks but the product assay does not call it (`wgs_e2e_thin_test.go:290-303`).  
**Impact:** Truncated, mislabeled, wrong-sample or stand-in outputs can satisfy the WGS success/release/resume path, making a local end-to-end pass misleading as scientific evidence.  
**Evidence:** Exact test paths; `testing.md`, `bioinformatics-domain.md`.  
**Cause:** The product assay prioritizes a cheap lifecycle run over a domain-specific output oracle.  
**Uncertainty:** Runtime output was excluded; no bad BAM was observed.  
**Severity (descriptive, not a score):** High for scientific claims; otherwise an evidence gap.  
**Blocking relation (descriptive, not an approval gate):** Material to domain/product distribution evidence.  
**Contract relation:** `in-contract (scientific verification question)`  
**Improvement direction — discussion required:** Discuss named scientific proof families and what the product assay must establish, without treating any example validator as an accepted decision.

### P35 — Crash recovery is synthetic and live/consumer/release checks are not enforced in the project

**Primary category:** Verification  
**Expectation:** A product handoff needs exact evidence for owner termination/restart, real process/Docker behavior, installed/external consumers, final artifact bytes and release gates.  
**Observation:** Crash tests rewrite `run.json`/`tasks.json` with dead PIDs or running state rather than terminating the owner at the relevant boundary (`session_proof_test.go:12-62`; `release_test.go:76-107`; `resume_test.go:772-845`). Signal tests stop at cancellation/release and do not restart/resume a real interrupted binary. Live tests are `//go:build live`; the tracked tree has no project-owned CI/release workflow, final artifact/checksum, clean external module consumer or install smoke.  
**Impact:** A passing root suite cannot establish descendant/container cleanup, real restart recovery, live assets, installed CLI/library behavior, artifact identity or release reproducibility. Live-only regressions can remain invisible to the repository's normal handoff.  
**Evidence:** `tests/local-e2e/cli_live_test.go:13-104`; `cmd/gobble/occupy_test.go:136-204`; `README.md:47-65`; frozen-tree inventory; `testing.md`, `build-packaging-release.md`.  
**Cause:** Commands and build tags are documented, but execution/evidence retention and delivery gates are delegated to an unstated developer environment.  
**Uncertainty:** External CI or release systems may exist outside this frozen repository and cannot support a claim about this tree.  
**Severity (descriptive, not a score):** High for distribution evidence.  
**Blocking relation (descriptive, not an approval gate):** Material to any claim beyond hermetic source tests.  
**Contract relation:** `in-contract (verification/delivery question)`  
**Improvement direction — discussion required:** Discuss which live/crash/install/consumer/artifact checks belong to the project-owned evidence contract and how exact identity/results are retained; this report does not authorize running them.

### P36 — Version-skew and stored-state matrix is incomplete

**Primary category:** Compatibility  
**Expectation:** With exact schema 2 and no migration, old, current, future, missing, malformed and mixed `run`/`plan`/`tasks` states should have an explicit non-mutating behavior across Run/Inspect/Release/Resume API and CLI entry points, or a migration policy should exist.  
**Observation:** `internal/engine/engine.go:46-48` rejects schema 0/1 and says no migration. Existing tests cover selected historical documents but not future schema, absent/mismatched files, unknown field/type corruption, or a complete operation matrix (`check_test.go:140-168`; `inspect_test.go:50-106`; `release_test.go:102-120`; `resume_test.go:32-55`).  
**Impact:** A workspace from a newer/partial writer can fail in an unclassified or untested mutation path; consumers have no supported upgrade/downgrade window or backup/disposition rule.  
**Evidence:** Exact paths above; `testing.md`, `compatibility-delivery.md`, `state-recovery.md`.  
**Cause:** Historical schema refusal was tested as a narrow case without a complete transition corpus or release policy.  
**Uncertainty:** No runtime matrix was executed in the specialist reports; root test evidence does not imply this corpus exists.  
**Severity (descriptive, not a score):** Medium to high.  
**Blocking relation (descriptive, not an approval gate):** Material to workspace compatibility.  
**Contract relation:** `in-contract (migration/compatibility question)`  
**Improvement direction — discussion required:** Discuss supported schema/version windows, fail-closed diagnostics, backup/migration/retirement disposition and an operation matrix before claiming compatible updates.

### P37 — Concurrency evidence is narrower than the lifecycle surface

**Primary category:** Testing  
**Expectation:** Shared mutable seams, cancellation, shutdown, public lifecycle concurrency, retention and backend interleavings need executed evidence in addition to static synchronization and serial tests.  
**Observation:** Existing tests use controlled channels/mutexes and restore globals, but no project-owned `t.Parallel`, fuzz, benchmark, stress or repeated cancellation layer was found in the specialist inventory. The supplied root `-race` pass is valuable but does not exercise live Docker, crash/power-loss or all public concurrent-call scenarios.  
**Impact:** A race, deadlock, leaked goroutine, global-hook interference or shutdown regression can remain outside evidence even though selected serial seam tests pass.  
**Evidence:** `internal/engine/exec_hook_test.go:58-104`; `internal/engine/exec/docker_test.go:124-220`; `internal/engine/lifecycle_test.go:317-413`; `concurrency.md`, `testing.md`; serialized root verification in Method.  
**Cause:** The test design emphasizes deterministic serial fakes and local synchronization, not a complete concurrent lifecycle workload.  
**Uncertainty:** No specific data race is asserted; the gap is coverage, not a race verdict.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to concurrency/support claims.  
**Contract relation:** `in-contract (concurrency verification question)`  
**Improvement direction — discussion required:** Discuss a contained concurrency evidence slice for public calls, cancellation, backend retention and shutdown, with exact workload/toolchain identity; preserve the passing race evidence as one bounded result, not a universal guarantee.

### P38 — Internal path policy has no package-local tests

**Primary category:** Testing  
**Expectation:** A security/correctness-critical path policy should have direct package-local tests for its invariants and failure boundaries, not only indirect root tests.  
**Observation:** `internal/path` has no package-local test suite; path behavior is covered only indirectly through broader engine tests. The code is responsible for containment/symlink/path normalization effects used by artifacts and control state.  
**Impact:** A future path-policy change can break containment or symlink behavior without a focused failure at the owning package, and reviewers must infer the invariant from distant lifecycle tests.  
**Evidence:** `internal/path` tracked inventory; `internal/engine/check.go:718-790`; `code-quality-refactoring.md`, `security-privacy.md`.  
**Cause:** The path owner is treated as an implementation helper rather than a package-level security/correctness boundary.  
**Uncertainty:** Indirect tests provide some coverage; no claim is made that current path checks are all wrong.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to maintainable workspace safety.  
**Contract relation:** `in-contract (testing/path-safety question)`  
**Improvement direction — discussion required:** Discuss the invariant set and package-local test ownership for containment, symlink, race-sensitive and control-tree paths, without treating test addition as a release decision.

### P39 — Public inspection and error shapes expose unstable wire details and mutable ownership

**Primary category:** Public API  
**Expectation:** A Go library should expose typed, stable concepts or an explicitly versioned wire contract, immutable error observations and distinguish absence from empty data.  
**Observation:** `Inspect` returns raw `[]byte` for view-dependent JSON/JSONL (`inspect.go:5-27`). Graph/Plan readers return strings, nil or empty values for missing selectors (`graph.go:89-220`; `plan.go:109-210`). `Error`/`Defect` expose mutable fields/slices, `Op` is a free string, multi-defect `Error()` loses individual messages, and Pipeline returns a stored pointer (`error.go:72-121`; `pipeline.go:150-160`; `compose.go:19-20`).  
**Impact:** External consumers duplicate decoding and cannot distinguish missing from empty; callers can mutate future Pipeline error observations; errors.Is/typed operation matching is unavailable and human diagnostics lose context.  
**Evidence:** Exact paths above; `public-api.md`.  
**Cause:** Structured JSON was treated as the complete public contract without an ownership/immutability layer or typed absence/error identity.  
**Uncertainty:** Raw views may be intentionally accepted for the first horizon; distribution stability remains a decision.  
**Severity (descriptive, not a score):** Medium.  
**Blocking relation (descriptive, not an approval gate):** Material to Go-library consumers and machine support.  
**Contract relation:** `in-contract (public API question)`  
**Improvement direction — discussion required:** Discuss typed versus versioned raw views, immutable error ownership, stable error identity and explicit not-found/empty semantics before committing to compatibility.

### P40 — Branch/Merge vocabulary does not express the topology consumers infer

**Primary category:** Vocabulary  
**Expectation:** Public DSL names should make graph topology, scope/grouping and binding ownership clear enough for consumers to review and extend a pipeline.  
**Observation:** `Branch` and `Merge` record named child scopes, while actual edges come from `Bind.From`; Merge takes no explicit branch list (`pipeline.go:226-238`). Scatter/Gather use the same source-wiring pattern (`pipeline.go:241-253`).  
**Impact:** Consumers can infer that Branch/Merge establish fan-out/fan-in even though bindings do the work, making generated code, review and extension harder to understand.  
**Evidence:** `pipeline.go:226-253`; `feature/compose-pipeline.md:25,33,73`; `public-api.md`.  
**Cause:** Scope/group names and graph-topology concepts share operators while relationships are encoded elsewhere.  
**Uncertainty:** Naming may be accepted first-horizon vocabulary; the mismatch is still a distribution usability risk.  
**Severity (descriptive, not a score):** Low to medium.  
**Blocking relation (descriptive, not an approval gate):** Material to API readability and DSL evolution.  
**Contract relation:** `in-contract (public vocabulary question)`  
**Improvement direction — discussion required:** Discuss whether these names are scope labels or topology operations and document/bind the chosen meaning before treating them as stable API terms.

### P41 — Exported PlanOption advertises an unusable extension point

**Primary category:** Modularization  
**Expectation:** An exported option type should be implementable by external consumers or remain internal; public API should not create an accidental long-lived promise.  
**Observation:** `plan.go:37-49` exports `PlanOption func(*planConfig)` while `planConfig` is unexported and only the package's `WriteTo` constructor can populate its writer.  
**Impact:** External packages cannot author a new PlanOption, yet the exported name suggests they can. Future changes must preserve or remove an unusable compatibility surface.  
**Evidence:** `plan.go:37-63`; `public-api.md`.  
**Cause:** Functional-option shape was selected before deciding whether the option boundary belongs to the public contract.  
**Uncertainty:** Type-level restriction is direct; no external implementation was needed.  
**Severity (descriptive, not a score):** Low to medium.  
**Blocking relation (descriptive, not an approval gate):** Material to API clarity and compatibility hygiene.  
**Contract relation:** `in-contract (public extension question)`  
**Improvement direction — discussion required:** Discuss whether PlanOption is a supported external extension or an internal helper, and align visibility/documentation with that decision.

## Improvements

These are improvement directions for discussion, not an implementation plan or acceptance list. They do not alter the non-issued verdict.

### Make one product/release contract legible across module, CLI, workspace and assets

**Current:** The repository has a useful local first-check and explicit Linux/Go guidance, but public support, version identity, schema window, final artifact, legal metadata and lifecycle disposition are unbound.  
**Evidence:** P1–P3; `README.md:7-23,43-65`; `go.mod`; `streams.go:61-75`.  
**Benefit:** Consumers and maintainers could distinguish current source behavior, supported installation, unsupported/pre-release boundaries, workspace transitions and artifact identity.  
**Suggestion:** Discuss one versioned vocabulary and owner for module/API, generated CLI protocol, workspace schema, assets, support/deprecation/retirement, license/notice and exact final bytes.  
**Cost:** This creates durable policy and compatibility obligations; it may deliberately keep the product pre-release or narrow the supported surface.  
**Contract relation:** `in-contract (distribution question)`

### Define durable recovery and coherent state ownership

**Current:** Fail-closed unknown occupancy and per-file atomic rename are valuable safety primitives, but they do not provide cross-engine process proof or a three-file generation owner.  
**Evidence:** P4–P6; `occupancy.go`; `run.go:1360-1529`; `inspect.go:231-265`.  
**Benefit:** A consumer could understand what happens after engine loss, write interruption, artifact replacement failure or stale workspace state.  
**Suggestion:** Discuss backend-specific recovery authority, generation/journal semantics, transaction/recovery rules and artifact/provenance relationships while preserving unknown safety.  
**Cost:** More persisted state, platform qualification, migration and crash evidence.  
**Contract relation:** `in-contract (recovery/state question)`

### Bind trust, privacy and scientific evidence boundaries

**Current:** Direct argv, bounded task environment and contained paths are useful; scientific assets and local workspaces have no complete trust/data/semantic-output contract.  
**Evidence:** P7–P13; security/privacy and bioinformatics prepared reports.  
**Benefit:** Consumers could know whether the engine is for trusted local workflows, what data persists, what Docker guarantees, and what a successful assay means scientifically.  
**Suggestion:** Discuss trusted-caller/workspace model, host/container boundary, immutable dependencies, data classes/retention, domain assay support and semantic output truth sets.  
**Cost:** Security/privacy and scientific policy require named owners and may narrow current claims.  
**Contract relation:** `in-contract (trust/privacy/scientific question)`

### Make the CLI/API boundary explicit and stable before expanding automation

**Current:** README verbs, structured errors and external-package tests make the ordinary path discoverable; raw views, mutable errors, hidden configuration, generated output and lifecycle vocabulary remain difficult to consume.  
**Evidence:** P2–P3, P14–P15, P19–P22, P39–P41.  
**Benefit:** Independent Go and CLI consumers could classify completion, absence, errors, identity and recovery without private implementation knowledge.  
**Suggestion:** Discuss protocol framing, envelope/exit mapping, typed/raw view policy, immutable error ownership, explicit configuration, lifecycle naming and provider scope.  
**Cost:** Each stable type/name/protocol field becomes a long-lived compatibility obligation.  
**Contract relation:** `in-contract (consumer API/CLI question)`

### Discuss evidence and structure proportional to the intended product claim

**Current:** The serialized root checks passed and local tests are broad; fakes, synthetic crash state, weak WGS oracle, absent path package tests, large-state cost and missing release/consumer gates narrow interpretation.  
**Evidence:** P27–P38 and Verification evidence.  
**Benefit:** Product claims could be tied to exact workload, scientific, runtime, artifact and consumer evidence rather than inferred from a hermetic first check.  
**Suggestion:** Discuss representative DAG/data workloads, focused package/security tests, scheduler/validation ownership, real adapter bridge, crash/consumer/artifact evidence and which are project-owned gates.  
**Cost:** More evidence maintenance and potentially narrower stated support; no run or gate is authorized by this report.  
**Contract relation:** `in-contract (verification/product claim question)`

## Strengths

### Coherent local lifecycle and explicit engine boundary

**Benefit:** Compose creates an immutable Graph and one internal plan translation; the engine owns execution state while the public root exposes a recognizable local lifecycle.  
**Evidence:** `compose.go:5-25`; `plan.go:299-372`; `gobble.go:1-40`; `README.md:9-45`; `architecture.md`.  
**Preserve:** Keep the public model from leaking local executor/storage details while resolving the public support/API decision.

### Conservative occupancy and unknown-state handling

**Benefit:** The engine does not force-release an identity it cannot prove stopped; leases and unknown defects preserve safety over convenience.  
**Evidence:** `internal/engine/occupancy.go:17-20,139-173,269-288`; `internal/engine/release.go:99-126`; `internal/engine/lifecycle_test.go:61-78,102-113`.  
**Preserve:** Do not trade this for blind PID killing or a forced release; any recovery authority must be backend-specific and durable.

### Bounded local admission and contained publication primitives

**Benefit:** Cap and known CPU/memory admission constrain in-flight work; isolates, temporary publication and tree replacement use containment/exclusive/atomic primitives in ordinary paths.  
**Evidence:** `internal/engine/run.go:124-161,896-921`; `internal/engine/check.go:718-790`; `internal/engine/exec/publish.go:57-75,115-122`; `internal/engine/tree.go:180-245`; `performance-resources.md`, `state-recovery.md`.  
**Preserve:** Retain path/symlink safety and no-partial-output intentions when discussing performance or generation changes.

### Reuse identity, checksums and lineage are richer than file-presence checks

**Benefit:** Command/script/params, environment digest, image/executable identity, input fingerprints, output checksums and producer/consumer lineage are represented for reuse and Inspect.  
**Evidence:** `internal/engine/reuse.go:66-164,167-207,305-346`; `internal/engine/identity.go:28-41,167-203,231-267`; `internal/engine/run.go:173-225,1247-1259`.  
**Preserve:** Keep rich proof while resolving missing manifest/staged-byte relationships and protected-data policy.

### Hermetic verification and platform scope are comparatively honest

**Benefit:** The root evidence is reproducible for Go `go1.26.5 linux/amd64`, and README separates ordinary tests from live Docker/network-tagged paths.  
**Evidence:** Serialized root verification in Method; `README.md:43-65`; `go.mod:1-3`.  
**Preserve:** Keep hermetic and live/external evidence separate; do not broaden the platform/artifact claim until direct evidence exists.

### Boundary tests, lifecycle assertions and CLI stream checks provide a useful base

**Benefit:** Process/Docker adapter construction, resource flags, lifecycle decisions, occupancy, reuse, stream/panic/signal paths and external-package API tests are materially covered.  
**Evidence:** `internal/engine/exec/process_test.go:14-155`; `internal/engine/exec/docker_test.go:18-220`; `internal/engine/lifecycle_test.go`; `internal/engine/run_test.go`; `cmd/gobble/main_test.go`; `tests/cli-valid/harness_test.go:37-63`; `testing.md`.  
**Preserve:** Add bridge/oracle/runtime evidence around this base rather than replacing it with only broader but weaker fakes.

### Must-preserve conditions

- Preserve fail-closed occupancy, unknown-backend retention and explicit unsupported-state refusal while discussing recovery; these are safety strengths, not permission to force-release.
- Preserve immutable post-Compose Graph/Plan ownership copies and contained artifact paths while discussing public API or performance changes.
- Preserve checksums, lineage, environment omission/digest treatment and bounded Inspect tails, while extending their policy to staged bytes, manifest identity and protected data.
- Preserve deterministic, hermetic first-check evidence for its Linux/Go scope and explicitly separate it from live Docker/network, scientific, performance and release claims.
- Preserve external-package and real adapter boundary tests while adding output, crash, consumer and artifact oracles where the product claim requires them.
- Treat current `Release` vocabulary accurately: it currently means reconcile-and-close occupancy, not publish/delete. Any rename, alias, ownership change or new cleanup authority requires product/API discussion.

## Gaps

| Gap | Effect | Needed |
|---|---|---|
| Release/support/legal/artifact identity | No exact module/CLI artifact, support window, license/notice, checksum, installation/consumer route, deprecation or retirement claim can be made. | An accepted release/support/lifecycle policy and exact candidate/artifact identity; no publication is authorized by this review. |
| Public API/CLI compatibility | Consumers cannot know which exports, raw views, errors, lifecycle names, generated driver, module or schema versions are supported together. | A product/API compatibility decision, machine protocol envelope and consumer compatibility position. |
| Cross-engine process recovery | Static source proves loss of Process adapter proof but not real child behavior or a safe durable handoff. | Authorized process-boundary/restart evidence and a decided backend recovery contract. |
| Control-state durability | No crash/power-loss injection, fsync qualification, journal/generation proof or complete recovery matrix exists. | Controlled mixed-snapshot/write-interruption evidence and a state transaction/disposition decision. |
| Live Docker/network and dependency provenance | Image tags, host client behavior, logs/cleanup, external assets and fixture bounds are not live-qualified. | Authorized live/runtime and dependency evidence with redaction and exact identities. |
| Scientific truth and domain support | DESeq2 design validity, WGS/BAM/BAI/report truth, reference/sample provenance and assay ownership are unbound. | Accepted domain support matrix, semantic truth sets and scientific provenance policy. |
| Performance and resource envelope | Large DAG, sample sheet, tree/file, polling, memory, disk and long-lived executor behavior lack representative workloads/budgets. | An accepted workload/metric/toolchain contract and retained measurements. |
| Testing oracle and concurrency scope | Fakes can skip artifact effects; crash recovery is synthetic; path package, schema skew, parallel public calls and some cleanup paths lack direct tests. | Focused bridge, scientific, crash, schema, path, concurrency and cleanup evidence tied to a chosen contract. |
| Privacy/protected data | Parameters/scripts/logs/control state can persist sensitive content without a complete classification/redaction/retention policy. | Data classification, access, redaction, retention/deletion and support-handling decisions. |
| Runtime correlation/history | Repeated runs and prior attempts are not stably correlated or publicly inspectable. | Stable run/attempt identity and a bounded history/event/retention contract if support requires it. |

## Handoff

| Field | Value |
|---|---|
| Rechecked subject identity | Unchanged: commit `c9ffeb84c611814fbf00a4ec15036b5939fa46b7`, tree `abb9d73f6b1b297e8dc8ba610033905391c96706`. |
| Record-state reason | `partial`: every core category and overlay was accounted for and the frozen static/Linux verification evidence was revalidated, but live Docker/network, destructive crash/power-loss, installed-consumer/final-artifact, scientific truth-set, representative performance, privacy-policy, legal and release evidence remains absent or excluded. |
| Original report ownership | At handoff, the caller took ownership of the caller-bound report. |
| Next owner | Product/API, engine-runtime, security/privacy, scientific-domain, testing, and release owners in a discussion authorized by the caller. |
| First action | Discuss and accept or revise the problem boundaries and desired outcomes before selecting any concrete implementation, refactor, compatibility mechanism, release policy, or test architecture. |
| Problems handed over | P1–P41 in this report. |
| Gaps handed over | The ten Gaps above. |
| Authority boundary | This report grants no target mutation, approval, merge, publication, release, cleanup, or compatibility decision authority. |
| Restart condition | Re-run Code Review if the frozen source/tree, accepted design, public support/release contract, checklist/specialist source set, or cited runtime evidence changes. |

The best supported version would make its local trust and scientific scope explicit, provide one coherent release/API/CLI/workspace identity, durable recovery and artifact provenance, truthful observability, bounded resource behavior, and evidence matched to installed consumers and scientific claims. This statement describes the reviewed gap; it is not an overall score, verdict, gate, or implementation plan.

### References

| Name | Location | Use |
|---|---|---|
| Gobbi Code Review skill | `/home/jeonhh0061/.codex/plugins/cache/gobbi-workspace/gobbi/1.2.2/skills/code-review/SKILL.md` | Read-only workflow, locked checklist-free record, source-order coverage, reconciliation and handoff rules. |
| Canonical Code Review checklist | `/home/jeonhh0061/.codex/plugins/cache/gobbi-workspace/gobbi/1.2.2/skills/code-review/checklist.md` | Full core category and item coverage; SHA-256 `89aaf24bad0d5f0deddba8df22103f7380c6aa348effe3468c5573cbc43fce07`. |
| Code Review report template | `/home/jeonhh0061/.codex/plugins/cache/gobbi-workspace/gobbi/1.2.2/skills/code-review/templates/report.md` | Nine-section report structure; SHA-256 `1ea3d640df200cc807398363a62e7cf7a61b4e980baeeaca1177580645b7c17e`. |
| Frozen Gobble source | `/playinganalytics/git/gobble` at commit `c9ffeb84c611814fbf00a4ec15036b5939fa46b7`, tree `abb9d73f6b1b297e8dc8ba610033905391c96706` | Subject evidence and identity. |
| Tracked README/module/Design Memory | `README.md`, `go.mod`, tracked `docs/`, tracked accepted `.gobbi` design sources | Product intent, support language, commands, schema and recovery boundaries. |
| Prepared specialist reconciliation | Temporary review workspace (not retained in project Memory) | Sixteen independent category reports supplied evidence after the locked Phase 2 record; they were not subject and were not backfilled into that record. |

### Identity revalidation

At the original review handoff, the frozen identity remained `HEAD=c9ffeb84c611814fbf00a4ec15036b5939fa46b7` and `HEAD^{tree}=abb9d73f6b1b297e8dc8ba610033905391c96706`. Worktree changes remained limited to excluded dirty/untracked `.gobbi` Memory. No source, test, report input, checklist, template, Memory, Git state or external state was changed during the review. Temporary inspection binaries were not retained. The prepared category reports and caller-bound aggregate were present only in the temporary review workspace at that time. This durable copy grants no mutation, approval, merge, publication, release or cleanup authority.
