// Package assets holds temporary constructors for first-party assay products.
//
// WGS, RNASeq, and MethylSeq delegate to their assets/pipelines owners and
// preserve source names across the ownership migration. RNASeq targets the
// lifted STAR-Salmon generation, and MethylSeq targets the lifted directional
// Bismark generation. Their pre-lift workspaces require new workspaces. WGS
// retains its checkpoint graph until its own named lift. Command adders live
// only in assets/modules child packages.
package assets
