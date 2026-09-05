package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/HahyeonJeon/gobble"
)

// Snapshot contains one control revision. Log bytes can advance after that
// revision; ReadAt is the reader's observation time, not an engine event time.
type Snapshot struct {
	SchemaVersion int       `json:"schema_version"`
	Revision      string    `json:"snapshot"`
	Pipeline      string    `json:"pipeline"`
	Run           Run       `json:"run"`
	Tasks         []Task    `json:"tasks"`
	Edges         []Edge    `json:"edges"`
	Logs          []Log     `json:"logs"`
	ReadAt        time.Time `json:"-"`
}

type Run struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Started   string `json:"started"`
	Ended     string `json:"ended"`
	Unknown   bool   `json:"unknown"`
	Occupancy struct {
		Active bool `json:"active"`
		Live   bool `json:"live"`
	} `json:"occupancy"`
}

type Task struct {
	Identity  string             `json:"identity"`
	TaskID    string             `json:"task_id"`
	Name      string             `json:"name"`
	Module    string             `json:"module"`
	Display   gobble.TaskDisplay `json:"display"`
	Status    string             `json:"status"`
	Attempt   int                `json:"attempt"`
	Executor  string             `json:"executor"`
	Image     string             `json:"image"`
	Command   []string           `json:"command"`
	Script    string             `json:"script"`
	Resources struct {
		CPU    float64 `json:"cpu"`
		Memory string  `json:"memory"`
	} `json:"resources"`
	Started  string `json:"started"`
	Ended    string `json:"ended"`
	Reason   string `json:"reason"`
	Decision string `json:"decision"`
	Template bool   `json:"template"`
	Expanded bool   `json:"expanded"`
}

// Edge connects authored task IDs (not the expanded instance identities).
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Log struct {
	Identity   string `json:"identity"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	StdoutSize int64  `json:"stdout_size"`
	StderrSize int64  `json:"stderr_size"`
	StdoutTail string `json:"stdout_tail"`
	StderrTail string `json:"stderr_tail"`
}

// Read obtains the global snapshot and, optionally, one instance's log tails.
// It applies the same identity, schema, and containment gates as Inspect.
func Read(workspace, instance string, opts ...gobble.OccupyOption) (Snapshot, error) {
	data, err := gobble.Inspect(workspace, gobble.ViewMonitor, instance, opts...)
	// A resume may remove a previously selected dynamic member. Read the new
	// global snapshot so the view can retire its selection instead of freezing.
	if instance != "" && instanceGone(err, instance) {
		data, err = gobble.Inspect(workspace, gobble.ViewMonitor, "", opts...)
	}
	if err != nil {
		return Snapshot{}, err
	}
	var result Snapshot
	if err := json.Unmarshal(data, &result); err != nil {
		return Snapshot{}, fmt.Errorf("decode monitor snapshot: %w", err)
	}
	if result.SchemaVersion != 2 {
		return Snapshot{}, fmt.Errorf("unsupported monitor schema %d", result.SchemaVersion)
	}
	result.ReadAt = time.Now()
	return result, nil
}

func instanceGone(err error, instance string) bool {
	var ge *gobble.Error
	if !errors.As(err, &ge) || ge == nil || len(ge.Defects) != 1 {
		return false
	}
	d := ge.Defects[0]
	return d.Code == gobble.DefectNotFound && d.Unit == instance
}
