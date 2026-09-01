// Package pipelineevidence anchors cross-product benchmark, manifest, pin, and
// default-image consistency checks for first-party assay pipeline evidence.
//
// Each tests/pipelines/<assay> child owns sheet parsing, typed sample
// invariants, defaults, graph identity, required stages, outputs, and one
// authoritative testdata directory. Its manifest names every remote byte used
// directly or after extraction. Shared decoding and staging mechanics live in
// tests/internal/fixture without owning an assay fact. Lifecycle scenarios read
// each assay owner instead of copying URLs, hashes, samplesheets, references, or
// assay builders.
//
// Hermetic tests never fetch the network. Test-only live preparation may fetch
// a manifest URL into the assay owner's ignored host cache, verify the declared
// byte identity, and stage it as an ordinary workspace input. Product Build
// functions and product tasks do not download, rewrite manifests, or accept a
// changed cached byte. A missing source, size or checksum mismatch, undeclared
// archive member, or unknown license stops data preparation. There is no
// cross-workspace product cache or duplicate scenario cache.
package pipelineevidence
