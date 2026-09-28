# Owned pipeline-flow qualification

This development-only Project supplies a real branching Gobble pipeline for
P2A-1 and an explicit-settings example for P2A-2. It is not bundled application data, a production template or a runtime
release. Commands are deliberately `must-not-run`: inspection composes and checks
the graph without running tasks or creating a Run. The declared paths need no
research files for this inspection.

The local qualification lock records the exact cached runtime and daemon
used on 2026-09-09. It is machine-specific, ignored by Git and not a default for another Project.
Its exact bytes are retained as `runtime-lock.json` in the stage evidence.
The native test copies this entire Project into a temporary directory, adds a test
image, uses a separate App profile and removes only those owned temporary files.

## Runtime provenance

- Base cached image: `sha256:bccd458a724b795ac507e3dd0be43e8431269db2819fa544784d73dc751ba22b`.
- Local derived image: `sha256:1d4dccd578363c104aa65127e33a4d924f1f2e4a15ae3d9e6ee8ab4c46695adb`.
- Linux/amd64, Go 1.26.8 and cached module dependencies; no network downloads.
- The build context contains trusted repository root non-test Go files, module
  files, `cmd/gobble`, `internal`, `assets`, `distribution` and `monitor`, copied
  under `gobble/`. The Dockerfile compiles the new `gobble flow` command into the
  image. Its `/opt/gobble` also provides the library used by the fixture's module
  replacement. The base must provide `/usr/bin/timeout` (coreutils).
- The retained context inventory and image build output are in the
  [current stage evidence](../../../docs/desktop-workspace/stages/p2a2-flow-discussion/verification.md).

To rebuild, first prepare that explicit context from the reviewed repository and
assign a local tag to an already cached compatible base. Pass its local tag as
`BASE_IMAGE` to `docker build --network=none --pull=false --platform linux/amd64`
with this Dockerfile. Record the resulting image ID, daemon ID and exact context
hashes, and update only an owned qualification Project lock. This is an explicit
developer operation, never an App fallback installation.

## Qualification commands

From `app/`, after confirming that this owned lock resolves locally:

```sh
GOBBLE_FLOW_LIVE=1 npm run check
```

The normal `npm run check` skips the two live flow/discussion tests when that variable is
absent. Its ordinary native import/missing-runtime scenarios remain enabled.

For just the actual service-to-runtime check, from the repository root:

```sh
GOBBLE_FLOW_QUALIFICATION_PROJECT="$PWD/app/qualification/pipeline-flow/project" \
  go test ./internal/appservice -run '^TestLivePipeline(Inspection|ModuleSettings)$' -count=1 -v
```

The `trim-review` example uses the real Trim Galore module with quality 25 Phred
and minimum length 40 bp. Its declared inspection files are Project-relative;
no analysis command is executed. The native discussion test uses a protocol test
Agent to verify exact observation/mark policy and stale refusal. Actual signed-in
Agent qualification is recorded separately in the stage evidence. None of these
checks proves analysis execution.
