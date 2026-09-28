package main

import (
	"strings"
	"testing"
)

func TestCreationCommandsKeepExecutionSeparate(t *testing.T) {
	for _, args := range [][]string{{"creation-scaffold"}, {"creation-review", "./pipeline"}} {
		req, err := parse(args)
		if err != nil {
			t.Fatal(err)
		}
		if req.workspace != "" || req.cap != 0 {
			t.Fatal("execution flags leaked")
		}
		if args[0] == "creation-review" {
			source := driverSource("gobble.local/creation/pipeline", req, installIdentityResult{})
			if !strings.Contains(source, "gobble.InspectPipelineCreation(g, inputPath)") {
				t.Fatal("whole check not dispatched")
			}
		}
		if commandHelp[args[0]] == "" {
			t.Fatal("missing help")
		}
	}
	for _, args := range [][]string{{"creation-scaffold", "other"}, {"creation-scaffold", "--workspace", "work"}, {"creation-review", "--workspace", "work"}, {"creation-review", "--cap", "4"}, {"creation-scaffold", "--sample", "data"}} {
		if _, err := parse(args); err == nil {
			t.Fatalf("unexpected invocation accepted: %v", args)
		}
	}
}

func TestPreparationDriverKeepsEvaluationOutsideInstalledExecutionIdentity(t *testing.T) {
	req, err := parse([]string{"prepare", "./pipeline"})
	if err != nil {
		t.Fatal(err)
	}
	source := driverSource("example.test/pipeline", req, installIdentityResult{})
	if !strings.Contains(source, `verb != "creation-review" && verb != "prepare"`) || !strings.Contains(source, "gobble.PreparePipeline(g, binding)") {
		t.Fatal("preparation driver requires execution identity or lacks sealed preparation")
	}
	if commandHelp["prepare"] == "" {
		t.Fatal("missing private-payload CLI guidance")
	}
}
