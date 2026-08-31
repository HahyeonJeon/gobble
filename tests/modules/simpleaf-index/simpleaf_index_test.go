package simpleafindex_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	simpleafindex "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-index"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSimpleafIndexDeclaresCompleteTreeAndProtectsOutput(t *testing.T) {
	p := simpleafindex.Pipeline(gobble.PathSpec{Base: "transcripts", Ext: ".fa"}, simpleafindex.Options{})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "simpleaf_index" || task.Image != string(simpleafindex.DefaultImage) || !strings.Contains(task.Script, "'simpleaf' 'index'") || !strings.Contains(task.Script, "'simpleaf' 'set-paths'") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertTreeIO(t, task.Outputs, "index", "results/scrnaseq/reference/simpleaf_index/index")
	bad := simpleafindex.Pipeline(gobble.PathSpec{Base: "transcripts", Ext: ".fa"}, simpleafindex.Options{Options: modules.Options{ExtraArgs: []string{"-oelsewhere"}}})
	if graph, err := gobble.Compose(bad); graph != nil || err == nil {
		t.Fatalf("protected output compose = (%v, %v), want defect", graph, err)
	}
}
