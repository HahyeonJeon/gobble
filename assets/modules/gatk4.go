package modules

import "github.com/HahyeonJeon/gobble"

// GATK4Image is the Sarek 3.10.0 GATK image resolved for linux/amd64.
const GATK4Image Image = "community.wave.seqera.io/library/gatk4_gcnvkernel:edb12e4f0bf02cd3@sha256:ced519873646379e287bc28738bdf88e975edd39a92e7bc6a34bccd37153d9d0"

// ResolveGATK4Options applies the shared Sarek GATK image and rejects every
// unique long-option prefix that could take ownership from a named field.
func ResolveGATK4Options(unit string, options Options, defaults gobble.Resources, protected []string) ([]string, string, gobble.Resources, error) {
	if err := RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return nil, "", gobble.Resources{}, err
	}
	return ResolveOptions(unit, options, GATK4Image, defaults, nil, protected)
}
