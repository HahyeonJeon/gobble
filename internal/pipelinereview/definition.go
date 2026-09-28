// Package pipelinereview owns Gobble's portable review comparison. It consumes
// checked, bounded facts, never source text or an executable Plan serialization.
package pipelinereview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Setting struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
	Value *int   `json:"value"`
}
type Step struct {
	ID          string    `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	Residual    string    `json:"residual"`
	Recipe      string    `json:"recipe"`
	Settings    []Setting `json:"settings"`
}
type Edge struct {
	FromTask string   `json:"fromTask"`
	FromPort string   `json:"fromPort"`
	ToTask   string   `json:"toTask"`
	ToPort   string   `json:"toPort"`
	Wait     []string `json:"wait"`
}
type Definition struct {
	SchemaVersion int    `json:"schemaVersion"`
	Context       string `json:"context"`
	Steps         []Step `json:"steps"`
	Edges         []Edge `json:"edges"`
}
type Change struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	StepID string `json:"stepId"`
	Key    string `json:"key"`
	Label  string `json:"label"`
	Unit   string `json:"unit"`
	Before *int   `json:"before"`
	After  *int   `json:"after"`
}
type Comparison struct {
	SchemaVersion int      `json:"schemaVersion"`
	Changes       []Change `json:"changes"`
	Gaps          []string `json:"gaps"`
}

// Digest has deterministic map ordering and rejects unencodable values. Its
// output is provenance, not a signature or permission to execute.
func Digest(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "invalid:" + fmt.Sprintf("%T", value)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Compare qualifies only unchanged definitions, explicit Trim Galore settings,
// and a canonical FastQC leaf added to an existing output. Other differences
// stay visible as gaps instead of being silently excluded from acceptance.
func Compare(base, proposed Definition) Comparison {
	out := Comparison{1, []Change{}, []string{}}
	gap := func(message string) { out.Gaps = append(out.Gaps, message) }
	if base.SchemaVersion != 1 || proposed.SchemaVersion != 1 {
		gap("This review definition version is unsupported.")
		return out
	}
	if base.Context != proposed.Context {
		gap("Pipeline inputs, identity or check context changed.")
	}
	old := map[string]Step{}
	next := map[string]Step{}
	for _, step := range base.Steps {
		if _, ok := old[step.ID]; ok {
			gap("Duplicate base step identity.")
		}
		old[step.ID] = step
	}
	for _, step := range proposed.Steps {
		if _, ok := next[step.ID]; ok {
			gap("Duplicate proposed step identity.")
		}
		next[step.ID] = step
	}
	for _, after := range proposed.Steps {
		before, exists := old[after.ID]
		if !exists {
			if after.Recipe != "fastqc-v1" {
				gap("An added processing step needs more review support: " + after.ID)
				continue
			}
			out.Changes = append(out.Changes, Change{ID: "added:" + after.ID, Kind: "added-step", StepID: after.ID, Label: "Quality check", Key: "", Unit: ""})
			continue
		}
		if before.Fingerprint == after.Fingerprint {
			continue
		}
		if before.Recipe != "trim-galore-v1" || after.Recipe != before.Recipe || before.Residual != after.Residual {
			gap("A processing change cannot yet be explained: " + after.ID)
			continue
		}
		if len(before.Settings) != len(after.Settings) {
			gap("The setting definition changed: " + after.ID)
			continue
		}
		count := 0
		for i, value := range after.Settings {
			previous := before.Settings[i]
			if value.Key != previous.Key || value.Label != previous.Label || value.Unit != previous.Unit {
				gap("The setting identity changed: " + after.ID)
				continue
			}
			if Digest(previous.Value) == Digest(value.Value) {
				continue
			}
			count++
			out.Changes = append(out.Changes, Change{ID: "setting:" + after.ID + ":" + value.Key, Kind: "setting", StepID: after.ID, Key: value.Key, Label: value.Label, Unit: value.Unit, Before: copyInteger(previous.Value), After: copyInteger(value.Value)})
		}
		if count == 0 {
			gap("A non-setting change needs review: " + after.ID)
		}
	}
	for _, before := range base.Steps {
		if _, exists := next[before.ID]; !exists {
			gap("A removed or renamed step needs review: " + before.ID)
		}
	}
	oldEdges := map[string]int{}
	nextEdges := map[string]int{}
	for _, e := range base.Edges {
		oldEdges[Digest(e)]++
	}
	addedInputs := map[string]int{}
	for _, e := range proposed.Edges {
		key := Digest(e)
		nextEdges[key]++
		if nextEdges[key] <= oldEdges[key] {
			continue
		}
		_, existingTarget := old[e.ToTask]
		_, existingSource := old[e.FromTask]
		if existingTarget || !existingSource || next[e.ToTask].Recipe != "fastqc-v1" || e.ToPort != "reads" {
			gap("A changed connection needs more review support.")
			continue
		}
		addedInputs[e.ToTask]++
	}
	for key, count := range oldEdges {
		if nextEdges[key] < count {
			gap("An existing input connection was removed or replaced.")
		}
	}
	for _, after := range proposed.Steps {
		if _, exists := old[after.ID]; !exists && after.Recipe == "fastqc-v1" && addedInputs[after.ID] != 1 {
			gap("A new quality check requires one connection from an existing step.")
		}
	}
	return out
}

func copyInteger(value *int) *int {
	if value == nil {
		return nil
	}
	n := *value
	return &n
}
