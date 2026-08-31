package atacseqevidence_test

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	atacseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/atacseq"
)

type operationPlan struct {
	Tasks []pc.Task `json:"tasks"`
	DAG   struct {
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	} `json:"dag"`
}

func TestOfficialStagedIdentitiesReachEverySelectedOperation(t *testing.T) {
	raw := pc.MustPlanJSON(t, atacseq.Build(loadSamples(t), atacseq.DefaultConfig()))
	plan := decodeOperationPlan(t, raw)
	staged := atacseqevidence.StagedInputs()
	byPath := make(map[string]string, len(staged))
	byName := make(map[string]bool, len(staged))
	for _, input := range staged {
		if input.Destination == "" || input.Pin.Name == "" || byPath[input.Destination] != "" || byName[input.Pin.Name] {
			t.Fatalf("invalid or duplicate ATAC staging record: %+v", input)
		}
		byPath[input.Destination] = input.Pin.Name
		byName[input.Pin.Name] = true
	}
	for _, pin := range atacseqevidence.MustPins() {
		if !byName[pin.Name] {
			t.Errorf("ATAC staging plan omits manifest pin %s", pin.Name)
		}
	}
	if len(byPath) != len(atacseqevidence.MustPins()) || len(byPath) != 10 {
		t.Fatalf("ATAC staging plan has %d identities, want all 10 manifest pins", len(byPath))
	}

	ancestors := make(map[string]map[string]bool, len(plan.Tasks))
	taskIDs := make(map[string]bool, len(plan.Tasks))
	directUse := make(map[string]bool, len(staged))
	for _, task := range plan.Tasks {
		taskIDs[task.ID] = true
		ancestors[task.ID] = make(map[string]bool)
		for _, path := range inputPaths(task.Inputs) {
			if name := byPath[path]; name != "" {
				ancestors[task.ID][name] = true
				directUse[name] = true
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, edge := range plan.DAG.Edges {
			from := edgeTaskID(edge.From, taskIDs)
			to := edgeTaskID(edge.To, taskIDs)
			if from == "" || to == "" {
				continue
			}
			for name := range ancestors[from] {
				if !ancestors[to][name] {
					ancestors[to][name] = true
					changed = true
				}
			}
		}
	}
	for _, input := range staged {
		if !directUse[input.Pin.Name] {
			t.Errorf("official staged identity %s at %s has no product input consumer", input.Pin.Name, input.Destination)
		}
	}
	for _, task := range plan.Tasks {
		if len(ancestors[task.ID]) == 0 {
			t.Errorf("selected operation %s (%s) has no official staged identity ancestor", task.ID, task.Name)
		}
	}
	wantAll := make(map[string]bool, len(staged))
	for _, input := range staged {
		wantAll[input.Pin.Name] = true
	}
	for _, endpoint := range []string{"multiqc", "igv.igv_session", "ataqv.ataqv_mkarv"} {
		if got := ancestors[endpoint]; !maps.Equal(got, wantAll) {
			t.Errorf("official identities reaching %s = %#v, want all %#v", endpoint, got, wantAll)
		}
	}
}

func edgeTaskID(endpoint string, taskIDs map[string]bool) string {
	matched := ""
	for id := range taskIDs {
		if len(id) > len(matched) && strings.HasPrefix(endpoint, id+".") {
			matched = id
		}
	}
	return matched
}

func decodeOperationPlan(t *testing.T, raw []byte) operationPlan {
	t.Helper()
	var plan operationPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("decode ATAC operation plan: %v", err)
	}
	return plan
}

func inputPaths(inputs []pc.IO) []string {
	var paths []string
	for _, input := range inputs {
		if input.Path != "" {
			paths = append(paths, input.Path)
		}
		if input.Source != "" {
			paths = append(paths, input.Source)
		}
		for _, member := range input.Members {
			paths = append(paths, member.Path)
		}
	}
	return paths
}
