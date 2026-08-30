// Package moduleevidence defines shared evidence metadata for first-party
// command modules.
//
// Each tests/modules/<module> child owns one command's options, argv, image,
// resources, binds, typed ports, invalid inputs, and bounded execution
// evidence. It does not own assay stage order, cohort outputs, or lifecycle
// recovery. Small command-only fixture bytes may stay with that child.
package moduleevidence
