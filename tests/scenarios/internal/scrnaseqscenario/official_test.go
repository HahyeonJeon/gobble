package scrnaseqscenario

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestOfficialMatrixToH5ADMetadataOracleRejectsSubjectMetadata(t *testing.T) {
	raw := Plan(t, scrnaseq.DefaultConfig())
	for _, taskID := range []string{"Sample_X.matrix_to_h5ad", "Sample_Y.matrix_to_h5ad"} {
		task := pc.TaskByID(t, raw, taskID)
		if _, err := commandInputPaths(task); err != nil {
			t.Fatalf("commandInputPaths(%s) baseline error = %v", taskID, err)
		}
		for _, field := range []struct {
			name       string
			operandPos int
			wrong      string
		}{
			{name: "sample", operandPos: 3, wrong: "Wrong_Sample"},
			{name: "expected_cells", operandPos: 2, wrong: "9999"},
			{name: "seq_center", operandPos: 1, wrong: "Wrong Center"},
		} {
			t.Run(taskID+"/"+field.name, func(t *testing.T) {
				changed := task
				changed.Command = append([]string(nil), task.Command...)
				changed.Params = append([]pc.Param(nil), task.Params...)
				foundParam := false
				for i := range changed.Params {
					if changed.Params[i].Name == field.name {
						changed.Params[i].Value = field.wrong
						foundParam = true
					}
				}
				if !foundParam {
					t.Fatalf("official task %s has no %s Param", taskID, field.name)
				}
				changed.Command[len(changed.Command)-field.operandPos] = field.wrong
				if _, err := commandInputPaths(changed); err == nil {
					t.Fatalf("commandInputPaths(%s) accepted %s %q propagated through Params and argv", taskID, field.name, field.wrong)
				} else if message := err.Error(); !strings.Contains(message, "embedded python operands") || !strings.Contains(message, field.wrong) || !strings.Contains(message, "want") {
					t.Fatalf("commandInputPaths(%s) error = %q, want independent metadata-oracle mismatch", taskID, err)
				}
			})
		}
	}
}
