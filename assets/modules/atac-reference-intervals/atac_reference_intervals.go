// Package atacreferenceintervals owns one Python reference-interval projection command.
package atacreferenceintervals

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Python image selected by nf-core/atacseq 2.1.2 reference utilities.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.8.3@sha256:4965e8f9078ba50c7148d49dcbc41c1827f21cb74329013deeca366204f0e317"

const projectionScript = `
import re, sys
fai, gtf, sizes_out, genes_out, tss_out, autosomes_out, mito, read_length = sys.argv[1:]
read_length = int(read_length)
contigs = []
with open(fai) as src, open(sizes_out, "w") as sizes:
    for line in src:
        fields = line.rstrip("\n").split("\t")
        if len(fields) < 2:
            continue
        contigs.append(fields[0])
        sizes.write(fields[0] + "\t" + fields[1] + "\n")
with open(autosomes_out, "w") as out:
    for name in contigs:
        if name != mito:
            out.write(name + "\n")
with open(gtf) as src, open(genes_out, "w") as genes, open(tss_out, "w") as tss:
    for line in src:
        if not line or line.startswith("#"):
            continue
        fields = line.rstrip("\n").split("\t")
        if len(fields) != 9 or fields[2] != "gene":
            continue
        start, end, strand = int(fields[3]) - 1, int(fields[4]), fields[6]
        match = re.search(r'gene_id "([^"]+)"', fields[8])
        gene = match.group(1) if match else "gene"
        genes.write("\t".join((fields[0], str(start), str(end), gene, "0", strand)) + "\n")
        point = start if strand == "+" else end - 1
        tss.write("\t".join((fields[0], str(max(0, point - read_length)), str(point + read_length + 1), gene, "0", strand)) + "\n")
`

// Options controls the reference projection.
type Options struct {
	modules.Options
	OutDir     gobble.Directory
	MitoName   string
	ReadLength int
}

// Ports contains chromosome sizes, genes, TSSs, and autosomal contigs.
type Ports struct {
	ChromSizes gobble.Handle
	Genes      gobble.Handle
	TSS        gobble.Handle
	Autosomes  gobble.Handle
}

// Add records one deterministic reference projection command.
func Add(parent modules.Parent, fai, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "atac_reference_intervals"
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "reference projection does not accept ExtraArgs")
	}
	if options.MitoName == "" || options.ReadLength < 1 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "mitochondrial contig name and positive read length are required")
	}
	faiPath, err := modules.HandlePath(unit, fai)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/reference/atac")
	}
	sizes := gobble.PathSpec{Dir: outDir, Base: "chrom", Ext: ".sizes"}
	genes := gobble.PathSpec{Dir: outDir, Base: "genes", Ext: ".bed"}
	tss := gobble.PathSpec{Dir: outDir, Base: "tss", Ext: ".bed"}
	autosomes := gobble.PathSpec{Dir: outDir, Base: "autosomes", Ext: ".txt"}
	sizesPath, err := sizes.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "reference output path is invalid")
	}
	genesPath, _ := genes.Render()
	tssPath, _ := tss.Render()
	autosomesPath, _ := autosomes.Render()
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"python3", "-c", projectionScript, faiPath, gtfPath, sizesPath, genesPath, tssPath, autosomesPath, options.MitoName, strconv.Itoa(options.ReadLength)}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "fai", From: fai}, {Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "chrom_sizes", Spec: sizes}, {Name: "genes", Spec: genes}, {Name: "tss", Spec: tss}, {Name: "autosomes", Spec: autosomes}}})
	return Ports{ChromSizes: task.Out("chrom_sizes"), Genes: task.Out("genes"), TSS: task.Out("tss"), Autosomes: task.Out("autosomes")}, nil
}
