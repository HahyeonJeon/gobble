# P2A-1 — Verification evidence

Date: 2026-09-09. Final implementation and author self-review, not an independent
review or release qualification. No user research Project, normal App profile,
signed-in Agent, source application or analysis execution was used.

## Final checks

| Check                        | Result and evidence                                                                                                                                                                                                                                                                                            |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Full App check               | `GOBBLE_FLOW_LIVE=1 npm run check` exited 0: formatting, process TypeScript checks, schema generation consistency, 328 unit/contract tests in 30 files and 68 native Electron scenarios. [Final log](app-check.log).                                                                                           |
| Native service concurrency   | `go test -race ./internal/appservice ./cmd/gobble-service -count=1` exited 0. [Log](service-tests.log). The opt-in live service test is separate from this ordinary race suite.                                                                                                                                |
| Native service static checks | `go vet ./internal/appservice ./cmd/gobble-service` exited 0. [Log](service-vet.log), empty on success.                                                                                                                                                                                                        |
| Native service boundary      | `go list -deps ./cmd/gobble-service` contains neither the root Gobble package nor its internal engine. [Dependency inventory](service-dependencies.txt).                                                                                                                                                       |
| Gobble/CLI Linux checks      | Focused `TestPipelineInspection`, `TestBuildPlan`, `TestDriver`, `TestParse` and `TestHelp` matching tests in root and `cmd/gobble` passed in the cached Linux/amd64 Go 1.26.8 image. [Log](linux-tests.log).                                                                                                  |
| Actual service integration   | Opt-in `TestLivePipelineInspection` passed in 41.54 seconds: five real steps, seven exact connections, zero Runs and a retained artifact. [Log](live-service.log), [actual flow JSON](actual-flow.json).                                                                                                       |
| Final native integration     | The full App check's live flow scenario passed in 50.9 seconds using the final service/runtime invocation, including its container-local timeout. [Full log](app-check.log), [actual screen](pipeline-flow.png), [compact screen](pipeline-flow-compact.png).                                                  |
| Historical preservation      | Previous schema bundles and frozen storage match before hashes; P1 evidence and the accepted visual proposal match their recorded hashes. Engine files and package dependencies match the before hashes. Exported historical discussion/tool/storage schemas keep their meaning. [After manifest](after.json). |

The first complete App run passed 327 tests/68 native scenarios before the final
two-Pane reuse review. Its [log](app-check-before-final-review.log) is retained as
intermediate evidence. After consolidating reuse policy, adding its regression,
scoping late import completion and consolidating styles, the complete suite was
run again. There is no mixed claim that the earlier build is the final one.

## Native scenario and effects

The test copies `app/qualification/pipeline-flow/project` into an owned temporary
directory and launches the real built Electron app/native Go service with a fresh
profile. It uses the native chooser boundary to register `rnaseq`, then:

1. Keeps an unsent Chat draft and opens an illustrative quality image in the second Pane.
2. Imports the existing analysis without entering Go, package or command text.
3. Requests Check flow through real validated IPC/service/Docker. Observes two
   declared inputs, five steps and seven connections; checks the retained catalog
   has no Run.
4. Activates a step with the keyboard, reads its actual memory/output declarations,
   follows an exact port connection and switches the connected list/diagram.
5. Opens the same checked pipeline in the second Pane, verifies both views, closes
   only the duplicate and keeps the original image tab/draft.
6. Invalidates only the copied inspection setup, retries and verifies the prior
   artifact ID remains authoritative.
7. Normally closes/reopens the owned App profile. Confirms saved flow, draft and
   second Pane, then inspects readable details at 940×760 native bounds.
8. Checks source bytes were not edited and removes only its temporary Project/profile.

The built screenshot's flow is real inspection output; task commands are
`must-not-run`. The lower chart is an existing static image fixture, not CSV chart
creation or a result generated by this analysis. Agent roster zero is intentional:
P2A-2 must separately prove pipeline observations and references with an Agent.

## Runtime and reproduction

Qualified host: macOS 26.5.2 arm64, Node 25.7.0, Electron 44.2.0, React 19.2.8,
TypeScript 5.9.3 and host Go 1.27.1. Project module declares Go 1.26. Inspection
executes under Linux/amd64 Go 1.26.8 with cached dependencies and network disabled.

- Cached base image: `sha256:bccd458a724b795ac507e3dd0be43e8431269db2819fa544784d73dc751ba22b`.
- Derived local inspection image: `sha256:7f5b4a590b2afd5d96894dd1a93108f959a88bbdd6c335fd675a09ec0b07a71e`.
- Build-context hash: `1f38d61973e118fbe5a0eb8e3e2b42442485aae637877dd260b180d03ab6459e`;
  [246-file inventory](runtime-build-source.json), [build log](runtime-build.log).
- [Owned fixture, Dockerfile and explicit reproduction instructions](../../../../app/qualification/pipeline-flow/README.md).

Focused engine check from the repository root used the cached base image:

```sh
docker run --rm --pull=never --platform linux/amd64 --network=none --cpus=1 \
  --mount "type=bind,src=$PWD,dst=/workspace,readonly" --workdir /workspace \
  --env GOMAXPROCS=1 --env GIT_CONFIG_COUNT=1 \
  --env GIT_CONFIG_KEY_0=safe.directory --env GIT_CONFIG_VALUE_0=/workspace \
  sha256:bccd458a724b795ac507e3dd0be43e8431269db2819fa544784d73dc751ba22b \
  go test . ./cmd/gobble \
  -run '^Test(PipelineInspection|BuildPlan|Driver|Parse|Help)' -count=1
```

The actual service test can be run from the repository root with the fixture
Project environment variable described in the fixture README. The full App check
uses `GOBBLE_FLOW_LIVE=1`; without it the live flow scenario is explicitly skipped,
not reported as qualified.

## Scope of evidence and remaining limits

- Unit/service tests cover endpoint identity, duplicate/cycle/unknown-field
  rejection, actual port/fan-out semantics, copied ownership, old Plan serialization,
  branch/merge/scatter/gather/condition/tree facts, changed source identity,
  cancellation/late results and Project confinement. These are distinct from the
  non-empty native scenario.
- Actual container execution demonstrates the configured bounded invocation and
  successful cleanup. Cancellation/stale publication are tested at the service
  boundary. A full 110-second timeout expiration and disk-full/crash fault injection
  were not run; do not describe them as native end-to-end qualifications.
- The public root library still does not compile natively on this Mac because of
  the pre-existing Linux `useProjectOwner` boundary. Root semantic tests ran in
  Linux; the native service remains engine-free. No whole-engine native pass is claimed.
- Earlier packed-runner failures `TestPackPrintpipeArtifact` and
  `TestPackHostpipeEmptyInspectProtocol` remain outside this slice. Focused Plan/
  inspection checks do not resolve them.
- The local runtime is development qualification only. Arbitrary pipelines,
  large research data, all control visualizations, runtime installation, signed-in
  Agent flow tools, source edits and execution are not established by these tests.
- Early local attempts exposed disposable scratch needing `exec`, invalid fixture
  memory spelling and native macOS path alias handling. Those were corrected before
  the final checks. An early [failed fixture log](actual-flow.log) is retained; it
  is not the successful service/native result.

No commit, push, package, signing, publication or release was performed.
