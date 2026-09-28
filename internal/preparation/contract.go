// Package preparation defines the engine's safe preparation interchange values.
// It contains no evaluator, execution authority, host paths or private payload.
package preparation

// Binding names all preparation-time external facts. Start must additionally
// verify actual staged data, runtime availability and a new workspace's occupancy.
type Binding struct {
	SourceRevision string `json:"sourceRevision"`
	RuntimeID      string `json:"runtimeId"`
	InputIdentity  string `json:"inputIdentity"`
	InputPath      string `json:"inputPath"`
	Cap            int    `json:"cap"`
}

type Step struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	CPU     float64  `json:"cpu"`
	Memory  string   `json:"memory"`
	Image   string   `json:"image"`
	Outputs []string `json:"outputs"`
}
