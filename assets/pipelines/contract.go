package pipelines

import (
	"io"

	"github.com/HahyeonJeon/gobble"
)

// Contract gives each assay package a compile-time statement of its shared
// typed entry points. S and C are that package's Sample and Config types.
type Contract[S, C any] struct {
	// Parse converts assay CSV from r into typed samples.
	Parse func(r io.Reader) ([]S, error)
	// Load opens the supplied filesystem path and returns its typed samples.
	Load func(path string) ([]S, error)
	// DefaultConfig returns a fresh supported config.
	DefaultConfig func() C
	// Build constructs a pipeline only from the supplied samples and config.
	Build func(samples []S, config C) *gobble.Pipeline
	// Pipeline returns the default command-line adapter pipeline.
	Pipeline func() *gobble.Pipeline
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
