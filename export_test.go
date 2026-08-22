package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Preflight is the checks-only run gate.
func Preflight(g *Graph, workspace string, cap int) error {
	return preflight(g, workspace, cap)
}

// PlanDocument is the Graph-to-execution-view translation used by tests.
func PlanDocument(g *Graph) (engine.Document, error) {
	return planDocument(g)
}

// DropHeldLease closes this process's occupancy flock so tests can
// simulate occupying-process death.
func DropHeldLease(workspace string) {
	engine.DropHeldLease(workspace)
}

// ForgetHeldLease makes this process a later-process while the flock
// stays held, so tests can observe live-occupancy.
func ForgetHeldLease(workspace string) {
	engine.ForgetHeldLease(workspace)
}
