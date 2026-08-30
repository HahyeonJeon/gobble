// Package resume owns evidence for Inspect, Release, and Resume with the same
// accepted graph generation and identity. Matching work is reused while
// affected and downstream work is not. Active occupancy, identity mismatch,
// unknown backend state, and an old graph generation fail closed. It does not
// own an assay graph or fixture.
package resume
