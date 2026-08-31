//go:build live

package local_e2e_test

import (
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

func TestWGSJointMappedFixtureReachesUnfilteredCallset(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageWGSPins(t, dir)
	graph, err := gobble.Compose(wgsevidence.JointFixturePipeline())
	if err != nil {
		t.Fatalf("Compose(JointFixturePipeline()) error = %v", err)
	}
	if err := gobble.Run(t.Context(), graph, dir, 2, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run(JointFixturePipeline())", err)
	}
	requireRegularFile(t, filepath.Join(dir, "evidence", "wgs", "joint", "joint_germline.vcf.gz"))
}
