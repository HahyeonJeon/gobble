package preparation

// LaunchIntent is host-created, never Agent-authored. InputSHA256 names the
// staged file bytes; WorkspaceID names one reserved, fresh execution directory.
// Schema 1 retains the original single-admission format; schema 2 creates an
// execution-history checkpoint. Neither version authorizes continuation.
type LaunchIntent struct {
	SchemaVersion  int               `json:"schemaVersion"`
	RequestID      string            `json:"requestId"`
	PreparedDigest string            `json:"preparedDigest"`
	Binding        Binding           `json:"binding"`
	InputSHA256    string            `json:"inputSHA256"`
	InputSize      int64             `json:"inputSize"`
	WorkspaceID    string            `json:"workspaceId"`
	EngineImage    string            `json:"engineImage"`
	DaemonID       string            `json:"daemonId"`
	Tools          map[string]string `json:"tools"`
}

// Admission survives controller/client lifetime and never authorizes replay.
type Admission struct {
	SchemaVersion  int    `json:"schemaVersion"`
	IntentDigest   string `json:"intentDigest"`
	RequestID      string `json:"requestId"`
	PreparedDigest string `json:"preparedDigest"`
	WorkspaceID    string `json:"workspaceId"`
	Lease          string `json:"lease"`
}
