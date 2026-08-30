package moduleevidence

import (
	"reflect"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/modules"
)

func TestAppendExtraArgs(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		extra   []string
		want    []string
	}{
		{
			name:    "append after named flags",
			command: []string{"fastqc", "--quiet", "in/sample.fastq.gz"},
			extra:   []string{"--kmers", "7"},
			want:    []string{"fastqc", "--quiet", "in/sample.fastq.gz", "--kmers", "7"},
		},
		{
			name:    "empty extra",
			command: []string{"fastqc"},
			extra:   nil,
			want:    []string{"fastqc"},
		},
		{
			name:    "empty command",
			command: nil,
			extra:   []string{"--flag"},
			want:    []string{"--flag"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modules.AppendLegacyExtraArgs(tt.command, tt.extra)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("AppendExtraArgs() = %#v, want %#v", got, tt.want)
			}
			if len(tt.command) > 0 && len(got) > 0 && len(tt.extra) > 0 {
				got[0] = "mutated"
				if tt.command[0] == "mutated" {
					t.Fatalf("AppendExtraArgs() aliased command")
				}
			}
		})
	}
}
