// Package gobble provides a pre-1.0 trusted-local Linux pipeline engine.
//
// The supported preview operations are [Compose], [Validate], [BuildPlan],
// [Run], [Inspect], [Release], and [Resume]. Authors compose a [Pipeline]
// with [Module], [Branch], [Merge], [Scatter], [Gather], and [When]. Artifacts
// use [PathSpec] and File, [Group], or [Tree] binds. Samplesheet support
// includes [SampleRow], [SampleSheet], [ParseSampleSheet], and the explicit,
// concurrency-safe [LoadSampleSheetFile]. Failures use [Error], [Defect], and
// stable [DefectCode] values.
//
// [WriteTo] is the supported plan option constructor. Graph readers Name,
// TaskIDs, InputNames, and Edges, plus Plan JSON, support the documented loop.
// Other exported names are provisional. In particular, [SetSampleSheetPath],
// [SampleSheetPath], [LoadSampleSheet], raw Inspect Go types, direct
// [PlanOption] construction, [TaskSpec.Backend], and package assets are not
// supported compatibility promises.
//
// Agents use the library and a generic command selected from the same local
// module graph. Humans receive one packed linux/amd64 runner for one pipeline;
// they do not need Go at run time. Exact-tag install is not yet available.
// Gobble is licensed under the MIT License in LICENSE.
//
// The caller provides trusted pipeline code and an exclusive caller-owned
// workspace. Docker --network=none and UID/GID settings are isolation
// conveniences, not a sandbox. Run and Resume take a context; cancellation
// leaves occupancy active. Run and Resume require [WithIdentity] when called
// through the external API. Inspect and Release may omit it and then bind the
// current executable. A mismatch fails closed before workspace mutation.
// Recovery is Inspect, then Release, then Resume remaining work. Release never
// signals an unproved process PID and does not delete control files or
// artifacts. A proved-stopped Docker task may retain a runtime id for cleanup
// retry without keeping occupancy active; unproved disposition remains
// unknown-backend and blocks Resume.
//
// File dest path is a present regular file, not a symlink. Group: every named
// member is a present regular file, not a symlink. Tree: dest directory exists
// as a directory and dest .gobble-tree.json is a present regular file.
// Directory presence alone is not dest-complete, and the Tree check does not
// walk every member.
//
// Gobble stages and publishes by copy. Input fingerprints cover staged isolate
// bytes. Missing hashes are reuse misses, and Resume re-evaluates When
// predicates. First-horizon installed-path evidence passed on linux/amd64 for
// local-pin agents and packed human runners with
// go test -tags=live ./tests/install-e2e. No exact-tag release is claimed.
package gobble
