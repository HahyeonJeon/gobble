# A test toolchain image, not the distributed Gobble runtime.
ARG GO_IMAGE=golang:1.26-bookworm
FROM ${GO_IMAGE}

WORKDIR /opt/gobble-dependencies
COPY go.mod go.sum ./
RUN go mod download all

ENV GOTOOLCHAIN=local GOPROXY=off
CMD ["go", "test", "-count=1", "./..."]
