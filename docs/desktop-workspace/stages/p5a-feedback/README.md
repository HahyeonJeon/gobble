# P5A — linked feedback and a fresh analysis

Implementation approved on 2026-09-12; implementation and verification completed on 2026-09-13. This stage completes the existing Flow → Chat → checked change → adoption → new analysis path for App-created single-end Trim Galore → FastQC and supported setting refinements.

## Observable behavior

- Select an observed task or log excerpt and discuss it in the existing composer. Open current design explicitly to authorize a supported proposal. Earlier Run Flow and evidence stay historical.
- The sent observations accompany the proposal. Change spotlight keeps its Current/Proposed visual comparison; original evidence opens through the existing captured-evidence preview.
- Adoption changes Current only. Review a new analysis leads back to preparation and data checks. The Chat confirmation discloses a fresh Run, all steps running, input-content comparison and a separate result location.
- After explicit Start, the new Run retains its earlier-analysis relation. The original Run and sent capture remain available after restart.

## Design and ownership

[Architecture and ownership diagram](architecture.md) records why provenance is stored atomically inside the proposal and launch records. Renderer owns navigation/presentation; Main derives the relation from a verified sent submission; native service validates the Project/Pipeline/Run relationship; Gobble owns execution. The Agent cannot supply arbitrary origin metadata through its proposal tool.

Current contract bundle: v26. Workspace v19, catalog v5 and shared toolset v13 remain unchanged. Frozen v25 and earlier schemas are preserved. No new IPC method, source editor, independent provenance store or engine execution policy is introduced.

[Verification record](verification.md) separates construction checks, native/Main behavior, actual Electron/engine evidence and untested claims.

## Scope limits

This relation does not establish why an analysis failed, that a suggested setting is scientifically effective, or that previous outputs can be reused. Only evidence attached to the actual proposal-request message participates. Multiple origins and another Pipeline are rejected for a linked proposal. Existing limits remain: 40 preserved Project messages, 8 retained Pipeline proposals, up to 16 attached evidence manifests and 48 KiB of follow-up metadata. Reaching a limit refuses new work; history is not silently pruned.

P5B (same-design continuation/Resume) remains a separate design and engine qualification stage. Existing prepared-Run Resume restrictions remain enforced. No real Agent provider, research data, release packaging, commit or publication is part of this acceptance run.

[Changed source paths](changed-paths.json) identify the P5A delta against the recorded P4 dirty-tree baseline; inherited changes have not been reset.

## Verified result

369 TypeScript tests across 39 files passed, along with native app-service race tests, static checks, type checking, current schema checks and the build. The final real-Electron workflow passed: synthetic first analysis → exact sent task capture → 40/20 bp checked visual proposal → adoption without execution → fresh preparation/check → explicit second analysis → restart and original capture preview. A separate real-Electron case checks missing-capture recovery, earlier design labeling, bounded Flow layout and compact Workspace/Chat switching. Agent responses use a deterministic peer; Gobble checks and both analyses execute for real.

[Visual comparison](evidence/feedback-comparison.png) · [New analysis confirmation](evidence/feedback-ready.png) · [Restored origin](evidence/feedback-origin-expanded.png) · [Compact work area](evidence/feedback-compact.png) · [Compact Chat](evidence/feedback-compact-chat.png)

P5A implementation is complete. The [P5B continuation design proposal](../../proposals/p5b-continuation/README.md) is ready for owner review; production Resume implementation has not started.
