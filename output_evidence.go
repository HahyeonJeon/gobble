package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

type OutputRequest = engine.OutputRequest
type OutputEvidence = engine.OutputEvidence

// ReadOutputEvidence attributes a declared result to its exact latest successful
// attempt. The trusted host supplies the original sealed launch, never Project code.
func ReadOutputEvidence(workspace string, raw []byte, launch LaunchIntent, request OutputRequest) (OutputEvidence, error) {
	identity, err := engine.PreparedIdentity(launch)
	if err != nil {
		return OutputEvidence{}, err
	}
	return engine.ReadOutputEvidence(workspace, raw, launch, request, identity)
}
