// Package gobble is the Go library for composing, validating, planning,
// and running bioinformatics pipelines.
//
// Authors build a [Pipeline] with modules, branches, merges, and tasks, then
// call [Compose] to obtain an immutable [Graph]. [Validate] re-checks compose
// defects and rejects rendered-path conflicts, unsupported backends, and
// non-finite CPU. [BuildPlan] validates first and returns an inspectable
// [Plan]. [WriteTo] is a [PlanOption] that writes the same JSON
// [Plan.WriteJSON] emits. [Run] executes a valid graph in a caller workspace
// after the same checks. The default concurrency cap is 1. Compose,
// Validate, BuildPlan, and Run report defects as [*Error] values inspected
// with errors.As. A WriteTo failure returns the writer's own error and
// the built Plan. WriteJSON on a nil Plan is not an [*Error].
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
// Shipped types and functions include PathSpec, Directory, Literal, Dir,
// DeriveRule, DeriveAppend, DeriveReplaceExt, Pipeline, NewPipeline,
// Module, Branch, Merge, Task, Handle, TaskSpec, Bind, Group, Member,
// Param, Resources, Script, Env, Graph, Compose, Validate, BuildPlan,
// WriteTo, PlanOption, Plan, Run, Inspect, Resume, Error, Defect, DefectCode, and the
// Defect* constants. PathSpec methods and builder methods belong to
// those types.
// The public surface is unsupported except these locked PathSpec concepts.
// This package does not inspect or resume a pipeline.
package gobble
