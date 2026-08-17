// Package gobble is the Go library for composing, validating, and planning
// bioinformatics pipelines.
//
// Authors build a [Pipeline] with modules, branches, merges, and tasks, then
// call [Compose] to obtain an immutable [Graph]. [Validate] re-checks compose
// defects and rejects rendered-path conflicts, unsupported backends, and
// non-finite CPU. [BuildPlan] validates first and returns an inspectable
// [Plan]. [WriteTo] is a [PlanOption] that writes the same JSON
// [Plan.WriteJSON] emits. Compose, Validate, and BuildPlan report defects
// as [*Error] values inspected with errors.As. A WriteTo failure returns
// the writer's own error and the built Plan. WriteJSON on a nil Plan is
// not an [*Error].
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
// Module, Branch, Merge, Task, Handle, TaskSpec, Bind, Param, Resources,
// Graph, Compose, Validate, BuildPlan, WriteTo, PlanOption, Plan,
// Error, Defect, DefectCode, and the Defect* constants. PathSpec methods
// and builder methods belong to those types.
// The public surface is unsupported except these locked PathSpec concepts.
// This package does not run, inspect, or resume a pipeline.
package gobble
