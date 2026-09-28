# P2B-2.2 verification

2026-09-12. author mode used no credentials and performed no external mutation.
Local implementation, build caches, disposable service profiles and local Docker
qualification images were used within the approved development scope. Research
Projects, signed-in Agent accounts and existing App profiles were not mutated.

## Passed checks

| Evidence | Result |
| --- | --- |
| `go test -race -json ./internal/appservice ./internal/pipelinereview` | 61 service tests (93 including subtests) and 4 portable-review tests passed; no race report |
| `go vet ./internal/appservice ./internal/pipelinereview ./cmd/gobble-service` | Passed |
| Linux/amd64 selected engine/root/CLI suite | 24 top-level tests passed: engine4, public library12, CLI8; 167 including subtests |
| `npm --prefix app run typecheck` | Contracts, Main, preload, renderer and tools passed |
| `npm --prefix app test` | All 345 tests in 33 files passed, including real native-service consumers |
| `npm --prefix app run build` | Default native-service and Electron Main/preload/renderer build passed |
| Schema/format checks and `git diff --check` | Passed; existing frozen App schema bundle unchanged |

Raw [native race results](go-tests.jsonl) and [Linux results](linux-tests.jsonl)
retain test names and outcomes. Native selected toolchain: go1.27.1 darwin/arm64;
Go module language1.26. Linux tests ran offline with the installed pinned
Linux/amd64 runtime against the read-only current source checkout.

Native tests cover request replay/conflict, wrong Project, setup/path/extra-file
refusal, retained source integrity, interrupted/cancelled and stale publication,
draft update/discard, input metadata drift, runtime changes, restored runtime and
historical artifacts, combined check budget, strict native HTTP bodies and
malformed/mismatched graph evidence. Root/engine tests cover all four declared
FASTQ suffixes, complete module graphs, extra input/step/command, environment,
resources, control, params, executable identity, IO source, report, module grouping,
edge multiplicity and wait changes. CLI tests retain invocation/help separation
from execution controls.

## Actual isolated Gobble qualification

Final local development image:
`sha256:4dbd1fb658d8aa6967bf8de6b2368a0dad2c59244306d2a2bf1c9e90e4d702dc`.
The [local build log](runtime-build.log) records construction from the installed
P2B-1 image without dependency installation, publishing or an analysis Run.

Final actual-runtime test: **passed in 196.25 seconds**. See [live test log](live-check.log).
The test exercises scaffold export and native runtime selection, a complete
single-end proposal with quality25/length40, refusal of an unrepresented extra
command, and another generation selecting an uncompressed `.fastq` input. The
input files deliberately contain metadata-only fixtures, so this is evidence that
research bytes are not required by a design check, not data/scientific validity.

Two preceding actual-runtime runs passed before the input-descriptor refinement.
Their old runtime pin and timings are not substituted for the final-image result.
Three opt-in tests in the ordinary native suite are skipped and not counted as
passed: creation runtime, older live inspection and live settings. The creation
runtime test is executed separately with an explicit image pin.

## Diagnostics and bounded claims

An initial native engine test hit the existing macOS root-library build limitation
(`internal/containerenv: undefined useProjectOwner`). Engine/CLI checks were moved
to their supported Linux/amd64 target; the unrelated platform implementation was
not modified. The first restart fixture attempted to close its owned service
twice; fixture cleanup now observes its lifetime. Production close semantics were
not weakened. A raw `sha256:` Docker FROM argument was initially interpreted as a
registry name and failed metadata resolution; the local image tag was used and its
resolved exact digest verified. No new base image was downloaded.

No new UI or Agent integration is claimed. Electron visual tests were not rerun
for this native-only slice; App source/test/build verification and actual isolated
Gobble tests are its evidence. Prior B visual/adoption evidence remains historical.
This is not a signed-in LLM run, first-creation adoption, execution preparation,
Start/Stop/Resume, FASTQ-content validation, packaging, or resolution of the two
previously recorded packed-CLI engine baseline failures.
