// Package customize owns evidence that a named typed option or argv ExtraArgs
// change has an exact visible command or graph effect without mutating
// defaults. Experiment data, analysis config, and engine controls remain
// separate. Named-option collisions and route-changing extras fail closed. It
// does not own an assay graph or fixture.
package customize
