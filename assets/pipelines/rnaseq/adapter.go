package rnaseq

import "github.com/HahyeonJeon/gobble"

// Pipeline loads the process-owned CLI samplesheet path, applies fresh
// defaults, and delegates to Build. Reusable callers use Load and Build.
func Pipeline() *gobble.Pipeline {
	samples, err := Load(gobble.SampleSheetPath())
	if err != nil {
		pipeline := gobble.NewPipeline("rnaseq")
		pipeline.RecordComposeError(err)
		return pipeline
	}
	return Build(samples, DefaultConfig())
}
