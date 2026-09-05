# Docker test environments

The image supplies Go and cached dependencies. Mount the checkout at runtime so
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
Release/Resume. Installed generic/packed Docker recovery, live streaming logs, daemon loss,
and Windows/WSL are additional gates in the
[v0.2.0 plan](../../docs/v0.2.0/plan.md). The existing broader installed suite is
`go test -tags=live ./tests/install-e2e`; it has fixture/network prerequisites
documented in [Provenance](../../docs/provenance.md).

A future controller-container live harness must mount the workspace at the
same absolute path seen by the daemon and put temporary task workspaces on that
mount. Do not copy the hermetic command and add a Docker socket while leaving
temporary work under the controller's private `/tmp`.

The Dockerfile and CI jobs were added in this review environment, which has no
Docker CLI or daemon. Their actual image builds and container runs are pending
CI execution. Linux containers do not validate native Windows or WSL behavior.
