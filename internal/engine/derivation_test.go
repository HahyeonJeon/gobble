package engine

import (
	"encoding/json"
	"testing"
)

func TestDeferredPathsSurviveRecordedPlan(t *testing.T) {
	task := TaskPlan{ID: "each.copy", Scatter: "each", ScatterMemberSpecs: []Path{
		{Dir: "in", Base: "sample", Ext: ".vcf.gz"},
		{Dir: "in", Literal: true, Opaque: "opaque.name.txt"},
	}, Outputs: []IO{{Name: "out", Spec: Path{Dir: "out", Ext: ".table"}, Rule: DeriveReplaceExt}}}
	raw, err := json.Marshal(encodeTask(task))
	if err != nil {
		t.Fatal(err)
	}
	var recorded jsonTask
	if err := json.Unmarshal(raw, &recorded); err != nil {
		t.Fatal(err)
	}
	decoded := decodeTask(recorded)
	if operatorFieldsDiffer(task, decoded) {
		t.Fatalf("Recorded plan changed deferred paths: %s", raw)
	}
	if decoded.Outputs[0].Rule != DeriveReplaceExt || decoded.ScatterMemberSpecs[1].Opaque != "opaque.name.txt" {
		t.Fatalf("Lost derivation data: %#v", decoded)
	}
	changed := cloneTaskPlan(decoded)
	changed.Outputs[0].Rule = DeriveAppend
	if !operatorFieldsDiffer(decoded, changed) {
		t.Fatal("Resume ignored a changed derivation rule")
	}
	changed = cloneTaskPlan(decoded)
	changed.ScatterMemberSpecs[0].Base = "sample.vcf"
	changed.ScatterMemberSpecs[0].Ext = ".gz"
	if !operatorFieldsDiffer(decoded, changed) {
		t.Fatal("Resume ignored changed compound extension semantics")
	}
}
