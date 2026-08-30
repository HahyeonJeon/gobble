// Package pipelineevidence defines the common benchmark and fixture-manifest
// contract for first-party assay pipeline evidence.
//
// Each tests/pipelines/<assay> child owns sheet parsing, typed sample
// invariants, defaults, graph identity, required stages, outputs, and one
// authoritative testdata directory. Its manifest names every remote byte used
// directly or after extraction. Lifecycle scenarios read this owner instead of
// copying URLs, hashes, samplesheets, references, or assay builders.
//
// Hermetic tests never fetch the network. Test-only live preparation may fetch
// a manifest URL into the assay owner's ignored host cache, verify the declared
// byte identity, and stage it as an ordinary workspace input. Product Build
// functions and product tasks do not download, rewrite manifests, or accept a
// changed cached byte. A missing source, size or checksum mismatch, undeclared
// archive member, or unknown license stops data preparation. There is no
// cross-workspace product cache or duplicate scenario cache.
package pipelineevidence
