# Gobble — Product

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. Keep one product file. If Interview named
several products, keep `## Products` as the inventory and add one subsection per
product under the remaining product headings.

## Products

- Products:
  - Gobble
- Source: `products`

## First useful outcome

### Gobble

- Statement: A consumer installs the Go module, writes a small pipeline in Go that uses modules, branch, and merge, validates and inspects the plan, runs it locally in containers, and can resume after a contained failure. Construction order is library, then engine, then CLI. First-horizon exit still requires the same loop on the CLI. Command names live in [system.md](../architecture/system.md) Interfaces Current. First useful outcome is that agent loop, not Nextflow- or Snakemake-class feature parity. Today consumers write Nextflow, Snakemake, Cromwell/WDL, or ad-hoc scripts and operate them as humans; the hypothesized switch is agent-native compose and recover without a Gobble DSL. No named external consumer is recorded.
- Source: `first-use`, `current-alternative`

## Refused uses

### Gobble

- Statement: Reject requiring a Gobble-specific language, reproducing Nextflow or Snakemake syntax, a GUI, unguarded deletion of artifacts, persisting secrets in logs or metadata, and treating compile or a DAG drawing as a successful agent-operable run. Also refuse built-in assay-specific tools as product features and HPC or cloud as first-horizon backends.
- Source: `refused-use`, `boundary`

## Audience and experience direction

### Gobble

- Statement: The human-facing surface is a small CLI. Audience is coding agents first and human developers second. Interaction is the Go API plus that CLI. Default output is structured JSON or JSONL. Human text is secondary. There is no TUI or GUI. The CLI is used from a local terminal and from agent tool runners, often unattended. Color-only status is assumed insufficient. Assumed CLI style follows `go` and `git`: small verbs, stable flags, structured or plain output. Do not take Nextflow or Snakemake CLI or DSL as the interaction reference.
- Current: Installed binary is `gobble` (`cmd/gobble`). Seven product verbs. `--sample PATH` is on compose, validate, plan, run, and resume. Inspect and release reject it. JSON or JSONL only. No color or pretty output this session. Contract in [system.md](../architecture/system.md) Interfaces Current.
- Source: `experience-direction`, `accessibility-needs`, `design-reference`

## Access and data promises

### Gobble

- Access: First horizon uses local file permissions only. There is no Gobble identity, account, or entitlement system.
- Data promise: Gobble holds pipeline code references, run state, artifacts, and logs. Artifacts may be valuable and may include sequence or clinical files the consumer supplies. Gobble does not collect user profiles. Credentials and secrets must not be written to logs or persisted metadata. Inspect omits Env. The consumer owns the run workspace. Deletion is not a shipped Engine verb. Consumers may delete their own files. Guarded clean stays designed and not shipped.
- Source: `access`, `data-promise`

## Failure and recovery

### Gobble

- Statement: The consumer sees structured task and run state, an error that names the failed unit, and which outputs remain reusable. Recovery is inspect, release occupancy, resume remaining work, or inspect-then-modify the Go pipeline and resume. Human translation of raw logs is not the recovery path.
- Current: After a contained `Run` failure or `ctx` cancel, recovery is `Inspect`, then occupying-process or later-process `Release` to close occupancy, then dest-scope `Resume`. Cancel is context on `Run` and `Resume`; occupancy stays until `Release`. Occupancy does not close and Resume does not occupy while any identity remains unknown. A dead-PID helper is not recovery authority. `output-exists` applies only to dests this Resume would publish that are not authorized replace dests. Topology edits classify as Change. There is no public Cancel. Named retry and guarded clean are not shipped.
- Source: `failure-recovery`

## Support and updates

### Gobble

- Statement: Assumption — help and updates come from the GitHub repository. There is no support SLA. Versioning will follow Go modules once the library is published.
- Source: `support-update`

## End of life

### Gobble

- Statement: Assumption — no export, deprecation, or retirement promise is made yet. Local run workspaces and artifacts remain the consumer’s files. A later promise can add export of run state and provenance.
- Source: `end-of-life`

## Feature index

Record `core-tasks` names only. If `core-tasks` is explicitly none, write no
feature files and say so here. Roadmap owns feature-to-horizon placement.

| Id | Name | Product | Consumer | Source topic |
|---|---|---|---|---|
| compose-pipeline | compose-pipeline | Gobble | Agent author or human Go developer | `core-tasks` |
| validate-plan | validate-plan | Gobble | Agent operator or human operator | `core-tasks` |
| run-local | run-local | Gobble | Agent operator or human operator | `core-tasks` |
| inspect-run | inspect-run | Gobble | Agent operator or human operator | `core-tasks` |
| recover-run | recover-run | Gobble | Agent operator or human operator | `core-tasks` |

- Source: `core-tasks`

## Open questions

| Id | Question | What would resolve it |
|---|---|---|
| visual-principles | Which few visual principles must stay recognizable on the CLI? | A recorded CLI visual rule |
| support-update | Is GitHub the lasting help channel? | A named support or release process |
| end-of-life | What export or retirement is promised? | An accepted end-of-life promise |

- Source: open topic ids
