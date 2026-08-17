# Gobble — Initial Design Brief

> **Document status:** Discussion Draft v0.2 · Non-binding  
> **Purpose:** This working draft defines Gobble’s initial direction and required capability baseline. It is not an approved or finalized architecture, requirements specification, or roadmap.  
> **Subject to change:** The vision, terminology, layers, components, responsibilities, technology candidates, and development sequence described here are provisional. They may be added, removed, combined, separated, replaced, or redefined as the project is discussed and validated.  
> **Interpretation:** Terms such as `provides`, `includes`, and `is responsible for` describe capabilities that Gobble may ultimately need. They do not prescribe the current implementation or settle detailed policies.  
> **Current scope:** This document defines the problem Gobble intends to solve, its primary differentiation, its high-level architecture, and its required feature baseline. Specific Go APIs, state schemas, scheduling algorithms, cache keys, and backend-specific behavior are intentionally deferred.

---

## 1. Vision

**Gobble aims to become an agent-native bioinformatics pipeline engine, built in Go, for designing, validating, planning, scheduling, executing, observing, modifying, recovering, and resuming pipelines.**

Gobble treats the core capabilities of established workflow engines such as Nextflow, Snakemake, and Cromwell as its functional baseline. Over time, it should broadly support pipeline composition and reuse, DAG-based execution, resource-aware scheduling, failure recovery and resume, reproducible execution, and portability across local, distributed cluster, and cloud environments.

Gobble’s primary distinction, however, is not simply that it is another pipeline engine. It is that the system is designed from the beginning so that **agents such as Claude Code and Codex can reliably act as both pipeline authors and pipeline operators**.

```text
Analysis Goal
      ↓
Pipeline Construction & Modification
      ↓
Validation · Planning · Scheduling
      ↓
Local · Cluster · Cloud Execution
      ↓
Observation · Recovery · Resume · Revision
```

The first validation target is an end-to-end workflow in which an agent can understand the Gobble library and documentation, create or modify a bioinformatics pipeline in Go, inspect its execution plan, run it, interpret its status and failures, and safely recover or resume affected work.

---

## 2. Positioning and Authoring Model

| Area | Direction |
|---|---|
| **Functional baseline** | Broadly address the problems already handled by Nextflow- and Snakemake-class engines, including pipeline design, scheduling, parallel execution, reproducibility, failure handling, caching and resume, and portability across execution environments. |
| **Primary differentiation** | Treat agent-driven discovery, construction, validation, execution, modification, diagnosis, and recovery as first-class use cases, in addition to direct human authoring and operation. |
| **Initial authoring model** | Allow pipelines to be composed directly through a Go library, without requiring users or agents to learn a separate language. |
| **Long-term authoring model** | Enable agents and applications to construct pipelines and control their execution lifecycle through stable APIs. |
| **Declarative representations** | JSON or YAML may later be introduced as interchange, persistence, or configuration formats. Creating a proprietary Gobble DSL with an independent semantic model is not the default direction. |
| **Compatibility principle** | Reinterpret the capabilities required in practice through Gobble’s own model and interfaces rather than reproducing another engine’s syntax or internal architecture. |
| **Development approach** | Build a reusable Go core library first, then construct the engine, backend adapters, and agent-facing interfaces on top of it. |

Go code, future APIs, and optional JSON or YAML representations should be treated as **different interfaces for constructing or expressing the same underlying pipeline model**, rather than as separate pipeline languages.

---

## 3. Architecture

The current draft organizes Gobble around **Core Library → Engine Services → Execution Backends → Interfaces**. A central architectural principle is to treat the **Scheduler and Executors as separate responsibilities**.

```text
Claude Code · Codex · Human Developer · Application
                         ↓
       Go API · CLI · Agent API · Optional Adapters
                         ↓
          Pipeline Model · Validator · Planner
                         ↓
                       Scheduler
       Dependency · Resource · Priority · Concurrency
                         ↓
                      Executors
       Local · Container · HPC Scheduler · Cloud Batch

       State / Cache · Artifact / Storage · Observability
               support the entire execution lifecycle
```

| Area | Provisional responsibility |
|---|---|
| **Gobble Core Library** | Provide general concepts for composing pipelines, such as Task, Pipeline, Input, Output, Resource, Environment, and execution contracts. |
| **Pipeline Model / Planner** | Validate the authored pipeline, interpret dependencies and execution units, and produce an inspectable execution plan. |
| **Scheduler** | Determine which tasks are eligible to run and decide when they should run based on dependencies, resources, priority, concurrency, and other constraints. |
| **Executor / Backend Adapter** | Submit scheduler decisions to a local process, container runtime, HPC scheduler such as Slurm, or cloud batch environment; then monitor, reconcile, and cancel backend jobs. |
| **Run State / Cache** | Persist pipeline-run and task state, execution history, reuse decisions, and recovery information. |
| **Artifact / Storage Manager** | Manage staging, isolated work directories, validation, retention, cleanup, and publication of inputs, reference data, intermediate results, and final outputs. |
| **Observability / Control** | Expose structured state, events, logs, resource metrics, provenance, and execution-control operations. |
| **Interface Layer** | Begin with the Go API and CLI, then expand to agent tools and APIs, remote services, and application interfaces. |

The boundaries between the core library and engine, the division of responsibility between Gobble’s scheduler and backend schedulers, and the design of state and artifact storage remain central topics for later discussion.

---

## 4. Required Feature Baseline

The capabilities below are considered **required for Gobble to become a complete bioinformatics pipeline engine over the long term**. The `Foundation`, `Portability`, and `Scale` labels indicate validation order, not whether a capability is optional.

| Capability | Required scope | Introduction stage |
|---|---|---|
| **Pipeline Model and DAG** | Express reusable tasks and sub-pipelines; explicit inputs, outputs, parameters, resources, and environments; dependency graphs; fan-out and fan-in; scatter/gather; conditional execution; and data-dependent dynamic expansion. | Foundation |
| **Validation and Planning** | Detect configuration errors, missing inputs, dependency cycles, conflicting outputs, and unsupported backend requirements before execution. Expose dry runs, DAG inspection, execution plans, and the impact of proposed changes. | Foundation |
| **Scheduling** | Select runnable tasks from the DAG and order execution using CPU, memory, GPU, disk, wall time, custom resources, priority, concurrency limits, queue limits, and backpressure. At scale, support job grouping, job arrays, and submission throttling. | Foundation → Scale |
| **Execution Backend Abstraction** | Execute the same pipeline model first through local-process and container backends, then extend it to HPC schedulers such as Slurm and cloud or orchestrated environments such as AWS Batch and Kubernetes. Backend-specific implementations should be replaceable behind consistent submit, poll, cancel, and reconcile contracts. | Foundation → Portability |
| **Persistent State, Cache, and Resume** | Persist enough state to recover runs after the engine or host terminates. Decide whether completed work can be reused using inputs, commands, parameters, code, tool environments, and output validity, and re-execute only changed work and affected downstream dependencies. | Foundation |
| **Failure Handling and Recovery** | Support failure policies, retries, backoff, timeouts, cancellation, optional continuation, resource adjustment on retry, incomplete-output handling, and reconciliation of jobs orphaned or detached at the backend. | Foundation |
| **Artifact and Storage Lifecycle** | Provide isolated task work directories; input and reference-data staging; intermediate and temporary artifacts; output validation; safe publication after success; and cleanup and retention policies. Connect local filesystems, shared filesystems, and object storage through consistent concepts. | Foundation → Portability |
| **Environment and Reproducibility** | Allow each task to declare a container, package environment, or HPC environment module. Record tool, image, environment, pipeline version, and execution parameters so that results can be reproduced. | Foundation |
| **Observability, Provenance, and Control** | Expose run and task state, structured events and errors, stdout/stderr, timing and resource metrics, execution history, DAGs, and artifact lineage. Provide consistent contracts for run, status, logs, cancel, retry, resume, and cleanup operations. | Foundation |
| **Configuration, Profiles, and Secrets** | Separate pipeline logic from execution-environment configuration. Apply local, HPC, and cloud profiles, resource overrides, and backend settings without changing pipeline code. Manage credentials and secrets separately so they are neither exposed nor persisted in logs or metadata. | Foundation → Portability |
| **Modularity and Extensibility** | Compose, version, and test task wrappers, components, sub-pipelines, and workflow modules. Extend executors, storage, metadata, and future scheduling policies through plugins or explicit adapter boundaries. | Foundation → Scale |
| **Testing and Quality Assurance** | Support linting, static validation, stub or lightweight execution, unit and integration testing, and output-integrity checks so agent-generated pipelines can be validated before expensive production execution. | Foundation |
| **Agent API and Safety** | Expose capability discovery, plan inspection, execution, and lifecycle control through machine-readable schemas and stable identifiers. Operations should be idempotent where practical and should provide structured errors, change diffs, execution impact, and guardrails around destructive or high-risk actions. | Foundation |

Scheduling is not merely the act of forwarding a job to a backend. Gobble requires a clear contract between the **Gobble scheduler, which understands pipeline dependencies and global resource constraints**, and the **local, HPC, or cloud executor, which places jobs onto the actual compute system**.

---

## 5. Development Maturity

| Stage | Minimum capabilities to validate |
|---|---|
| **Local Agent-operable Core** | A Go pipeline model; validation and planning; a DAG- and resource-aware scheduler; local and container executors; persistent state; caching and resume; retry handling; artifact lifecycle management; and structured CLI/API responses and logs. |
| **HPC-ready Engine** | A first HPC adapter such as Slurm; shared and node-local storage; queue and resource mapping; job grouping and arrays; submission-rate control; backend reconciliation; and execution profiles. |
| **Cloud and Service-ready Engine** | Object storage; a cloud batch or Kubernetes adapter; scheduling across multiple runs; persistent service state for long-running operations; credential management; and the feasibility of a standardized remote API. |
| **Agent-native Ecosystem** | Reusable component and pipeline catalogs; capability discovery; plan diffs; automated diagnosis and recovery assistance; policy-governed execution; and application integration. |

These stages do not define a committed product roadmap. They are provisional milestones for discussing the order in which major capabilities should be validated.

---

## 6. Agent-native Design

In Gobble, `agent-native` does not merely mean that an agent can execute shell commands. To operate pipelines reliably, an agent must be able to understand the system’s capabilities and current state, detect problems before execution, and safely determine the next action from execution results.

| Requirement | Meaning |
|---|---|
| **Discoverable** | Components, backends, resources, operations, and constraints must be available through structured discovery mechanisms. |
| **Structured** | Plans, states, artifacts, metrics, and errors must be machine-readable rather than accessible only through natural-language logs. |
| **Explainable** | The system should explain why a task ran, waited, reran, produced a cache hit, or failed, and which changes affected that decision. |
| **Controllable** | Validate, plan, run, inspect, cancel, retry, resume, and clean should be explicit operations and idempotent where practical. |
| **Recoverable** | Agents must be able to identify failure causes and reusable results, then safely rerun or resume only affected portions of a modified pipeline. |
| **Guarded** | High-risk actions—such as large resource consumption, artifact deletion, external-system modification, or secret access—must be constrained by explicit policies and boundaries. |

The first reference scenario is an end-to-end flow completed with minimal human intervention:

```text
Understand Request
      ↓
Create or Modify Go Pipeline
      ↓
Validate · Inspect DAG · Review Plan
      ↓
Schedule and Execute Locally in Containers
      ↓
Observe Structured State, Logs and Artifacts
      ↓
Diagnose Failure or Change Impact
      ↓
Retry · Modify · Resume only affected work
```

---

## 7. Design Principles

| Principle | Meaning |
|---|---|
| **Library-first** | Build an independently usable Go library and stable core concepts first; compose the engine and applications from that foundation. |
| **Agent-native and Human-friendly** | Enable structured agent discovery and control while keeping the interfaces readable and practical for direct human use. |
| **Modular by Design** | Separate the responsibilities of the pipeline model, planner, scheduler, state system, artifact manager, executors, and interfaces so that individual parts can be replaced or extended. |
| **Intuitive Interface** | Express common pipelines with little code and sensible defaults while preserving access to lower-level policies and components for advanced requirements. |
| **Go-first, Model-centered** | Use Go as the most direct initial authoring interface, while ensuring that all future interfaces share the same pipeline model and lifecycle. |
| **No Proprietary DSL** | Do not make a Gobble-specific language a prerequisite for constructing or operating pipelines. |
| **Backend-independent Core** | Prevent any particular scheduler, container runtime, storage system, or cloud provider from determining the core pipeline model. |
| **Observable and Recoverable by Default** | Record not only final results but also state transitions and decision context, and treat recovery after failures and interruptions as a normal part of the lifecycle. |
| **Reproducible and Incremental** | Make executions reproducible while safely reusing unchanged, valid results. |
| **Stable Core, Replaceable Edges** | Evolve core concepts and contracts carefully while keeping backends, storage systems, and external integrations behind replaceable boundaries. |

---

## 8. Scope

### Current focus

- A core library and shared pipeline model for composing pipelines in Go
- Responsibilities and contracts for the validator, planner, and resource-aware scheduler
- Local and container executors and the execution lifecycle
- Persistent state, caching, resume, retries, and artifact lifecycle management
- Structured CLI/API operations, statuses, logs, and error models for Claude Code and Codex
- Adapter boundaries for future HPC and cloud backends
- Foundations for handling bioinformatics tools, large files, and reference data naturally

### Not decided at this stage

- Concrete Go package, type, function, and public API designs
- DAG-construction mechanisms and scheduling algorithms
- Technology choices for the state store, metadata store, and artifact catalog
- Detailed cache fingerprints, invalidation rules, and resume semantics
- Implementation of work directories, staging, atomic publication, and cleanup
- Resource naming, scope, priority, fairness, and quota policies
- Initial container runtime, HPC scheduler, and cloud-provider support
- Whether or when to support JSON, YAML, GA4GH WES/TES, or other standards
- Component registry, distribution ecosystem, and compatibility policy
- GUI, desktop application, or web application design
- Built-in support for particular bioinformatics tools or pipelines

---

## 9. Discussion Topics

| Topic | Possible directions for discussion |
|---|---|
| **Boundary between the Core Library and Engine** | Whether the library should stop at the model and validation layers or also include local scheduling and execution. |
| **Scheduler Responsibilities** | Which guarantees for dependencies, resources, priority, fairness, and backpressure belong in Gobble’s core scheduler. |
| **Executor Contract** | The minimum common contract for submit, poll, cancel, logs, artifact staging, reconciliation, and recovery. |
| **Pipeline Model** | Common representations for tasks, artifacts, parameters, resources, environments, and dynamic expansion. |
| **State and Resume Semantics** | How to reconcile engine restarts, pipeline changes, missing or corrupted outputs, and backend job state. |
| **Cache and Provenance Boundary** | Which inputs, code, tools, environments, and reference data must participate in reuse decisions and lineage. |
| **Agent Interface** | The minimum operations and schemas an agent needs to discover, validate, execute, diagnose, and modify pipelines. |
| **Initial Validation Workflow** | The first bioinformatics pipeline that can jointly validate scheduling, scatter/gather, retries, caching and resume, and container execution. |
| **Compatibility Scope** | Which Nextflow- and Snakemake-class capabilities form the initial baseline and which belong to later scale-oriented validation. |

---

## 10. Research Basis

This feature baseline reorganizes capabilities repeatedly found in the **official documentation reviewed as of August 2026** into a direction suitable for Gobble.

- **Nextflow:** [Executors](https://www.nextflow.io/docs/latest/executor.html), [Caching and resuming](https://www.nextflow.io/docs/latest/cache-and-resume.html), [Process reference](https://www.nextflow.io/docs/latest/reference/process.html), [Reports](https://www.nextflow.io/docs/latest/tracing.html), [Configuration](https://www.nextflow.io/docs/latest/config.html)
- **Snakemake:** [Command-line execution and scheduling](https://snakemake.readthedocs.io/en/stable/executing/cli.html), [Rules, resources, priorities and checkpoints](https://snakemake.readthedocs.io/en/stable/snakefiles/rules.html), [Deployment and reproducibility](https://snakemake.readthedocs.io/en/stable/snakefiles/deployment.html), [Storage support](https://snakemake.readthedocs.io/en/stable/snakefiles/storage.html), [Reports](https://snakemake.readthedocs.io/en/stable/snakefiles/reporting.html), [Modularization](https://snakemake.readthedocs.io/en/stable/snakefiles/modularization.html), [Testing](https://snakemake.readthedocs.io/en/stable/snakefiles/testing.html)
- **Cromwell / WDL:** [Call caching](https://cromwell.readthedocs.io/en/latest/cromwell_features/CallCaching/), [Execution backends](https://cromwell.readthedocs.io/en/latest/backends/Backends/), [Persistent job and workflow stores](https://cromwell.readthedocs.io/en/latest/developers/bitesize/workflowExecution/workflowSubworkflowAndJobStores/)
- **GA4GH:** [Workflow Execution Service](https://ga4gh.github.io/workflow-execution-service-schemas/docs/), [Task Execution Service](https://ga4gh.github.io/task-execution-schemas/docs/)

These references do not imply that Gobble should reproduce their syntax or implementation. They serve as a functional checklist so that the project does not overlook problems that established engines and standards have repeatedly needed to solve.

---

> **Gobble’s current objective is not to settle detailed implementation choices in advance. It is to define the capabilities and responsibility boundaries that an engine must provide for agents to design and operate bioinformatics pipelines reliably. The architecture, feature grouping, technologies, and development sequence in this document remain provisional and should be validated through subsequent discussion and experimentation before being recorded as decisions.**
