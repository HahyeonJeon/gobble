# R4b verification

Date: 2026-09-09. Scope: isolated investigation, not production release acceptance.
Final command: `node app/qualification/jupyter/qualify.cjs` from the qualification
folder, with the existing App Electron/TypeScript tooling and isolated pinned Jupyter
Python/JS dependencies. The command launches actual Electron windows and a local TLS
Jupyter Server, then removes its temporary server, kernel data and Electron profile.

## Results

- Strict TypeScript qualification build passed; final full native/protocol run passed.
- Component: real keyboard selection and edit, UTF-16 mapping after an emoji,
  revision rejection, stable cell ID after reorder, second widget sharing the model,
  and document survival after that widget's disposal. No captured renderer errors.
- Connected Jupyter: document context and kernel ready, exact cell selection,
  renderer token presence measured as a boolean only, real save-conflict dialog,
  Cancel preserved external contents, and awaited WebContents destruction while the
  Jupyter session survived Pane closure.
- REST: stale If-Match saved with 200 and overwrote newer synthetic content. Missing
  file read returned 404; subsequent save recreated it with 201. These **observed
  unsafe defaults** are successful qualification findings, not accepted App behavior.
- Kernel: correlated output/display updates, disconnect after execution began,
  reconnect observed exactly one synthetic increment without original-request replay,
  and restart preserved REST ID while changing process epoch and losing variables.
- Actual screenshots were opened and visually reviewed. Candidate A fits the compact
  shared workspace. Candidate B still carries Jupyter controls and a news prompt;
  no production UX approval is inferred from this comparison.

Raw final results: `native-results.json`, `protocol-results.json`, `qualification.log`.
Source/dependency identity: `source-manifest.json`, qualification lockfiles.
Preservation: `preservation.json`. No production tests are represented as newly run.
R4a3's 320 unit checks and 65 distinct native scenarios remain predecessor evidence.

## Corrections and evidence limits

The first successful functional run used BrowserWindow.capturePage, which omitted
its child WebContentsView, producing blank connected captures. It also recorded
WebContents.isDestroyed immediately after requesting closure, before destruction
completed; no assertion caught that timing in the first harness. Initial results
and log are retained as `native-initial.json` and `qualification-initial.log`.

The harness now captures the child WebContents directly and awaits the destroyed
event with an assertion. It waits for a ready kernel before recording the connected
state. Native keyboard selection, explicit shared-widget lifetime and missing-file
probes were added. A focused native rerun and the final full run passed. Final
screenshots replace the invalid captures and were visually inspected. Type skeleton
construction initially needed an exact-optional property correction; no production
code was involved.

Screenshots are real renderer captures, not a complete operating-system screenshot
of the composite window. Candidate B's image intentionally shows the child surface.
Candidate A's Chat area is labeled a communication layout preview; no live Agent was
invoked for R4b. No independent reviewer, complete accessibility study, adversarial
output review, packaging verification or remote environment support is claimed.

## Isolation

TLS used a disposable self-signed certificate pinned to the exact loopback hostname
and leaf fingerprint in the connected Electron session; verification was not globally
disabled. Python clients verified the generated certificate. Authentication used a
private file and request headers, not a token in the browser URL. Private config/logs
were removed with the scratch directory. Persistent evidence contains no token.

Electron used sandbox, context isolation, no Node in renderers, no App preload or
IPC on the connected surface, denied permissions/popups/webviews and restricted
network origins. These settings contain the experiment; they are not a full security
qualification for arbitrary notebooks or production Jupyter credentials. The normal
App profile and existing engine/memory work were not used or edited.
