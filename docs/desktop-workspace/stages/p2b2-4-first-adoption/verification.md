# First adoption verification

Request: `p2b2-4-adoption-20260912` (Development → Testing), defined in README.

Initial failures retained by classification:

- Test defect: native fixture asserted four Agent-editable files, although only the one Pipeline source file is editable. The four-file retained bundle includes three immutable setup/input files. Corrected assertion.
- Test defect: live refinement reused the five-second mock-test polling budget. Use the existing native two-inspection-timeout job budget plus bounded observation margin; production timeout unchanged.
- Test execution defect: first Electron invocation used repository cwd rather than app cwd, resolving the launch target incorrectly. Stopped its owned processes, removed isolated copied credentials, then reran from app.
- Environment lookup: Docker tag inspection returned not-found despite listing the tag. Exact immutable image ID resolves and is the native qualification input. No retag or download.

Further visual/testing findings:

- Test defect: the Current graph contains one input and two steps; the initial assertion incorrectly counted every node as a processing step. Corrected to three nodes, retaining checked two-step assertions.
- Test defect: exact implicit label text included select options; model metadata lookup now uses its visible label without a false exact-name assumption.
- Test defect found by screenshot inspection: first real-Agent attempt reused an earlier step pointer and captured before navigation completed. Its real discussion-completion/Current screenshot claims are invalidated. The final scenario waits for the new setting-specific note, both completed replies and actual departure from the creation view.
- Product defect, routed Testing → Development: a long real-Agent note overflowed the creation panel. Added bounded wrapping, rebuilt, and reran affected visual scenarios.

- Test defect: completion status is lowercase in the DOM and capitalized by CSS. The real-Agent completion assertion now targets the actual status field. The rerun also verifies the retained setting target against Gobble's `Quality threshold = 25 Phred` metadata, not only its note text.

- Test defect: the existing refinement Electron scenario still expected catalog4. Its post-adoption/restart assertion is updated to catalog5, preserving the zero-Run and exact Current checks; the full affected case is rerun.

- Environment gap: the real account run lost foreground window authority, and Agent pointing correctly refused with Project window unavailable. Stopped that attempt and added actual native focus plus a document-focus assertion before each message, as in the deterministic fixture. The long live network turn lost focus again after the initial assertion. Final qualification maintains the owned native test window in the foreground until completion and clears that test-only driver on quit. No visibility-policy override or synthetic receipt is used.

## Lower-layer results

| Claim                             | Observation                                                                                                                                                                               | Result |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Atomic first adoption and retry   | Eight concurrent requests converge on one Pipeline; identical retry, conflicting intent, wrong Project, immutable copies, preserved sibling Current and birth after later Current/restart | passed |
| Refusal and recovery              | Stale generation/input/runtime/source/artifact, closed/unready candidates and cancelled adoption publish no Pipeline; failures before/after durable write reconcile from disk             | passed |
| Catalog compatibility             | v4 draft migration and byte-for-byte archive; new birth fields rejected inside frozen v4; older migration regressions and missing-primary refusal                                         | passed |
| Concurrency/static native checks  | Native service suite under race detector; vet for service and service command                                                                                                             | passed |
| Contracts/Main association        | 353 Vitest tests; strict success/refusal association, closed lifecycle result, existing Agent authoring policy regressions                                                                | passed |
| Construction                      | TypeScript checks, generated bundle23 consistency, development build and changed-source formatting                                                                                        | passed |
| Real Gobble creation → refinement | Exact pinned image, checked creation adopted, quality25→30 comparison with one explained change adopted as later Current; birth unchanged and zero Runs                                   | passed |

## Electron results

The final development bundle passes creation/discard and creation/adoption scenarios,
and all eight workspace regression cases (10 cases in `electron-regression.log`).
The live account case is separately passed after correcting the foreground test
environment: `real-agent/acceptance.json` identifies the model and isolated Project;
`real-agent/agent-exact-reference.json` binds the actual Agent mark to the checked
candidate's setting ID and artifact. Both responses completed before adoption.
The User created exactly one Pipeline, reopened Current after restart, retained
unsent Chat, and created no Run. The copied credential file was removed after the
test and is absent from evidence. The underlying research fixture bytes are unchanged.

The final existing-refinement case also passes: scoped proposal, exact Chat
reference, adoption, reopened Current and catalog5 ownership. Together the final
records establish **12 distinct passed Electron scenarios** (10 regression cases
plus the two corrected reruns), without claiming that earlier failing runs passed.

Visual review inspected the actual confirmation, completed real-Agent setting
reference and registered Current screenshots. The long Agent note now wraps within
the creation panel. A final read-only navigation check on the isolated real-Agent
profile records `real-agent/creation-history.png` and `visual-layout.json`: the
626-pixel note stays inside the 658-pixel panel, with normal wrapping. The central flow and right Chat remain primary; no additional
message composer, source editor or Run action was added.

## Result and next checkpoint

Part4 is implemented and verified for this recorded macOS development environment.
Return to the owner's result review. P3 execution preparation needs its own sketch,
engine-owned payload/identity definitions and explicit next-stage authorization.
No Run-control, general biological correctness, FASTQ-content validation, packaging,
installation, update or non-macOS qualification is established by these records.

Evidence identities: `before.json`, `source-delta.json`, `tested-build.json` and
`environment.json`. Native/type/unit/build logs are adjacent. Real-Agent artifacts
are in `real-agent/`; deterministic creation/workspace artifacts are in
`electron-regression/`; the corrected refinement case is in
`refinement-regression/`. Earlier attempts are retained and classified above. No packaged, installed, update, non-macOS or execution-control claim.
