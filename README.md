# Gobble

Gobble is a Go pipeline engine for agents and developers to compose, validate, plan, run, inspect, and resume bioinformatics pipelines.

PathSpec is the public parameterized path model. DirName, Prefix, BaseName, Suffixes, and Extension map to `Dir`, `Lead`, `Name`, `Steps`, and `Ext` (JSON `dir`, `lead`, `name`, `steps`, `ext`). The public surface is unsupported except these locked PathSpec concepts.

## First check

Requires Go 1.26 or newer.

```sh
go test ./...
```
