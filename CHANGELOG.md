# Changelog

## Unreleased

### macOS and existing-assay walkthroughs

- Add native Intel/Apple Silicon launchers, a Mac/Linux preview setup script,
  and explicit linux/amd64 runtime and task selection for Docker Desktop.
- Add `gobble demo NAME DIR` for all five existing assays, using their official
  checksum-verified test data and unchanged pipeline defaults. Downloads can
  be retried without replacing existing projects or publishing corrupt files.
- Add an installation-to-results guide with resource requirements, agent
  prompts, real assay outputs, monitoring, Stop, and Resume.
- Add native Mac launcher CI and installed-runtime RNA-seq/WGS Docker checks.
  Actual Mac/Windows Docker Desktop validation remains a release gate.

### Terminal monitoring

- Add read-only `watch --workspace DIR` to the generic command and packed
  runners, with a Bubble Tea dashboard, pipeline graph, global statistics,
  exact sample selection, attention list, and live stdout/stderr tails.
- Add coherent `inspect monitor` snapshots and optional copied `TaskDisplay`
  labels. Bundled pipelines label sample, shared reference, and cohort work.
  Labels do not change execution commands, artifact paths, or reuse decisions.
- Keep terminal dependencies in `monitor/tui`; the engine only projects
  existing controls. Quitting the monitor leaves pipeline execution running.
- See [Monitoring](docs/monitoring.md) for controls and progress semantics.

### Multiomics product family

- Current source contains five engineering-supported products: WGS joint
  germline, bulk RNA-seq, Methyl-seq, ATAC-seq, and scRNA-seq.
- Their graph generations are `wgs-joint-germline-v1`,
  `rnaseq-star-salmon-v1`, `methylseq-bismark-v1`, `atacseq-bwa-v1`, and
  `scrnaseq-simpleaf-v1`. The executable baseline is commit
  `f21a858c66a2d95ce8eff469e6db2bfa3240c3a5`, tree
  `c90dfe77192c2528f8fd54d17f4d9547b09a6998`.
- The WGS, RNA-seq, and Methyl-seq lifts are graph breaks. Their old proof
  workspaces require new workspaces. Temporary `assets.WGS`, `assets.RNASeq`,
  and `assets.MethylSeq` names point to the lifted defaults, not old graph
  bytes. ATAC-seq and scRNA-seq have no earlier workspace generation.
- Defaults use immutable image tags and digests. Each assay manifest under
  `tests/pipelines/<assay>/testdata/manifest.json` records exact benchmark and
  fixture-byte authority plus that assay's image metadata. Image rows are not a
  uniform license, provenance, attribution, or redistribution authority.
- Support is engineering-only on trusted-local `linux/amd64` Docker execution.
  It does not claim nf-core endorsement or scientific, clinical, diagnostic,
  or regulatory validity.

This family is not part of `v0.1.0`. A later release must name these graph
generations and their source, workspace, and recovery compatibility effects.

## v0.1.0 — 2026-08-30

First public pre-1.0 release of `github.com/HahyeonJeon/gobble`. The module path has no `/v0` suffix. This tag is immutable and is never reused. Supported installs never use `@latest`.

This is the first numbered release. There is no prior published version to migrate from.

### Go API

- Supported verbs: `Compose`, `Validate`, `BuildPlan`, `Run`, `Inspect`, `Release`, `Resume`.
- Supported composition: `Module`, `Branch`, `Merge`, `Scatter`, `Gather`, `When`.
- Artifacts: `PathSpec` with File, `Group`, or `Tree` binds.
- Occupy identity: `Run` and `Resume` require `WithIdentity`. `Inspect` and `Release` may omit it and then bind the current executable. Mismatch fails closed before workspace mutation.
- SchemaVersion remains 2.

### CLI

- Generic `gobble` on `linux/amd64`, Go 1.26 or newer.
- Agent install: `go get github.com/HahyeonJeon/gobble@v0.1.0` and `go install github.com/HahyeonJeon/gobble/cmd/gobble@v0.1.0`.
- Human artifact: `gobble pack [package] --output PATH` writes one packed `linux/amd64` trampoline. The runner needs no Go at run time. Gobble portions are MIT; the embedded pipeline may differ.

### Workspace and recovery

- Recovery is Inspect, later-process Release without an unproved PID signal, then Resume remaining work.
- A proved-stopped Docker task may keep a `runtime_id` for log-copy or `docker rm` retry without wedging occupancy. Unproved Docker remains `unknown-backend` and blocks Resume.

### License and evidence

- MIT License, Copyright (c) 2026 HahyeonJeon.
- Hermetic first check: `go test ./...`.
- First-horizon local-pin and packed evidence: `go test -tags=live ./tests/install-e2e`.
