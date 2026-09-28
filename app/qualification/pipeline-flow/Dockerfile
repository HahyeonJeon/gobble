# Local development qualification only; no registry publication or runtime release.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
USER root
COPY gobble /opt/gobble
WORKDIR /opt/gobble
RUN CGO_ENABLED=0 GOMAXPROCS=1 go build -buildvcs=false -o /usr/local/bin/gobble ./cmd/gobble
USER 10001:10001
ENV GOBBLE_CONTAINER_BOOTSTRAP=0 GOTOOLCHAIN=local GOPROXY=off
ENTRYPOINT ["/usr/local/bin/gobble"]
