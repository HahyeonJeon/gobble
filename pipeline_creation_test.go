package gobble_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	trim "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
)

func TestCreationCompleteModuleGraph(t *testing.T) {
	for _, inputPath := range []string{"inputs/reads.fastq", "inputs/reads.fq", "inputs/reads.fastq.gz", "inputs/reads.fq.gz"} {
		for _, quality := range []int{0, 25, 30} {
			p := gobble.NewPipeline("read_quality")
			reads := p.AddInput("reads", gobble.Literal(inputPath))
			ports, err := trim.Add(modules.WithDisplay(p, gobble.TaskDisplay{Stage: "Trim reads"}), reads, gobble.Handle{}, trim.Options{Quality: quality, Length: 40})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = fastqc.Add(p, ports.Read1, fastqc.Options{}); err != nil {
				t.Fatal(err)
			}
			graph, err := gobble.Compose(p)
			if err != nil {
				t.Fatal(err)
			}
			got, err := gobble.InspectPipelineCreation(graph, inputPath)
			if err != nil || len(got.Gaps) != 0 {
				t.Fatalf("quality %d: %+v %v", quality, got, err)
			}
			hash := "sha256:" + strings.Repeat("a", 64)
			b := gobble.PreparationBinding{SourceRevision: hash, RuntimeID: hash, InputIdentity: hash, InputPath: inputPath, Cap: 1}
			sealed, e := gobble.PreparePipeline(graph, b)
			if e != nil {
				t.Fatal(e)
			}
			verified, e := gobble.InspectPreparedPipeline(sealed.Payload, sealed.Digest, b)
			if e != nil {
				t.Fatal(e)
			}
			want, _ := json.Marshal(got.Review.Flow)
			actual, _ := json.Marshal(verified.Flow)
			if string(want) != string(actual) {
				t.Fatalf("trusted review mismatch\n%s\n%s", want, actual)
			}
			p.AddInput("unused", gobble.Literal("other.fastq"))
			graph, err = gobble.Compose(p)
			if err != nil {
				t.Fatal(err)
			}
			got, err = gobble.InspectPipelineCreation(graph, inputPath)
			if _, e := gobble.PreparePipeline(graph, b); e == nil {
				t.Fatal("unrepresented graph input prepared")
			}
			if err == nil && len(got.Gaps) == 0 {
				t.Fatal("unrepresented input accepted")
			}
		}
	}
}
