// Package atacconsensuspeaks owns one Python consensus-peak command.
package atacconsensuspeaks

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 consensus image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/mulled-v2-2f48cc59b03027e31ead6d383fe1b8057785dd24:5d182f583f4696f4c4d9f3be93052811b383341f-0@sha256:2bb8995555bc3505389b8534063f1743c9f31dd3055a3b559b8c4f077a9add53"

const consensusScript = `
import sys
bed_out, saf_out, presence_out, minimum, *paths = sys.argv[1:]
minimum = int(minimum)
records = []
for source, path in enumerate(paths):
    with open(path) as handle:
        for line in handle:
            if not line.strip() or line.startswith("#"):
                continue
            fields = line.rstrip("\n").split("\t")
            if len(fields) >= 3:
                records.append((fields[0], int(fields[1]), int(fields[2]), source))
records.sort()
merged = []
for chrom, start, end, source in records:
    if not merged or merged[-1][0] != chrom or start > merged[-1][2]:
        merged.append([chrom, start, end, {source}])
    else:
        merged[-1][2] = max(merged[-1][2], end)
        merged[-1][3].add(source)
accepted = [record for record in merged if len(record[3]) >= minimum]
if not accepted:
    raise SystemExit("no consensus peaks met the typed membership threshold")
with open(bed_out, "w") as bed, open(saf_out, "w") as saf, open(presence_out, "w") as presence:
    saf.write("GeneID\tChr\tStart\tEnd\tStrand\n")
    presence.write("peak_id\t" + "\t".join("member_%d" % (i + 1) for i in range(len(paths))) + "\n")
    for index, (chrom, start, end, sources) in enumerate(accepted, 1):
        peak = "peak_%06d" % index
        bed.write("%s\t%d\t%d\t%s\n" % (chrom, start, end, peak))
        saf.write("%s\t%s\t%d\t%d\t+\n" % (peak, chrom, start + 1, end))
        presence.write(peak + "\t" + "\t".join("1" if i in sources else "0" for i in range(len(paths))) + "\n")
`

// Options controls one strict consensus operation.
type Options struct {
	modules.Options
	OutDir  gobble.Directory
	Prefix  string
	Minimum int
}

// Ports contains consensus BED, SAF, and member-presence outputs.
type Ports struct {
	BED      gobble.Handle
	SAF      gobble.Handle
	Presence gobble.Handle
}

// Add records one strict fan-in over every declared peak set.
func Add(parent modules.Parent, peaks []gobble.Handle, options Options) (Ports, error) {
	const unit = "atac_consensus_peaks"
	if len(peaks) < 2 || options.Minimum < 1 || options.Minimum > len(peaks) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "consensus requires at least two peak sets and a reachable threshold")
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "consensus command does not accept ExtraArgs")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/consensus")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "consensus"
	}
	bed := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bed"}
	saf := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".saf"}
	presence := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".presence.tsv"}
	bedPath, err := bed.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "consensus output path is invalid")
	}
	safPath, _ := saf.Render()
	presencePath, _ := presence.Render()
	command := []string{"python3", "-c", consensusScript, bedPath, safPath, presencePath, strconv.Itoa(options.Minimum)}
	inputs := make([]gobble.Bind, len(peaks))
	for i, peak := range peaks {
		peakPath, pathErr := modules.HandlePath(unit, peak)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, peakPath)
		inputs[i] = gobble.Bind{Name: "peaks_" + strconv.Itoa(i), From: peak}
	}
	base := options.Options
	base.ExtraArgs = nil
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "bed", Spec: bed}, {Name: "saf", Spec: saf}, {Name: "presence", Spec: presence}}})
	return Ports{BED: task.Out("bed"), SAF: task.Out("saf"), Presence: task.Out("presence")}, nil
}
