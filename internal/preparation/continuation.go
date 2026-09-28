package preparation

// MaxContinuations bounds retained admissions. History is never silently pruned.
const MaxContinuations = 32

// ContinuationIntent binds one host-created confirmation to a reviewed Run.
// It carries no executable source or caller-selected reuse decisions.
type ContinuationIntent struct {
	SchemaVersion int    `json:"schemaVersion"`
	RequestID     string `json:"requestId"`
	WorkspaceID   string `json:"workspaceId"`
	OriginDigest  string `json:"originDigest"`
	ReviewDigest  string `json:"reviewDigest"`
	PreviousHead  string `json:"previousHead"`
}

// ContinuationAdmission records an accepted ownership interval. The engine owns
// these values; merely decoding one never authorizes execution.
type ContinuationAdmission struct {
	Intent       ContinuationIntent `json:"intent"`
	IntentDigest string             `json:"intentDigest"`
	Lease        string             `json:"lease"`
	Snapshot     string             `json:"snapshot"`
}

// ExecutionHistory retains the continuation chain beside the immutable initial
// Admission. Readers own decoded slices; writers must not mutate retained entries.
type ExecutionHistory struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Continuations []ContinuationAdmission `json:"continuations"`
}

// ExecutionEpoch identifies current ownership independently of the initial Start.
type ExecutionEpoch struct {
	Head  string `json:"head"`
	Lease string `json:"lease"`
}

// ContinuationStep explains one decision without exposing executable source.
type ContinuationStep struct {
	TaskID         string `json:"taskId"`
	PriorAttempt   int    `json:"priorAttempt"`
	PlannedAttempt int    `json:"plannedAttempt"`
	Action         string `json:"action"` // reuse, restart, or start
}

// ContinuationFile binds reviewed file contents, not mutable timestamps.
type ContinuationFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ContinuationReview is read-only evidence, never execution permission. Digest
// covers all fields with Digest empty; slices belong to the receiving caller.
type ContinuationReview struct {
	SchemaVersion int                `json:"schemaVersion"`
	WorkspaceID   string             `json:"workspaceId"`
	OriginDigest  string             `json:"originDigest"`
	PreviousHead  string             `json:"previousHead"`
	Snapshot      string             `json:"snapshot"`
	StateDigest   string             `json:"stateDigest"`
	Steps         []ContinuationStep `json:"steps"`
	Files         []ContinuationFile `json:"files"`
	Digest        string             `json:"digest"`
}
