# Live Notebook integration — Study

> **Result role:** Requested saved study and qualification conclusion.<br>
> **Question:** Should Gobble App reuse Jupyter components or host a connected Jupyter surface?<br>
> **Purpose:** Choose the boundary before production editing and execution.<br>
> **Consumer:** Project owner reviewing R4b and the proposed R4c scope.<br>
> **Criteria:** One document authority; precise references; Project/Panes/Chat fit; save correctness; credential isolation; recovery; maintainable scope.<br>
> **Scope:** Two isolated Electron candidates, synthetic local notebooks, REST and kernel probes. No production integration or CSV chart creation.<br>
> **Evidence boundary:** Official Jupyter/Electron documentation, pinned source and runnable qualification. Stop when decision-critical comparisons and limitations are explicit.<br>
> **As of:** 2026-09-09; JupyterLab 4.6.3, Server 2.21.0, ipykernel 7.3.0, Electron 44.2.0, macOS arm64. Locks: `app/qualification/jupyter/`.<br>
> **Output boundary:** Study advises. The owner retains design acceptance and authorization of R4c.

## Conclusion

Prefer **reusable Jupyter NotebookModel/editor components** inside the existing
Project interaction model. Both candidates support exact cell/source selection.
Component reuse gives the App a focused interaction surface to own and allows
connection credentials to stay outside the editor renderer. That credential
boundary is an architectural opportunity, **not a verified production transport**.

Connected JupyterLab supplies more ready-made behavior, including a working file
conflict dialog. It also retains another application shell, broad server capability
and renderer-visible authentication state in this tested setup. Trimming appearance
does not reduce those capabilities. It remains a credible future option for projects
that need a full Jupyter environment.

## Recommendation

> **Best-supported direction:** Reuse the document model and editor; separate App-owned presentation, references, file authority and connection operations.<br>
> **Next action:** Review R4c1: live document identity, source editing, exact unsaved references and explicit **Save a copy**. Kernel connection/execution follows a separate R4c2 checkpoint.<br>
> **Reconsider when:** Full JupyterLab extensions or remote collaboration become essential, or a focused service adapter proves disproportionate to a scoped connected extension.

Do not implement a second editable JSON copy behind the Jupyter model. Do not promote
qualification debug globals or raw server routes into production APIs. In-place
save is deferred until conflict guarantees and external-writer limits are accepted.
A hash preflight alone cannot be described as an atomic conditional save.

## Evidence

| Claim                                                                                                                                                                                         | Type                                         | Source                                                                                                                       | Assessment and effect                                                                                                                  |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Models, contexts and widgets are distinct. Widgets can share a model; independent models for one path are not automatically synchronized.                                                     | Verified documentation and model probe       | [Jupyter document architecture](https://jupyterlab.readthedocs.io/en/stable/extension/documents.html); `native-results.json` | Reuse one document session for linked Panes.                                                                                           |
| Native keyboard selection after an emoji maps to the exact Korean substring. Keyboard edits update the model, old references fail after revision changes, and cell identity survives reorder. | Verified native experiment                   | `native-results.json`; `component-selection.png`                                                                             | Stable cell identity and explicit UTF-16 offsets work. This small probe is not a production validator or complete Unicode suite.       |
| Connected Jupyter detects an external change when context.save runs. Cancelling preserves the external file.                                                                                  | Verified native experiment                   | `native-results.json`; `connected-save-conflict.png`                                                                         | Contrary evidence favoring connected Jupyter: useful save UX already exists.                                                           |
| Server 2.21.0 accepted stale If-Match PUT with 200 and replaced newer content. A save after deletion recreated the file with 201.                                                             | Verified REST experiment                     | `protocol-results.json`; [Contents REST API](https://jupyter-server.readthedocs.io/en/stable/developers/rest-api.html)       | Generic Contents.save is not compare-and-swap and does not inherently refuse missing files in the tested default manager.              |
| Context checks hashes before a separate save request and has a missing-file save branch.                                                                                                      | Verified pinned source                       | `@jupyterlab/docregistry/lib/context.js`, `_maybeSave`; `source-manifest.json`                                               | Client checking helps but leaves a check/write race. Other providers may offer stronger contracts.                                     |
| The connected renderer page configuration contained the server token despite Main injecting authentication.                                                                                   | Verified native experiment and pinned source | `native-results.json`; JupyterLab `labapp.py`; `source-manifest.json`                                                        | No secret is retained in evidence. This finding concerns the tested local configuration, not every JupyterHub/auth setup.              |
| Display updates reuse a display ID. Replies correlate to a request. Restart preserved REST kernel ID but changed message-session identity and cleared variables.                              | Verified protocol experiment                 | `protocol-results.json`; [Jupyter messaging](https://jupyter-client.readthedocs.io/en/stable/messaging.html)                 | Distinguish request identity, output revision and kernel epoch. Kernel ID alone does not prove continuity.                             |
| A disconnected request still incremented a counter once. Reconnect observed the effect without replaying the original request.                                                                | Verified protocol experiment                 | `protocol-results.json`                                                                                                      | Disconnect means unknown outcome. The controlled counter is not general recovery for arbitrary side effects.                           |
| Closing the connected view destroyed its WebContents while its Jupyter session survived. Closing a linked widget preserved its model.                                                         | Verified lifecycle probes                    | `native-results.json`; [Electron WebContentsView](https://www.electronjs.org/docs/latest/api/web-contents-view)              | Pane closure, document closure and kernel shutdown are separate actions.                                                               |
| Component reuse fits the accepted App boundary better.                                                                                                                                        | Supported inference                          | Evidence above; accepted Project/Panes/Chat design                                                                           | It still requires service adaptation and bounded output/lifecycle work. No lower total cost for a full Jupyter replacement is claimed. |

## Alternatives

| Direction                                             | Evidence                                         | Trade-offs                                                                                                                                                                                                                                                        | Prefer when                                                 |
| ----------------------------------------------------- | ------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| Reusable model/editor                                 | Candidate A, real model and keyboard probes      | Fits App UI and narrow authority; needs focused IO/session integration. Development bundle: 6,039,777 bytes JS, 220,204 bytes CSS, 1,888 input modules. This is not shipping size. Jupyter brings React 18 alongside App React 19: isolate build/style ownership. | Current focused Notebook collaboration.                     |
| Connected Jupyter with an actual extension bridge     | Candidate B, real context/kernel and conflict UI | Mature environment; broader authority, auth and second-shell management. CSS trimming still left controls and a news prompt in this probe.                                                                                                                        | Full environment compatibility outweighs compact ownership. |
| Existing saved reader plus opening Jupyter separately | Accepted R4a behavior                            | Lowest integration scope; no shared unsaved/live selection.                                                                                                                                                                                                       | Fallback when live integration prerequisites are unmet.     |

## Limits

| Gap or conflict                                                                                                             | Effect                                                                      | Resolve or reconsider when                 |
| --------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- | ------------------------------------------ |
| A has no production IO, preload or kernel bridge.                                                                           | Credential separation is a design opportunity, not an end-to-end result.    | R4c contract and implementation review.    |
| B uses expose_app_in_browser and test-side inspection, not a deployed extension.                                            | Selection reachability is proven; authenticated extension messaging is not. | Before production connected Jupyter.       |
| Synthetic local Python only; no remote auth, SSH, containers, Conda, Windows or Linux.                                      | No support/packaging claim for those environments.                          | Before expanding the support tuple.        |
| Linked-model probe creates a second widget; no multi-window editing or cross-device collaboration.                          | R4c1 should keep one editable session in the Project window.                | Separate transfer/shared-document design.  |
| No widget comms, arbitrary HTML/JS outputs, debugging, completion, large notebooks, IME composition or accessibility audit. | Do not expose all standard renderers as an accepted App feature.            | Focused qualification for each capability. |
| Stale If-Match proves default-server behavior, not impossibility of stronger storage.                                       | Do not generalize to conditional custom providers.                          | Concrete storage selection.                |
| No independent reviewer or representative-user usability study.                                                             | Visual findings are the implementing assistant's review.                    | Owner review and later user tests.         |

## Verification

| Check                 | Result                                                                                      |
| --------------------- | ------------------------------------------------------------------------------------------- |
| Question and criteria | Answered by both native candidates and separate protocol probes.                            |
| Citations             | Decision-critical documentation reopened; corresponding pinned source inspected and hashed. |
| Boundaries            | Facts, inference and deferred implementation are separated.                                 |
| Execution evidence    | Final full qualification passed; see `verification.md` for corrections and scope.           |
