# Jupyter integration qualification (R4b)

Isolated synthetic experiment, not a production plugin or an App workspace package.
Read the [stage result](../../../docs/desktop-workspace/stages/r4b-live-integration.md)
and [study](../../../docs/desktop-workspace/stages/r4b-review/study.md) before reuse.

## Boundaries

- `component.ts` / `model.ts`: actual Jupyter model/editor and typed test observation.
- `style.css`: comparison shell and component styling, qualification only.
- `host.cjs`: disposable Electron sessions, exact TLS pin, restricted resources and
  explicit WebContentsView lifetime. No production preload or IPC.
- `server.py`: synthetic project, private auth/certificate, owned server cleanup.
- `protocol.py`: REST conflict and kernel message/reconnect/restart probes.
- `qualify.cjs`: build, native keyboard/UI assertions, captures and teardown.

The connected probe enables Jupyter debug application exposure to inspect public
model/editor/context APIs. It is not an implemented extension messaging bridge.
Neither global test API is suitable for a production preload.

## Reproduce

Requires App development dependencies and Python 3.12. Create `.venv` with that
interpreter and install `requirements.lock.txt`. Run `npm ci --ignore-scripts` in
this folder for its separate JS dependencies, then:

```sh
node qualify.cjs
```

`R4B_NATIVE_ONLY=1 node qualify.cjs` skips protocol probes for a focused native rerun;
it does not regenerate protocol evidence. Locks pin this tested local tuple. Setup
does not modify global Python or the App package lock. The harness removes temporary
data and its Electron profile on normal completion or caught failure. It writes
only R4b review evidence. Forced termination requires checking for its specifically
named temporary server resources.

Rich HTML/JS output, remote credentials, cross-window editing and general Jupyter
extension compatibility are outside this probe. Bundle sizes are unminified and
include qualification code; they are not a production-size forecast.
