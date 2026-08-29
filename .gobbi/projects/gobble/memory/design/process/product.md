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

- Statement: A consumer uses the module from a trusted local source revision, writes a small Go pipeline with modules, branch, and merge, validates and inspects the plan, runs it locally, and recovers after a contained failure through Inspect, Release, and Resume. Construction order is library, then engine, then CLI. First-horizon exit still requires the complete API and CLI loop and remains unclaimed. Command names live in [system.md](../architecture/system.md) Interfaces Current. First useful outcome is that agent loop, not Nextflow- or Snakemake-class feature parity. No named external consumer is recorded.
- Current: Gobble is a pre-1.0 trusted-local `linux/amd64` preview licensed under MIT. Agents use the Go library and generic command through an explicit local path pin. Humans receive one packed runner for one embedded pipeline and need no Go at run time. First-horizon installed-path exit is proved for both families by `go test -tags=live ./tests/install-e2e`. This is local-pin and packed-artifact evidence; an exact-tag route is not yet available because no tag exists.
- Source: `first-use`, `current-alternative`

## Refused uses

### Gobble

- Statement: Reject requiring a Gobble-specific language, reproducing Nextflow or Snakemake syntax, a GUI, unguarded deletion of artifacts, persisting secrets in logs or metadata, and treating compile or a DAG drawing as a successful agent-operable run. Also refuse built-in assay-specific tools as product features and HPC or cloud as first-horizon backends.
- Source: `refused-use`, `boundary`

## Audience and experience direction

### Gobble

- Statement: The human-facing surface is a small CLI. Audience is coding agents first and human developers second. Interaction is the Go API plus that CLI. Default output is structured JSON or JSONL. Human text is secondary. There is no TUI or GUI. The CLI is used from a local terminal and from agent tool runners, often unattended. Color-only status is assumed insufficient. Assumed CLI style follows `go` and `git`: small verbs, stable flags, structured or plain output. Do not take Nextflow or Snakemake CLI or DSL as the interaction reference.
- Current: The agent binary is generic `gobble` (`cmd/gobble`) selected from the consumer module graph. The human binary is one packed `linux/amd64` runner with no Go, `pack` command, or package operand at run time. Both expose the seven product verbs. `--sample PATH` is on compose, validate, plan, run, and resume; Inspect and Release reject it. The installed command, selected module, pipeline, platform, install family, and workspace identity must match. JSON or JSONL only. Contract in [system.md](../architecture/system.md) Interfaces Current.
- Source: `experience-direction`, `accessibility-needs`, `design-reference`

## Access and data promises

### Gobble

- Access: First horizon uses local file permissions only. There is no Gobble identity, account, or entitlement system.
- Data promise: Gobble holds pipeline code references, run state, artifacts, and logs. Artifacts may be valuable and may include sequence or clinical files the consumer supplies. Gobble does not collect user profiles. Credentials and secrets must not be written to logs or persisted metadata. Inspect omits Env. The consumer owns the run workspace. Deletion is not a shipped Engine verb. Consumers may delete their own files. Guarded clean stays designed and not shipped.
- Current trust boundary: One trusted author and OS user owns one exclusive workspace and trusted pipeline code. Docker `--network=none` and UID/GID are isolation conveniences, not a sandbox. Untrusted, multi-user, regulated, and clinical deployment are unsupported.
- Source: `access`, `data-promise`

## Failure and recovery

### Gobble

- Statement: The consumer sees structured task and run state, an error that names the failed unit, and which outputs remain reusable. Recovery is inspect, release occupancy, resume remaining work, or inspect-then-modify the Go pipeline and resume. Human translation of raw logs is not the recovery path.
- Current: After a contained failure, cancellation, or controller death, recovery is `Inspect`, then occupying-process or later-process `Release`, then `Resume` remaining work. Later-process Release never signals an unproved process PID. Dest-complete process work persists `published-unfinalized` and is omitted from remaining. Incomplete process work reruns. A Docker task whose stopped state and exit code were proved may retain a RuntimeID for log-copy or removal retry without becoming unknown or wedging occupancy. Unproved Docker disposition remains `unknown-backend`, keeps occupancy active, and blocks Resume. Dest-incomplete outputs from the recorded incomplete or succeeded producer may be replaced on Resume; failed, blocked, and repathed foreign destinations remain `output-exists`. There is no public Cancel, named retry, repair verb, PID adoption, or guarded Clean.
- Source: `failure-recovery`

## Support and updates

### Gobble

- Statement: Assumption — help and updates come from the GitHub repository. There is no support SLA. Versioning will follow Go modules once the library is published.
- Current: MIT permits redistribution of the current tree. No public tag or remote release exists. Future pre-1.0 publication uses one immutable repository-root `v0.x.y` tag for the root module, library, and `cmd/gobble`, with no `/v0` module path, no retag, and no supported `@latest`. A patch means no intended Go API, CLI protocol, workspace-schema, or recovery break. A minor may add features and may declare a pre-1.0 break; its release notes name Go API, CLI, workspace, and recovery effects. The first number remains Deferred until the user names it. `v0.x.y-rc.1` is used only if the user later asks. This session creates no tag, push, GitHub Release, or other remote publication. A future tag still requires release notes, MIT bytes, an exact commit, and installed external-consumer proof.
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
