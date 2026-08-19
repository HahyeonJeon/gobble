package gobble_test

import (
	"errors"
	"math"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestValidateAccept(t *testing.T) {
	g, err := gobble.Compose(workflowCasePipeline())
	if err != nil {
		t.Fatalf("case workflow-case: Compose() error = %v, want nil", err)
	}
	if err := gobble.Validate(g); err != nil {
		t.Fatalf("case workflow-case: Validate() error = %v, want nil", err)
	}

	local := oneTask("local-backend", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Backend: "local",
		Outputs: []gobble.Bind{fileOut("out")},
	})
	g, err = gobble.Compose(local)
	if err != nil {
		t.Fatalf("case local-backend: Compose() error = %v, want nil", err)
	}
	if err := gobble.Validate(g); err != nil {
		t.Fatalf("case local-backend: Validate() error = %v, want nil", err)
	}
}

func TestValidateReject(t *testing.T) {
	tests := []struct {
		name string
		pipe *gobble.Pipeline
		code gobble.DefectCode
		unit string
		path string
	}{
		{
			name: "derived-path conflict",
			pipe: derivedRelatedCollisionPipeline(),
			code: gobble.DefectConflict,
			unit: "collide.out",
			path: "aln.bam.bai",
		},
		{
			name: "derived index path equals another output",
			pipe: derivedIndexCollisionPipeline(),
			code: gobble.DefectConflict,
			unit: "collide.out",
			path: "aln.bam.bai",
		},
		{
			name: "same-task input/output conflict",
			pipe: sameTaskIOPipeline(),
			code: gobble.DefectConflict,
			unit: "copy.out",
			path: "sample.txt",
		},
		{
			name: "unsupported-backend",
			pipe: unsupportedBackendPipeline(),
			code: gobble.DefectUnsupportedBackend,
			unit: "copy",
		},
		{
			name: "NaN CPU",
			pipe: nanCPUPipeline(),
			code: gobble.DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "+Inf CPU",
			pipe: infCPUPipeline(),
			code: gobble.DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "-Inf CPU",
			pipe: negInfCPUPipeline(),
			code: gobble.DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "group member path conflict",
			pipe: groupMemberConflictPipeline(),
			code: gobble.DefectConflict,
			unit: "index.collide",
			path: "ref.amb",
		},
		{
			name: "negative CPU",
			pipe: negativeCPUPipeline(),
			code: gobble.DefectInvalidValue,
			unit: "copy",
		},
		{
			name: "unparseable Memory",
			pipe: junkMemoryPipeline(),
			code: gobble.DefectInvalidMemory,
			unit: "copy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := gobble.Compose(tt.pipe)
			if err != nil {
				t.Fatalf("case %s: Compose() error = %v, want nil", tt.name, err)
			}
			if g == nil {
				t.Fatalf("case %s: Compose() graph = nil, want compose-valid graph", tt.name)
			}
			err = gobble.Validate(g)
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: Validate() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "validate" {
				t.Fatalf("case %s: Error.Op got %q, want %q", tt.name, ge.Op, "validate")
			}
			found := false
			codes := make([]gobble.DefectCode, len(ge.Defects))
			units := make([]string, len(ge.Defects))
			for i, d := range ge.Defects {
				codes[i] = d.Code
				units[i] = d.Unit
				if d.Code != tt.code || d.Unit != tt.unit {
					continue
				}
				if tt.path != "" && !hasPath(d.Paths, tt.path) {
					t.Fatalf("case %s: Validate() defect Paths got %v, want %q", tt.name, d.Paths, tt.path)
				}
				found = true
			}
			if !found {
				t.Fatalf("case %s: Validate() defects codes got %v units %v, want code %s unit %q", tt.name, codes, units, tt.code, tt.unit)
			}
		})
	}
}

func derivedRelatedCollisionPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("related-collision")
	align := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Base: "aln", Ext: ".bam"}}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"samtools"},
		Inputs:  []gobble.Bind{{Name: "bam", From: align.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "bai",
			Spec: gobble.PathSpec{Ext: ".bai"},
			From: align.Out("bam"),
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "collide",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "aln", Ext: ".bam.bai"}}},
	})
	return p
}

func derivedIndexCollisionPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("index-collision")
	bam := gobble.PathSpec{Base: "aln", Ext: ".bam"}
	p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{
			{Name: "bam", Spec: bam},
			{Name: "bai", Spec: bam.AppendExt(".bai")},
		},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "collide",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "aln", Ext: ".bam.bai"}}},
	})
	return p
}

func sameTaskIOPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("same-io")
	in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Prefix: "sam", Base: "ple", Ext: ".txt"}}},
	})
	return p
}

func unsupportedBackendPipeline() *gobble.Pipeline {
	return oneTask("backend", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Backend: "slurm",
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func nanCPUPipeline() *gobble.Pipeline {
	return oneTask("nan-cpu", gobble.TaskSpec{
		Name:      "copy",
		Command:   []string{"cp"},
		Outputs:   []gobble.Bind{fileOut("out")},
		Resources: gobble.Resources{CPU: math.NaN()},
	})
}

func infCPUPipeline() *gobble.Pipeline {
	return oneTask("inf-cpu", gobble.TaskSpec{
		Name:      "copy",
		Command:   []string{"cp"},
		Outputs:   []gobble.Bind{fileOut("out")},
		Resources: gobble.Resources{CPU: math.Inf(1)},
	})
}

func negInfCPUPipeline() *gobble.Pipeline {
	return oneTask("neginf-cpu", gobble.TaskSpec{
		Name:      "copy",
		Command:   []string{"cp"},
		Outputs:   []gobble.Bind{fileOut("out")},
		Resources: gobble.Resources{CPU: math.Inf(-1)},
	})
}

func negativeCPUPipeline() *gobble.Pipeline {
	return oneTask("neg-cpu", gobble.TaskSpec{
		Name:      "copy",
		Command:   []string{"cp"},
		Outputs:   []gobble.Bind{fileOut("out")},
		Resources: gobble.Resources{CPU: -1},
	})
}

func junkMemoryPipeline() *gobble.Pipeline {
	return oneTask("junk-mem", gobble.TaskSpec{
		Name:      "copy",
		Command:   []string{"cp"},
		Outputs:   []gobble.Bind{fileOut("out")},
		Resources: gobble.Resources{Memory: "not-a-size"},
	})
}

func hasPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
