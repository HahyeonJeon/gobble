# A test toolchain image, not the distributed Gobble runtime.
ARG GO_IMAGE=golang:1.26-bookworm
FROM ${GO_IMAGE}

WORKDIR /opt/gobble-dependencies
COPY go.mod go.sum ./
RUN go mod download all

# Permission-sensitive tests must run without root's DAC bypass. Keep the
# verified module cache writable for temporary consumer-module builds.
RUN useradd --create-home --uid 10001 gobble && chown -R gobble:gobble /go
USER gobble

ENV GOTOOLCHAIN=local GOPROXY=off
CMD ["go", "test", "-count=1", "./..."]
