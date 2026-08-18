# Go Mistakes

## Compose and Run must share one edge contract

**Context:** Related-file output `From` is legal at compose. Thin-slice BAI used it.

**Mistake:** `upstreamReady` only looked up inputs. Output-port `From` composed cleanly and deadlocked at run (`not-started`).

**Correction:** Resolve wait paths at `BuildPlan`. Run reads `Edge.Wait` only. Do not guess from `ToPort`. Add a process-level regression.

## Do not classify from a record this operation will overwrite

**Context:** Resume occupy overwrites `plan.json`. Inspect remaining can take an instance filter. Dest replace needs an owner.

**Mistake:** Classifying resume identity from the plan after occupy overwrites it. Filtering latest attempts before remaining classification. Treating any executed identity as dest owner.

**Correction:** Persist script and env on the attempt, or classify before replacing `plan.json`. Classify remaining on all latest attempts, then instance-filter emit. Attribute dests by checksum or producer lineage, not executed identity.
