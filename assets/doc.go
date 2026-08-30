// Package assets holds temporary constructors for the three pre-lift assay
// graphs.
//
// WGS, RNASeq, and MethylSeq delegate to their assets/pipelines owners and
// preserve graph bytes at the mechanical ownership checkpoint. They preserve
// source names for compatible workspaces only while the graph is unchanged.
// Each later assay lift is a named new graph generation and requires a new
// workspace. Command adders live only in assets/modules child packages.
package assets
