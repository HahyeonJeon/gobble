# Docker test environments

The image supplies Go and verified, cached dependencies and runs tests as an
unprivileged user so permission checks exercise normal filesystem access.
Mount the checkout at runtime so
Git metadata and the pipeline source identity remain available. Build it from
the repository root:

```sh
docker build -f tests/docker/Dockerfile -t gobble-tests .
docker run --rm --network=none \
  --mount "type=bind,src=$PWD,dst=$PWD" \
  --env GIT_CONFIG_COUNT=1 --env GIT_CONFIG_KEY_0=safe.directory \
  --env "GIT_CONFIG_VALUE_0=$PWD" \
  --workdir "$PWD" gobble-tests
```

Run a second Linux userspace by rebuilding with
`--build-arg GO_IMAGE=golang:1.26-trixie`. Image construction needs network access;
the mounted hermetic suite runs with network disabled and no Docker socket.
The checkout should be a complete clone. Tests that build consumer binaries need
readable Git metadata. Generated fixture caches stay outside version control.
The Git settings trust only the explicitly mounted checkout, whose owner can
differ from the test image's user.

CI also pins each test container to one available CPU. Assay command doubles
use small fixture resource requests; real pipeline defaults remain covered by
plan tests. The temporary packed-runner consumer intentionally lacks `go.sum`,
so that test disables fresh checksum-database lookups after dependencies have
been verified during image construction. Product checksum verification remains
enabled.

For race checks, replace the default command:

```sh
docker run --rm --network=none \
  --mount "type=bind,src=$PWD,dst=$PWD" \
  --env GIT_CONFIG_COUNT=1 --env GIT_CONFIG_KEY_0=safe.directory \
  --env "GIT_CONFIG_VALUE_0=$PWD" \
  --workdir "$PWD" gobble-tests \
  go test -race -count=1 ./internal/engine/... ./monitor/...
```

## Actual containers

The hermetic image does not prove that Gobble works with a real Docker daemon.
The separate `docker-smoke` CI job executes the live engine smoke tests on a
Linux host. To run the same tests on a configured host:

```sh
docker info
go test -tags=live -count=1 -run '^(TestRunDocker(Publishes|BadImageContained)|TestDockerLiveControllerDeathRecovery)$' ./internal/engine
```

This suite covers Docker publication/log collection/cleanup, a contained image
failure, and controller death around container creation/start followed by
Release/Resume. The separate `runtime-install` job builds the runtime and native
launcher, then checks init/doctor, live logs, Stop, repeated Stop, controller
death, and automatic Resume without host Go. Broader installed generic/packed
Docker recovery, daemon loss, and Windows Desktop remain additional gates in the
[v0.2.0 plan](../../docs/v0.2.0/plan.md). The existing broader installed suite is
`go test -tags=live ./tests/install-e2e`; it has fixture/network prerequisites
documented in [Provenance](../../docs/provenance.md).

The runtime launcher mounts the project and translates task workspace paths
through the controller's inspected mounts. Do not copy the hermetic command
and add a Docker socket while leaving temporary work in private `/tmp`.

Both `docker-smoke` and `runtime-install` passed on Linux in the
[first Docker CI run](https://github.com/HahyeonJeon/gobble/actions/runs/33964259175).
That run also exposed environment assumptions in the hermetic matrix, which
the fixture, user, and checksum setup above address. Linux container results
do not validate Windows Desktop behavior.
