package engine

import (
	"math"
	"testing"
)

func TestValidateDocumentPlanDefects(t *testing.T) {
	file := IO{Name: "out", Kind: ArtifactFile, Path: "out.txt", Spec: Path{Base: "out", Ext: ".txt"}}
	tests := []struct {
		name string
		doc  Document
		code string
		unit string
	}{
		{
			name: "unsupported-backend",
			doc: Document{
				Name: "plan-only",
				Tasks: []TaskPlan{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Backend: "slurm",
					Outputs: []IO{file},
				}},
			},
			code: DefectUnsupportedBackend,
			unit: "copy",
		},
		{
			name: "NaN CPU",
			doc: Document{
				Name: "nan-cpu",
				Tasks: []TaskPlan{{
					ID:        "copy",
					Name:      "copy",
					Command:   []string{"cp"},
					Resources: ResourcePlan{CPU: math.NaN()},
					Outputs:   []IO{file},
				}},
			},
			code: DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "negative CPU",
			doc: Document{
				Name: "neg-cpu",
				Tasks: []TaskPlan{{
					ID:        "copy",
					Name:      "copy",
					Command:   []string{"cp"},
					Resources: ResourcePlan{CPU: -1},
					Outputs:   []IO{file},
				}},
			},
			code: DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "invalid memory",
			doc: Document{
				Name: "junk-mem",
				Tasks: []TaskPlan{{
					ID:        "copy",
					Name:      "copy",
					Command:   []string{"cp"},
					Resources: ResourcePlan{Memory: "not-a-size"},
					Outputs:   []IO{file},
				}},
			},
			code: DefectInvalidMemory,
			unit: "copy",
		},
		{
			name: "duplicate param names",
			doc: Document{
				Name: "dup-param",
				Tasks: []TaskPlan{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Params:  []ParamPlan{{Name: "mode", Value: "fast"}, {Name: "mode", Value: "slow"}},
					Outputs: []IO{file},
				}},
			},
			code: DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "empty env key",
			doc: Document{
				Name: "env",
				Tasks: []TaskPlan{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Env:     map[string]string{"": "x"},
					Outputs: []IO{file},
				}},
			},
			code: DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "group and spec both set",
			doc: Document{
				Name: "xor",
				Tasks: []TaskPlan{{
					ID:      "index",
					Name:    "index",
					Command: []string{"bwa"},
					Outputs: []IO{{
						Name:    "idx",
						Kind:    ArtifactFile,
						Path:    "ref.amb",
						Spec:    Path{Base: "ref", Ext: ".amb"},
						Members: []IOMember{{Name: "amb", Path: "ref.amb"}},
					}},
				}},
			},
			code: DefectInvalidValue,
			unit: "index.idx",
		},
		{
			name: "group and tree both set",
			doc: Document{
				Name: "xor-tree",
				Tasks: []TaskPlan{{
					ID:      "index",
					Name:    "index",
					Command: []string{"star"},
					Outputs: []IO{{
						Name:     "idx",
						Kind:     ArtifactTree,
						Path:     "work/idx",
						Manifest: "work/idx/.gobble-tree.json",
						Members:  []IOMember{{Name: "amb", Path: "ref.amb"}},
					}},
				}},
			},
			code: DefectInvalidValue,
			unit: "index.idx",
		},
		{
			name: "malformed image",
			doc: Document{
				Name: "image",
				Tasks: []TaskPlan{{
					ID:      "copy",
					Name:    "copy",
					Command: []string{"cp"},
					Image:   "--privileged",
					Outputs: []IO{file},
				}},
			},
			code: DefectInvalidValue,
			unit: "copy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.doc)
			if !hasDefect(got, tt.code, tt.unit) {
				t.Fatalf("case %s: Validate() defects %v, want code %s unit %q", tt.name, formatDefects(got), tt.code, tt.unit)
			}
		})
	}
}

func TestParseMemoryGrammar(t *testing.T) {
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"", 0, true},
		{"0", 0, true},
		{"0m", 0, true},
		{"512m", 512 << 20, true},
		{"1g", 1 << 30, true},
		{"1.5g", 1610612736, true},
		{"1K", 1024, true},
		{"not-a-size", 0, false},
		{"-1g", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseMemory(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Fatalf("parseMemory(%q) = %d, %v, want %d, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
	got := Validate(Document{
		Name: "mem-15g",
		Tasks: []TaskPlan{{
			ID:        "copy",
			Name:      "copy",
			Command:   []string{"cp"},
			Resources: ResourcePlan{Memory: "1.5g"},
			Outputs:   []IO{{Name: "out", Kind: ArtifactFile, Path: "out.txt", Spec: Path{Base: "out", Ext: ".txt"}}},
		}},
	})
	if hasDefect(got, DefectInvalidMemory, "copy") {
		t.Fatalf("1.5g: Validate() defects %v, want accepted", formatDefects(got))
	}
}

func TestValidateEmptyMemoryOK(t *testing.T) {
	got := Validate(Document{
		Name: "empty-mem",
		Tasks: []TaskPlan{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"cp"},
			Outputs: []IO{{Name: "out", Kind: ArtifactFile, Path: "out.txt", Spec: Path{Base: "out", Ext: ".txt"}}},
		}},
	})
	if hasDefect(got, DefectInvalidMemory, "copy") {
		t.Fatalf("empty-mem: Validate() defects %v, want no invalid-memory", formatDefects(got))
	}
}

func TestValidateConflict(t *testing.T) {
	got := Validate(Document{
		Name: "conflict",
		Tasks: []TaskPlan{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"cp"},
			Inputs:  []IO{{Name: "in", Kind: ArtifactFile, Path: "sample.txt"}},
			Outputs: []IO{{Name: "out", Kind: ArtifactFile, Path: "sample.txt"}},
		}},
	})
	if !hasDefect(got, DefectConflict, "copy.out") {
		t.Fatalf("conflict: Validate() defects %v, want conflict unit copy.out", formatDefects(got))
	}
}

func TestBuildPlanEncodeFailureHasCode(t *testing.T) {
	doc := Document{
		Name: "encode",
		Tasks: []TaskPlan{{
			ID:        "copy",
			Name:      "copy",
			Command:   []string{"cp"},
			Resources: ResourcePlan{CPU: math.NaN()},
			Outputs:   []IO{{Name: "out", Path: "out.txt", Spec: Path{Base: "out", Ext: ".txt"}}},
		}},
	}
	plan, defects := BuildPlan(doc)
	if plan != nil {
		t.Fatalf("case encode-fail: BuildPlan() plan != nil, want nil")
	}
	if len(defects) == 0 {
		t.Fatalf("case encode-fail: BuildPlan() defects empty, want DefectInvalidValue")
	}
	if defects[0].Code != DefectInvalidValue {
		t.Fatalf("case encode-fail: defect Code got %q, want %s", defects[0].Code, DefectInvalidValue)
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
