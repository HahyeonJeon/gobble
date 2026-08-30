// Package assets holds temporary constructors for first-party assay products.
//
// WGS, RNASeq, and MethylSeq delegate to their assets/pipelines owners and
// preserve source names across the ownership migration. RNASeq now targets the
// lifted STAR-Salmon generation; pre-lift RNA workspaces require a new
// workspace. WGS and MethylSeq retain their checkpoint graphs until their own
// named lifts. Command adders live only in assets/modules child packages.
package assets
