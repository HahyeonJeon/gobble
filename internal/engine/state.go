package engine

type jsonRun struct {
	SchemaVersion int              `json:"schema_version"`
	Identity      *InstallIdentity `json:"identity,omitempty"`
	Snapshot      string           `json:"snapshot,omitempty"`
	ID            string           `json:"id"`
	Status        string           `json:"status"`
	Started       string           `json:"started"`
	Ended         string           `json:"ended,omitempty"`
	Occupancy     *jsonOccupancy   `json:"occupancy"`
}

type jsonTaskState struct {
	ID               string         `json:"id"`
	Instance         string         `json:"instance"`
	ShardIndex       int            `json:"shard_index"`
	ShardCount       int            `json:"shard_count"`
	Attempt          int            `json:"attempt"`
	Status           string         `json:"status"`
	Executor         string         `json:"executor"`
	Image            string         `json:"image"`
	Command          []string       `json:"command"`
	Script           string         `json:"script,omitempty"`
	Resources        jsonResources  `json:"resources"`
	Params           []jsonParam    `json:"params"`
	EnvDigest        string         `json:"env_digest,omitempty"`
	RuntimeID        string         `json:"runtime_id,omitempty"`
	ImageDigest      string         `json:"image_digest,omitempty"`
	ExecutablePath   string         `json:"executable_path,omitempty"`
	ExecutableSHA256 string         `json:"executable_sha256,omitempty"`
	Reason           string         `json:"reason"`
	Error            *jsonTaskErr   `json:"error,omitempty"`
	Stdout           string         `json:"stdout,omitempty"`
	Stderr           string         `json:"stderr,omitempty"`
	Started          string         `json:"started,omitempty"`
	Ended            string         `json:"ended,omitempty"`
	Fingerprints     []jsonFileHash `json:"fingerprints,omitempty"`
	Checksums        []jsonFileHash `json:"checksums,omitempty"`
	Lineage          []jsonLineage  `json:"lineage,omitempty"`
	Decision         string         `json:"decision,omitempty"`
	ReuseReason      string         `json:"reuse_reason,omitempty"`
	Differing        []string       `json:"differing,omitempty"`
	Change           string         `json:"change,omitempty"`
	Scatter          string         `json:"scatter,omitempty"`
	Gather           string         `json:"gather,omitempty"`
	When             string         `json:"when,omitempty"`
	Condition        string         `json:"condition,omitempty"`
	Expansion        *jsonExpansion `json:"expansion,omitempty"`
}

type jsonExpansion struct {
	Producer string   `json:"producer,omitempty"`
	Members  []string `json:"members"`
}

type jsonTaskErr struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

type jsonTasksFile struct {
	SchemaVersion int             `json:"schema_version"`
	Snapshot      string          `json:"snapshot,omitempty"`
	Tasks         []jsonTaskState `json:"tasks"`
}
