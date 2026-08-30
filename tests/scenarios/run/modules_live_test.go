//go:build live

package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	bismarkmethylationextractor "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	staralign "github.com/HahyeonJeon/gobble/assets/modules/star-align"
	stargenomegenerate "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	bismarkevidence "github.com/HahyeonJeon/gobble/tests/modules/bismark-genome"
	fastpevidence "github.com/HahyeonJeon/gobble/tests/modules/fastp"
	fastqcevidence "github.com/HahyeonJeon/gobble/tests/modules/fastqc"
	multiqcevidence "github.com/HahyeonJeon/gobble/tests/modules/multiqc"
	methylseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/methylseq"
	rnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/rnaseq"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

func TestFastQCStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, fastqcevidence.CacheDir, fastqcevidence.SARSCoV2R1)
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/test_1.fastq.gz", src)
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	p := fastqc.FastQCPipeline(reads, fastqc.FastQCOptions{
		ExtraArgs: []string{"--quiet"},
		Resources: gobble.Resources{CPU: 1},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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
	src := cachePin(t, fastqcevidence.CacheDir, fastqcevidence.SARSCoV2R1)
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/test_1.fastq.gz", src)
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	opts := fastqc.FastQCOptions{ExtraArgs: []string{"--quiet"}, Resources: gobble.Resources{CPU: 1}}
	g, err := gobble.Compose(fastqc.FastQCPipeline(reads, opts))
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
	}
	if err := gobble.Release(dir); err != nil {
		fatalAPIError(t, "Release()", err)
	}
	opts.ExtraArgs = []string{"--quiet", "--kmers", "7"}
	g2, err := gobble.Compose(fastqc.FastQCPipeline(reads, opts))
	if err != nil {
		t.Fatalf("Compose(changed extra-args) error = %v", err)
	}
	if err := gobble.Resume(t.Context(), g2, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Resume()", err)
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
	src := cachePin(t, wgsevidence.CacheDir, wgsevidence.MustPins()[2])
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/genome.fasta", src)
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	p := bwaindex.BWAIndexPipeline(fasta, bwaindex.BWAIndexOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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
	src1 := cachePin(t, fastqcevidence.CacheDir, fastqcevidence.SARSCoV2R1)
	src2 := cachePin(t, fastpevidence.CacheDir, fastpevidence.SARSCoV2R2)
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/test_1.fastq.gz", src1)
	pc.StageFile(t, dir, "in/test_2.fastq.gz", src2)
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := fastp.FastpPipeline(r1, r2, fastp.FastpOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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
	src := cachePin(t, multiqcevidence.CacheDir, multiqcevidence.SARSCoV2FastQCZip)
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/test_fastqc.zip", src)
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	p := multiqc.MultiQCPipeline([]gobble.PathSpec{report}, multiqc.MultiQCOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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
	srcFASTA := cachePin(t, rnaseqevidence.CacheDir, rnaseqevidence.GenomeFASTA)
	srcGTF := cachePin(t, rnaseqevidence.CacheDir, rnaseqevidence.GTF)
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/genome.fasta", srcFASTA)
	pc.StageFile(t, dir, "in/genes.gtf", srcGTF)
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	p := stargenomegenerate.STARGenomeGeneratePipeline(fasta, stargenomegenerate.STARGenomeGenerateOptions{
		GTF:       gtf,
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 1},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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
	srcFASTA := cachePin(t, rnaseqevidence.CacheDir, rnaseqevidence.GenomeFASTA)
	srcGTF := cachePin(t, rnaseqevidence.CacheDir, rnaseqevidence.GTF)
	srcR1 := cachePin(t, rnaseqevidence.CacheDir, rnaseqevidence.Test1FASTQ)
	srcR2 := cachePin(t, rnaseqevidence.CacheDir, rnaseqevidence.Test2FASTQ)
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/genome.fasta", srcFASTA)
	pc.StageFile(t, dir, "in/genes.gtf", srcGTF)
	pc.StageFile(t, dir, "in/SRR6357072_1.fastq.gz", srcR1)
	pc.StageFile(t, dir, "in/SRR6357072_2.fastq.gz", srcR2)
	p := gobble.NewPipeline("rna")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"})
	hg := p.AddInput("gtf", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "SRR6357072_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "SRR6357072_2", Ext: ".fastq.gz"})
	idx := stargenomegenerate.AddSTARGenomeGenerate(p, hf, hg, stargenomegenerate.STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 1},
	})
	ports := staralign.AddSTARAlign(p, idx.Index, h1, h2, staralign.STARAlignOptions{
		Resources: gobble.Resources{CPU: 1},
	})
	if ports.LogFinalOut.IsZero() {
		t.Fatalf("ports.LogFinalOut IsZero = true, want false")
	}
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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

func TestBismarkGenomeStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, methylseqevidence.CacheDir, methylseqevidence.GenomeFASTA)
	dir := t.TempDir()
	pc.StageFile(t, dir, "in/genome.fa", src)
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"}
	p := bismarkgenome.BismarkGenomePipeline(fasta, bismarkgenome.BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into in/: %v", err)
	}
	for _, rel := range bismarkevidence.PublishedPaths("work/bismark-genome") {
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
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "Ecoli_10K_methylated_R1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "Ecoli_10K_methylated_R2", Ext: ".fastq.gz"})
	idx := bismarkgenome.AddBismarkGenome(p, hf, bismarkgenome.BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	bismarkalign.AddBismarkAlign(p, hf, idx.Index, h1, h2, bismarkalign.BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "Ecoli_10K_methylated_R1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "Ecoli_10K_methylated_R2", Ext: ".fastq.gz"})
	idx := bismarkgenome.AddBismarkGenome(p, hf, bismarkgenome.BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	aln := bismarkalign.AddBismarkAlign(p, hf, idx.Index, h1, h2, bismarkalign.BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	bismarkmethylationextractor.AddBismarkMethylationExtractor(p, aln.BAM, bismarkmethylationextractor.BismarkMethylationExtractorOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run()", err)
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
