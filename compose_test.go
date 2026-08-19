package gobble_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestComposeWorkflowCase(t *testing.T) {
	p := workflowCasePipeline()
	var zero gobble.Handle
	if !zero.IsZero() {
		t.Fatalf("case workflow-case: zero Handle.IsZero() got false, want true")
	}

	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case workflow-case: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case workflow-case: Compose() graph = nil, want non-nil")
	}
}

func TestComposeHandles(t *testing.T) {
	p := gobble.NewPipeline("handles")
	in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"})
	if in.IsZero() {
		t.Fatalf("case handles: AddInput().IsZero() got true, want false")
	}
	if in.Name() != "reads" {
		t.Fatalf("case handles: AddInput().Name() got %q, want %q", in.Name(), "reads")
	}
	if !in.Spec().Equal(gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"}) {
		t.Fatalf("case handles: AddInput().Spec() got %+v, want Name=sample Ext=.fastq.gz", in.Spec())
	}

	task := p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{{Name: "dst", Spec: gobble.PathSpec{Base: "out", Ext: ".txt"}}},
	})
	out := task.Out("dst")
	inPort := task.In("src")
	if out.IsZero() || inPort.IsZero() {
		t.Fatalf("case handles: Out/In IsZero() got true, want false")
	}
	if out.Name() != "dst" || inPort.Name() != "src" {
		t.Fatalf("case handles: port names got %q %q, want dst src", out.Name(), inPort.Name())
	}
	missing := gobble.NewPipeline("missing-port").AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}).Out("nope")
	if missing.IsZero() {
		t.Fatalf("case handles: Out(%q).IsZero() got true, want false", "nope")
	}
	if missing.Name() != "nope" {
		t.Fatalf("case handles: Out(%q).Name() got %q, want %q", "nope", missing.Name(), "nope")
	}

	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case handles: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case handles: Compose() graph = nil, want non-nil")
	}
}

func TestComposeReject(t *testing.T) {
	tests := []struct {
		name string
		pipe *gobble.Pipeline
		code gobble.DefectCode
		unit string
	}{
		{
			name: "cycle",
			pipe: cyclePipeline(),
			code: gobble.DefectCycle,
			unit: "loop",
		},
		{
			name: "missing-input zero From",
			pipe: missingInputPipeline(),
			code: gobble.DefectMissingInput,
			unit: "copy.reads",
		},
		{
			name: "missing-input In request",
			pipe: missingInRequestPipeline(),
			code: gobble.DefectMissingInput,
			unit: "copy.nope",
		},
		{
			name: "missing-output no binds",
			pipe: missingOutputPipeline(),
			code: gobble.DefectMissingOutput,
			unit: "copy",
		},
		{
			name: "missing-output Out request",
			pipe: missingOutRequestPipeline(),
			code: gobble.DefectMissingOutput,
			unit: "copy.nope",
		},
		{
			name: "missing-command",
			pipe: missingCommandPipeline(),
			code: gobble.DefectMissingCommand,
			unit: "copy",
		},
		{
			name: "invalid-name empty pipeline",
			pipe: invalidNameEmptyPipeline(),
			code: gobble.DefectInvalidName,
			unit: "pipeline",
		},
		{
			name: "invalid-name empty task",
			pipe: invalidNameEmptyTaskPipeline(),
			code: gobble.DefectInvalidName,
			unit: "prep",
		},
		{
			name: "invalid-name spelling",
			pipe: invalidNameSpellingPipeline(),
			code: gobble.DefectInvalidName,
			unit: "1copy",
		},
		{
			name: "invalid-name duplicate sibling",
			pipe: invalidNameDuplicatePipeline(),
			code: gobble.DefectInvalidName,
			unit: "copy",
		},
		{
			name: "invalid-path slash name",
			pipe: invalidPathSlashPipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "invalid-path empty lead and name",
			pipe: invalidPathEmptyPipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "invalid-path dir escape",
			pipe: invalidPathEscapePipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "invalid-path literal method",
			pipe: invalidPathLiteralPipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "foreign From colliding task id",
			pipe: foreignFromCollidingIDPipeline(),
			code: gobble.DefectMissingInput,
			unit: "use.in",
		},
		{
			name: "foreign output From colliding task id",
			pipe: foreignOutputFromCollidingIDPipeline(),
			code: gobble.DefectMissingInput,
			unit: "use.out",
		},
		{
			name: "foreign output From non-colliding task id",
			pipe: foreignOutputFromNonCollidingIDPipeline(),
			code: gobble.DefectMissingInput,
			unit: "use.out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := gobble.Compose(tt.pipe)
			if g != nil {
				t.Fatalf("case %s: Compose() graph != nil, want nil", tt.name)
			}
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: Compose() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "compose" {
				t.Fatalf("case %s: Error.Op got %q, want %q", tt.name, ge.Op, "compose")
			}
			found := false
			codes := make([]gobble.DefectCode, len(ge.Defects))
			units := make([]string, len(ge.Defects))
			for i, d := range ge.Defects {
				codes[i] = d.Code
				units[i] = d.Unit
				if d.Code == tt.code && d.Unit == tt.unit {
					found = true
				}
			}
			if !found {
				t.Fatalf("case %s: Compose() defects codes got %v units %v, want code %s unit %q", tt.name, codes, units, tt.code, tt.unit)
			}
		})
	}
}

func TestComposeForeignFromCollidingID(t *testing.T) {
	g, err := gobble.Compose(foreignFromCollidingIDPipeline())
	if g != nil {
		t.Fatalf("case foreign-from: Compose() graph != nil, want nil")
	}
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("case foreign-from: Compose() error = %v, want *Error", err)
	}
	if ge.Op != "compose" {
		t.Fatalf("case foreign-from: Error.Op got %q, want %q", ge.Op, "compose")
	}
	found := false
	codes := make([]gobble.DefectCode, len(ge.Defects))
	units := make([]string, len(ge.Defects))
	for i, d := range ge.Defects {
		codes[i] = d.Code
		units[i] = d.Unit
		if d.Code == gobble.DefectMissingInput && d.Unit == "use.in" {
			found = true
		}
	}
	if !found {
		t.Fatalf("case foreign-from: Compose() defects codes got %v units %v, want code %s unit %q", codes, units, gobble.DefectMissingInput, "use.in")
	}
}

func TestComposeForeignOutputFrom(t *testing.T) {
	tests := []struct {
		name string
		pipe *gobble.Pipeline
		unit string
	}{
		{
			name: "colliding task id",
			pipe: foreignOutputFromCollidingIDPipeline(),
			unit: "use.out",
		},
		{
			name: "non-colliding task id",
			pipe: foreignOutputFromNonCollidingIDPipeline(),
			unit: "use.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := gobble.Compose(tt.pipe)
			if g != nil {
				t.Fatalf("case foreign-output-from %s: Compose() graph != nil, want nil (no DAG edge)", tt.name)
			}
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case foreign-output-from %s: Compose() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "compose" {
				t.Fatalf("case foreign-output-from %s: Error.Op got %q, want %q", tt.name, ge.Op, "compose")
			}
			found := false
			codes := make([]gobble.DefectCode, len(ge.Defects))
			units := make([]string, len(ge.Defects))
			for i, d := range ge.Defects {
				codes[i] = d.Code
				units[i] = d.Unit
				if d.Code == gobble.DefectMissingInput && d.Unit == tt.unit {
					found = true
				}
			}
			if !found {
				t.Fatalf("case foreign-output-from %s: Compose() defects codes got %v units %v, want code %s unit %q", tt.name, codes, units, gobble.DefectMissingInput, tt.unit)
			}
		})
	}
}

func TestComposeRejectDeclarations(t *testing.T) {
	tests := []struct {
		name    string
		pipe    *gobble.Pipeline
		code    gobble.DefectCode
		unit    string
		message string
	}{
		{
			name:    "group and spec both set",
			pipe:    groupAndSpecPipeline(),
			code:    gobble.DefectInvalidValue,
			unit:    "index.idx",
			message: "group and spec both set",
		},
		{
			name:    "empty group",
			pipe:    emptyGroupPipeline(),
			code:    gobble.DefectInvalidValue,
			unit:    "index.idx",
			message: "empty group",
		},
		{
			name:    "empty member name",
			pipe:    emptyMemberNamePipeline(),
			code:    gobble.DefectInvalidName,
			unit:    "index.idx",
			message: "empty name",
		},
		{
			name:    "duplicate member name",
			pipe:    duplicateMemberNamePipeline(),
			code:    gobble.DefectInvalidName,
			unit:    "index.idx.amb",
			message: "duplicate name",
		},
		{
			name: "invalid member name",
			pipe: invalidMemberNamePipeline(),
			code: gobble.DefectInvalidName,
			unit: "index.idx.1amb",
		},
		{
			name: "group from single-file",
			pipe: groupFromSingleFilePipeline(),
			code: gobble.DefectMissingInput,
			unit: "mem.idx",
		},
		{
			name: "single-file from group",
			pipe: singleFileFromGroupPipeline(),
			code: gobble.DefectMissingInput,
			unit: "mem.idx",
		},
		{
			name: "group from member-set mismatch",
			pipe: groupFromMismatchPipeline(),
			code: gobble.DefectMissingInput,
			unit: "mem.idx",
		},
		{
			name: "group from single-file pipeline input",
			pipe: groupFromSingleFilePipelineInputPipeline(),
			code: gobble.DefectMissingInput,
			unit: "mem.idx",
		},
		{
			name:    "empty input group",
			pipe:    emptyInputGroupPipeline(),
			code:    gobble.DefectInvalidValue,
			unit:    "idx",
			message: "empty group",
		},
		{
			name:    "command and script both set",
			pipe:    commandAndScriptPipeline(),
			code:    gobble.DefectInvalidValue,
			unit:    "copy",
			message: "command and script both set",
		},
		{
			name:    "empty env key",
			pipe:    emptyEnvKeyPipeline(),
			code:    gobble.DefectInvalidValue,
			unit:    "copy",
			message: "empty env key",
		},
		{
			name:    "empty env value",
			pipe:    emptyEnvValuePipeline(),
			code:    gobble.DefectInvalidValue,
			unit:    "copy",
			message: "empty env value",
		},
		{
			name:    "env key contains =",
			pipe:    envKeyEqualsPipeline(),
			code:    gobble.DefectInvalidValue,
			unit:    "copy",
			message: "env key contains =",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := gobble.Compose(tt.pipe)
			if g != nil {
				t.Fatalf("case %s: Compose() graph != nil, want nil", tt.name)
			}
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: Compose() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "compose" {
				t.Fatalf("case %s: Error.Op got %q, want %q", tt.name, ge.Op, "compose")
			}
			found := false
			codes := make([]gobble.DefectCode, len(ge.Defects))
			units := make([]string, len(ge.Defects))
			messages := make([]string, len(ge.Defects))
			for i, d := range ge.Defects {
				codes[i] = d.Code
				units[i] = d.Unit
				messages[i] = d.Message
				if d.Code == tt.code && d.Unit == tt.unit && (tt.message == "" || d.Message == tt.message) {
					found = true
				}
			}
			if !found {
				t.Fatalf("case %s: Compose() defects codes got %v units %v messages %v, want code %s unit %q message %q",
					tt.name, codes, units, messages, tt.code, tt.unit, tt.message)
			}
		})
	}
}

func TestComposeGroupFromPipelineInput(t *testing.T) {
	p := gobble.NewPipeline("group-from-in")
	in := p.AddInputGroup("idx", gobble.Group{
		{Name: "amb", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}},
	})
	if in.IsZero() {
		t.Fatalf("case group from pipeline input: AddInputGroup().IsZero() got true, want false")
	}
	if in.Name() != "idx" {
		t.Fatalf("case group from pipeline input: AddInputGroup().Name() got %q, want %q", in.Name(), "idx")
	}
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  in,
			Group: gobble.Group{{Name: "amb"}},
		}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case group from pipeline input: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case group from pipeline input: Compose() graph = nil, want non-nil")
	}
	raw := mustBuildPlanJSON(t, groupFromPipelineInputPipeline())
	type member struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	var decoded struct {
		Tasks []struct {
			ID     string `json:"id"`
			Inputs []struct {
				Name    string   `json:"name"`
				Members []member `json:"members"`
			} `json:"inputs"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case group from pipeline input: Unmarshal() error = %v", err)
	}
	var inMembers []member
	for _, task := range decoded.Tasks {
		if task.ID != "mem" {
			continue
		}
		for _, b := range task.Inputs {
			if b.Name == "idx" {
				inMembers = b.Members
			}
		}
	}
	want := []member{{Name: "amb", Path: "ref.amb"}}
	if !jsonEqual(inMembers, want) {
		t.Fatalf("case group from pipeline input: mem.idx members got %#v, want %#v", inMembers, want)
	}
}

func TestComposeGroupHandle(t *testing.T) {
	p := gobble.NewPipeline("group-handle")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{groupOut("idx")},
	})
	out := src.Out("idx")
	if out.IsZero() {
		t.Fatalf("case group-handle: Out(%q).IsZero() got true, want false", "idx")
	}
	if out.Name() != "idx" {
		t.Fatalf("case group-handle: Out(%q).Name() got %q, want %q", "idx", out.Name(), "idx")
	}
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  out,
			Group: gobble.Group{{Name: "amb"}, {Name: "ann"}},
		}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case group-handle: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case group-handle: Compose() graph = nil, want non-nil")
	}
}

func TestComposeGroupPlanMembers(t *testing.T) {
	raw := mustBuildPlanJSON(t, matchingGroupPipeline())
	type member struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	var decoded struct {
		Tasks []struct {
			ID     string `json:"id"`
			Inputs []struct {
				Name    string   `json:"name"`
				Path    string   `json:"path"`
				Members []member `json:"members"`
			} `json:"inputs"`
			Outputs []struct {
				Name    string   `json:"name"`
				Path    string   `json:"path"`
				Members []member `json:"members"`
			} `json:"outputs"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case group-plan: Unmarshal() error = %v", err)
	}
	var outMembers []member
	var inMembers []member
	var inPath string
	var outPath string
	for _, task := range decoded.Tasks {
		if task.ID == "index" {
			for _, b := range task.Outputs {
				if b.Name == "idx" {
					outPath = b.Path
					outMembers = b.Members
				}
			}
		}
		if task.ID == "mem" {
			for _, b := range task.Inputs {
				if b.Name == "idx" {
					inPath = b.Path
					inMembers = b.Members
				}
			}
		}
	}
	if outPath != "" {
		t.Fatalf("case group-plan: index.idx path got %q, want empty", outPath)
	}
	if inPath != "" {
		t.Fatalf("case group-plan: mem.idx path got %q, want empty", inPath)
	}
	want := []member{{Name: "amb", Path: "ref.amb"}, {Name: "ann", Path: "ref.ann"}}
	if !jsonEqual(outMembers, want) {
		t.Fatalf("case group-plan: index.idx members got %#v, want %#v", outMembers, want)
	}
	if !jsonEqual(inMembers, want) {
		t.Fatalf("case group-plan: mem.idx members got %#v, want %#v", inMembers, want)
	}
	for _, task := range decoded.Tasks {
		if task.ID != "mem" {
			continue
		}
		for _, b := range task.Outputs {
			if b.Name == "out" && b.Members != nil {
				t.Fatalf("case group-plan: single-file out members got %#v, want omitted", b.Members)
			}
		}
	}
}

func TestComposeScriptPlan(t *testing.T) {
	raw := mustBuildPlanJSON(t, oneTask("script", gobble.TaskSpec{
		Name:    "copy",
		Script:  "echo hi",
		Outputs: []gobble.Bind{fileOut("out")},
	}))
	var decoded struct {
		Tasks []struct {
			Command []string `json:"command"`
			Script  string   `json:"script"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case script-plan: Unmarshal() error = %v", err)
	}
	if len(decoded.Tasks) != 1 {
		t.Fatalf("case script-plan: tasks got %d, want 1", len(decoded.Tasks))
	}
	task := decoded.Tasks[0]
	if task.Script != "echo hi" {
		t.Fatalf("case script-plan: script got %q, want %q", task.Script, "echo hi")
	}
	wantCmd := []string{"sh", "-c", "set -eu\necho hi"}
	if !jsonEqual(task.Command, wantCmd) {
		t.Fatalf("case script-plan: command got %#v, want %#v", task.Command, wantCmd)
	}

	raw = mustBuildPlanJSON(t, oneTask("script-set", gobble.TaskSpec{
		Name:    "copy",
		Script:  "set -eu\necho hi",
		Outputs: []gobble.Bind{fileOut("out")},
	}))
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case script-set: Unmarshal() error = %v", err)
	}
	wantCmd = []string{"sh", "-c", "set -eu\nset -eu\necho hi"}
	if !jsonEqual(decoded.Tasks[0].Command, wantCmd) {
		t.Fatalf("case script-set: command got %#v, want %#v", decoded.Tasks[0].Command, wantCmd)
	}
}

func TestComposeEnvPlan(t *testing.T) {
	raw := mustBuildPlanJSON(t, oneTask("env", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Env:     map[string]string{"HOME": "/tmp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}))
	var decoded struct {
		Tasks []struct {
			Env map[string]string `json:"env"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case env-plan: Unmarshal() error = %v", err)
	}
	if len(decoded.Tasks) != 1 {
		t.Fatalf("case env-plan: tasks got %d, want 1", len(decoded.Tasks))
	}
	if decoded.Tasks[0].Env["HOME"] != "/tmp" {
		t.Fatalf("case env-plan: env got %#v, want HOME=/tmp", decoded.Tasks[0].Env)
	}

	raw = mustBuildPlanJSON(t, oneTask("no-env", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}))
	var rawObj map[string]any
	if err := json.Unmarshal(raw, &rawObj); err != nil {
		t.Fatalf("case no-env: Unmarshal() error = %v", err)
	}
	tasks, _ := rawObj["tasks"].([]any)
	if len(tasks) != 1 {
		t.Fatalf("case no-env: tasks got %d, want 1", len(tasks))
	}
	task, _ := tasks[0].(map[string]any)
	if _, ok := task["env"]; ok {
		t.Fatalf("case no-env: env key present, want omitted")
	}
	if _, ok := task["script"]; ok {
		t.Fatalf("case no-env: script key present, want omitted")
	}
}

func TestComposePipelineBranchMerge(t *testing.T) {
	p := gobble.NewPipeline("root-branch")
	align := p.Branch("align")
	qc := p.Branch("qc")
	join := p.Merge("join")
	src := align.AddTask(gobble.TaskSpec{
		Name:    "bwa",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Base: "aln", Ext: ".bam"}}},
	})
	qc.AddTask(gobble.TaskSpec{
		Name:    "fastqc",
		Command: []string{"fastqc"},
		Outputs: []gobble.Bind{{Name: "html", Spec: gobble.PathSpec{Base: "qc", Ext: ".html"}}},
	})
	join.AddTask(gobble.TaskSpec{
		Name:    "report",
		Command: []string{"report"},
		Inputs:  []gobble.Bind{{Name: "bam", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{Name: "summary", Spec: gobble.PathSpec{Base: "report", Ext: ".json"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case pipeline-branch-merge: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case pipeline-branch-merge: Compose() graph = nil, want non-nil")
	}
	if err := gobble.Validate(g); err != nil {
		t.Fatalf("case pipeline-branch-merge: Validate() error = %v, want nil", err)
	}
}

func TestComposeNestedModuleID(t *testing.T) {
	p := gobble.NewPipeline("nested")
	p.AddModule("a").AddModule("b").AddTask(gobble.TaskSpec{
		Name:    "task",
		Command: []string{"echo"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	raw := mustBuildPlanJSON(t, p)
	var decoded struct {
		Tasks []struct {
			ID     string `json:"id"`
			Module string `json:"module"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case nested-module: Unmarshal() error = %v", err)
	}
	if len(decoded.Tasks) != 1 {
		t.Fatalf("case nested-module: tasks got %d, want 1", len(decoded.Tasks))
	}
	if decoded.Tasks[0].ID != "a.b.task" {
		t.Fatalf("case nested-module: task id got %q, want %q", decoded.Tasks[0].ID, "a.b.task")
	}
	if decoded.Tasks[0].Module != "b" {
		t.Fatalf("case nested-module: module got %q, want %q", decoded.Tasks[0].Module, "b")
	}
}

func TestComposeDeriveReplaceExt(t *testing.T) {
	p := gobble.NewPipeline("picard-bai")
	align := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Base: "aln", Ext: ".bam"}}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"picard"},
		Inputs:  []gobble.Bind{{Name: "bam", From: align.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "bai",
			Spec: gobble.PathSpec{Ext: ".bai"},
			From: align.Out("bam"),
			Rule: gobble.DeriveReplaceExt,
		}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planOutputPath(t, raw, "index", "bai")
	if got != "aln.bai" {
		t.Fatalf("case derive-replace-ext: index.bai path got %q, want %q", got, "aln.bai")
	}
}

func TestComposeDirOnlyRestage(t *testing.T) {
	p := gobble.NewPipeline("dir-restage")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name: "bam",
			Spec: gobble.PathSpec{Dir: gobble.Dir("work/align"), Base: "sample", Suffixes: []string{"sorted"}, Ext: ".bam"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "in", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: src.Out("bam"),
			Spec: gobble.PathSpec{Dir: gobble.Dir("out")},
		}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planOutputPath(t, raw, "copy", "out")
	if got != "out/sample.sorted.bam" {
		t.Fatalf("case dir-only-restage: copy.out path got %q, want %q", got, "out/sample.sorted.bam")
	}
}

func TestComposeStepsOnlyRestage(t *testing.T) {
	p := gobble.NewPipeline("steps-restage")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name: "bam",
			Spec: gobble.PathSpec{Dir: gobble.Dir("work"), Base: "sample", Suffixes: []string{"sorted"}, Ext: ".bam"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "mark",
		Command: []string{"picard"},
		Inputs:  []gobble.Bind{{Name: "in", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: src.Out("bam"),
			Spec: gobble.PathSpec{Suffixes: []string{"markdup"}},
		}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planOutputPath(t, raw, "mark", "out")
	if got != "work/sample.markdup.bam" {
		t.Fatalf("case steps-only-restage: mark.out path got %q, want %q", got, "work/sample.markdup.bam")
	}
}

func TestComposeBranchAddModule(t *testing.T) {
	p := gobble.NewPipeline("branch-mod")
	p.Branch("align").AddModule("bwa").AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	raw := mustBuildPlanJSON(t, p)
	var decoded struct {
		Tasks []struct {
			ID     string `json:"id"`
			Module string `json:"module"`
			Branch string `json:"branch"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case branch-add-module: Unmarshal() error = %v", err)
	}
	if len(decoded.Tasks) != 1 {
		t.Fatalf("case branch-add-module: tasks got %d, want 1", len(decoded.Tasks))
	}
	if decoded.Tasks[0].ID != "align.bwa.mem" {
		t.Fatalf("case branch-add-module: task id got %q, want %q", decoded.Tasks[0].ID, "align.bwa.mem")
	}
	if decoded.Tasks[0].Module != "bwa" {
		t.Fatalf("case branch-add-module: module got %q, want %q", decoded.Tasks[0].Module, "bwa")
	}
	if decoded.Tasks[0].Branch != "align" {
		t.Fatalf("case branch-add-module: branch got %q, want %q", decoded.Tasks[0].Branch, "align")
	}
}

func TestComposeFromInPort(t *testing.T) {
	p := gobble.NewPipeline("from-in")
	in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fq"})
	prep := p.AddTask(gobble.TaskSpec{
		Name:    "prep",
		Command: []string{"prep"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "prep", Ext: ".fq"}}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "in", From: prep.In("src")}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "copy", Ext: ".fq"}}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planInputPath(t, raw, "copy", "in")
	if got != "sample.fq" {
		t.Fatalf("case from-in: copy.in path got %q, want %q", got, "sample.fq")
	}
}

func TestHandleSpecIsAuthored(t *testing.T) {
	p := gobble.NewPipeline("authored-spec")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Base: "aln", Ext: ".bam"}}},
	})
	h := src.Out("bam")
	if !h.Spec().Equal(gobble.PathSpec{Base: "aln", Ext: ".bam"}) {
		t.Fatalf("case authored-spec: Out.Spec() got %+v, want Name=aln Ext=.bam", h.Spec())
	}
	p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"samtools"},
		Inputs:  []gobble.Bind{{Name: "bam", From: h}},
		Outputs: []gobble.Bind{{Name: "bai", Spec: gobble.PathSpec{Ext: ".bai"}, From: h}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planOutputPath(t, raw, "index", "bai")
	if got != "aln.bam.bai" {
		t.Fatalf("case authored-spec: index.bai path got %q, want %q", got, "aln.bam.bai")
	}
	if !h.Spec().Equal(gobble.PathSpec{Base: "aln", Ext: ".bam"}) {
		t.Fatalf("case authored-spec: Out.Spec() after plan got %+v, want authored Name=aln Ext=.bam", h.Spec())
	}
}

func ExampleCompose() {
	p := gobble.NewPipeline("demo")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Base: "sample", Ext: ".bam"}}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"samtools"},
		Inputs:  []gobble.Bind{{Name: "bam", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{Name: "bai", Spec: gobble.PathSpec{Ext: ".bai"}, From: src.Out("bam")}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		return
	}
	if err := gobble.Validate(g); err != nil {
		return
	}
	if _, err := gobble.BuildPlan(g); err != nil {
		return
	}
	fmt.Println("ok")
	// Output: ok
}

func workflowCasePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("workflow-case")
	readsR1 := p.AddInput("reads_r1", gobble.PathSpec{
		Dir:    gobble.Dir("in"),
		Prefix: "sample_S1_L001_R1_",
		Base:   "001",
		Ext:    ".fastq.gz",
	})
	readsR2 := p.AddInput("reads_r2", gobble.PathSpec{
		Dir:    gobble.Dir("in"),
		Prefix: "sample_S1_L001_R2_",
		Base:   "001",
		Ext:    ".fastq.gz",
	})

	prep := p.AddModule("prep")
	fastp := addFastp(prep, readsR1, readsR2)

	call := p.AddModule("call")
	align := call.Branch("align")
	qc := call.Branch("qc")
	join := call.Merge("join")

	bam := gobble.PathSpec{
		Dir:      gobble.Dir("work/align"),
		Base:     "sample",
		Suffixes: []string{"sorted"},
		Ext:      ".bam",
	}
	bwa := align.AddTask(gobble.TaskSpec{
		Name:    "bwa",
		Command: []string{"bwa"},
		Image:   "example/bwa:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: fastp.Out("clean_r1")},
			{Name: "r2", From: fastp.Out("clean_r2")},
		},
		Outputs: []gobble.Bind{
			{Name: "bam", Spec: bam},
			{Name: "bai", Spec: bam.AppendExt(".bai")},
		},
	})

	fastqc := qc.AddTask(gobble.TaskSpec{
		Name:    "fastqc",
		Command: []string{"fastqc"},
		Image:   "example/fastqc:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: fastp.Out("clean_r1")},
			{Name: "r2", From: fastp.Out("clean_r2")},
		},
		Outputs: []gobble.Bind{
			{Name: "html", Spec: gobble.Literal("sample_clean_fastqc.html").WithDir(gobble.Dir("work/qc"))},
		},
	})

	join.AddTask(gobble.TaskSpec{
		Name:    "report",
		Command: []string{"report"},
		Inputs: []gobble.Bind{
			{Name: "bam", From: bwa.Out("bam")},
			{Name: "bai", From: bwa.Out("bai")},
			{Name: "html", From: fastqc.Out("html")},
		},
		Outputs: []gobble.Bind{
			{Name: "summary", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "report", Ext: ".json"}},
		},
	})
	return p
}

func addFastp(prep *gobble.Module, r1, r2 gobble.Handle) *gobble.Task {
	return prep.AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Image:   "example/fastp:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs: []gobble.Bind{
			{
				Name: "clean_r1",
				Spec: gobble.PathSpec{
					Dir:      gobble.Dir("work/prep"),
					Prefix:   "sample_S1_L001_R1_",
					Base:     "001",
					Suffixes: []string{"clean"},
					Ext:      ".fastq.gz",
				},
			},
			{
				Name: "clean_r2",
				Spec: gobble.PathSpec{
					Dir:      gobble.Dir("work/prep"),
					Prefix:   "sample_S1_L001_R2_",
					Base:     "001",
					Suffixes: []string{"clean"},
					Ext:      ".fastq.gz",
				},
			},
		},
	})
}

func oneTask(name string, spec gobble.TaskSpec) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	p.AddTask(spec)
	return p
}

func fileOut(name string) gobble.Bind {
	return gobble.Bind{Name: name, Spec: gobble.PathSpec{Base: "out", Ext: ".txt"}}
}

func cyclePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("cycle")
	inputs := []gobble.Bind{{Name: "in"}}
	t := p.AddTask(gobble.TaskSpec{
		Name:    "loop",
		Command: []string{"echo"},
		Inputs:  inputs,
		Outputs: []gobble.Bind{fileOut("out")},
	})
	inputs[0].From = t.Out("out")
	return p
}

func missingInputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-input")
	p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "reads"}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func missingInRequestPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-in")
	in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"})
	t := p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	t.In("nope")
	return p
}

func missingOutputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-output")
	in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
	})
	return p
}

func missingOutRequestPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-out")
	in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq.gz"})
	t := p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	t.Out("nope")
	return p
}

func missingCommandPipeline() *gobble.Pipeline {
	return oneTask("missing-command", gobble.TaskSpec{
		Name:    "copy",
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func invalidNameEmptyPipeline() *gobble.Pipeline {
	return oneTask("", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func invalidNameEmptyTaskPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("empty-task")
	p.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func invalidNameSpellingPipeline() *gobble.Pipeline {
	return oneTask("invalid-spelling", gobble.TaskSpec{
		Name:    "1copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func invalidNameDuplicatePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("dup")
	spec := gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}
	p.AddTask(spec)
	p.AddTask(spec)
	return p
}

func invalidPathSlashPipeline() *gobble.Pipeline {
	return oneTask("slash", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "a/b", Ext: ".txt"}}},
	})
}

func invalidPathEmptyPipeline() *gobble.Pipeline {
	return oneTask("empty-path", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Ext: ".txt"}}},
	})
}

func invalidPathEscapePipeline() *gobble.Pipeline {
	return oneTask("escape", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("../out"), Base: "x", Ext: ".txt"}}},
	})
}

func invalidPathLiteralPipeline() *gobble.Pipeline {
	return oneTask("literal", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.Literal("aln.bam").AppendSuffix("sorted")}},
	})
}

func foreignFromCollidingIDPipeline() *gobble.Pipeline {
	a := gobble.NewPipeline("run-a")
	ta := a.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Base: "foreign", Ext: ".fq"}}},
	})
	b := gobble.NewPipeline("run-b")
	b.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Base: "local", Ext: ".fq"}}},
	})
	b.AddTask(gobble.TaskSpec{
		Name:    "use",
		Command: []string{"use"},
		Inputs:  []gobble.Bind{{Name: "in", From: ta.Out("clean")}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return b
}

func foreignOutputFromCollidingIDPipeline() *gobble.Pipeline {
	a := gobble.NewPipeline("run-a")
	ta := a.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Base: "foreign", Ext: ".fq"}}},
	})
	b := gobble.NewPipeline("run-b")
	b.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Base: "local", Ext: ".fq"}}},
	})
	b.AddTask(gobble.TaskSpec{
		Name:    "use",
		Command: []string{"use"},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: ta.Out("clean"),
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "use", Ext: ".txt"},
		}},
	})
	return b
}

func groupOut(name string) gobble.Bind {
	return gobble.Bind{
		Name: name,
		Group: gobble.Group{
			{Name: "amb", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}},
			{Name: "ann", Spec: gobble.PathSpec{Base: "ref", Ext: ".ann"}},
		},
	}
}

func matchingGroupPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("group")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{groupOut("idx")},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  src.Out("idx"),
			Group: gobble.Group{{Name: "amb"}, {Name: "ann"}},
		}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func groupAndSpecPipeline() *gobble.Pipeline {
	return oneTask("group-xor", gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name:  "idx",
			Spec:  gobble.PathSpec{Base: "ref", Ext: ".amb"},
			Group: gobble.Group{{Name: "amb", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}}},
		}},
	})
}

func emptyGroupPipeline() *gobble.Pipeline {
	return oneTask("empty-group", gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name:  "idx",
			Group: gobble.Group{},
		}},
	})
}

func emptyMemberNamePipeline() *gobble.Pipeline {
	return oneTask("empty-member", gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name: "idx",
			Group: gobble.Group{
				{Name: "", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}},
			},
		}},
	})
}

func duplicateMemberNamePipeline() *gobble.Pipeline {
	return oneTask("dup-member", gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name: "idx",
			Group: gobble.Group{
				{Name: "amb", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}},
				{Name: "amb", Spec: gobble.PathSpec{Base: "ref", Ext: ".ann"}},
			},
		}},
	})
}

func invalidMemberNamePipeline() *gobble.Pipeline {
	return oneTask("bad-member", gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name: "idx",
			Group: gobble.Group{
				{Name: "1amb", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}},
			},
		}},
	})
}

func groupFromSingleFilePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("group-from-file")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "idx", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  src.Out("idx"),
			Group: gobble.Group{{Name: "amb"}},
		}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func singleFileFromGroupPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("file-from-group")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{groupOut("idx")},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs:  []gobble.Bind{{Name: "idx", From: src.Out("idx")}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func groupFromMismatchPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("group-mismatch")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{groupOut("idx")},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  src.Out("idx"),
			Group: gobble.Group{{Name: "amb"}, {Name: "pac"}},
		}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func groupFromPipelineInputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("group-from-in")
	in := p.AddInputGroup("idx", gobble.Group{
		{Name: "amb", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  in,
			Group: gobble.Group{{Name: "amb"}},
		}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func groupFromSingleFilePipelineInputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("group-from-file-in")
	in := p.AddInput("idx", gobble.PathSpec{Base: "ref", Ext: ".amb"})
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Inputs: []gobble.Bind{{
			Name:  "idx",
			From:  in,
			Group: gobble.Group{{Name: "amb"}},
		}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func emptyInputGroupPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("empty-input-group")
	p.AddInputGroup("idx", nil)
	p.AddTask(gobble.TaskSpec{
		Name:    "mem",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func commandAndScriptPipeline() *gobble.Pipeline {
	return oneTask("cmd-script", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Script:  "echo hi",
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func emptyEnvKeyPipeline() *gobble.Pipeline {
	return oneTask("empty-env-key", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Env:     map[string]string{"": "x"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func emptyEnvValuePipeline() *gobble.Pipeline {
	return oneTask("empty-env-value", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Env:     map[string]string{"HOME": ""},
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func envKeyEqualsPipeline() *gobble.Pipeline {
	return oneTask("env-eq", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Env:     map[string]string{"HO=ME": "/tmp"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func groupMemberConflictPipeline() *gobble.Pipeline {
	return oneTask("group-conflict", gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{
			groupOut("idx"),
			{Name: "collide", Spec: gobble.PathSpec{Base: "ref", Ext: ".amb"}},
		},
	})
}

func groupOutputFromPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("group-out-from")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{groupOut("idx")},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{
			Name: "idx",
			From: src.Out("idx"),
			Group: gobble.Group{
				{Name: "amb", Spec: gobble.PathSpec{Dir: gobble.Dir("out")}},
				{Name: "ann", Spec: gobble.PathSpec{Dir: gobble.Dir("out")}},
			},
		}},
	})
	return p
}

func outputPortFromPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("out-from")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Base: "aln", Ext: ".bam"}}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"samtools"},
		Inputs:  []gobble.Bind{{Name: "bam", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "bai",
			Spec: gobble.PathSpec{Ext: ".bai"},
			From: src.Out("bam"),
		}},
	})
	return p
}

func foreignOutputFromNonCollidingIDPipeline() *gobble.Pipeline {
	a := gobble.NewPipeline("run-a")
	ta := a.AddTask(gobble.TaskSpec{
		Name:    "src",
		Command: []string{"echo"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "foreign", Ext: ".txt"}}},
	})
	b := gobble.NewPipeline("run-b")
	b.AddTask(gobble.TaskSpec{
		Name:    "use",
		Command: []string{"use"},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: ta.Out("out"),
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "use", Ext: ".txt"},
		}},
	})
	return b
}

func mustBuildPlanJSON(t *testing.T, p *gobble.Pipeline) []byte {
	t.Helper()
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("Compose() graph = nil, want non-nil")
	}
	var buf bytes.Buffer
	plan, err := gobble.BuildPlan(g, gobble.WriteTo(&buf))
	if err != nil {
		t.Fatalf("BuildPlan() error = %v, want nil", err)
	}
	if plan == nil {
		t.Fatalf("BuildPlan() plan = nil, want non-nil")
	}
	return buf.Bytes()
}

func planOutputPath(t *testing.T, raw []byte, taskID, port string) string {
	t.Helper()
	return planBindPath(t, raw, taskID, port, true)
}

func planInputPath(t *testing.T, raw []byte, taskID, port string) string {
	t.Helper()
	return planBindPath(t, raw, taskID, port, false)
}

func planBindPath(t *testing.T, raw []byte, taskID, port string, output bool) string {
	t.Helper()
	type bind struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	var decoded struct {
		Tasks []struct {
			ID      string `json:"id"`
			Inputs  []bind `json:"inputs"`
			Outputs []bind `json:"outputs"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, task := range decoded.Tasks {
		if task.ID != taskID {
			continue
		}
		binds := task.Inputs
		if output {
			binds = task.Outputs
		}
		for _, b := range binds {
			if b.Name == port {
				return b.Path
			}
		}
	}
	t.Fatalf("plan missing %s.%s", taskID, port)
	return ""
}
