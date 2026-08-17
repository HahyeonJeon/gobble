// Package gobble is the Go pipeline library for composing bioinformatics pipelines.
//
// PathSpec is the public parameterized path model. Locked concepts map to
// exported fields and JSON keys:
//
//	DirName    → Dir    (JSON dir)
//	Prefix     → Lead   (JSON lead)
//	BaseName   → Name   (JSON name)
//	Suffixes   → Steps  (JSON steps)
//	Extension  → Ext    (JSON ext)
//
// Authors build a [Pipeline] and call [Compose] to obtain an immutable [Graph].
// The public surface is unsupported except these locked PathSpec concepts.
package gobble
