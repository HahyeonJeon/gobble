---
name: gobbi
description: "Gobbi is the read-only entry operation that establishes and routes General, Cowork, or Workflow session state."
allowed-tools: Read, Grep, Glob, Bash, AskUserQuestion
skill-type: operation
---

# Gobbi

Gobbi establishes the manager's durable entry state and routes it without mutation. Use it at session start
and after any boundary that may discard manager context.

## Principles

### Load the durable foundation

Read the canonical principles, project rules, role, and entry owners before governed action. Runtime memory
and a surviving task list do not replace these sources.

### Let the user select the mode

General, Cowork, and Workflow make different commitments. Present all three at fresh entry and let the user
choose.

### Preserve proved entry state

Keep a validated mode, slug, partner policy, and root pair across a context boundary. Ask again only when
evidence is missing, ambiguous, or conflicting.

### Leave work to its owner

Gobbi owns entry and routing only. The selected mode owns session state, and task skills own their work.

## Rules

- **NEVER mutate from Gobbi entry.** Create no branch, worktree, session record, artifact, configuration, or
  implementation.
- **MUST obtain an explicit mode selection at every fresh entry.** Use `AskUserQuestion` in Claude Code,
  `request_user_input` in Codex, the official Ask questions tool in Cursor (identifier pending), or
  `ask_user_question` in Grok; a recommendation cannot select the mode.
- **MUST validate one Gobbi root pair and load the entry foundation before routing.** Hold the pair unchanged
  for the session and carry it into every specialist brief.
- **MUST preserve skill ownership.** References expose owners but do not load them, and task triggers still
  decide which task skill applies.
- **MUST apply the session-wide finding gate.** Every correction receives fresh evaluation, and only a verified
  PASS continues automatically.
- **MUST keep the manager as the only authority for assignment, scope, user decisions, acceptance, and
  external or destructive action.** Build specialist prompts through Delegation and keep writes in one ordered
  chain.

## Procedure

### Phase 1 — Establish the Entry

#### 1.1 Resolve the Gobbi roots

- Take the loaded Gobbi skill path reported by the runtime. Treat that path and its parent as the only two
  candidate `{gobbi-skills-root}` values, and derive each candidate's sibling `agents/` directory.
- Accept exactly one candidate pair that resolves all three readable sentinels:

  | Sentinel | Proves |
  |---|---|
  | `{gobbi-skills-root}/gobbi/SKILL.md` | The entry skill resolves from the skills root. |
  | `{gobbi-skills-root}/principles/SKILL.md` | A sibling skill resolves from the same root. |
  | `{gobbi-agents-root}/manager.md` or `{gobbi-agents-root}/claude/manager.md` | Role contracts resolve from a runtime-flat or runtime-folder agents root. |

- Expand and record the accepted pair with the runtime and entry trigger. Re-derive it after every context
  boundary; stop with both observations when no pair, two pairs, a partial pair, or a changed pair appears.

#### 1.2 Resolve the project layout

- Derive the project key with
  `basename(dirname(git rev-parse --path-format=absolute --git-common-dir))`. Accept at most 64 lowercase
  alphanumeric or hyphen characters matching `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`; ask before deriving paths
  when it fails.
- Use this layout:

  ```text
  .gobbi/                          tracked
  ├── .gitignore                   tracked
  └── projects/<project>/          tracked
      ├── memory/                  tracked
      ├── sessions/                ignored
      └── worktrees/               ignored
  ```

  `.gobbi/.gitignore` owns these exact runtime-state entries:

  ```text
  # Gobbi runtime state. Session evidence and linked worktrees are never tracked.
  projects/*/sessions/
  projects/*/worktrees/
  ```

- Gobbi writes none of this layout. A selected owner may bootstrap only the namespace roots and ignore file
  when authorized; it creates no category, session, marker, or `rules/` path until that path is needed.

#### 1.3 Report missing prerequisites

- Run the read-only [prerequisite checker](scripts/check-prerequisites.sh) from the active repository or
  worktree. It reports `PASS`, `WARN`, and `FAIL` for these project-local conditions:

  | Scope | Required observation |
  |---|---|
  | Claude Code | Team environment and display settings, role and skill discovery, and entry permissions. |
  | Codex | Agent definitions, agent settings, skill discovery, and instruction entrypoints. |
  | Grok | Agent definitions under `.grok/agents`, skill discovery, and the `.agents/agents` root-pair sibling. |
  | Cursor | Agent definitions under `.cursor/agents`, skill discovery under `.cursor/skills`, and the `.cursor` pair. |
  | Gobbi | The project namespace and tracked or ignored state, including effective `.gitignore` ownership. |
  | All runtimes | Installed `claude`, `codex`, `cursor-agent`, and `grok` CLIs respond to a version probe. |

- For plugin consumers, recommend namespaced permissions such as `Agent(gobbi:leader)` and
  `Skill(gobbi:principles)`; repository-local Claude skills use bare names. Partner availability belongs to
  the [Partner Manual](partner/SKILL.md#availability).
- Continue after reporting ordinary missing configuration. Stop before routing when root resolution or layout
  evidence is partial, contradictory, unreadable, or unsafe to repair without the user's decision.

#### 1.4 Load the entry foundation

- Read [Principles](../principles/SKILL.md), [Discussion](../discussion/SKILL.md), and
  [Delegation](../delegation/SKILL.md), in that order.
- Read applicable repository instructions, every applicable project rule, and the canonical
  [manager role](../../agents/manager.md) for Claude and Grok. Codex custom agents are not a plugin component; load the project `.codex/agents/manager.toml` when that file exists. Cursor custom agents are not a plugin component; load the project `.cursor/agents/manager.md` when that file exists. Record `NO_PROJECT_RULES: rules/ absent-or-empty` when the rules
  directory is absent or empty.
- Confirm the foundation and fixed root pair. Defer every other skill to the selected mode or its own trigger.

### Phase 2 — Select and Route the Mode

#### 2.1 Obtain or preserve the mode

- At fresh entry, use Discussion and the active structured input control to present all three choices:

  | Mode | Use when | Commitment |
  |---|---|---|
  | **General** | Ordinary assistance needs no Gobbi lifecycle. | Task owners decide participants and evaluation. |
  | **Cowork** | The user wants bounded topics with Fast or Light delivery. | The user controls topic decisions, evaluation calls, and closure. |
  | **Workflow** | Work needs durable phase checkpoints and autonomous delivery. | Phase 1 closes user decisions; later phases continue or stop from accepted evidence. |

- After selection, publish the selected owner's complete native TODO template before asking for a slug or
  partner policy. General publishes no Gobbi TODO; Cowork and Workflow supply their own fixed templates.
- Across a boundary, preserve a validated selection. Ask again only when mode evidence is missing, ambiguous,
  or conflicting.

#### 2.2 Resolve the slug and partner policy

- For Cowork or Workflow, warn that the slug enters paths and branch names. Ask for the slug and session-wide
  partner policy together through one structured request; General records `slug: not-applicable` and asks only
  for the policy.
- Normalize the slug by lowercasing each maximal ASCII alphanumeric sequence, joining sequences with one
  hyphen, and trimming separators. Do not transliterate, truncate, or append a suffix; accept 1–20 characters
  matching `^[a-z0-9]+(?:-[a-z0-9]+)*$` and reject Windows device names from `con`, `prn`, `aux`, and `nul`
  through `com1`–`com9` and `lpt1`–`lpt9`.
- Ask one Partner policy with that slug: `disabled`, or a multi-select of `{claude-code, codex, cursor, grok}`
  limited to one or two names. Record `disabled` as that word, or the distinct names in lexicographic order
  joined by one comma and no spaces. Valid values are `disabled`, `claude-code`, `codex`, `cursor`, `grok`,
  `claude-code,codex`, `claude-code,cursor`, `claude-code,grok`, `codex,cursor`, `codex,grok`, and
  `cursor,grok`.
- Record mode, normalized slug when applicable, and that one Partner field together. A named set authorizes
  launch of the selected names minus the active runtime; `disabled` authorizes no launch. A value outside the
  grammar, including recovered `enabled`, is invalid and a stop; recovered `disabled` stays valid. Do not add
  a second policy field or rewrite an empty launch set to `disabled`.

#### 2.3 Apply the session-wide finding gate

- Correct a finding automatically only when its severity is High, Medium, or Low; `blocking: no`; it stays
  inside the locked contract; and it is reversible, authority-neutral, non-destructive, and non-external.
- Send every other finding to the user in General, Cowork, and Workflow Phase 1. After a Complete Workflow
  Phase 1 handoff, the manager decides from the accepted design, authority, available subagents or teammates,
  and remaining Partner runtimes, or writes a stopped handoff without asking the user.
- Run fresh evaluation after every correction. Continue automatically only from a verified PASS.

#### 2.4 Hand off the selected route

- Hand the complete entry state to one owner:

  | Mode | Handoff |
  |---|---|
  | **General** | Mode, `slug: not-applicable`, and partner policy; no orchestration owner or session state. |
  | **Cowork** | Mode, normalized slug, partner policy, runtime, and validated root pair to [Cowork](../cowork/SKILL.md). |
  | **Workflow** | Mode, normalized slug, partner policy, runtime, and validated root pair to [Workflow](../workflow/SKILL.md). |

- Before specialist work, load Delegation, add the selected owner's fields, and resolve every role and skill
  from the validated root pair. Load further task skills only when their triggers apply.
- Stop with the exact blocker when mode evidence, owner evidence, identity, path, or authority is invalid.
  Never invent a fallback mode, cursor, worktree, session directory, or participant route.

## References

| Name | Description |
|---|---|
| [Principles](../principles/SKILL.md) | Defines the behavioral foundation loaded at entry. |
| [Discussion](../discussion/SKILL.md) | Defines structured questions, evidence-backed options, and user decisions. |
| [Delegation](../delegation/SKILL.md) | Defines every specialist prompt and final Handoff. |
| [Manager role](../../agents/manager.md) | Defines session authority, routing, assignment, and acceptance for Claude and Grok plugin consumers. Codex roles load from the project `.codex/agents/manager.toml`, not from this package. Cursor roles load from the project `.cursor/agents/manager.md`, not from this package. |
| [Cowork](../cowork/SKILL.md) | Owns user-led bounded topics, explicit evaluation, and explicit closure. |
| [Workflow](../workflow/SKILL.md) | Owns checkpointed phases and autonomous continuation after Phase 1. |
| [Partner](partner/SKILL.md) | Defines each write-bounded opposite-runtime invocation. |
| [Agent Teams](agent-teams/SKILL.md) | Defines Claude Code teammate coordination and context-aware re-delegation. |
| [Prerequisite checker](scripts/check-prerequisites.sh) | Checks project-local Gobbi, Claude Code, Codex, Cursor, Grok, Git-ignore, and CLI prerequisites without mutation. |
