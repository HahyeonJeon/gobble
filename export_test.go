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
