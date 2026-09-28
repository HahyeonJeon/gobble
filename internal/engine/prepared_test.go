package engine

import (
	"bytes"
	"encoding/json"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
	"github.com/HahyeonJeon/gobble/internal/preparation"
	"reflect"
	"strings"
	"testing"
)

func preparationBinding() preparation.Binding {
	id := "sha256:" + strings.Repeat("a", 64)
	return preparation.Binding{id, id, id, pipelinereview.CreationReadPath, 1}
}
func TestPreparedRoundTripAndExactAdmission(t *testing.T) {
	doc, b := creationDocument(), preparationBinding()
	raw, id, err := Prepare(doc, b)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePrepared(raw, id, b)
	if err != nil || !reflect.DeepEqual(got, doc) {
		t.Fatalf("round trip: %v %#v", err, got)
	}
	got.Tasks[0].Command[0] = "changed"
	fresh, err := DecodePrepared(raw, id, b)
	if err != nil || fresh.Tasks[0].Command[0] == "changed" {
		t.Fatal("decoded alias", err)
	}
	for name, change := range map[string]func(*preparation.Binding){"source": func(x *preparation.Binding) { x.SourceRevision += "b" }, "input": func(x *preparation.Binding) { x.InputIdentity += "b" }, "runtime": func(x *preparation.Binding) { x.RuntimeID += "b" }, "cap": func(x *preparation.Binding) { x.Cap = 2 }, "path": func(x *preparation.Binding) { x.InputPath = "elsewhere" }} {
		t.Run(name, func(t *testing.T) {
			other := b
			change(&other)
			if _, err := DecodePrepared(raw, id, other); err == nil {
				t.Fatal("changed binding accepted")
			}
		})
	}
	for _, bad := range [][]byte{append(bytes.Clone(raw), ' '), bytes.Replace(raw, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":2`), 1), bytes.Replace(raw, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":1,"schemaVersion":1`), 1)} {
		if _, err := DecodePrepared(bad, preparedDigest(bad), b); err == nil {
			t.Fatal("noncanonical/unsupported payload accepted")
		}
	}
	plan, defects := BuildPlan(doc)
	if len(defects) > 0 {
		t.Fatal(defects)
	}
	public, _ := plan.MarshalJSON()
	if _, err := DecodePrepared(public, preparedDigest(public), b); err == nil {
		t.Fatal("public Plan accepted as execution payload")
	}
	raw[0] = '['
	if _, err := DecodePrepared(raw, id, b); err == nil {
		t.Fatal("tamper accepted")
	}
}
func TestPreparedRejectsHiddenExecutionBehavior(t *testing.T) {
	for _, mutate := range []func(*TaskPlan){func(t *TaskPlan) { t.Env = map[string]string{"TOKEN": "private"} }, func(t *TaskPlan) { t.ExecutableSHA256 = "other" }, func(t *TaskPlan) { t.Replace = true }, func(t *TaskPlan) { t.When = "later" }, func(t *TaskPlan) { t.ScatterMemberSpecs = []Path{{Opaque: "other"}} }} {
		doc := creationDocument()
		mutate(&doc.Tasks[0])
		if _, _, err := Prepare(doc, preparationBinding()); err == nil {
			t.Fatal("unreviewed behavior accepted")
		}
	}
}
func TestPreparedCodecPreservesCompletePrivateRepresentation(t *testing.T) {
	// A codec test, not permission to execute unqualified behavior: hidden fields
	// must survive the private encoding before qualification refuses them.
	d := creationDocument()
	d.Tasks[0].Env = map[string]string{"TOKEN": "private"}
	d.Tasks[0].ExecutablePath = "/tool"
	d.Tasks[0].Replace = true
	d.Tasks[0].ScatterMemberSpecs = []Path{{Literal: true, Opaque: "input", BadLit: true}}
	raw, e := json.Marshal(preparedV1{1, preparationBinding(), d})
	if e != nil {
		t.Fatal(e)
	}
	var v preparedV1
	if e = json.Unmarshal(raw, &v); e != nil || !reflect.DeepEqual(v.Document, d) {
		t.Fatal("private representation is lossy", e)
	}
}

func TestPreparedV1FieldSetRequiresExplicitFormatReview(t *testing.T) {
	// This private interchange deliberately tracks complete engine fields. A new
	// field must trigger a format/compatibility review rather than silently vanish.
	cases := []struct {
		value  any
		fields string
	}{
		{Document{}, "Name Tasks Edges"},
		{TaskPlan{}, "Display ID Name Instance ShardIndex ShardCount Attempt Module Branch Merge Scatter Gather When ScatterFromKind ScatterFromTask ScatterFromPort ScatterFromPath ScatterMembers ScatterMemberPaths ScatterMemberSpecs SkipIfMissingTask SkipIfMissingPort SkipIfMissingPath SkipIfFalse Command Script Image Backend Resources Params Env EnvDigest ExecutablePath ExecutableSHA256 Inputs Outputs Replace"},
		{Path{}, "Dir Prefix Base Suffixes Ext Literal Opaque BadLit"},
		{IO{}, "Name Kind Path Source Spec Rule Members Manifest"},
		{IOMember{}, "Name Path Source Spec"},
		{Edge{}, "FromTask FromPort ToTask ToPort Wait"},
		{ResourcePlan{}, "CPU Memory"},
		{ParamPlan{}, "Name Value"},
		{Display{}, "Stage Samples Scope"},
	}
	for _, c := range cases {
		typ := reflect.TypeOf(c.value)
		names := []string{}
		for i := range typ.NumField() {
			names = append(names, typ.Field(i).Name)
		}
		if strings.Join(names, " ") != c.fields {
			t.Fatalf("%s changed: review prepared format v1 compatibility", typ.Name())
		}
	}
}
