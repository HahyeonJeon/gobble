package gobble_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestDisplayInheritanceIsCopiedAndDoesNotChangeExecution(t *testing.T) {
	build := func(label bool) map[string]any {
		p := gobble.NewPipeline("display")
		m := p.AddModule("sample")
		owners := []string{"S01"}
		if label { m.WithDisplay(gobble.TaskDisplay{Samples: owners}) }
		owners[0] = "mutated"
		m.AddBranch("branch").AddTask(gobble.TaskSpec{Name: "align", Command: []string{"true"}, Outputs: []gobble.Bind{fileOut("out")}})
		shared := m.AddModule("report")
		if label { shared.WithDisplay(gobble.TaskDisplay{Scope: gobble.DisplayCohort}) }
		shared.AddTask(gobble.TaskSpec{Name: "report", Command: []string{"true"}, Outputs: []gobble.Bind{fileOut("report")}})
		var doc map[string]any
		if err := json.Unmarshal(mustBuildPlanJSON(t, p), &doc); err != nil { t.Fatal(err) }
		return doc
	}
	want, got := build(false), build(true)
	tasks := got["tasks"].([]any)
	first := tasks[0].(map[string]any)["display"].(map[string]any)
	if first["scope"] != "sample" || first["samples"].([]any)[0] != "S01" { t.Fatalf("inheritance/copy: %v", first) }
	second := tasks[1].(map[string]any)["display"].(map[string]any)
	if second["scope"] != "cohort" || second["samples"] != nil { t.Fatalf("cohort ownership: %v", second) }
	for _, task := range tasks { delete(task.(map[string]any), "display") }
	if !reflect.DeepEqual(got, want) { t.Fatal("display metadata changed execution plan facts") }
}
