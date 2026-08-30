package featurecounts

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 Subread image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/subread:2.0.6--he4a0461_2@sha256:114390a783c77f7739d86e474bedfa5a4e65309a2f71d4db430803fb04601f5d"

// BiotypeOptions controls featureCounts when it is used only for RNA biotype
// quality control. It is not an abundance or differential-analysis surface.
type BiotypeOptions struct {
	modules.Options
	OutDir       gobble.Directory
	Strandedness string
	Paired       bool
}

// BiotypePorts are the biotype QC table and command summary.
type BiotypePorts struct {
	Counts  gobble.Handle
	Summary gobble.Handle
}

// AddBiotype records one validated featureCounts biotype-QC command.
func AddBiotype(parent modules.Parent, bam, gtf gobble.Handle, options BiotypeOptions) (BiotypePorts, error) {
	return addBiotype(parent, bam, gtf, gobble.Handle{}, options)
}

// AddBiotypeInferred records biotype QC with its -s value selected from a
// completed strandedness inference task.
func AddBiotypeInferred(parent modules.Parent, bam, gtf, strandedness gobble.Handle, options BiotypeOptions) (BiotypePorts, error) {
	return addBiotype(parent, bam, gtf, strandedness, options)
}

func addBiotype(parent modules.Parent, bam, gtf, strandedness gobble.Handle, options BiotypeOptions) (BiotypePorts, error) {
	const unit = "featurecounts_biotype_qc"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return BiotypePorts{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return BiotypePorts{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/featurecounts-biotype")
	}
	counts := gobble.PathSpec{Dir: outDir, Base: "biotype", Ext: ".featureCounts.tsv"}
	summary := counts.AppendExt(".summary")
	countsPath, _ := counts.Render()
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	commandFor := func(strand string) ([]string, string, gobble.Resources, error) {
		if strand == "" {
			strand = gobble.StrandednessUnstranded
		}
		if strand != gobble.StrandednessUnstranded && strand != gobble.StrandednessForward && strand != gobble.StrandednessReverse {
			return nil, "", gobble.Resources{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "strandedness must be unstranded, forward, or reverse")
		}
		command := []string{"featureCounts", "-a", gtfPath, "-o", countsPath, "-s", featureCountsStrand(strand), "-T", strconv.Itoa(modules.ThreadCount(resources.CPU))}
		if options.Paired {
			command = append(command, "-p")
		}
		command = append(command, bamPath)
		base := options.Options
		base.Resources = resources
		return modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-a", "-o", "-s", "-T", "-p"})
	}
	spec := gobble.TaskSpec{Name: unit, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "counts", Spec: counts}, {Name: "summary", Spec: summary}}}
	if strandedness.IsZero() {
		command, image, resolvedResources, commandErr := commandFor(options.Strandedness)
		if commandErr != nil {
			return BiotypePorts{}, commandErr
		}
		spec.Command, spec.Image, spec.Resources = command, image, resolvedResources
	} else {
		strandPath, pathErr := modules.HandlePath(unit, strandedness)
		if pathErr != nil {
			return BiotypePorts{}, pathErr
		}
		commands := make(map[string][]string, 3)
		for _, strand := range []string{gobble.StrandednessUnstranded, gobble.StrandednessForward, gobble.StrandednessReverse} {
			command, image, resolvedResources, commandErr := commandFor(strand)
			if commandErr != nil {
				return BiotypePorts{}, commandErr
			}
			commands[strand] = command
			spec.Image, spec.Resources = image, resolvedResources
		}
		spec.Script = modules.StrandedCommand(strandPath, commands[gobble.StrandednessUnstranded], commands[gobble.StrandednessForward], commands[gobble.StrandednessReverse])
		spec.Inputs = append(spec.Inputs, gobble.Bind{Name: "strandedness", From: strandedness})
	}
	task := parent.AddTask(spec)
	return BiotypePorts{Counts: task.Out("counts"), Summary: task.Out("summary")}, nil
}

// BiotypePipeline returns a standalone validated featureCounts biotype-QC
// module.
func BiotypePipeline(bam, gtf gobble.PathSpec, options BiotypeOptions) *gobble.Pipeline {
	return modules.StandaloneChecked("featurecounts-biotype-qc", []modules.Input{{Name: "bam", Spec: bam}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := AddBiotype(parent, handles[0], handles[1], options)
		return err
	})
}
