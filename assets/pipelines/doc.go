// Package pipelines defines the shared construction contract for first-party
// assay pipeline products.
//
// Each direct child package owns one assay. It declares assay-specific Sample
// and Config types and supplies every [Contract] callable: Parse, Load,
// DefaultConfig, Build, and Pipeline. No universal sample row, arbitrary module
// list, serialized parameters, or cross-assay pipeline import belongs in this
// layer.
//
// Parse converts strict assay CSV from only the supplied reader into typed
// Sample values. The caller retains ownership of the reader. Load opens only
// its supplied filesystem path, closes the file it opens, and applies the same
// parser and assay rules. Neither callable discovers input from process
// globals, environment variables, or the network. Malformed CSV and unusable
// paths return errors inspectable as [*gobble.Error], with structured
// invalid-samplesheet or invalid-path defects, instead of panicking.
//
// DefaultConfig returns a fresh, internally consistent value on every call.
// Config owns reference locations, selected graph policy, nested module
// options, output policy, images, and task resources. Sample owns experiment
// identity, input paths, and assay metadata. Engine controls such as workspace,
// execution identity, concurrency, occupancy, cancellation, Inspect, Release,
// and Resume remain with package gobble and its caller.
//
// Default reference and output locations are deterministic and
// workspace-relative. Defaults do not choose a species, contain fixture URLs,
// or hide reference bytes. Callers stage compatible reference members or
// supply different typed locations. Missing members fail through compose or
// preflight instead of triggering a download or guess.
//
// Build accepts only supplied Sample and Config values. It copies every
// caller-owned slice, map, and other mutable value before retaining it. Build
// reads no process global, environment variable, command-line flag, current
// working directory default, or network location. Invalid data, config,
// reference members, paths, or module options are recorded with
// [RecordComposeDefects], so gobble.Compose returns structured defects rather
// than a panic.
//
// Pipeline is only the default exclusive-process CLI adapter. It obtains the
// injected samplesheet path, calls the assay loader and DefaultConfig, and
// delegates to Build. Customized runners provide their own small Pipeline
// function and call the same Build. File, gobble.Group, and gobble.Tree handles
// retain the artifact kind consumed by each tool.
//
// The wgs child retains its graph-stable checkpoint constructor until its named
// lift. The rnaseq and methylseq children own lifted products and complete typed
// contracts. Each lift is a new graph generation and does not promise resume of
// a pre-lift workspace.
package pipelines
