# Installation and version selection

The current supported execution target is Linux/amd64. Windows/WSL setup is
proposed for v0.2.0 and is not yet a validated installation path. See the
[v0.2.0 design](v0.2.0/design.md).

For a first local example, follow [Hello Gobble](../examples/hello/README.md).
This guide explains exact revision selection for an agent-owned Go project.
Create the consumer module with `go mod init YOUR_MODULE` before using the
module-graph commands below.

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

## Agent install

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

