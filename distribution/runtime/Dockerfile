# Build from an exact Git checkout. This is the distributed authoring runtime,
# distinct from tests/docker/Dockerfile, which contains only a test toolchain.
ARG GO_IMAGE=golang:1.26-bookworm
ARG DOCKER_IMAGE=docker:28-cli
FROM ${DOCKER_IMAGE} AS docker-client
FROM ${GO_IMAGE} AS build
WORKDIR /opt/gobble
COPY . .
RUN test -d .git && go mod download all
RUN go build -o /usr/local/bin/gobble ./cmd/gobble
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /launchers/linux-amd64/gobble ./cmd/gobble-container \
 && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /launchers/windows-amd64/gobble.exe ./cmd/gobble-container

FROM scratch AS launcher-artifacts
COPY --from=build /launchers/ /

FROM build AS runtime
COPY --from=docker-client /usr/local/bin/docker /usr/local/bin/docker
RUN git config --system --add safe.directory /opt/gobble \
 && git config --system --add safe.directory /gobble/project \
 && mkdir -p /gobble/project
ENV GOTOOLCHAIN=local GOPROXY=off GOBBLE_SOURCE=/opt/gobble
WORKDIR /gobble/project
ENTRYPOINT ["/usr/local/bin/gobble"]
