package engine

import (
	"math"
	"testing"
)

func TestComposeCheckIllegalSnapshots(t *testing.T) {
	file := Path{Name: "out", Ext: ".txt"}
	tests := []struct {
		name string
		snap Snapshot
		code string
		unit string
	}{
		{
			name: "cycle",
			snap: Snapshot{
				Name: "cycle",
				Tasks: []Task{{
					ID:      "loop",
					Name:    "loop",
					Command: []string{"echo"},
					Inputs: []Bind{{
						Name:     "in",
						FromKind: FromOut,
						FromTask: "loop",
						FromName: "out",
					}},
					Outputs: []Bind{{Name: "out", Spec: file}},
				}},
			},
			code: DefectCycle,
			unit: "loop",
		},
		{
			name: "missing-input zero From",
			snap: Snapshot{
				Name: "missing-input",
				Tasks: []Task{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Inputs:  []Bind{{Name: "reads"}},
					Outputs: []Bind{{Name: "out", Spec: file}},
				}},
			},
			code: DefectMissingInput,
			unit: "copy.reads",
		},
		{
			name: "missing-output no binds",
			snap: Snapshot{
				Name: "missing-output",
				Tasks: []Task{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Inputs: []Bind{{
						Name:     "src",
						FromKind: FromInput,
						FromName: "reads",
					}},
				}},
				Inputs: []Input{{Name: "reads", Spec: Path{Name: "sample", Ext: ".fastq.gz"}}},
			},
			code: DefectMissingOutput,
			unit: "copy",
		},
		{
			name: "missing-output Out request",
			snap: Snapshot{
				Name: "missing-out",
				Tasks: []Task{{
					ID:       "copy",
					Name:     "copy",
					Command:  []string{"cp"},
					Outputs:  []Bind{{Name: "out", Spec: file}},
					OutCalls: []string{"nope"},
				}},
			},
			code: DefectMissingOutput,
			unit: "copy.nope",
		},
		{
			name: "missing-command",
			snap: Snapshot{
				Name: "missing-command",
				Tasks: []Task{{
					ID:      "copy",
					Name:    "copy",
					Outputs: []Bind{{Name: "out", Spec: file}},
				}},
			},
			code: DefectMissingCommand,
			unit: "copy",
		},
		{
			name: "invalid-name empty pipeline",
			snap: Snapshot{
				Name: "",
				Tasks: []Task{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Outputs: []Bind{{Name: "out", Spec: file}},
				}},
			},
			code: DefectInvalidName,
			unit: "pipeline",
		},
		{
			name: "invalid-name spelling",
			snap: Snapshot{
				Name: "invalid-spelling",
				Tasks: []Task{{
					ID:      "1copy",
					Name:    "1copy",
					Command: []string{"cp"},
					Outputs: []Bind{{Name: "out", Spec: file}},
				}},
			},
			code: DefectInvalidName,
			unit: "1copy",
		},
		{
			name: "invalid-path slash name",
			snap: Snapshot{
				Name: "slash",
				Tasks: []Task{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Outputs: []Bind{{Name: "out", Spec: Path{Name: "a/b", Ext: ".txt"}}},
				}},
			},
			code: DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "invalid-path dir escape",
			snap: Snapshot{
				Name: "escape",
				Tasks: []Task{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Outputs: []Bind{{Name: "out", Spec: Path{Dir: "../out", Name: "x", Ext: ".txt"}}},
				}},
			},
			code: DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "empty step token",
			snap: Snapshot{
				Name: "empty-step",
				Tasks: []Task{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Outputs: []Bind{{Name: "out", Spec: Path{Name: "sample", Steps: []string{""}, Ext: ".fastq"}}},
				}},
			},
			code: DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "foreign output From complete spec",
			snap: Snapshot{
				Name: "run-b",
				Tasks: []Task{{
					ID:      "use",
					Name:    "use",
					Command: []string{"use"},
					Outputs: []Bind{{
						Name:     "out",
						FromKind: FromOut,
						FromName: "clean",
						Spec:     Path{Dir: "out", Name: "use", Ext: ".txt"},
					}},
				}},
			},
			code: DefectMissingInput,
			unit: "use.out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComposeCheck(tt.snap)
			if !hasDefect(got, tt.code, tt.unit) {
				t.Fatalf("case %s: ComposeCheck() defects %v, want code %s unit %q", tt.name, formatDefects(got), tt.code, tt.unit)
			}
		})
	}
}

func TestValidateDirEscapeStaysInvalidPath(t *testing.T) {
	snap := Snapshot{
		Name: "escape",
		Tasks: []Task{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"cp"},
			Outputs: []Bind{{Name: "out", Spec: Path{Dir: "../out", Name: "x", Ext: ".txt"}}},
		}},
	}
	got := Validate(snap)
	if !hasDefect(got, DefectInvalidPath, "copy.out") {
		t.Fatalf("case dir-escape: Validate() defects %v, want code %s unit %q", formatDefects(got), DefectInvalidPath, "copy.out")
	}
	for _, d := range got {
		if d.Code == DefectConflict {
			t.Fatalf("case dir-escape: Validate() also reported conflict, want invalid-path only for escape")
		}
	}
}

func TestRestageClearsLiteralOpacity(t *testing.T) {
	snap := Snapshot{
		Name: "restage-literal",
		Tasks: []Task{
			{
				ID:      "src",
				Name:    "src",
				Command: []string{"echo"},
				Outputs: []Bind{{
					Name: "html",
					Spec: Path{Literal: true, Opaque: "sample.html", Dir: "work"},
				}},
			},
			{
				ID:      "copy",
				Name:    "copy",
				Command: []string{"cp"},
				Inputs: []Bind{{
					Name:     "in",
					FromKind: FromOut,
					FromTask: "src",
					FromName: "html",
				}},
				Outputs: []Bind{{
					Name:     "out",
					FromKind: FromOut,
					FromTask: "src",
					FromName: "html",
					Spec:     Path{Name: "a/b", Ext: ".txt"},
				}},
			},
		},
	}
	got := ComposeCheck(snap)
	if !hasDefect(got, DefectInvalidPath, "copy.out") {
		t.Fatalf("case restage-literal: ComposeCheck() defects %v, want invalid-path unit copy.out from author Name", formatDefects(got))
	}
}

func TestComposeCheckDoesNotReportPlanDefects(t *testing.T) {
	snap := Snapshot{
		Name: "plan-only",
		Tasks: []Task{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"cp"},
			Backend: "slurm",
			Outputs: []Bind{{Name: "out", Spec: Path{Name: "out", Ext: ".txt"}}},
		}},
	}
	got := ComposeCheck(snap)
	if hasDefect(got, DefectUnsupportedBackend, "copy") {
		t.Fatalf("case plan-only: ComposeCheck() reported unsupported-backend, want compose defects only")
	}
	got = Validate(snap)
	if !hasDefect(got, DefectUnsupportedBackend, "copy") {
		t.Fatalf("case plan-only: Validate() defects %v, want unsupported-backend unit copy", formatDefects(got))
	}

	nan := Snapshot{
		Name: "nan-cpu",
		Tasks: []Task{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"cp"},
			CPU:     math.NaN(),
			Outputs: []Bind{{Name: "out", Spec: Path{Name: "out", Ext: ".txt"}}},
		}},
	}
	got = ComposeCheck(nan)
	if hasDefect(got, DefectInvalidName, "copy") {
		t.Fatalf("case nan-cpu: ComposeCheck() reported invalid-name, want compose defects only")
	}
	got = Validate(nan)
	if !hasDefect(got, DefectInvalidName, "copy") {
		t.Fatalf("case nan-cpu: Validate() defects %v, want invalid-name unit copy", formatDefects(got))
	}
}

func TestBuildPlanEncodeFailureHasCode(t *testing.T) {
	snap := Snapshot{
		Name: "encode",
		Tasks: []Task{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"cp"},
			Outputs: []Bind{{Name: "out", Spec: Path{Name: "out", Ext: ".txt"}}},
		}},
	}
	doc := Document{
		Name: "encode",
		Tasks: []TaskPlan{{
			ID:        "copy",
			Name:      "copy",
			Command:   []string{"cp"},
			Resources: ResourcePlan{CPU: math.NaN()},
			Outputs:   []IO{{Name: "out", Path: "out.txt", Spec: Path{Name: "out", Ext: ".txt"}}},
		}},
	}
	plan, defects := BuildPlan(snap, doc)
	if plan != nil {
		t.Fatalf("case encode-fail: BuildPlan() plan != nil, want nil")
	}
	if len(defects) == 0 {
		t.Fatalf("case encode-fail: BuildPlan() defects empty, want DefectInvalidName")
	}
	if defects[0].Code != DefectInvalidName {
		t.Fatalf("case encode-fail: defect Code got %q, want %s", defects[0].Code, DefectInvalidName)
	}
	if defects[0].Message == "" {
		t.Fatalf("case encode-fail: defect Message empty")
	}
}

func hasDefect(ds []Defect, code, unit string) bool {
	for _, d := range ds {
		if d.Code == code && d.Unit == unit {
			return true
		}
	}
	return false
}

func formatDefects(ds []Defect) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Code + ":" + d.Unit
	}
	return out
}
