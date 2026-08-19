//go:build live

package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestFastQCStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, pinSARSCoV2R1)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_1.fastq.gz", src)
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	p := FastQCPipeline(reads, FastQCOptions{
		ExtraArgs: []string{"--quiet"},
		Resources: gobble.Resources{CPU: 1},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{"work/fastqc/test_1_fastqc.html", "work/fastqc/test_1_fastqc.zip"} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func TestFastQCExtraArgsResume(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, pinSARSCoV2R1)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_1.fastq.gz", src)
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	opts := FastQCOptions{ExtraArgs: []string{"--quiet"}, Resources: gobble.Resources{CPU: 1}}
	g, err := gobble.Compose(FastQCPipeline(reads, opts))
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	forceDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	opts.ExtraArgs = []string{"--quiet", "--kmers", "7"}
	g2, err := gobble.Compose(FastQCPipeline(reads, opts))
	if err != nil {
		t.Fatalf("Compose(changed extra-args) error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g2, dir, 1); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	instances := inspectJSONL(t, dir, "instances")
	if len(instances) == 0 {
		t.Fatalf("Inspect(instances) empty")
	}
	if instances[0]["reuse_reason"] != "command-or-script-changed" {
		t.Fatalf("reuse_reason = %#v, want command-or-script-changed", instances[0]["reuse_reason"])
	}
}

func TestBWAIndexStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, PinWGSGenomeFASTA)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", src)
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	p := BWAIndexPipeline(fasta, BWAIndexOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{
		"in/genome.fasta.amb",
		"in/genome.fasta.ann",
		"in/genome.fasta.bwt",
		"in/genome.fasta.pac",
		"in/genome.fasta.sa",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func TestFastpStandaloneRun(t *testing.T) {
	requireDocker(t)
	src1 := cachePin(t, pinSARSCoV2R1)
	src2 := cachePin(t, pinSARSCoV2R2)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_1.fastq.gz", src1)
	stageFile(t, dir, "in/test_2.fastq.gz", src2)
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := FastpPipeline(r1, r2, FastpOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{
		"work/fastp/test_1.clean.fastq.gz",
		"work/fastp/test_2.clean.fastq.gz",
		"work/fastp/test_1.fastp.json",
		"work/fastp/test_1.fastp.html",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func TestMultiQCStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, pinSARSCoV2FastQCZip)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_fastqc.zip", src)
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	p := MultiQCPipeline([]gobble.PathSpec{report}, MultiQCOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{"work/multiqc/multiqc_report.html", "work/multiqc/multiqc_data.zip"} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func TestSTARGenomeGenerateStandaloneRun(t *testing.T) {
	requireDocker(t)
	srcFASTA := cachePin(t, PinRNAGenomeFASTA)
	srcGTF := cachePin(t, PinRNAGTF)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", srcFASTA)
	stageFile(t, dir, "in/genes.gtf", srcGTF)
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	p := STARGenomeGeneratePipeline(fasta, STARGenomeGenerateOptions{
		GTF:       gtf,
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 1},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := starGenomePublishedPaths("work/star-genome", true)
	for _, rel := range want {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
	got := listRegularRel(t, filepath.Join(dir, filepath.FromSlash("work/star-genome")), "work/star-genome")
	want = append(want, "work/star-genome/.gobble-tree.json")
	if !sameStringSet(got, want) {
		t.Fatalf("genome dir regular files = %v, want %v", got, want)
	}
}

func TestSTARAlignNestedRun(t *testing.T) {
	requireDocker(t)
	srcFASTA := cachePin(t, PinRNAGenomeFASTA)
	srcGTF := cachePin(t, PinRNAGTF)
	srcR1 := cachePin(t, PinRNATest1FASTQ)
	srcR2 := cachePin(t, PinRNATest2FASTQ)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", srcFASTA)
	stageFile(t, dir, "in/genes.gtf", srcGTF)
	stageFile(t, dir, "in/SRR6357072_1.fastq.gz", srcR1)
	stageFile(t, dir, "in/SRR6357072_2.fastq.gz", srcR2)
	p := gobble.NewPipeline("rna")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"})
	hg := p.AddInput("gtf", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "SRR6357072_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "SRR6357072_2", Ext: ".fastq.gz"})
	idx := AddSTARGenomeGenerate(p, hf, hg, STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 1},
	})
	ports := AddSTARAlign(p, idx.Index, h1, h2, STARAlignOptions{
		Resources: gobble.Resources{CPU: 1},
	})
	if ports.LogFinalOut.IsZero() {
		t.Fatalf("ports.LogFinalOut IsZero = true, want false")
	}
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	bam := filepath.Join(dir, filepath.FromSlash("work/star-align/Aligned.out.bam"))
	info, err := os.Stat(bam)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published BAM: %v", err)
	}
	logPath := filepath.Join(dir, filepath.FromSlash("work/star-align/Log.final.out"))
	assertUniquelyMappedAbove(t, logPath, 10)
	assertSplicesRecorded(t, logPath)
}

func TestRNASeqRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", cachePin(t, PinRNAGenomeFASTA))
	stageFile(t, dir, "in/genes.gtf", cachePin(t, PinRNAGTF))
	stageFile(t, dir, "in/SRR6357072_1.fastq.gz", cachePin(t, PinRNATest1FASTQ))
	stageFile(t, dir, "in/SRR6357072_2.fastq.gz", cachePin(t, PinRNATest2FASTQ))
	g, err := gobble.Compose(RNASeq())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 2); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	logPath := filepath.Join(dir, filepath.FromSlash("work/star-align/Log.final.out"))
	assertUniquelyMappedAbove(t, logPath, 10)
	assertSplicesRecorded(t, logPath)
	for _, rel := range []string{
		"work/star-align/Aligned.out.bam",
		"work/multiqc/multiqc_report.html",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func TestBismarkGenomeStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, PinMethylGenomeFASTA)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fa", src)
	fasta := pinnedMethylFASTA()
	p := BismarkGenomePipeline(fasta, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into in/: %v", err)
	}
	for _, rel := range bismarkGenomePublishedPaths("work/bismark-genome") {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func TestBismarkAlignNestedRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	p := gobble.NewPipeline("methyl")
	hf := p.AddInput("fasta", pinnedMethylFASTA())
	h1 := p.AddInput("r1", pinnedMethylFASTQ1())
	h2 := p.AddInput("r2", pinnedMethylFASTQ2())
	idx := AddBismarkGenome(p, hf, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_pe.bam")))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published BAM: %v", err)
	}
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_PE_report.txt")))
	t.Logf("unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
}

func TestBismarkMethylationExtractorNestedRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	p := gobble.NewPipeline("methyl")
	hf := p.AddInput("fasta", pinnedMethylFASTA())
	h1 := p.AddInput("r1", pinnedMethylFASTQ1())
	h2 := p.AddInput("r2", pinnedMethylFASTQ2())
	idx := AddBismarkGenome(p, hf, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	aln := AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	AddBismarkMethylationExtractor(p, aln.BAM, BismarkMethylationExtractorOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{
		"work/bismark-extractor/aligned_pe.bedGraph.gz",
		"work/bismark-extractor/aligned_pe.bismark.cov.gz",
		"work/bismark-extractor/aligned_pe_splitting_report.txt",
		"work/bismark-extractor/aligned_pe.M-bias.txt",
		"work/bismark-extractor/CpG_context_aligned_pe.txt.gz",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_PE_report.txt")))
	t.Logf("unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
	assertMethylationCallRows(t, unique,
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/CpG_context_aligned_pe.txt.gz")),
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/aligned_pe.bismark.cov.gz")),
	)
}

func TestMethylSeqRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	g, err := gobble.Compose(MethylSeq())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into in/: %v", err)
	}
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_PE_report.txt")))
	t.Logf("unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
	assertMethylationCallRows(t, unique,
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/CpG_context_aligned_pe.txt.gz")),
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/aligned_pe.bismark.cov.gz")),
	)
}
