// Package gobble is the Go library for composing, validating, planning,
// and running bioinformatics pipelines.
//
// Authors build a [Pipeline] with modules, branches, merges, and tasks, then
// call [Compose] to obtain an immutable [Graph]. [Validate] re-checks compose
// defects and rejects rendered-path conflicts, unsupported backends, and
// non-finite CPU. [BuildPlan] validates first and returns an inspectable
// [Plan]. [WriteTo] is a [PlanOption] that writes the same JSON
// [Plan.WriteJSON] emits. [Run] executes a valid graph in a caller workspace
// after the same checks. Run and Resume take a context as the first
// argument; a done context cancels in-flight work and leaves occupancy
// active. The default concurrency cap is 1. [Inspect]
// returns a read-only workspace view selected by [View]. [Resume] occupies
// a released run and continues remaining work. [Release] closes occupancy
// and leaves documents and artifacts. Compose, Validate, BuildPlan, Run,
// Inspect, Resume, and Release report defects as [*Error] values inspected
// with errors.As. A WriteTo failure returns the writer's own error and the
// built Plan. WriteJSON on a nil Plan is not an [*Error].
//
// PathSpec is the public parameterized path model. Fields are Dir, Prefix,
// Base, Suffixes, and Ext (JSON dir, prefix, base, suffixes, ext).
//
// A Bind is File, Group, or Tree. Tree is a declared directory plus a
// dest .gobble-tree.json. Directory remains placement, not an artifact.
//
// Guarded Clean is designed and not shipped. Occupancy must be closed.
// Default scope is isolate directories under .gobble/tasks. Dry-run a
// manifest before delete. Never unguarded dest delete. Refuse while
// occupied. Tree remains correct without this verb.
//
// Shipped types and functions include PathSpec, Directory, Literal, Dir,
// DeriveRule, DeriveAppend, DeriveReplaceExt, Pipeline, NewPipeline,
// Module, Branch, Merge, Task, Handle, TaskSpec, Bind, Group, Member,
// Tree, DeclareTree, Param, Resources, Graph, Edge, Compose, Validate,
// BuildPlan, WriteTo, PlanOption, Plan, Run, Inspect, View, Resume,
// Release, Error, Defect, DefectCode, and the Defect* constants.
// PathSpec methods and builder methods belong to those types.
// The public surface is unsupported except these locked PathSpec concepts.
package gobble
