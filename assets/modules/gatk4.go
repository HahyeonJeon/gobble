package modules

import "github.com/HahyeonJeon/gobble"

// GATK4Image is the Sarek 3.10.0 GATK image resolved for linux/amd64.
const GATK4Image Image = "community.wave.seqera.io/library/gatk4_gcnvkernel:edb12e4f0bf02cd3@sha256:ced519873646379e287bc28738bdf88e975edd39a92e7bc6a34bccd37153d9d0"

// ResolveGATK4Options applies the shared Sarek GATK image and rejects every
// unique long-option prefix and documented short alias that could take
// ownership from a named field.
func ResolveGATK4Options(unit string, options Options, defaults gobble.Resources, protected []string) ([]string, string, gobble.Resources, error) {
	protected = gatk4ProtectedOptions(protected)
	if err := RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return nil, "", gobble.Resources{}, err
	}
	return ResolveOptions(unit, options, GATK4Image, defaults, nil, protected)
}

func gatk4ProtectedOptions(protected []string) []string {
	aliases := map[string]string{
		"--input":               "-I",
		"--INPUT":               "-I",
		"--output":              "-O",
		"--OUTPUT":              "-O",
		"--reference":           "-R",
		"--intervals":           "-L",
		"--variant":             "-V",
		"--emit-ref-confidence": "-ERC",
		"--dbsnp":               "-D",
		"--bqsr-recal-file":     "-bqsr",
		"--METRICS_FILE":        "-M",
		"--SEQUENCE_DICTIONARY": "-D",
	}
	out := make([]string, 0, 2*len(protected))
	seen := make(map[string]bool, 2*len(protected))
	for _, option := range protected {
		for _, name := range []string{option, aliases[option]} {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}
