package gobble_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

// Exercise deferred path rules through staging, sibling tasks, publication,
// Gather and Resume. A successful tool exit alone cannot prove these contracts.
func TestScatterReplaceExtRunGatherAndResume(t *testing.T) {
	dir := t.TempDir()
	p := gobble.NewPipeline("scatter-derivation")
	var members gobble.Group
	for _, name := range []string{"region.one", "region.two"} {
		writeRunFile(t, filepath.Join(dir, "in", name+".bed"), name+"\n")
		members = append(members, gobble.Member{Name: name[len("region."):], Spec: gobble.PathSpec{Dir: gobble.Dir("in"), Base: name, Ext: ".bed"}})
	}
	intervals := p.AddInputGroup("intervals", members)
	scatter := p.Scatter("each").From(intervals)
	table := scatter.AddTask(gobble.TaskSpec{
		Name: "table",
		Script: `for f in staged/*.interval; do
name=${f##*/}; stem=${name%.interval}
cp "$f" "tables/$stem.table"
done`,
		Inputs:  []gobble.Bind{{Name: "interval", From: intervals, Spec: gobble.PathSpec{Dir: gobble.Dir("staged"), Ext: ".interval"}, Rule: gobble.DeriveReplaceExt}},
		Outputs: []gobble.Bind{{Name: "table", From: intervals, Spec: gobble.PathSpec{Dir: gobble.Dir("tables"), Ext: ".table"}, Rule: gobble.DeriveReplaceExt}},
	})
	converted := scatter.AddTask(gobble.TaskSpec{
		Name: "convert",
		Script: `for f in staged/*.txt; do
name=${f##*/}; stem=${name%.txt}
cp "$f" "converted/$stem.bam"
cp "$f" "indexes/$stem.table.idx"
done`,
		Inputs: []gobble.Bind{{Name: "table", From: table.Out("table"), Spec: gobble.PathSpec{Dir: gobble.Dir("staged"), Ext: ".txt"}, Rule: gobble.DeriveReplaceExt}},
		Outputs: []gobble.Bind{
			{Name: "bam", From: table.Out("table"), Spec: gobble.PathSpec{Dir: gobble.Dir("converted"), Ext: ".bam"}, Rule: gobble.DeriveReplaceExt},
			{Name: "index", From: table.Out("table"), Spec: gobble.PathSpec{Dir: gobble.Dir("indexes"), Ext: ".idx"}},
		},
	})
	p.Gather("all").AddTask(gobble.TaskSpec{
		Name: "join", Script: `cat converted/*.bam > out/all.txt`,
		Inputs:  []gobble.Bind{{Name: "parts", From: converted.Out("bam")}},
		Outputs: []gobble.Bind{{Name: "all", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "all", Ext: ".txt"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		t.Fatalf("%#v", err)
	}
	for _, name := range []string{"region.one", "region.two"} {
		for _, relative := range []string{"tables/" + name + ".table", "converted/" + name + ".bam", "indexes/" + name + ".table.idx"} {
			got, err := os.ReadFile(filepath.Join(dir, relative))
			if err != nil || string(got) != name+"\n" {
				t.Fatalf("%s = %q, %v", relative, got, err)
			}
		}
	}
	got, err := os.ReadFile(filepath.Join(dir, "out/all.txt"))
	if err != nil || string(got) != "region.one\nregion.two\n" {
		t.Fatalf("Gather = %q, %v", got, err)
	}
	before, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	if err != nil {
		t.Fatalf("%#v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("%#v", err)
	}
	if err := gobble.Resume(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		t.Fatalf("%#v", err)
	}
	after, err := gobble.Inspect(dir, gobble.ViewInstances, "")
	type instance struct {
		Identity string `json:"identity"`
		Status   string `json:"status"`
		Attempt  int    `json:"attempt"`
	}
	readInstances := func(raw []byte) []instance {
		t.Helper()
		var instances []instance
		for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
			var row instance
			if err := json.Unmarshal(line, &row); err != nil {
				t.Fatal(err)
			}
			instances = append(instances, row)
		}
		return instances
	}
	if err != nil || !reflect.DeepEqual(readInstances(before), readInstances(after)) {
		t.Fatalf("Unchanged Resume changed instances: %v\nbefore: %s\nafter: %s", err, before, after)
	}
}
