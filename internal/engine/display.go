package engine

// Display is presentation metadata, persisted separately from execution facts.
type Display struct {
	Stage   string   `json:"stage,omitempty"`
	Samples []string `json:"samples,omitempty"`
	Scope   string   `json:"scope,omitempty"`
}

func cloneDisplay(d *Display) *Display {
	if d == nil {
		return nil
	}
	copy := *d
	copy.Samples = append([]string(nil), d.Samples...)
	return &copy
}
