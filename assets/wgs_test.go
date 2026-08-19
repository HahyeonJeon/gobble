package assets

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestWGSComposeBuildPlan(t *testing.T) {
	raw := mustPlanJSON(t, WGS())
	tasks := planAllTasks(t, raw)

	if got := countTasksNamed(tasks, "bwa_index"); got != 1 {
		t.Fatalf("bwa_index count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "bwa_mem"); got != 2 {
		t.Fatalf("bwa_mem count = %d, want 2", got)
	}
	if got := countTasksNamed(tasks, "fastp"); got != 2 {
		t.Fatalf("fastp count = %d, want 2", got)
	}
	if got := countTasksNamed(tasks, "samtools_sort"); got != 2 {
		t.Fatalf("samtools_sort count = %d, want 2", got)
	}
	if got := countTasksNamed(tasks, "samtools_index"); got != 2 {
		t.Fatalf("samtools_index count = %d, want 2", got)
	}
	mustHaveTaskID(t, tasks, "raw.fastqc")
	mustHaveTaskID(t, tasks, "clean.fastqc")
	mustHaveTaskID(t, tasks, "sample1.fastp")
	mustHaveTaskID(t, tasks, "sample2.fastp")
	mustHaveTaskID(t, tasks, "sample1.bwa_mem")
	mustHaveTaskID(t, tasks, "sample2.bwa_mem")
	mustHaveTaskID(t, tasks, "bwa_index")
	mustHaveTaskID(t, tasks, "multiqc")

	rawQC := planTask(t, raw, "raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := planTask(t, raw, "clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	assertIOPath(t, planTask(t, raw, "sample1.fastp").Inputs, "r1", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "sample2.fastp").Inputs, "r1", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/raw/fastqc/test_1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_4", "work/sample1/fastp/test_1.fastp.json")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_6", "work/sample2/fastp/test_1.fastp.json")

	assertNoTaskName(t, tasks, "star_align", "star_genome_generate", "bismark_align", "bismark_genome", "bismark_methylation_extractor")
}

func TestWGSOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "wgs.go", "AddTask")
}

func TestWGSAddsPinnedFAI(t *testing.T) {
	if !pinnedWGSFAI().Equal(gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta.fai"}) {
		t.Fatalf("pinnedWGSFAI() = %+v", pinnedWGSFAI())
	}
	assertCalls(t, "wgs.go", "AddInput")
}

func planAllTasks(t *testing.T, raw []byte) []planTaskRec {
	t.Helper()
	var decoded struct {
		Tasks []planTaskRec `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal plan: %v", err)
	}
	return decoded.Tasks
}

func countTasksNamed(tasks []planTaskRec, name string) int {
	n := 0
	for _, task := range tasks {
		if task.Name == name {
			n++
		}
	}
	return n
}

func mustHaveTaskID(t *testing.T, tasks []planTaskRec, id string) {
	t.Helper()
	for _, task := range tasks {
		if task.ID == id {
			return
		}
	}
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	t.Fatalf("plan missing task id %q in %v", id, ids)
}

func assertNoTaskName(t *testing.T, tasks []planTaskRec, names ...string) {
	t.Helper()
	banned := make(map[string]bool, len(names))
	for _, name := range names {
		banned[name] = true
	}
	for _, task := range tasks {
		if banned[task.Name] {
			t.Fatalf("plan has task %s id %q, want none", task.Name, task.ID)
		}
	}
}

func assertNoCall(t *testing.T, path string, names ...string) {
	t.Helper()
	got := fileCallNames(t, path)
	for _, name := range names {
		if got[name] {
			t.Fatalf("%s calls %s, want no raw %s", path, name, name)
		}
	}
}

func assertCalls(t *testing.T, path string, names ...string) {
	t.Helper()
	got := fileCallNames(t, path)
	for _, name := range names {
		if !got[name] {
			t.Fatalf("%s missing call %s", path, name)
		}
	}
}

func fileCallNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s) error = %v", path, err)
	}
	got := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			got[fun.Name] = true
		case *ast.SelectorExpr:
			got[fun.Sel.Name] = true
		}
		return true
	})
	return got
}
