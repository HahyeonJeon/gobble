# Study and design discussion

Index: [Accepted design](ideation-index.md). Recommendations confirmed on 2026-09-28.

## Study

**S1 — Actual workflow and output.** The prior actual Electron walkthrough found
HTML displayed as source (`internal/appservice/files.go`, UTF-8 fallback). The
current bounded FastQC module declares `html` and `zip` outputs in
`assets/modules/fastqc/product.go`. The original qualified report is 529,856 bytes.
Structural inspection of its prior sketch copy (only an injected CSP differs)
found ten module blocks, eight substantive PNG charts, 29 image elements including
icons, no scripts and one external footer link. Text is small but graphs contain
material content. [Measurements](study.json) retain the distinction between original
bytes and the display-policy-modified copy. These are fixture facts, not universal
FastQC limits or representative research findings.

**S2 — Output authority.** `internal/engine/state.go`, `identity.go`, `run.go` and
`reuse.go` retain per-attempt output checksums and history. `monitor.go`/`inspect.go`
currently expose latest attempts without complete output attribution. Lineage has
paths/checksums but not the requested Run/attempt/status/port tuple. App cannot safely
join separate best-effort views to invent that tuple. `checkpoint.go` retains only
current/previous generations, so a historical checkpoint ID cannot be the saved
report's long-term address. `continuation_review.go` already checks declared outputs
against recorded checksums, demonstrating engine ownership of these facts. App-created tasks have no reliable
module-type field. The existing canonical FastQC recognizer returns `fastqc-v1`;
this recognition, bound to the admitted declaration, must accompany report eligibility.
Checkpoint runtime defaults require validated declaration reconstruction or a sealed
prepared-document tieback. A task name or `html` port alone is insufficient.

**S3 — Runtime compatibility.** `internal/appservice/runs.go` and `runtime.go` bind
registered Runs to exact engine image identities. A newly built image cannot silently
stand in for the recorded engine. A new output-evidence capability therefore needs a
fresh qualified Run; old engines return unsupported for the new report operation.
This restriction does not disable existing monitor/log behavior. Existing prepared
capability consumers reject unknown fields; a separate versioned output-capabilities
query preserves that launch/continuation contract.

**S4 — Existing evidence boundaries.** `app/desktop/src/main/evidence/storage.ts`
limits individual stored blobs to 1 MiB and Project storage to 64 MiB.
`evidence/service.ts` and `collaboration/coordinator.ts` both enforce a two-image
message limit. Ordinary JSON text has a 64 KiB budget. `shared-context/host.ts` bounds
calls/results per turn. Existing immutable draft/sent capture and hash validation
are the correct retention foundation. New report content must be represented as a
report, not disguised as text or one image, and every accounting consumer must agree.

**S5 — Visibility versus addressed evidence.** Existing workspace observations are
ready/foreground-view scoped. Sent evidence is durable and available independently
of visible Panes. The new Agent chart-read operation must check active submission
recipient and attachment, rather than bypassing visibility inside `workspace_observe`.
Main's UI-only `readSentEvidence` is not by itself an Agent authorization boundary.

**S6 — Official sources.** FastQC produces an HTML report
([saving documentation](https://www.bioinformatics.babraham.ac.uk/projects/fastqc/Help/2%20Basic%20Operations/2.3%20Saving%20a%20Report.html));
its reported module evaluations are distinct from process completion
([evaluation documentation](https://www.bioinformatics.babraham.ac.uk/projects/fastqc/Help/2%20Basic%20Operations/2.2%20Evaluating%20Results.html)).
Electron recommends sandboxing and withholding privileged capabilities from untrusted
content ([security](https://www.electronjs.org/docs/latest/tutorial/security),
[embedding options](https://www.electronjs.org/docs/latest/tutorial/web-embeds)).
This supports isolated bounded parsing and trusted-data rendering; it does not prove
that a new reader is safe without implementation tests. Consulted 2026-09-27.

**S7 — Design principles.** Coding Principles and Coding OOP preferences were studied:
reuse current concrete owners; no default design pattern, generic viewer registry,
inheritance hierarchy, event bus or chunk/child-blob store. New isolated reader entry
has a current sandbox boundary, not a hypothetical future plugin force.

## Independent discussion

Two read-only subagents supplied separate advice. `report_identity` examined engine
and native-service evidence, rejected private-checkpoint reading and highlighted
old-image compatibility. `report_sharing` examined rendering, budgets and recipient
permissions, recommending a single normalized report model with original chart reads.
Primary study independently measured the actual report and compared these options.
Both agents then critiqued the combined draft. The review narrowed fresh capture to
the latest successful attempt, kept exact historical logs explicitly unavailable when
unsupported, required authoritative format eligibility, preserved source status/icon
labels and inert footer text, and made current-message attachment scope explicit.
These corrections avoid implying historical output APIs or whole-chart observation
that the current contracts do not provide.

### D1 — Where does output attribution come from?

| Option | Advantage | Cost / flaw |
| --- | --- | --- |
| App parses private checkpoints or guesses output filename | Small initial change | Duplicates engine semantics; cannot safely claim correct producer identity. Reject. |
| Join monitor and lineage | Reuses existing APIs | Missing atomic exact-attempt/port attribution. Reject as proof. |
| Explicit Gobble output-evidence query | One narrow authoritative read boundary | New capability and new-engine qualification required. Recommend. |

Design destination: Conceptual Definition and `ReadOutputEvidence`/`ReadRunReport`
contracts in [design](ideation-01.md).

### D2 — What do User and Agent read?

| Option | Advantage | Cost / flaw |
| --- | --- | --- |
| Original HTML host plus separate Agent extraction | Original appearance | Two representations, isolation host and consistency validation. Viable alternative if original appearance is required. |
| FastQC-specific saved reading model | Same original content for UI and Agent, one validation owner | Supports a bounded dialect; original CSS layout not retained. Recommend. |
| Whole report screenshot | Little initial modeling | Eight charts cannot remain legible in one bounded image; first viewport is incomplete. Reject. |

Design destination: Saved report concept, `FastqcReader` and `ReportView`.
No scientific data or chart regeneration is implied by a reading model. Table-of-
contents navigation and chart IDs are content addresses within that rendition;
semantic module/metric references remain deferred.

### D3 — How is the whole report shared within current limits?

| Option | Advantage | Cost / flaw |
| --- | --- | --- |
| Increase image limits and attach everything eagerly | Agent gets all graphs at once | Broader provider/message policy and payload cost; unnecessary for current need. |
| Save one report, send full text plus inventory, read charts on demand | Whole content stays available; preserves limits | Requires recipient-scoped attachment-read tool and honest read-coverage wording. Recommend. |
| Send only path or summary | Very small payload | Agent lacks original evidence; violates shared-result outcome. Reject. |

Design destination: delivery/retention policy and `read_report` contract. Do not
silently truncate report content or claim every chart has been observed.

### D4 — What happens to already registered older-engine Runs?

| Option | Advantage | Cost / flaw |
| --- | --- | --- |
| Substitute a new engine invisibly | Existing reports appear immediately | Violates exact runtime binding. Reject. |
| Add a historical compatible reader | Could support old qualified results | Separate compatibility/authority work; defer. |
| Capability-aware unavailable state; new-engine/new-Run qualification | Preserves runtime contract | New report feature initially unavailable for older Runs. Recommend explicitly for owner approval. |

Design destination: compatibility and stage acceptance. Existing saved evidence,
monitoring and logs remain available under their current contracts.

## Session learnings and boundaries

- An exploratory source read named a nonexistent `history.go`; actual ownership was
  found in continuation history, state, checkpoint and reuse sources. No conclusion
  relies on that failed read.
- The measured HTML copy contains a CSP injection from the prior sketch. Original
  source size/hash are recorded separately; never use the modified copy's hash as
  engine producer evidence.
- Whole report scope is a preservation/read-access promise, not a single screenshot
  or an assertion that the Agent consumed every image. Make this distinction in UI.
- Report materialization must account for saved bytes separately from provider text.
  Merely adding a new schema union without updating both delivery guards would fail.
- Prior completed-run manifest verified before work. This checkpoint changed only
  draft documentation/status. No production code, test runtime, engine or dependency
  was changed; no new implementation qualification is claimed.
