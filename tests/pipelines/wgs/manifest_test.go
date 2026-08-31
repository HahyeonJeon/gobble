package wgsevidence_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

type wgsManifest struct {
	Schema    int `json:"schema"`
	Benchmark struct {
		Pipeline          string            `json:"pipeline"`
		Release           string            `json:"release"`
		Commit            string            `json:"commit"`
		SelectedRoute     string            `json:"selected_route"`
		DatasetCommit     string            `json:"dataset_commit"`
		CoverageScenarios map[string]string `json:"coverage_scenarios"`
	} `json:"benchmark"`
	Entries    []wgsEntry          `json:"entries"`
	StageTrace map[string][]string `json:"stage_trace"`
	Images     []wgsImage          `json:"images"`
}

type wgsEntry struct {
	LogicalName      string   `json:"logical_name"`
	Name             string   `json:"name"`
	Role             string   `json:"role"`
	Repository       string   `json:"repository"`
	Commit           string   `json:"commit"`
	Path             string   `json:"path"`
	URL              string   `json:"url"`
	Bytes            int64    `json:"bytes"`
	SHA256           string   `json:"sha256"`
	LicenseAuthority string   `json:"license_authority"`
	Provenance       string   `json:"provenance"`
	AssayUse         []string `json:"assay_use"`
	Staged           bool     `json:"staged"`
}

type wgsImage struct {
	Reference    string   `json:"reference"`
	Digest       string   `json:"digest"`
	Modules      []string `json:"modules"`
	Tool         string   `json:"tool"`
	Version      string   `json:"version"`
	ModuleSource string   `json:"module_source"`
	License      string   `json:"license"`
	Platform     string   `json:"platform"`
}

func TestManifestIsExactPlanningByteAndImageAuthority(t *testing.T) {
	manifest := loadManifest(t)
	if manifest.Schema != 2 || manifest.Benchmark.Pipeline != "nf-core/sarek" || manifest.Benchmark.Release != "3.10.0" || manifest.Benchmark.Commit != "8ccac7ad37b05dd792447763bf9671b719824587" || manifest.Benchmark.DatasetCommit != "6c82958a6f302d8471a20855023ac59f9974fa8a" {
		t.Fatalf("benchmark = %+v, want exact Sarek and dataset commits", manifest.Benchmark)
	}
	if !strings.Contains(manifest.Benchmark.SelectedRoute, "without VQSR") || manifest.Benchmark.CoverageScenarios["F"] == "" || manifest.Benchmark.CoverageScenarios["J"] == "" {
		t.Fatalf("benchmark route/scenarios = %+v", manifest.Benchmark)
	}
	expected := expectedPlanningBytes()
	if len(manifest.Entries) != len(expected) {
		t.Fatalf("manifest entries = %d, want %d Planning-bound bytes", len(manifest.Entries), len(expected))
	}
	staged := make(map[string]wgsEntry)
	seen := make(map[string]bool, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		want, ok := expected[entry.LogicalName]
		if !ok || seen[entry.LogicalName] || entry.Name == "" || entry.Role == "" || entry.Repository == "" || entry.Path == "" || !strings.Contains(entry.URL, entry.Commit) || entry.Bytes != want.bytes || entry.SHA256 != want.sha256 || !lowerHex(entry.SHA256, 64) || entry.LicenseAuthority == "" || entry.Provenance == "" || len(entry.AssayUse) == 0 {
			t.Fatalf("invalid or substituted manifest entry: %+v", entry)
		}
		seen[entry.LogicalName] = true
		if entry.Staged {
			staged[entry.Name] = entry
		}
	}
	if len(staged) != 16 {
		t.Fatalf("staged entries = %d, want 16 official data bytes", len(staged))
	}
	for _, pin := range wgsevidence.MustPins() {
		entry, ok := staged[pin.Name]
		if !ok || pin.URL != entry.URL || pin.Bytes != entry.Bytes || pin.SHA256 != entry.SHA256 {
			t.Fatalf("typed pin %q differs from manifest authority", pin.Name)
		}
	}

	for _, stage := range []string{"reference_readiness", "raw_read_qc", "read_preparation", "alignment", "alignment_normalization", "duplicate_marking", "bqsr", "alignment_qc", "haplotypecaller", "sample_gvcf_gather", "genomicsdb", "joint_genotype", "joint_gather_and_qc"} {
		if len(manifest.StageTrace[stage]) == 0 {
			t.Errorf("stage trace omits %s", stage)
		}
	}

	imageSet := make(map[string]bool, len(manifest.Images))
	moduleSet := make(map[string]bool)
	for _, image := range manifest.Images {
		if image.Reference == "" || strings.Contains(image.Reference, "@") || !strings.HasPrefix(image.Digest, "sha256:") || !lowerHex(strings.TrimPrefix(image.Digest, "sha256:"), 64) || len(image.Modules) == 0 || image.Tool == "" || image.Version == "" || image.ModuleSource == "" || image.License == "" || image.Platform != "linux/amd64" {
			t.Fatalf("invalid image authority: %+v", image)
		}
		imageSet[image.Reference+"@"+image.Digest] = true
		for _, module := range image.Modules {
			moduleSet[module] = true
		}
	}
	for _, task := range pc.AllTasks(t, pc.MustPlanJSON(t, wgs.Build(loadSamples(t), wgs.DefaultConfig()))) {
		if task.Image != "" && !imageSet[task.Image] {
			t.Errorf("task %s image %q is absent from manifest", task.ID, task.Image)
		}
	}
	for _, module := range []string{"bwa-index", "bwa-mem", "fastp", "fastqc", "samtools-merge", "gatk4-haplotypecaller", "gatk4-genomicsdbimport", "gatk4-genotypegvcfs", "bcftools-sort", "bcftools-stats", "multiqc"} {
		if !moduleSet[module] {
			t.Errorf("image authority omits %s", module)
		}
	}
}

func TestFixtureSheetUsesOnlyManifestFASTQBytes(t *testing.T) {
	staged := make(map[string]bool)
	for _, pin := range wgsevidence.MustPins() {
		staged[pin.Name] = true
	}
	for _, sample := range loadSamples(t) {
		for _, lane := range sample.Lanes {
			for _, fastq := range []string{lane.Fastq1, lane.Fastq2} {
				if !staged[filepath.Base(fastq)] {
					t.Errorf("samplesheet FASTQ %q is absent from staged manifest bytes", fastq)
				}
			}
		}
	}
}

func TestJointMappedFixtureBytesReachCallingGraph(t *testing.T) {
	raw := pc.MustPlanJSON(t, wgsevidence.JointFixturePipeline())
	for _, test := range []struct {
		task string
		bam  string
		bai  string
	}{
		{task: "joint_fixture_haplotype_intervals.patient1.testN.gatk4_haplotypecaller", bam: "in/joint/testN/test.paired_end.sorted.bam", bai: "in/joint/testN/test.paired_end.sorted.bam.bai"},
		{task: "joint_fixture_haplotype_intervals.patient2.testT.gatk4_haplotypecaller", bam: "in/joint/testT/test2.paired_end.sorted.bam", bai: "in/joint/testT/test2.paired_end.sorted.bam.bai"},
	} {
		task := pc.TaskByID(t, raw, test.task)
		pc.AssertIOPath(t, task.Inputs, "bam", test.bam)
		pc.AssertIOPath(t, task.Inputs, "bai", test.bai)
	}
	pc.MustHaveTaskID(t, pc.AllTasks(t, raw), "joint_fixture_intervals.database.gatk4_genomicsdbimport")
	pc.MustHaveTaskID(t, pc.AllTasks(t, raw), "joint_fixture_intervals.genotype.gatk4_genotypegvcfs")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "joint_fixture_gather.joint.gatk4_mergevcfs").Outputs, "vcf", "evidence/wgs/joint/joint_germline.vcf.gz")
	pc.AssertNoTaskName(t, pc.AllTasks(t, raw), "VariantRecalibrator", "ApplyVQSR")
}

func TestJointMappedFixtureBytesAreStagedByLiveConsumers(t *testing.T) {
	for _, source := range []string{"../../local-e2e/helpers_test.go", "../../install-e2e/consumer_source_test.go"} {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", source, err)
		}
		for _, name := range []string{
			"test.paired_end.sorted.bam",
			"test.paired_end.sorted.bam.bai",
			"test2.paired_end.sorted.bam",
			"test2.paired_end.sorted.bam.bai",
		} {
			if !strings.Contains(string(data), name) {
				t.Errorf("%s does not stage %s", source, name)
			}
		}
	}
}

func loadManifest(t *testing.T) wgsManifest {
	t.Helper()
	data, err := os.ReadFile("testdata/manifest.json")
	if err != nil {
		t.Fatalf("ReadFile manifest: %v", err)
	}
	var manifest wgsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("Unmarshal manifest: %v", err)
	}
	return manifest
}

type byteIdentity struct {
	bytes  int64
	sha256 string
}

func expectedPlanningBytes() map[string]byteIdentity {
	return map[string]byteIdentity{
		"WG-FQ-TEST-R1":        {16013759, "bae22d1ab233d9a746d7f6413dbc99f3c4a24b278eba97286940b3460f4dac97"},
		"WG-FQ-TEST-R2":        {16698858, "d89ff7b7009e29c31f7722646dfec7fe6dd1ac90c2a82e631e59685065c9dddf"},
		"WG-FQ-TEST2-R1":       {16579912, "93b4d38834f6cd6c045f89a88a674618aa33c9ca52da1360bdea17c6f971cf8a"},
		"WG-FQ-TEST2-R2":       {17421020, "4c20267ace1585db77984a98308361a0565916444867ab8ab5e9956d3e09df49"},
		"WG-REF-FASTA":         {40675, "48d0bbb875d37e529640d43f2751ec2a25e0ba1144f1994773e9c643d3cf9d05"},
		"WG-REF-FAI":           {20, "aa684078e5bc8ec77af619fb6c3e6e9e51e143d997d84ab213b47eff844757d1"},
		"WG-REF-DICT":          {165, "06afa97cc34e3d5dada6919f8f6aa52490fb967717c7b1ebef46e1bc98b36a19"},
		"WG-INTERVALS":         {32, "074797ca478a33a5df610c11916ad446a73e4e5b3dfe38de0d21726825c37820"},
		"WG-DBSNP":             {54037, "25c4720a49c9cebc6e6de06a8d986511ff9499ab18f03609dd97a1073afe442f"},
		"WG-DBSNP-TBI":         {137, "ae1b7dcbec68408fdbdf80c7e079d0761af2d8309f9c00a07ebdde123f7d6847"},
		"WG-MILLS":             {4133, "85e739c69b759c8a04d878ec70bcfb1a824b5b9d2cd10546fd71b1e7d6293f82"},
		"WG-MILLS-TBI":         {132, "6df518da3e1069be23bc6b0c37c2b9585ceb11a31e3595b1e4509000961bdbb2"},
		"WG-JOINT-TEST-BAM":    {179552, "1ab90e4d913e6078c5c5234322b3db4e0de1f5a09e923c21891b1f9e5a37cc77"},
		"WG-JOINT-TEST-BAI":    {96, "c7cc6e280bca92fd713cde8d317ce221b1e2280b73dbfef0aa241bbd9aa7f673"},
		"WG-JOINT-TEST2-BAM":   {201184, "19bde1970bef46ef40c5b28e01c34205fde191423108d424fdd602f2820650a9"},
		"WG-JOINT-TEST2-BAI":   {96, "8bce3e466ae8c33dfc9670b828cbeb49692f99a2b460eb8d9ef92c912e8f2746"},
		"WG-SAREK-FASTQ-SHEET": {851, "4abf9a3f1a516eb3512086263524fde36e3377cc301ec1832d32b6ef7e686296"},
		"WG-SAREK-JOINT-SHEET": {592, "2e4ebd3c285624b237209113e72afbf69924cabd94838028a9b7c83ecc6dc264"},
		"WG-SAREK-JOINT-TEST":  {2602, "2ab3ba980ee7706d9f688221c6710236521969f0de14533b3336ad5ec52c4b10"},
		"WG-SAREK-TEST-CONFIG": {3657, "b3fc253cc872fb79a5c685093f24dae3f1a8f0213355b4ec2ea4dbbd72b2c92a"},
		"WG-SAREK-IGENOMES":    {28027, "e4a9a169342e0ed641c7c009626f723173b48a1a85a341059dd377f384a4643c"},
		"A1":                   {1064, "1fa2a62f20d23902c7aad04f3d728fbfc9d153df472e8f2ef163322a8c5557c8"},
		"A2":                   {30240, "35a2c50aad083968bc026b71381ae0c5a5fc0234d9aeaa0238e92a3fb8ea6c54"},
		"A3":                   {1074, "4b4d4dcfe4367910627dde3999ec28c001197f3d8167e5db14a6f1d0adbae48b"},
	}
}

func lowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
