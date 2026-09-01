package pipelines

import (
	"github.com/HahyeonJeon/gobble"
)

const (
	// SupportPlatform is the platform tuple covered by the five pinned default
	// products and their image manifests.
	SupportPlatform = "linux/amd64"
	// SupportExecutionBoundary is the current local execution boundary.
	SupportExecutionBoundary = "trusted-local Docker"
	// SupportClaim limits support to engineering behavior, not scientific or
	// clinical correctness.
	SupportClaim = "engineering-only"
)

// LifecycleParticipation states how one product enters the shared lifecycle
// evidence owners. Scenario packages verify the behavior; this value does not
// duplicate an assay graph or fixture.
type LifecycleParticipation struct {
	GraphGeneration  string
	Design           bool
	Build            bool
	Customize        bool
	Run              bool
	Resume           bool
	Stop             bool
	Failure          bool
	PreLiftResumable bool
}

// CompleteLifecycle returns the shared supported scenario set for one current
// graph generation. Lifted and newly introduced products do not resume a
// pre-lift workspace.
func CompleteLifecycle(graphGeneration string) LifecycleParticipation {
	return LifecycleParticipation{
		GraphGeneration: graphGeneration,
		Design:          true,
		Build:           true,
		Customize:       true,
		Run:             true,
		Resume:          true,
		Stop:            true,
		Failure:         true,
	}
}

// Complete reports whether a product participates in every supported
// lifecycle scenario and names its current graph generation.
func (p LifecycleParticipation) Complete() bool {
	return p.GraphGeneration != "" && p.Design && p.Build && p.Customize &&
		p.Run && p.Resume && p.Stop && p.Failure
}

// CopySlice returns a shallow copy of in. An assay package must also copy any
// reference-bearing values inside each element before retaining them.
func CopySlice[T any](in []T) []T {
	return append([]T(nil), in...)
}

// CopyMap returns a shallow copy of in. An assay package must also copy any
// reference-bearing keys or values before retaining them.
func CopyMap[K comparable, V any](in map[K]V) map[K]V {
	if in == nil {
		return nil
	}
	out := make(map[K]V, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// RecordComposeDefects records copied defects on p. Compose returns them with
// Op "compose" and does not build a graph. A nil pipeline or empty defects is
// a no-op.
func RecordComposeDefects(p *gobble.Pipeline, defects ...gobble.Defect) {
	if p == nil || len(defects) == 0 {
		return
	}
	copied := make([]gobble.Defect, len(defects))
	for i, defect := range defects {
		copied[i] = defect
		copied[i].Paths = append([]string(nil), defect.Paths...)
	}
	p.RecordComposeError(&gobble.Error{Op: "compose", Defects: copied})
}
