package gobble

// TaskDisplay describes a task for human monitoring. It does not change the
// task's command, scheduling, artifact identity, or reuse fingerprint.
// Empty fields inherit the nearest module's display information. WithDisplay
// must be called before adding a module's children.
type TaskDisplay struct {
	Stage   string   `json:"stage,omitempty"`
	Samples []string `json:"samples,omitempty"`
	Scope   string   `json:"scope,omitempty"`
}

const (
	DisplaySample = "sample"
	DisplayShared = "shared"
	DisplayCohort = "cohort"
)

// WithDisplay sets copied display defaults for subsequently added children.
// A shared or cohort scope clears inherited sample ownership. Use explicit
// sample IDs, including a patient or assay prefix when needed for uniqueness.
func (m *Module) WithDisplay(display TaskDisplay) *Module {
	m.display = cloneDisplay(display)
	return m
}

func cloneDisplay(d TaskDisplay) TaskDisplay {
	d.Samples = copyStrings(d.Samples)
	return d
}

func inheritDisplay(parent, child TaskDisplay) TaskDisplay {
	result := cloneDisplay(parent)
	if child.Stage != "" {
		result.Stage = child.Stage
	}
	if child.Scope != "" {
		result.Scope = child.Scope
		if child.Scope != DisplaySample {
			result.Samples = nil
		}
	}
	if child.Samples != nil && (child.Scope == "" || child.Scope == DisplaySample) {
		result.Samples = copyStrings(child.Samples)
		if child.Scope == "" {
			result.Scope = DisplaySample
		}
	}
	return result
}

func (m *Module) ancestors() []ancestor {
	anc := childAnc(m.anc, nodeModule, m.name)
	anc[len(anc)-1].display = cloneDisplay(m.display)
	return anc
}
