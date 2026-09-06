# Direct Linux development installation

For the common installation on all supported hosts, use [Docker Compose](../distribution/runtime/README.md).
This page covers working directly with the Go library and Linux engine.


The common installation uses [Docker Compose](../distribution/runtime/README.md).
This page covers direct Linux/amd64 development with Go, Git, and Docker managed
on the host. WSL can provide such a Linux development environment on Windows;
the common Docker installation does not require using an Ubuntu terminal.

## Release state

The immutable `v0.1.0` tag is the published engine preview. It predates the
five-product family and does not contain these product packages. The product
execution baseline is recorded at commit
`f21a858c66a2d95ce8eff469e6db2bfa3240c3a5` (tree
`c90dfe77192c2528f8fd54d17f4d9547b09a6998`). No product-family release tag has
been assigned. The monitoring changes add presentation labels without changing
execution commands or artifact paths. Until release notes name these graph
generations, use an exact trusted local checkout containing that baseline and
build the command from the same selected revision.
Do not expect `v0.1.0` or `@latest` to provide the products.

## Direct Linux installation

Agents, library consumers, and the machine creating a packed runner require Go
1.26 or newer. For the unreleased product baseline, put an exact trusted checkout
in the consumer module graph and build the command from that graph:

```sh
go mod edit -require=github.com/HahyeonJeon/gobble@v0.0.0
go mod edit -replace=github.com/HahyeonJeon/gobble=/absolute/path/to/gobble
go list -m github.com/HahyeonJeon/gobble
mkdir -p .gobbin
GOBIN="$PWD/.gobbin" go install github.com/HahyeonJeon/gobble/cmd/gobble
export PATH="$PWD/.gobbin:$PATH"
```

The unsuffixed `go install` uses the consumer's selected local module. The
`v0.0.0` requirement is only a local module-graph placeholder.

The released engine-only `v0.1.0` install remains:

```sh
go get github.com/HahyeonJeon/gobble@v0.1.0
GOBIN="$PWD/.gobbin" go install github.com/HahyeonJeon/gobble/cmd/gobble@v0.1.0
```

Keep the exact selected command on `PATH` for graph verbs. Consumer packages
under `internal/` are unsupported.

## Start a project

With the matching generic command installed, `gobble init my-pipeline` creates a
small runnable project, starter data, agent instructions, and initial local Git
history. If the build's source directory is unavailable, set `GOBBLE_SOURCE` to
the exact checkout used to build the command. Existing directories are refused.
Run `gobble doctor` to check Go, Git, and Docker before preparing an analysis.
