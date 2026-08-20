package cli_valid_test

import (
	"bytes"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/tests/cli-valid/runlocal"
)

func TestWorkflowCaseCLIPlan(t *testing.T) {
	bin := buildGobble(t)
	res := runGobble(t, bin, "plan", "./tests/cli-valid/workflowcase")
	if res.code != 0 {
		t.Fatalf("gobble plan exit = %d, want 0\nstderr: %s", res.code, res.stderr)
	}
	if len(res.stderr) != 0 {
		t.Fatalf("gobble plan stderr = %q, want empty", res.stderr)
	}
	want := readGolden(t, "testdata/workflow-case/plan.json")
	if !bytes.Equal(res.stdout, want) {
		t.Fatalf("gobble plan stdout != testdata/workflow-case/plan.json\ngot:\n%s\nwant:\n%s", res.stdout, want)
	}
}

func TestRunLocalWrapperPlanGolden(t *testing.T) {
	g, err := gobble.Compose(runlocal.Pipeline())
	if err != nil {
		t.Fatalf("Compose(runlocal.Pipeline()) error = %v, want nil", err)
	}
	var buf bytes.Buffer
	if _, err := gobble.BuildPlan(g, gobble.WriteTo(&buf)); err != nil {
		t.Fatalf("BuildPlan() error = %v, want nil", err)
	}
	want := readGolden(t, "testdata/run-local/plan.json")
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("BuildPlan JSON != testdata/run-local/plan.json\ngot:\n%s\nwant:\n%s", buf.Bytes(), want)
	}
}
