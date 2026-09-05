package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

func engineDisplay(d TaskDisplay) *engine.Display {
	if d.Stage == "" && d.Scope == "" && len(d.Samples) == 0 {
		return nil
	}
	return &engine.Display{Stage: d.Stage, Samples: copyStrings(d.Samples), Scope: d.Scope}
}
